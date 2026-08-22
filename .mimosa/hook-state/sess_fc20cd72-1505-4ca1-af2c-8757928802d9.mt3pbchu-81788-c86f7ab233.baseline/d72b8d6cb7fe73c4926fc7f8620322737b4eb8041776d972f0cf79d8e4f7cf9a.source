package commerce

import (
	"context"
	"time"
)

// OrderRepository is the persistence interface for orders. Implementations
// (Ent adapter, in-memory mock) must be safe for concurrent use.
type OrderRepository interface {
	// CreateOrder persists a new order. Returns the stored order and a boolean
	// indicating whether this was a fresh insert (true) or an idempotent replay
	// that returned an existing order (false).
	CreateOrder(ctx context.Context, order Order) (*Order, bool, error)

	// GetOrder retrieves an order by namespace and ID, including its line snapshots.
	GetOrder(ctx context.Context, namespace, id string) (*Order, error)

	// GetOrderByIdempotencyKey looks up an order by its idempotency key.
	GetOrderByIdempotencyKey(ctx context.Context, namespace, customerID, key string) (*Order, error)

	// UpdateOrderStatus applies a status transition atomically. It must reject
	// transitions where the current status does not match expectedFrom.
	UpdateOrderStatus(ctx context.Context, namespace, id string, expectedFrom, to OrderStatus) (*Order, error)

	// ListOrdersByCustomer lists orders for a customer, optionally filtered by status.
	ListOrdersByCustomer(ctx context.Context, namespace, customerID string, status *OrderStatus) ([]Order, error)
}

// ProductRepository is the persistence interface for catalog products.
type ProductRepository interface {
	// CreateProduct persists a new product.
	CreateProduct(ctx context.Context, product Product) (*Product, error)

	// GetProduct retrieves a product by namespace and ID.
	GetProduct(ctx context.Context, namespace, id string) (*Product, error)

	// GetProductBySKU retrieves a product by its SKU.
	GetProductBySKU(ctx context.Context, namespace, sku string) (*Product, error)

	// ListProducts lists products, optionally filtered by kind and active status.
	ListProducts(ctx context.Context, namespace string, kind *ProductKind, activeOnly bool) ([]Product, error)

	// UpdateProduct mutates a product's mutable fields (price, active, sale
	// interval, display order). Version and SKU are never changed.
	UpdateProduct(ctx context.Context, product Product) (*Product, error)
}

// AllocationGrant describes a credit grant from the Ledger that feeds a Wallet
// bucket. This is a read-model projection — it is never mutated by the commerce
// domain.
type AllocationGrant struct {
	GrantID    string
	Source     BucketSource
	Granted    int64
	Consumed   int64
	Priority   int
	ExpiresAt  *time.Time
	CreatedAt  time.Time
	Refundable int64
}

// EnterpriseReceivable describes the enterprise credit line for a customer.
type EnterpriseReceivable struct {
	AccountID      string
	CeilingCredits int64
	UsedCredits    int64
}

// WalletDataPort is the read interface the Wallet service needs. All data is
// sourced from the Credit Ledger and related immutable facts.
type WalletDataPort interface {
	// GetGrants returns all non-expired credit grants for the customer.
	GetGrants(ctx context.Context, namespace, customerID string) ([]AllocationGrant, error)

	// GetEnterpriseReceivable returns the enterprise receivable line, if any.
	GetEnterpriseReceivable(ctx context.Context, namespace, customerID string) (*EnterpriseReceivable, error)

	// GetRecentTransactions returns recent credit movements with Ledger provenance.
	GetRecentTransactions(ctx context.Context, namespace, customerID string, limit int) ([]WalletTransaction, error)
}

// CreditEngine is the interface the commerce domain uses to consume credits.
// It delegates to the Phase 1 settlement/collector layer.
type CreditEngine interface {
	// Debit charges credits using the fixed source priority. It returns the
	// allocations produced and an error if insufficient.
	Debit(ctx context.Context, namespace, customerID string, amount int64) (remaining int64, err error)
}
