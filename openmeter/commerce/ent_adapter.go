package commerce

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	entsql "entgo.io/ent/dialect/sql"

	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billingcustomerlock"
	"github.com/openmeterio/openmeter/openmeter/ent/db/commerceorder"
	"github.com/openmeterio/openmeter/openmeter/ent/db/commerceproduct"
	dbgrant "github.com/openmeterio/openmeter/openmeter/ent/db/grant"
	"github.com/openmeterio/openmeter/openmeter/ent/db/receivableaccount"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
)

// EntAdapter is the Ent-backed implementation of the commerce repository
// interfaces. It provides customer-locked transactions via WithCustomerLock.
type EntAdapter struct {
	db     *entdb.Client
	logger *slog.Logger
}

// EntAdapterConfig wires the adapter.
type EntAdapterConfig struct {
	Client *entdb.Client
	Logger *slog.Logger
}

// NewEntAdapter creates a new Ent-backed adapter.
func NewEntAdapter(cfg EntAdapterConfig) (*EntAdapter, error) {
	if cfg.Client == nil {
		return nil, errors.New("ent client is required")
	}
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}
	return &EntAdapter{db: cfg.Client, logger: logger}, nil
}

// ---------------------------------------------------------------------------
// transaction.Creator + entutils.TxUser (for WithCustomerLock)
// ---------------------------------------------------------------------------

func (a *EntAdapter) Tx(ctx context.Context) (context.Context, transaction.Driver, error) {
	txCtx, rawConfig, eDriver, err := a.db.HijackTx(ctx, &sql.TxOptions{ReadOnly: false})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to hijack transaction: %w", err)
	}
	return txCtx, entutils.NewTxDriver(eDriver, rawConfig), nil
}

func (a *EntAdapter) WithTx(_ context.Context, tx *entutils.TxDriver) *EntAdapter {
	txClient := entdb.NewTxClientFromRawConfig(context.Background(), *tx.GetConfig())
	return &EntAdapter{db: txClient.Client(), logger: a.logger}
}

func (a *EntAdapter) Self() *EntAdapter { return a }

// WithCustomerLock starts a transaction, acquires a per-customer advisory lock,
// and runs fn inside that transaction.
func (a *EntAdapter) WithCustomerLock(ctx context.Context, namespace, customerID string, fn func(*EntAdapter) error) error {
	return transaction.RunWithNoValue(ctx, a, func(ctx context.Context) error {
		return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, txa *EntAdapter) error {
			err := txa.db.BillingCustomerLock.Create().
				SetNamespace(namespace).
				SetCustomerID(customerID).
				OnConflict(entsql.DoNothing()).
				Exec(ctx)
			if err != nil && !errors.Is(err, sql.ErrNoRows) {
				return fmt.Errorf("upsert customer lock: %w", err)
			}

			_, err = txa.db.BillingCustomerLock.Query().
				Where(
					billingcustomerlock.Namespace(namespace),
					billingcustomerlock.CustomerID(customerID),
				).
				ForUpdate().
				First(ctx)
			if err != nil {
				return fmt.Errorf("lock customer: %w", err)
			}

			return fn(txa)
		})
	})
}

// ---------------------------------------------------------------------------
// ProductRepository
// ---------------------------------------------------------------------------

// CreateProduct persists a new catalog product. Extended domain attributes are
// stored in the Ent metadata JSON column.
func (a *EntAdapter) CreateProduct(ctx context.Context, p Product) (*Product, error) {
	meta := productMetadata(p)

	builder := a.db.CommerceProduct.Create().
		SetNamespace(p.Namespace).
		SetSku(p.SKU).
		SetName(p.DisplayName).
		SetKind(must1(mapKindToEnt(p.Kind))).
		SetPriceCents(p.AmountMinor).
		SetCurrency(p.Currency).
		SetMetadata(meta)

	if p.Description != "" {
		builder.SetDescription(p.Description)
	}

	saved, err := builder.Save(ctx)
	if err != nil {
		if entdb.IsConstraintError(err) {
			return nil, ErrSKUNotUnique
		}
		return nil, fmt.Errorf("ent: create product: %w", err)
	}

	return mapEntProduct(saved), nil
}

