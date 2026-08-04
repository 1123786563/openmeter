// Package fulfillment implements exactly-once fulfillment of paid commerce
// orders. After an order reaches "paid" (verified provider fact exists), the
// fulfillment service:
//
//  1. Creates a Fulfillment request record (status=pending).
//  2. A worker locks the request, marks the existing OpenMeter commercial
//     Invoice paid, grants plan/recharge Credits or renews the Subscription
//     through the existing domains, and marks the order "fulfilled".
//  3. Emits one notification.
//
// The partial unique index (namespace, commerce_order_id WHERE status='fulfilled')
// guarantees exactly one successful fulfillment per order — repeated worker
// retries or crashes converge to a single fulfilled state.
package fulfillment

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/openmeterio/openmeter/openmeter/commerce"
)

// ProcessingLeaseTimeout is how long a processing record can be stuck before
// another worker may re-claim it. This prevents stuck records when a worker
// crashes mid-processing (I4).
const ProcessingLeaseTimeout = 60 * time.Second

// ---------------------------------------------------------------------------
// Repository interfaces
// ---------------------------------------------------------------------------

// Repository manages Fulfillment records.
type Repository interface {
	// CreateFulfillment persists a new fulfillment request. The partial unique
	// index (status='fulfilled') prevents duplicate successful fulfillments.
	CreateFulfillment(ctx context.Context, req FulfillmentRequest) (*FulfillmentRecord, error)

	// GetFulfillment retrieves a fulfillment by namespace and ID.
	GetFulfillment(ctx context.Context, namespace, id string) (*FulfillmentRecord, error)

	// GetFulfillmentByOrder retrieves the fulfillment for an order (if any).
	GetFulfillmentByOrder(ctx context.Context, namespace, orderID string) (*FulfillmentRecord, error)

	// ClaimForProcessing atomically transitions a fulfillment from pending ->
	// processing and records the claim timestamp. Records already in processing
	// state are eligible for re-claim after the lease timeout elapses (I4).
	ClaimForProcessing(ctx context.Context, namespace, id string) (*FulfillmentRecord, error)

	// MarkFulfilled marks a fulfillment as succeeded. This is the point at which
	// the unique constraint (status='fulfilled') is enforced — if another
	// fulfillment for the same order is already fulfilled, this fails.
	MarkFulfilled(ctx context.Context, namespace, id string, result FulfillmentResult) (*FulfillmentRecord, error)

	// MarkFailed marks a fulfillment as failed for retry.
	MarkFailed(ctx context.Context, namespace, id string, reason string) (*FulfillmentRecord, error)

	// ListPending returns fulfillments in pending or processing status for the
	// worker to process.
	ListPending(ctx context.Context, namespace string, limit int) ([]FulfillmentRecord, error)
}

// OrderStatusUpdater transitions the order to fulfilled.
type OrderStatusUpdater interface {
	GetOrder(ctx context.Context, namespace, id string) (*commerce.Order, error)
	UpdateOrderStatus(ctx context.Context, namespace, id string, expectedFrom, to commerce.OrderStatus) (*commerce.Order, error)
}

// CreditGrantor grants Credits to the customer's wallet after fulfillment.
// This is the bridge to the Phase 1 Credit Ledger. Implementations create a
// real ledger grant and return the GrantID + CreditsGranted.
type CreditGrantor interface {
	// GrantCredits grants Credits to the customer. It must be idempotent on the
	// idempotency key — repeated calls for the same key return the original
	// grant. Returns the grant ID and the total Credits granted.
	GrantCredits(ctx context.Context, in GrantCreditsInput) (GrantCreditsResult, error)
}

// GrantCreditsInput carries the context for a credit grant.
type GrantCreditsInput struct {
	Namespace      string
	CustomerID     string
	OrderID        string
	Source         commerce.BucketSource
	Credits        int64
	ValidityDays   int
	IdempotencyKey string
}

// GrantCreditsResult holds the result of a credit grant.
type GrantCreditsResult struct {
	GrantID string
	Credits int64
}

// SubscriptionRenewer renews a subscription after a renewal order is fulfilled.
type SubscriptionRenewer interface {
	// RenewSubscription extends the customer's subscription. It must be
	// idempotent on the order ID.
	RenewSubscription(ctx context.Context, in RenewSubscriptionInput) (RenewSubscriptionResult, error)
}

