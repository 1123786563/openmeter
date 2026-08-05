package order

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/oklog/ulid/v2"

	"github.com/openmeterio/openmeter/openmeter/commerce"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
)

// Repository is the persistence interface for orders.
type Repository interface {
	CreateOrder(ctx context.Context, order commerce.Order) (*commerce.Order, bool, error)
	GetOrder(ctx context.Context, namespace, id string) (*commerce.Order, error)
	GetOrderByIdempotencyKey(ctx context.Context, namespace, customerID, key string) (*commerce.Order, error)
	UpdateOrderStatus(ctx context.Context, namespace, id string, expectedFrom, to commerce.OrderStatus) (*commerce.Order, error)
	ListOrdersByCustomer(ctx context.Context, namespace, customerID string, status *commerce.OrderStatus) ([]commerce.Order, error)
}

// ProductLookup resolves product IDs to Product snapshots at order creation time.
type ProductLookup interface {
	GetProduct(ctx context.Context, namespace, id string) (*commerce.Product, error)
}

// Config wires the order service.
type Config struct {
	Repo     Repository
	Products ProductLookup
	Logger   *slog.Logger
}

// Service is the order lifecycle interface.
type Service interface {
	CreateOrder(ctx context.Context, in commerce.CreateOrderInput) (*commerce.Order, bool, error)
	GetOrder(ctx context.Context, namespace, id string) (*commerce.Order, error)
	TransitionStatus(ctx context.Context, namespace, id string, to commerce.OrderStatus) (*commerce.Order, error)
}

type service struct {
	repo     Repository
	products ProductLookup
	logger   *slog.Logger
}

// New creates an order Service from the given Config.
func New(cfg Config) Service {
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}
	return &service{repo: cfg.Repo, products: cfg.Products, logger: logger}
}

// CreateOrder implements idempotent order creation. If an order with the same
// idempotency key already exists, it is returned without modification. The
// boolean return is true for a fresh insert and false for an idempotent replay.
//
// Each order line snapshots the product's immutable attributes at creation time
// so later catalog edits never affect existing orders.
func (s *service) CreateOrder(ctx context.Context, in commerce.CreateOrderInput) (*commerce.Order, bool, error) {
	if err := validateCreateInput(in); err != nil {
		return nil, false, err
	}

	// Idempotency: if an order with this key exists, return it.
	existing, err := s.repo.GetOrderByIdempotencyKey(ctx, in.Namespace, in.CustomerID, in.IdempotencyKey)
	if err == nil && existing != nil {
		return existing, false, nil
	}

	// Resolve products and build immutable line snapshots.
	lines := make([]commerce.OrderLineSnapshot, 0, len(in.ProductIDs))
	totalMinor := int64(0)
	totalCredits := int64(0)

	for _, pid := range in.ProductIDs {
		product, err := s.products.GetProduct(ctx, in.Namespace, pid)
		if err != nil {
			return nil, false, fmt.Errorf("order: resolve product %s: %w", pid, err)
		}

		line := snapshotProduct(*product)
		lines = append(lines, line)
		totalMinor += line.SubtotalMinor
		totalCredits += line.Credits
	}

	now := clock.Now()
	orderID := ulid.Make().String()
	order := commerce.Order{
		NamespacedID: models.NamespacedID{
			Namespace: in.Namespace,
			ID:        orderID,
		},
		ManagedModel: models.ManagedModel{
			CreatedAt: now,
			UpdatedAt: now,
		},
		PublicID:       orderID,
		CustomerID:     in.CustomerID,
		Kind:           in.Kind,
		Status:         commerce.OrderStatusCreated,
		AmountMinor:    totalMinor,
		Currency:       in.Currency,
		IdempotencyKey: in.IdempotencyKey,
		Lines:          lines,
	}

	if in.Description != "" {
		// Description is stored on the order header via the repo; we keep it
		// in the first line's metadata for the domain model.
		if len(order.Lines) > 0 && order.Lines[0].Metadata == nil {
			order.Lines[0].Metadata = make(map[string]string)
		}
		if len(order.Lines) > 0 {
			order.Lines[0].Metadata["order_description"] = in.Description
		}
	}

	return s.repo.CreateOrder(ctx, order)
}

// GetOrder retrieves an order by namespace and ID.
func (s *service) GetOrder(ctx context.Context, namespace, id string) (*commerce.Order, error) {
	if id == "" {
		return nil, fmt.Errorf("order id is required")
	}
	return s.repo.GetOrder(ctx, namespace, id)
}

// TransitionStatus applies a state-machine-validated status transition. If the
// transition is invalid, it returns commerce.ErrInvalidOrderTransition without
// modifying the order.
func (s *service) TransitionStatus(ctx context.Context, namespace, id string, to commerce.OrderStatus) (*commerce.Order, error) {
	current, err := s.repo.GetOrder(ctx, namespace, id)
	if err != nil {
		return nil, err
	}

	if err := MustTransition(current.Status, to); err != nil {
		return nil, err
	}

	return s.repo.UpdateOrderStatus(ctx, namespace, id, current.Status, to)
}