// GetProduct retrieves a product by namespace and ID.
func (a *EntAdapter) GetProduct(ctx context.Context, namespace, id string) (*Product, error) {
	ep, err := a.db.CommerceProduct.Get(ctx, id)
	if err != nil {
		if entdb.IsNotFound(err) {
			return nil, ErrProductNotFound
		}
		return nil, fmt.Errorf("ent: get product: %w", err)
	}
	if ep.Namespace != namespace {
		return nil, ErrProductNotFound
	}
	return mapEntProduct(ep), nil
}

// GetProductBySKU retrieves a product by its SKU within a namespace.
func (a *EntAdapter) GetProductBySKU(ctx context.Context, namespace, sku string) (*Product, error) {
	ep, err := a.db.CommerceProduct.Query().
		Where(
			commerceproduct.Namespace(namespace),
			commerceproduct.Sku(sku),
		).
		First(ctx)
	if err != nil {
		if entdb.IsNotFound(err) {
			return nil, ErrProductNotFound
		}
		return nil, fmt.Errorf("ent: get product by sku: %w", err)
	}
	return mapEntProduct(ep), nil
}

// ListProducts lists products with optional kind and active filters.
func (a *EntAdapter) ListProducts(ctx context.Context, namespace string, kind *ProductKind, activeOnly bool) ([]Product, error) {
	q := a.db.CommerceProduct.Query().Where(commerceproduct.Namespace(namespace))
	if kind != nil {
		q = q.Where(commerceproduct.KindEQ(must1(mapKindToEnt(*kind))))
	}

	eps, err := q.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("ent: list products: %w", err)
	}

	result := make([]Product, 0, len(eps))
	for _, ep := range eps {
		p := mapEntProduct(ep)
		if activeOnly && !p.Active {
			continue
		}
		result = append(result, *p)
	}
	return result, nil
}

// UpdateProduct updates mutable product fields.
func (a *EntAdapter) UpdateProduct(ctx context.Context, p Product) (*Product, error) {
	meta := productMetadata(p)

	builder := a.db.CommerceProduct.UpdateOneID(p.ID).
		SetName(p.DisplayName).
		SetPriceCents(p.AmountMinor).
		SetMetadata(meta)

	saved, err := builder.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("ent: update product: %w", err)
	}
	return mapEntProduct(saved), nil
}

// ---------------------------------------------------------------------------
// OrderRepository
// ---------------------------------------------------------------------------

// CreateOrder persists a new order with its line snapshots. Returns the stored
// order and a boolean (true = fresh insert, false = idempotent replay).
func (a *EntAdapter) CreateOrder(ctx context.Context, o Order) (*Order, bool, error) {
	// Idempotency check.
	existing, err := a.db.CommerceOrder.Query().
		Where(
			commerceorder.NamespaceEQ(o.Namespace),
			commerceorder.CustomerIDEQ(o.CustomerID),
			commerceorder.IdempotencyKeyEQ(o.IdempotencyKey),
		).
		WithLines().
		First(ctx)
	if err == nil && existing != nil {
		return mapEntOrder(existing), false, nil
	}
	if err != nil && !entdb.IsNotFound(err) {
		return nil, false, fmt.Errorf("ent: idempotency check: %w", err)
	}

	// Create order header.
	builder := a.db.CommerceOrder.Create().
		SetID(o.ID).
		SetNamespace(o.Namespace).
		SetPublicID(o.PublicID).
		SetCustomerID(o.CustomerID).
		SetKind(must1(mapOrderKindToEnt(o.Kind))).
		SetStatus(commerceorder.StatusCreated).
		SetTotalCents(o.AmountMinor).
		SetCurrency(o.Currency).
		SetIdempotencyKey(o.IdempotencyKey)

	if len(o.Lines) > 0 {
		if desc := o.Lines[0].Metadata["order_description"]; desc != "" {
			builder.SetDescription(desc)
		}
	}

	saved, err := builder.Save(ctx)
	if err != nil {
		if entdb.IsConstraintError(err) {
			// Concurrent insert with same idempotency key: fetch the winner order.
			existing, gErr := a.GetOrderByIdempotencyKey(ctx, o.Namespace, o.CustomerID, o.IdempotencyKey)
			if gErr != nil {
				return nil, false, fmt.Errorf("ent: concurrent insert recovery: %w", gErr)
			}
			return existing, false, nil
		}
		return nil, false, fmt.Errorf("ent: create order: %w", err)
	}

	// Create line snapshots.
	lineBuilders := make([]*entdb.CommerceOrderLineCreate, 0, len(o.Lines))
	for _, line := range o.Lines {
		snapData, _ := json.Marshal(line)
		lb := a.db.CommerceOrderLine.Create().
			SetNamespace(o.Namespace).
			SetCommerceOrderID(saved.ID).
			SetProductID(line.ProductID).
			SetProductSku(line.SKU).
			SetProductName(line.DisplayName).
			SetQuantity(line.Quantity).
			SetUnitPriceCents(line.UnitPriceMinor).
			SetSubtotalCents(line.SubtotalMinor)
		if len(snapData) > 2 {
			lb.SetSnapshotData(map[string]any{"line": string(snapData)})
		}
		lineBuilders = append(lineBuilders, lb)
	}
	if len(lineBuilders) > 0 {
		if _, err := a.db.CommerceOrderLine.CreateBulk(lineBuilders...).Save(ctx); err != nil {
			return nil, false, fmt.Errorf("ent: create order lines: %w", err)
		}
	}

	// Reload with lines eager-loaded.
	full, err := a.db.CommerceOrder.Query().
		Where(commerceorder.IDEQ(saved.ID)).
		WithLines().
		Only(ctx)
	if err != nil {
		return nil, false, fmt.Errorf("ent: reload order with lines: %w", err)
	}

	return mapEntOrder(full), true, nil
}