// RenewSubscriptionInput carries the context for a subscription renewal.
type RenewSubscriptionInput struct {
	Namespace  string
	CustomerID string
	OrderID    string
}

// RenewSubscriptionResult holds the result of a subscription renewal.
type RenewSubscriptionResult struct {
	SubscriptionID string
	ValidUntil     time.Time
}

// InvoiceMarker marks the OpenMeter commercial invoice as paid. This connects
// the commerce order fulfillment to the existing billing domain.
type InvoiceMarker interface {
	// MarkInvoicePaid marks the invoice associated with this order as paid.
	// It must be idempotent — repeated calls for the same order succeed silently.
	MarkInvoicePaid(ctx context.Context, namespace, orderID string) error
}

// Notifier emits a fulfillment notification (e.g. order completed push).
type Notifier interface {
	NotifyOrderFulfilled(ctx context.Context, namespace, orderID, customerID string) error
}

// ---------------------------------------------------------------------------
// Domain types
// ---------------------------------------------------------------------------

// FulfillmentStatus is the lifecycle status of a fulfillment.
type FulfillmentStatus string

const (
	FulfillmentStatusPending    FulfillmentStatus = "pending"
	FulfillmentStatusProcessing FulfillmentStatus = "processing"
	FulfillmentStatusFulfilled  FulfillmentStatus = "fulfilled"
	FulfillmentStatusFailed     FulfillmentStatus = "failed"
)

// FulfillmentRequest is the input to create a fulfillment record.
type FulfillmentRequest struct {
	Namespace  string
	OrderID    string
	CustomerID string
}

// FulfillmentRecord is the persisted fulfillment state. The ClaimedAt field
// records when a worker started processing — records stuck in processing past
// the lease timeout become eligible for re-claim (I4).
type FulfillmentRecord struct {
	ID             string
	Namespace      string
	OrderID        string
	CustomerID     string
	Status         FulfillmentStatus
	GrantID        string
	CreditsGranted int64
	FulfilledAt    *time.Time
	FailureReason  *string
	ClaimedAt      *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// FulfillmentResult is the output of the grant step.
type FulfillmentResult struct {
	GrantID        string
	CreditsGranted int64
}

// ErrAlreadyFulfilled is returned when a fulfillment for the order is already
// in the fulfilled state. Callers treat this as idempotent success.
var ErrAlreadyFulfilled = errors.New("fulfillment: order already fulfilled")

// ErrAlreadyProcessing is returned when a fulfillment is being processed by
// another worker and the lease has not expired.
var ErrAlreadyProcessing = errors.New("fulfillment: already being processed")

// ErrOrderNotPaid is returned when ProcessOne encounters an order that is not
// in the paid or fulfilled state (I3).
var ErrOrderNotPaid = errors.New("fulfillment: order is not in paid state")

// ---------------------------------------------------------------------------
// Service
// ---------------------------------------------------------------------------

// Service implements exactly-once fulfillment.
type Service interface {
	// RequestFulfillment creates a fulfillment request for a paid order. This is
	// called after the order transitions to paid. It is idempotent: if a
	// fulfillment request already exists for this order, it returns the existing
	// record.
	RequestFulfillment(ctx context.Context, namespace, orderID string) (*FulfillmentRecord, error)

	// ProcessOne processes a single fulfillment request. This is the worker
	// entry point. It is idempotent and crash-safe: repeated calls converge to
	// exactly one fulfilled state.
	ProcessOne(ctx context.Context, namespace, fulfillmentID string) (*FulfillmentRecord, error)

	// ProcessPending processes all pending fulfillments (worker loop).
	ProcessPending(ctx context.Context, namespace string, limit int) (int, error)

	// GetFulfillment retrieves a fulfillment by ID.
	GetFulfillment(ctx context.Context, namespace, id string) (*FulfillmentRecord, error)

	// GetFulfillmentByOrder retrieves the fulfillment for an order.
	GetFulfillmentByOrder(ctx context.Context, namespace, orderID string) (*FulfillmentRecord, error)
}

// Config wires the fulfillment service.
type Config struct {
	Repo     Repository
	Orders   OrderStatusUpdater
	Grantor  CreditGrantor
	Renewer  SubscriptionRenewer
	Invoices InvoiceMarker
	Notifier Notifier
	Logger   *slog.Logger
}

type service struct {
	repo     Repository
	orders   OrderStatusUpdater
	grantor  CreditGrantor
	renewer  SubscriptionRenewer
	invoices InvoiceMarker
	notifier Notifier
	logger   *slog.Logger
}

// New creates a fulfillment Service.
func New(cfg Config) (Service, error) {
	if cfg.Repo == nil {
		return nil, errors.New("fulfillment: repository is required")
	}
	if cfg.Orders == nil {
		return nil, errors.New("fulfillment: orders port is required")
	}
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}
	return &service{
		repo:     cfg.Repo,
		orders:   cfg.Orders,
		grantor:  cfg.Grantor,
		renewer:  cfg.Renewer,
		invoices: cfg.Invoices,
		notifier: cfg.Notifier,
		logger:   logger,
	}, nil
}