// snapshotProduct captures the immutable attributes of a product into an order
// line. This snapshot is frozen at order creation and never changes, even if
// the catalog product is later edited.
func snapshotProduct(p commerce.Product) commerce.OrderLineSnapshot {
	return commerce.OrderLineSnapshot{
		ProductID:          p.ID,
		SKU:                p.SKU,
		DisplayName:        p.DisplayName,
		Quantity:           1,
		UnitPriceMinor:     p.AmountMinor,
		SubtotalMinor:      p.AmountMinor,
		Credits:            p.Credits,
		Currency:           p.Currency,
		ValidityDays:       p.ValidityDays,
		IncludedPlanCredit: planCreditForKind(p),
		Metadata: map[string]string{
			"product_version": fmt.Sprintf("%d", p.Version),
			"refund_policy":   string(p.RefundPolicy),
			"bonus_credits":   fmt.Sprintf("%d", p.BonusCredits),
		},
	}
}

// planCreditForKind extracts the included plan credit for plan purchases. For
// non-plan products this is 0.
func planCreditForKind(p commerce.Product) int64 {
	if p.Kind == commerce.ProductKindPlanPurchase || p.Kind == commerce.ProductKindSubscriptionRenewal {
		return p.Credits
	}
	return 0
}

// validateCreateInput checks the create-order input for required fields.
func validateCreateInput(in commerce.CreateOrderInput) error {
	var errs []error

	if in.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}
	if in.CustomerID == "" {
		errs = append(errs, errors.New("customer_id is required"))
	}
	if in.IdempotencyKey == "" {
		errs = append(errs, errors.New("idempotency_key is required"))
	}
	if in.Currency == "" {
		errs = append(errs, errors.New("currency is required"))
	}

	switch in.Kind {
	case commerce.OrderKindPlanPurchase, commerce.OrderKindSubscriptionRenewal, commerce.OrderKindWalletTopUp:
	default:
		errs = append(errs, fmt.Errorf("invalid order kind: %s", in.Kind))
	}

	if len(in.ProductIDs) == 0 {
		errs = append(errs, errors.New("at least one product_id is required"))
	}

	// Check for duplicate product IDs.
	seen := make(map[string]bool)
	for _, pid := range in.ProductIDs {
		if seen[pid] {
			errs = append(errs, fmt.Errorf("duplicate product_id: %s", pid))
		}
		seen[pid] = true
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// RenewalScheduleEntry is one monthly grant in a yearly subscription renewal
// schedule. A yearly renewal never grants the whole annual credit at payment
// time — it schedules twelve monthly grants starting from fulfillment.
type RenewalScheduleEntry struct {
	OrderID    string
	GrantDate  time.Time
	Credits    int64
	MonthIndex int // 1-based: 1 through 12
}

// ScheduleYearlyRenewal produces twelve concrete monthly grant entries for a
// yearly Pro/Team subscription renewal. Each entry grants exactly 1/12 of the
// total plan credits, scheduled one month apart starting from the order's
// UpdatedAt (which represents fulfillment time). The annual credit is never
// granted at payment time.
func ScheduleYearlyRenewal(order commerce.Order) []RenewalScheduleEntry {
	if order.Kind != commerce.OrderKindSubscriptionRenewal {
		return nil
	}

	totalCredits := int64(0)
	for _, line := range order.Lines {
		totalCredits += line.IncludedPlanCredit
	}
	if totalCredits <= 0 {
		return nil
	}

	monthly := totalCredits / 12
	if monthly == 0 {
		monthly = 1 // floor at 1 Credit per month
	}

	start := order.UpdatedAt
	if start.IsZero() {
		start = clock.Now()
	}

	entries := make([]RenewalScheduleEntry, 12)
	for i := 0; i < 12; i++ {
		entries[i] = RenewalScheduleEntry{
			OrderID:    order.ID,
			GrantDate:  start.AddDate(0, i, 0),
			Credits:    monthly,
			MonthIndex: i + 1,
		}
	}
	return entries
}

// IsTerminal returns true if the status is a terminal state (no further
// transitions possible).
func IsTerminal(s commerce.OrderStatus) bool {
	switch s {
	case commerce.OrderStatusCancelled, commerce.OrderStatusExpired, commerce.OrderStatusRefunded:
		return true
	default:
		return false
	}
}

// DescribeTransitionTable returns a human-readable summary of the state machine
// for documentation and debugging.
func DescribeTransitionTable() string {
	var sb strings.Builder
	for from, dests := range ValidTransitions {
		for to := range dests {
			sb.WriteString(fmt.Sprintf("%s -> %s\n", from, to))
		}
	}
	return sb.String()
}

var _ Service = (*service)(nil)