// GetOrder retrieves an order by namespace and ID with its lines.
func (a *EntAdapter) GetOrder(ctx context.Context, namespace, id string) (*Order, error) {
	eo, err := a.db.CommerceOrder.Query().
		Where(commerceorder.IDEQ(id), commerceorder.NamespaceEQ(namespace)).
		WithLines().
		Only(ctx)
	if err != nil {
		if entdb.IsNotFound(err) {
			return nil, ErrOrderNotFound
		}
		return nil, fmt.Errorf("ent: get order: %w", err)
	}
	return mapEntOrder(eo), nil
}

// GetOrderByIdempotencyKey looks up an order by idempotency key.
func (a *EntAdapter) GetOrderByIdempotencyKey(ctx context.Context, namespace, customerID, key string) (*Order, error) {
	eo, err := a.db.CommerceOrder.Query().
		Where(
			commerceorder.NamespaceEQ(namespace),
			commerceorder.CustomerIDEQ(customerID),
			commerceorder.IdempotencyKeyEQ(key),
		).
		WithLines().
		First(ctx)
	if err != nil {
		if entdb.IsNotFound(err) {
			return nil, ErrOrderNotFound
		}
		return nil, fmt.Errorf("ent: get order by idempotency key: %w", err)
	}
	return mapEntOrder(eo), nil
}

// UpdateOrderStatus applies a status transition atomically with optimistic
// concurrency control via expectedFrom.
func (a *EntAdapter) UpdateOrderStatus(ctx context.Context, namespace, id string, expectedFrom, to OrderStatus) (*Order, error) {
	n, err := a.db.CommerceOrder.Update().
		Where(
			commerceorder.IDEQ(id),
			commerceorder.NamespaceEQ(namespace),
			commerceorder.StatusEQ(must1(mapOrderStatusToEnt(expectedFrom))),
		).
		SetStatus(must1(mapOrderStatusToEnt(to))).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("ent: update order status: %w", err)
	}
	if n == 0 {
		return nil, ErrInvalidOrderTransition
	}

	return a.GetOrder(ctx, namespace, id)
}

// ListOrdersByCustomer lists orders for a customer, optionally filtered by status.
func (a *EntAdapter) ListOrdersByCustomer(ctx context.Context, namespace, customerID string, status *OrderStatus) ([]Order, error) {
	q := a.db.CommerceOrder.Query().
		Where(
			commerceorder.NamespaceEQ(namespace),
			commerceorder.CustomerIDEQ(customerID),
		).
		WithLines()

	if status != nil {
		q = q.Where(commerceorder.StatusEQ(must1(mapOrderStatusToEnt(*status))))
	}

	eos, err := q.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("ent: list orders: %w", err)
	}

	result := make([]Order, 0, len(eos))
	for _, eo := range eos {
		result = append(result, *mapEntOrder(eo))
	}
	return result, nil
}