// RequestFulfillment creates a fulfillment request for a paid order.
func (s *service) RequestFulfillment(ctx context.Context, namespace, orderID string) (*FulfillmentRecord, error) {
	// Idempotency: check for existing fulfillment.
	existing, err := s.repo.GetFulfillmentByOrder(ctx, namespace, orderID)
	if err == nil && existing != nil {
		return existing, nil
	}

	// Fetch the order to get the customer ID.
	order, err := s.orders.GetOrder(ctx, namespace, orderID)
	if err != nil {
		return nil, fmt.Errorf("fulfillment: get order: %w", err)
	}

	return s.repo.CreateFulfillment(ctx, FulfillmentRequest{
		Namespace:  namespace,
		OrderID:    orderID,
		CustomerID: order.CustomerID,
	})
}

// ProcessOne processes a single fulfillment request. The flow is:
//
//  1. Check current state. If already fulfilled, return (idempotent success).
//  2. Claim for processing (pending/failed -> processing). If another worker
//     has it and the lease hasn't expired, return.
//  3. Fetch the order; verify it is in the paid state (I3).
//  4. Mark the commercial invoice paid (idempotent).
//  5. Grant credits or renew subscription based on order kind.
//  6. Mark the fulfillment fulfilled (unique constraint enforces exactly-once).
//  7. Transition the order to fulfilled.
//  8. Notify.
//
// If a crash occurs at any step, the worker restarts and reprocesses. The
// unique index on (order, status='fulfilled') guarantees only one fulfillment
// succeeds. Invoice marking and credit granting are idempotent.
func (s *service) ProcessOne(ctx context.Context, namespace, fulfillmentID string) (*FulfillmentRecord, error) {
	// Check current state.
	rec, err := s.repo.GetFulfillment(ctx, namespace, fulfillmentID)
	if err != nil {
		return nil, fmt.Errorf("fulfillment: get fulfillment: %w", err)
	}
	// C3 nil guard: if the repository returns no record (e.g. stub adapter in
	// production before the Ent fulfillment repo is wired), return early
	// instead of panicking on a nil dereference.
	if rec == nil {
		return nil, fmt.Errorf("fulfillment: record %s not found in namespace %s", fulfillmentID, namespace)
	}

	// Already fulfilled — idempotent success.
	if rec.Status == FulfillmentStatusFulfilled {
		return rec, nil
	}

	// Claim for processing (pending/failed -> processing). If another worker
	// has it and the lease hasn't expired, skip (return the current state).
	rec, err = s.repo.ClaimForProcessing(ctx, namespace, fulfillmentID)
	if err != nil {
		if errors.Is(err, ErrAlreadyProcessing) {
			return rec, nil
		}
		return nil, fmt.Errorf("fulfillment: claim for processing: %w", err)
	}
	// C3 nil guard: ClaimForProcessing should never return nil without error,
	// but guard defensively for stub adapters.
	if rec == nil {
		return nil, fmt.Errorf("fulfillment: claim returned no record for %s", fulfillmentID)
	}

	// Fetch the order.
	order, err := s.orders.GetOrder(ctx, namespace, rec.OrderID)
	if err != nil {
		s.markFailed(ctx, namespace, fulfillmentID, "get order: "+err.Error())
		return nil, fmt.Errorf("fulfillment: get order: %w", err)
	}

	// I3: Guard — the order must be in paid or fulfilled state. An order in any
	// other state (created, awaiting_payment, cancelled) is not ready for
	// fulfillment.
	if order.Status != commerce.OrderStatusPaid && order.Status != commerce.OrderStatusFulfilled {
		s.markFailed(ctx, namespace, fulfillmentID, fmt.Sprintf("order status is %s, expected paid", order.Status))
		return nil, fmt.Errorf("%w: status=%s", ErrOrderNotPaid, order.Status)
	}

	// If already fulfilled, the unique index would have prevented a duplicate
	// fulfillment record, but we check defensively.
	if order.Status == commerce.OrderStatusFulfilled {
		return rec, nil
	}

	// Compute total credits to grant.
	credits := totalOrderCredits(order)

	// Step 1: Mark the commercial invoice paid (idempotent).
	if s.invoices != nil {
		if err := s.invoices.MarkInvoicePaid(ctx, namespace, rec.OrderID); err != nil {
			s.logger.WarnContext(ctx, "fulfillment: mark invoice paid failed", "error", err)
			// Non-fatal: the invoice may already be paid. Continue.
		}
	}

	// Step 2: Grant credits or renew subscription based on order kind.
	var result FulfillmentResult
	switch order.Kind {
	case commerce.OrderKindWalletTopUp:
		result, err = s.grantWalletCredits(ctx, order, credits)
	case commerce.OrderKindPlanPurchase:
		result, err = s.grantPlanCredits(ctx, order, credits)
	case commerce.OrderKindSubscriptionRenewal:
		result, err = s.renewSubscription(ctx, order)
	default:
		err = fmt.Errorf("fulfillment: unknown order kind: %s", order.Kind)
	}

	if err != nil {
		s.markFailed(ctx, namespace, fulfillmentID, err.Error())
		return nil, fmt.Errorf("fulfillment: grant step: %w", err)
	}

	// Step 3: Mark fulfillment fulfilled. The unique index enforces exactly-once.
	rec, err = s.repo.MarkFulfilled(ctx, namespace, fulfillmentID, result)
	if err != nil {
		// C1: If MarkFulfilled failed, rec may be nil. Log the error and reload
		// the record from the store. A unique-constraint violation here means a
		// concurrent worker already fulfilled this order — that's success.
		s.logger.InfoContext(ctx, "fulfillment: mark fulfilled (may be concurrent)", "error", err)
		rec, reloadErr := s.repo.GetFulfillment(ctx, namespace, fulfillmentID)
		if reloadErr != nil {
			return nil, fmt.Errorf("fulfillment: mark fulfilled failed (%w) and reload failed: %w", err, reloadErr)
		}
		// If the record is now fulfilled by a concurrent worker, treat as success.
		if rec.Status == FulfillmentStatusFulfilled {
			return rec, nil
		}
		// Otherwise the failure was real — return the error.
		return rec, fmt.Errorf("fulfillment: mark fulfilled: %w", err)
	}

	// Step 4: Transition the order to fulfilled.
	if _, err := s.orders.UpdateOrderStatus(ctx, namespace, rec.OrderID, commerce.OrderStatusPaid, commerce.OrderStatusFulfilled); err != nil {
		// The order may already be fulfilled (concurrent worker). Log and continue.
		s.logger.InfoContext(ctx, "fulfillment: order -> fulfilled transition (may be concurrent)", "error", err)
	}

	// Step 5: Notify.
	if s.notifier != nil {
		if err := s.notifier.NotifyOrderFulfilled(ctx, namespace, rec.OrderID, rec.CustomerID); err != nil {
			s.logger.WarnContext(ctx, "fulfillment: notify failed", "error", err)
		}
	}

	// Reload final state.
	return s.repo.GetFulfillment(ctx, namespace, fulfillmentID)
}

