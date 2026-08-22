// Package commerce implements the Phase 2 commerce domain: wallet read-model,
// product catalog, and order lifecycle. The Wallet is a read-only projection
// over the immutable Credit Ledger — it never holds a mutable second balance.
// Orders follow a strict state machine and snapshot product attributes at
// creation time so later catalog edits never mutate existing orders.
package commerce

import (
	"time"

	"github.com/openmeterio/openmeter/pkg/models"
)

// ---------------------------------------------------------------------------
// Wallet (read model)
// ---------------------------------------------------------------------------

// BucketSource identifies the origin of a credit bucket and determines its
// consumption priority. The burn order is plan < gift < recharge <
// enterprise_receivable.
type BucketSource string

const (
	BucketSourcePlan                 BucketSource = "plan"
	BucketSourceGift                 BucketSource = "gift"
	BucketSourceRecharge             BucketSource = "recharge"
	BucketSourceEnterpriseReceivable BucketSource = "enterprise_receivable"
)

// SourcePriority maps a BucketSource to its fixed consumption priority. Lower
// values burn first.
func SourcePriority(s BucketSource) int {
	switch s {
	case BucketSourcePlan:
		return 10
	case BucketSourceGift:
		return 20
	case BucketSourceRecharge:
		return 30
	case BucketSourceEnterpriseReceivable:
		return 40
	default:
		return 100
	}
}

// WalletBucket is a single credit bucket projected from the Credit Ledger. It
// is a read model — callers must never mutate AvailableCredits directly.
type WalletBucket struct {
	Source            BucketSource `json:"source"`
	AvailableCredits  int64        `json:"available_credits"`
	ExpiresAt         *time.Time   `json:"expires_at,omitempty"`
	RefundableCredits *int64       `json:"refundable_credits,omitempty"`
}

// WalletTransactionKind enumerates the movement types on the Wallet.
type WalletTransactionKind string

const (
	TransactionKindFunded   WalletTransactionKind = "funded"
	TransactionKindConsumed WalletTransactionKind = "consumed"
	TransactionKindExpired  WalletTransactionKind = "expired"
	TransactionKindRefunded WalletTransactionKind = "refunded"
	TransactionKindAdjusted WalletTransactionKind = "adjusted"
)

// LedgerProvenance links a Wallet transaction to its underlying ledger entry
// without exposing internal Ent IDs.
type LedgerProvenance struct {
	GrantID  string       `json:"grant_id"`
	Priority int          `json:"priority"`
	Source   BucketSource `json:"source"`
}

// WalletTransaction is an immutable credit movement projected from the Ledger.
type WalletTransaction struct {
	ID         string                `json:"id"`
	Kind       WalletTransactionKind `json:"kind"`
	Amount     int64                 `json:"amount"` // signed: positive adds, negative deducts
	Provenance LedgerProvenance      `json:"provenance"`
	OccurredAt time.Time             `json:"occurred_at"`
}

// Wallet is the aggregate read view for a customer. It is recomputed on every
// retrieval from the Ledger and never stores a separate balance.
type Wallet struct {
	CustomerID            string              `json:"customer_id"`
	ContractVersion       string              `json:"contract_version"`
	TotalAvailableCredits int64               `json:"total_available_credits"`
	Buckets               []WalletBucket      `json:"buckets"`
	Transactions          []WalletTransaction `json:"transactions,omitempty"`
	RetrievedAt           time.Time           `json:"retrieved_at"`
}

// ContractVersion is bumped when the Wallet response shape changes. Clients use
// it for compatibility detection.
const ContractVersion = "commerce.phase2.v1"

// ---------------------------------------------------------------------------
// Catalog
// ---------------------------------------------------------------------------

// ProductKind classifies what a catalog entry purchases.
type ProductKind string

const (
	ProductKindPlanPurchase        ProductKind = "plan_purchase"
	ProductKindSubscriptionRenewal ProductKind = "subscription_renewal"
	ProductKindWalletTopUp         ProductKind = "wallet_top_up"
)

// RefundPolicy defines how a purchased product may be refunded.
type RefundPolicy string