// ---------------------------------------------------------------------------
// WalletDataPort (read-only)
// ---------------------------------------------------------------------------

// GetGrants reads allocation grants for a customer. The full implementation
// queries the Credit Ledger's grant balances via the collector; this provides
// the interface contract for the wallet service.
func (a *EntAdapter) GetGrants(ctx context.Context, namespace, customerID string) ([]AllocationGrant, error) {
	// I3: query the Phase 1 Grant table directly. Grants with VoidedAt=nil and
	// not expired are active. The consumed amount requires the ledger collector
	// (Balance), so we report Granted as the grant amount and Consumed=0; the
	// refund service independently computes refundable credits from the wallet
	// snapshot. This returns real allocation data instead of an empty stub.
	q := a.db.Grant.Query().
		Where(
			dbgrant.NamespaceEQ(namespace),
			dbgrant.OwnerIDEQ(customerID),
			dbgrant.VoidedAtIsNil(),
		)

	grants, err := q.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("ent: list grants: %w", err)
	}

	result := make([]AllocationGrant, 0, len(grants))
	for _, g := range grants {
		// Skip expired grants.
		if g.ExpiresAt != nil && g.ExpiresAt.Before(clock.Now()) {
			continue
		}
		granted := int64(g.Amount)
		source := sourceFromPriority(int(g.Priority))
		result = append(result, AllocationGrant{
			GrantID:   g.ID,
			Source:    source,
			Granted:   granted,
			Consumed:  0, // requires ledger collector; refund path computes independently
			Priority:  SourcePriority(source),
			ExpiresAt: g.ExpiresAt,
			CreatedAt: g.CreatedAt,
			// Refundable: only recharge grants are refundable. We report the
			// full granted amount as refundable for recharge; the refund
			// service applies the actual unspent computation via ReserveCredits.
			Refundable: refundableForSource(source, granted),
		})
	}
	return result, nil
}

// sourceFromPriority maps a grant's numeric priority to a BucketSource using
// the SourcePriority ranges (plan=10, gift=20, recharge=30, enterprise=40).
func sourceFromPriority(p int) BucketSource {
	switch {
	case p < 15:
		return BucketSourcePlan
	case p < 25:
		return BucketSourceGift
	case p < 35:
		return BucketSourceRecharge
	default:
		return BucketSourceEnterpriseReceivable
	}
}

// refundableForSource returns the refundable amount for a source. Only recharge
// credits are refundable; the exact unspent computation happens in ReserveCredits.
func refundableForSource(source BucketSource, granted int64) int64 {
	if source == BucketSourceRecharge {
		return granted
	}
	return 0
}

// GetEnterpriseReceivable reads the enterprise receivable account for a customer.
func (a *EntAdapter) GetEnterpriseReceivable(ctx context.Context, namespace, customerID string) (*EnterpriseReceivable, error) {
	acct, err := a.db.ReceivableAccount.Query().
		Where(
			receivableaccount.NamespaceEQ(namespace),
			receivableaccount.CustomerIDEQ(customerID),
		).
		First(ctx)
	if err != nil {
		if entdb.IsNotFound(err) {
			return nil, nil // no enterprise receivable — normal for non-enterprise customers
		}
		return nil, fmt.Errorf("ent: get receivable account: %w", err)
	}

	used := -acct.CurrentBalanceCents // balance is negative when customer owes
	if used < 0 {
		used = 0
	}

	return &EnterpriseReceivable{
		AccountID:      acct.ID,
		CeilingCredits: acct.CreditLimitCents,
		UsedCredits:    used,
	}, nil
}

// GetRecentTransactions reads recent wallet transactions.
func (a *EntAdapter) GetRecentTransactions(ctx context.Context, namespace, customerID string, limit int) ([]WalletTransaction, error) {
	// Placeholder: full implementation queries the ledger for recent movements.
	return []WalletTransaction{}, nil
}

// ---------------------------------------------------------------------------
// Mapping helpers
// ---------------------------------------------------------------------------