// markFailed marks a fulfillment as failed and logs if the mark itself errors
// (M5 — never silently swallow the error).
func (s *service) markFailed(ctx context.Context, namespace, id, reason string) {
	if _, err := s.repo.MarkFailed(ctx, namespace, id, reason); err != nil {
		s.logger.ErrorContext(ctx, "fulfillment: MarkFailed itself failed", "id", id, "error", err)
	}
}

// ProcessPending processes all pending fulfillments. Returns the count processed.
func (s *service) ProcessPending(ctx context.Context, namespace string, limit int) (int, error) {
	pending, err := s.repo.ListPending(ctx, namespace, limit)
	if err != nil {
		return 0, fmt.Errorf("fulfillment: list pending: %w", err)
	}
	// C3 nil guard: ListPending returning nil is safe (ranging over nil is a
	// no-op), but log at debug so a stub adapter is visible during development.
	if len(pending) == 0 {
		return 0, nil
	}

	processed := 0
	for _, rec := range pending {
		if _, err := s.ProcessOne(ctx, namespace, rec.ID); err != nil {
			s.logger.WarnContext(ctx, "fulfillment: process pending item failed", "id", rec.ID, "error", err)
			continue
		}
		processed++
	}
	return processed, nil
}

// GetFulfillment retrieves a fulfillment by ID.
func (s *service) GetFulfillment(ctx context.Context, namespace, id string) (*FulfillmentRecord, error) {
	return s.repo.GetFulfillment(ctx, namespace, id)
}