const (
	RefundPolicyNone       RefundPolicy = "none"
	RefundPolicyUnspent    RefundPolicy = "unspent"
	RefundPolicyFullWindow RefundPolicy = "full_window"
)

// Product is a catalog entry for a purchasable item. Version, SKU and Credit
// fields are immutable once published; price and on-sale interval may change
// but never retroactively affect existing orders.
type Product struct {
	models.NamespacedID
	models.ManagedModel

	Version         int          `json:"version"`
	SKU             string       `json:"sku"`
	DisplayName     string       `json:"display_name"`
	Kind            ProductKind  `json:"kind"`
	Credits         int64        `json:"credits"`
	AmountMinor     int64        `json:"amount_minor"`
	Currency        string       `json:"currency"`
	Active          bool         `json:"active"`
	DisplayOrder    int          `json:"display_order"`
	ApplicablePlans []string     `json:"applicable_plans,omitempty"`
	OnSaleAt        *time.Time   `json:"on_sale_at,omitempty"`
	OffSaleAt       *time.Time   `json:"off_sale_at,omitempty"`
	PurchaseLimit   int          `json:"purchase_limit,omitempty"`
	RefundPolicy    RefundPolicy `json:"refund_policy"`
	ValidityDays    int          `json:"validity_days,omitempty"`
	BonusCredits    int64        `json:"bonus_credits,omitempty"`
	Description     string       `json:"description,omitempty"`
	Metadata        []byte       `json:"metadata,omitempty"`
}

// ---------------------------------------------------------------------------
// Orders
// ---------------------------------------------------------------------------

// OrderKind mirrors the API contract order kinds.
type OrderKind string

const (
	OrderKindPlanPurchase        OrderKind = "plan_purchase"
	OrderKindSubscriptionRenewal OrderKind = "subscription_renewal"
	OrderKindWalletTopUp         OrderKind = "wallet_top_up"
)

// OrderStatus is the lifecycle status of an order, governed by a state machine.
type OrderStatus string

const (
	OrderStatusCreated           OrderStatus = "created"
	OrderStatusAwaitingPayment   OrderStatus = "awaiting_payment"
	OrderStatusPaid              OrderStatus = "paid"
	OrderStatusFulfilled         OrderStatus = "fulfilled"
	OrderStatusCancelled         OrderStatus = "cancelled" //nolint:misspell // business domain value
	OrderStatusExpired           OrderStatus = "expired"
	OrderStatusRefundPending     OrderStatus = "refund_pending"
	OrderStatusPartiallyRefunded OrderStatus = "partially_refunded"
	OrderStatusRefunded          OrderStatus = "refunded"
)

// OrderLineSnapshot is an immutable line-item snapshot embedded in an order.
// It captures the product state at purchase time so later catalog edits never
// affect existing orders.
type OrderLineSnapshot struct {
	ProductID          string            `json:"product_id"`
	SKU                string            `json:"sku"`
	DisplayName        string            `json:"display_name"`
	Quantity           int32             `json:"quantity"`
	UnitPriceMinor     int64             `json:"unit_price_minor"`
	SubtotalMinor      int64             `json:"subtotal_minor"`
	Credits            int64             `json:"credits"`
	Currency           string            `json:"currency"`
	IncludedPlanCredit int64             `json:"included_plan_credit,omitempty"`
	ValidityDays       int               `json:"validity_days,omitempty"`
	Metadata           map[string]string `json:"metadata,omitempty"`
}

// Order is a commerce purchase intent with immutable total, currency, and lines.
// Only status transitions are allowed after creation.
type Order struct {
	models.NamespacedID
	models.ManagedModel

	PublicID               string              `json:"public_id"`
	CustomerID             string              `json:"customer_id"`
	Kind                   OrderKind           `json:"kind"`
	Status                 OrderStatus         `json:"status"`
	AmountMinor            int64               `json:"amount_minor"`
	Currency               string              `json:"currency"`
	IdempotencyKey         string              `json:"idempotency_key"`
	Lines                  []OrderLineSnapshot `json:"lines"`
	BusinessTrackingNumber *string             `json:"business_tracking_number,omitempty"`
	ExpiredAt              *time.Time          `json:"expired_at,omitempty"`
}