// productMetadata encodes extended Product attributes into the Ent metadata JSON.
func productMetadata(p Product) map[string]any {
	meta := map[string]any{
		"version":          p.Version,
		"credits":          p.Credits,
		"active":           p.Active,
		"display_order":    p.DisplayOrder,
		"refund_policy":    string(p.RefundPolicy),
		"bonus_credits":    p.BonusCredits,
		"applicable_plans": p.ApplicablePlans,
		"purchase_limit":   p.PurchaseLimit,
	}
	if p.OnSaleAt != nil {
		meta["on_sale_at"] = p.OnSaleAt.Format(time.RFC3339)
	}
	if p.OffSaleAt != nil {
		meta["off_sale_at"] = p.OffSaleAt.Format(time.RFC3339)
	}
	return meta
}

// mapEntProduct converts an Ent CommerceProduct to a domain Product.
func mapEntProduct(ep *entdb.CommerceProduct) *Product {
	p := &Product{
		NamespacedID: models.NamespacedID{
			Namespace: ep.Namespace,
			ID:        ep.ID,
		},
		ManagedModel: models.ManagedModel{
			CreatedAt: ep.CreatedAt,
			UpdatedAt: ep.UpdatedAt,
			DeletedAt: ep.DeletedAt,
		},
		SKU:         ep.Sku,
		DisplayName: ep.Name,
		Kind:        must1(mapKindFromEnt(ep.Kind)),
		AmountMinor: ep.PriceCents,
		Currency:    ep.Currency,
	}
	if ep.Description != nil {
		p.Description = *ep.Description
	}
	if ep.Metadata != nil {
		if v, ok := ep.Metadata["version"].(float64); ok {
			p.Version = int(v)
		}
		if v, ok := ep.Metadata["credits"].(float64); ok {
			p.Credits = int64(v)
		}
		if v, ok := ep.Metadata["active"].(bool); ok {
			p.Active = v
		}
		if v, ok := ep.Metadata["display_order"].(float64); ok {
			p.DisplayOrder = int(v)
		}
		if v, ok := ep.Metadata["bonus_credits"].(float64); ok {
			p.BonusCredits = int64(v)
		}
		if v, ok := ep.Metadata["refund_policy"].(string); ok {
			p.RefundPolicy = RefundPolicy(v)
		}
		if v, ok := ep.Metadata["purchase_limit"].(float64); ok {
			p.PurchaseLimit = int(v)
		}
		if v, ok := ep.Metadata["applicable_plans"].([]any); ok {
			for _, item := range v {
				if s, ok := item.(string); ok {
					p.ApplicablePlans = append(p.ApplicablePlans, s)
				}
			}
		}
		if v, ok := ep.Metadata["on_sale_at"].(string); ok {
			if t, err := time.Parse(time.RFC3339, v); err == nil {
				p.OnSaleAt = &t
			}
		}
		if v, ok := ep.Metadata["off_sale_at"].(string); ok {
			if t, err := time.Parse(time.RFC3339, v); err == nil {
				p.OffSaleAt = &t
			}
		}
	}
	return p
}

// mapEntOrder converts an Ent CommerceOrder (with lines) to a domain Order.
func mapEntOrder(eo *entdb.CommerceOrder) *Order {
	o := &Order{
		NamespacedID: models.NamespacedID{
			Namespace: eo.Namespace,
			ID:        eo.ID,
		},
		ManagedModel: models.ManagedModel{
			CreatedAt: eo.CreatedAt,
			UpdatedAt: eo.UpdatedAt,
			DeletedAt: eo.DeletedAt,
		},
		PublicID:       eo.PublicID,
		CustomerID:     eo.CustomerID,
		Kind:           must1(mapOrderKindFromEnt(eo.Kind)),
		Status:         must1(mapOrderStatusFromEnt(eo.Status)),
		AmountMinor:    eo.TotalCents,
		Currency:       eo.Currency,
		IdempotencyKey: eo.IdempotencyKey,
	}

	// Map line snapshots from edges.
	if lines, err := eo.Edges.LinesOrErr(); err == nil {
		o.Lines = make([]OrderLineSnapshot, 0, len(lines))
		for _, el := range lines {
			line := OrderLineSnapshot{
				ProductID:      el.ProductID,
				SKU:            el.ProductSku,
				DisplayName:    el.ProductName,
				Quantity:       el.Quantity,
				UnitPriceMinor: el.UnitPriceCents,
				SubtotalMinor:  el.SubtotalCents,
			}
			// Decode snapshot_data for extra fields.
			if el.SnapshotData != nil {
				if raw, ok := el.SnapshotData["line"].(string); ok {
					_ = json.Unmarshal([]byte(raw), &line)
				}
			}
			o.Lines = append(o.Lines, line)
		}
	}

	return o
}