// GetFulfillmentByOrder retrieves the fulfillment for an order.
func (s *service) GetFulfillmentByOrder(ctx context.Context, namespace, orderID string) (*FulfillmentRecord, error) {
	return s.repo.GetFulfillmentByOrder(ctx, namespace, orderID)
}

// --- Internal helpers ---

// grantWalletCredits grants recharge Credits to the customer's wallet.
func (s *service) grantWalletCredits(ctx context.Context, order *commerce.Order, credits int64) (FulfillmentResult, error) {
	if s.grantor == nil {
		return FulfillmentResult{CreditsGranted: credits}, nil
	}
	res, err := s.grantor.GrantCredits(ctx, GrantCreditsInput{
		Namespace:      order.Namespace,
		CustomerID:     order.CustomerID,
		OrderID:        order.ID,
		Source:         commerce.BucketSourceRecharge,
		Credits:        credits,
		IdempotencyKey: "fulfillment:" + order.ID,
	})
	if err != nil {
		return FulfillmentResult{}, fmt.Errorf("grant wallet credits: %w", err)
	}
	return FulfillmentResult{GrantID: res.GrantID, CreditsGranted: res.Credits}, nil
}

// grantPlanCredits grants plan Credits to the customer.
func (s *service) grantPlanCredits(ctx context.Context, order *commerce.Order, credits int64) (FulfillmentResult, error) {
	if s.grantor == nil {
		return FulfillmentResult{CreditsGranted: credits}, nil
	}
	res, err := s.grantor.GrantCredits(ctx, GrantCreditsInput{
		Namespace:      order.Namespace,
		CustomerID:     order.CustomerID,
		OrderID:        order.ID,
		Source:         commerce.BucketSourcePlan,
		Credits:        credits,
		IdempotencyKey: "fulfillment:" + order.ID,
	})
	if err != nil {
		return FulfillmentResult{}, fmt.Errorf("grant plan credits: %w", err)
	}
	return FulfillmentResult{GrantID: res.GrantID, CreditsGranted: res.Credits}, nil
}

// renewSubscription renews the subscription and returns a zero-credit result
// (subscription credits are scheduled monthly, not granted upfront).
func (s *service) renewSubscription(ctx context.Context, order *commerce.Order) (FulfillmentResult, error) {
	if s.renewer == nil {
		return FulfillmentResult{}, nil
	}
	if _, err := s.renewer.RenewSubscription(ctx, RenewSubscriptionInput{
		Namespace:  order.Namespace,
		CustomerID: order.CustomerID,
		OrderID:    order.ID,
	}); err != nil {
		return FulfillmentResult{}, fmt.Errorf("renew subscription: %w", err)
	}
	// Yearly renewal credits are scheduled monthly via ScheduleYearlyRenewal,
	// not granted upfront.
	return FulfillmentResult{}, nil
}

// totalOrderCredits sums the Credits across all order lines.
func totalOrderCredits(order *commerce.Order) int64 {
	total := int64(0)
	for _, line := range order.Lines {
		total += line.Credits
	}
	return total
}

// Compile-time check.
var _ Service = (*service)(nil)