// must1 is a helper for enum-mapper calls that returns the value or panics on
// error. The mappers only error on unknown enum values, which would indicate a
// bug at the call site (the domain type guarantees valid values at the API
// boundary).
func must1[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

// --- Enum mappers ---

func mapKindToEnt(k ProductKind) (commerceproduct.Kind, error) {
	switch k {
	case ProductKindPlanPurchase:
		return commerceproduct.KindPlanPurchase, nil
	case ProductKindSubscriptionRenewal:
		return commerceproduct.KindSubscriptionRenewal, nil
	case ProductKindWalletTopUp:
		return commerceproduct.KindWalletTopUp, nil
	default:
		return "", fmt.Errorf("unknown product kind: %s", k)
	}
}

func mapKindFromEnt(k commerceproduct.Kind) (ProductKind, error) {
	switch k {
	case commerceproduct.KindPlanPurchase:
		return ProductKindPlanPurchase, nil
	case commerceproduct.KindSubscriptionRenewal:
		return ProductKindSubscriptionRenewal, nil
	case commerceproduct.KindWalletTopUp:
		return ProductKindWalletTopUp, nil
	default:
		return "", fmt.Errorf("unknown ent product kind: %s", k)
	}
}

func mapOrderKindToEnt(k OrderKind) (commerceorder.Kind, error) {
	switch k {
	case OrderKindPlanPurchase:
		return commerceorder.KindPlanPurchase, nil
	case OrderKindSubscriptionRenewal:
		return commerceorder.KindSubscriptionRenewal, nil
	case OrderKindWalletTopUp:
		return commerceorder.KindWalletTopUp, nil
	default:
		return "", fmt.Errorf("unknown order kind: %s", k)
	}
}

func mapOrderKindFromEnt(k commerceorder.Kind) (OrderKind, error) {
	switch k {
	case commerceorder.KindPlanPurchase:
		return OrderKindPlanPurchase, nil
	case commerceorder.KindSubscriptionRenewal:
		return OrderKindSubscriptionRenewal, nil
	case commerceorder.KindWalletTopUp:
		return OrderKindWalletTopUp, nil
	default:
		return "", fmt.Errorf("unknown ent order kind: %s", k)
	}
}

func mapOrderStatusToEnt(s OrderStatus) (commerceorder.Status, error) {
	switch s {
	case OrderStatusCreated:
		return commerceorder.StatusCreated, nil
	case OrderStatusAwaitingPayment:
		return commerceorder.StatusAwaitingPayment, nil
	case OrderStatusPaid:
		return commerceorder.StatusPaid, nil
	case OrderStatusFulfilled:
		return commerceorder.StatusFulfilled, nil
	case OrderStatusCancelled:
		return commerceorder.StatusCancelled, nil
	case OrderStatusExpired:
		return commerceorder.StatusExpired, nil
	case OrderStatusRefundPending:
		return commerceorder.StatusRefundPending, nil
	case OrderStatusPartiallyRefunded:
		return commerceorder.StatusPartiallyRefunded, nil
	case OrderStatusRefunded:
		return commerceorder.StatusRefunded, nil
	default:
		return "", fmt.Errorf("unknown order status: %s", s)
	}
}

func mapOrderStatusFromEnt(s commerceorder.Status) (OrderStatus, error) {
	switch s {
	case commerceorder.StatusCreated:
		return OrderStatusCreated, nil
	case commerceorder.StatusAwaitingPayment:
		return OrderStatusAwaitingPayment, nil
	case commerceorder.StatusPaid:
		return OrderStatusPaid, nil
	case commerceorder.StatusFulfilled:
		return OrderStatusFulfilled, nil
	case commerceorder.StatusCancelled:
		return OrderStatusCancelled, nil
	case commerceorder.StatusExpired:
		return OrderStatusExpired, nil
	case commerceorder.StatusRefundPending:
		return OrderStatusRefundPending, nil
	case commerceorder.StatusPartiallyRefunded:
		return OrderStatusPartiallyRefunded, nil
	case commerceorder.StatusRefunded:
		return OrderStatusRefunded, nil
	default:
		return "", fmt.Errorf("unknown ent order status: %s", s)
	}
}
