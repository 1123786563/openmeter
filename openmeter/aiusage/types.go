package aiusage

import (
	"errors"
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/pkg/models"
)

// BillingMode determines how a line item or batch is charged.
type BillingMode string

const (
	// BillingModeComponent charges per resource line item.
	BillingModeComponent BillingMode = "component"

	// BillingModeBundle charges a flat ceiling amount with no per-resource breakdown.
	BillingModeBundle BillingMode = "bundle"
)

func (m BillingMode) Validate() error {
	switch m {
	case BillingModeComponent, BillingModeBundle:
		return nil
	default:
		return ErrInvalidBillingMode
	}
}

// BatchStatus tracks the lifecycle of an AI Usage Batch.
type BatchStatus string

const (
	BatchStatusPending     BatchStatus = "pending"
	BatchStatusSettled     BatchStatus = "settled"
	BatchStatusRejected    BatchStatus = "rejected"
	BatchStatusCompensated BatchStatus = "compensated"
)

func (s BatchStatus) Validate() error {
	switch s {
	case BatchStatusPending, BatchStatusSettled, BatchStatusRejected, BatchStatusCompensated:
		return nil
	default:
		return ErrInvalidBatchStatus
	}
}

// UsageLineItem represents one billable resource within a batch.
type UsageLineItem struct {
	// ResourceCode identifies the billable resource type.
	ResourceCode ResourceCode `json:"resource_code"`

	// Quantity is the raw consumption amount (tokens, calls, seconds, etc.).
	Quantity int64 `json:"quantity"`

	// Provider is the LLM vendor (e.g., "openai", "anthropic").
	Provider string `json:"provider"`

	// Model is the canonical model identifier.
	Model string `json:"model"`

	// ProviderManaged is true when the platform calls the provider's API.
	// BYOK (bring-your-own-key) resources have this set to false.
	ProviderManaged bool `json:"provider_managed"`

	// Dimensions carries optional resource-specific attributes.
	Dimensions map[string]string `json:"dimensions,omitempty"`
}

func (i UsageLineItem) Validate() error {
	var errs []error

	if err := i.ResourceCode.Validate(); err != nil {
		errs = append(errs, err)
	}

	if i.Quantity <= 0 {
		errs = append(errs, fmt.Errorf("quantity must be positive"))
	}

	// Provider-managed resources require provider and model identification for cost lookup.
	if i.ProviderManaged && i.Provider == "" {
		errs = append(errs, fmt.Errorf("provider must not be empty for provider-managed resource"))
	}
	if i.ProviderManaged && i.Model == "" {
		errs = append(errs, fmt.Errorf("model must not be empty for provider-managed resource"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// CorrectionInput reverses a previously settled batch by creating linked
// reversing allocations. The correction is itself a batch that offsets the
// original — useful for refunds, billing errors, or tenant-driven disputes.
type CorrectionInput struct {
	Namespace       string `json:"namespace"`
	CustomerID      string `json:"customer_id"`
	SubjectID       string `json:"subject_id"`
	OriginalBatchID string `json:"original_batch_id"`
	TenantSeq       int64  `json:"tenant_seq"`
	PayloadHash     string `json:"payload_hash"`
	Reason          string `json:"reason"`
}

func (i CorrectionInput) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, fmt.Errorf("namespace must not be empty"))
	}
	if i.CustomerID == "" {
		errs = append(errs, fmt.Errorf("customer_id must not be empty"))
	}
	if i.SubjectID == "" {
		errs = append(errs, fmt.Errorf("subject_id must not be empty"))
	}
	if i.OriginalBatchID == "" {
		errs = append(errs, fmt.Errorf("original_batch_id must not be empty"))
	}
	if i.TenantSeq <= 0 {
		errs = append(errs, fmt.Errorf("tenant_seq must be positive"))
	}
	if i.PayloadHash == "" {
		errs = append(errs, fmt.Errorf("payload_hash must not be empty"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// AIUsageBatch is the canonical unit of AI billing — one business action produces one batch.
type AIUsageBatch struct {
	models.ManagedModel `json:",inline"`

	// Namespace isolates tenants.
	Namespace string `json:"namespace"`

	// CustomerID is the Billing Customer (payer) in OpenMeter.
	CustomerID string `json:"customer_id"`

	// SubjectID is the Tenant/Subject that produced the usage.
	SubjectID string `json:"subject_id"`

	// UsageBatchID is the client-generated unique identifier for idempotency.
	UsageBatchID string `json:"usage_batch_id"`

	// TenantSeq is the monotonic per-tenant sequence number for watermark tracking.
	TenantSeq int64 `json:"tenant_seq"`

	// OccurredAt is when the business action happened (used for rate version resolution).
	OccurredAt time.Time `json:"occurred_at"`

	// ReservationID links to the WeKnora runtime reservation, if any.
	ReservationID *string `json:"reservation_id,omitempty"`

	// CeilingCredits caps the total Credit charge; platform absorbs any excess.
	CeilingCredits *int64 `json:"ceiling_credits,omitempty"`

	// RateVersion is the rate card version snapshot used for settlement.
	RateVersion string `json:"rate_version"`

	// BillingMode determines component vs bundle charging.
	BillingMode BillingMode `json:"billing_mode"`

	// PayloadHash is a SHA-256 of the canonical request body for idempotency verification.
	PayloadHash string `json:"payload_hash"`

	// Status is the current settlement status.
	Status BatchStatus `json:"status"`

	// LineItems contains the individual resource consumption entries.
	LineItems []UsageLineItem `json:"line_items"`
}

func (b AIUsageBatch) Validate() error {
	var errs []error

	if b.Namespace == "" {
		errs = append(errs, fmt.Errorf("namespace must not be empty"))
	}
	if b.CustomerID == "" {
		errs = append(errs, fmt.Errorf("customer_id must not be empty"))
	}
	if b.SubjectID == "" {
		errs = append(errs, fmt.Errorf("subject_id must not be empty"))
	}
	if b.UsageBatchID == "" {
		errs = append(errs, fmt.Errorf("usage_batch_id must not be empty"))
	}
	if b.TenantSeq <= 0 {
		errs = append(errs, fmt.Errorf("tenant_seq must be positive"))
	}
	if b.PayloadHash == "" {
		errs = append(errs, fmt.Errorf("payload_hash must not be empty"))
	}

	if err := b.BillingMode.Validate(); err != nil {
		errs = append(errs, err)
	}

	switch b.BillingMode {
	case BillingModeComponent:
		if len(b.LineItems) == 0 {
			errs = append(errs, fmt.Errorf("line_items must not be empty for component billing mode"))
		}
		for i, item := range b.LineItems {
			if err := item.Validate(); err != nil {
				errs = append(errs, fmt.Errorf("line_item[%d]: %w", i, err))
			}
		}
	case BillingModeBundle:
		if b.CeilingCredits == nil {
			errs = append(errs, fmt.Errorf("ceiling_credits must be set for bundle billing mode"))
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// CostSnapshot is the provider cost for a resource (typically in USD).
type CostSnapshot struct {
	Currency string            `json:"currency"`
	Amount   alpacadecimal.Decimal `json:"amount"`
	Source   string            `json:"source"`
}

// SalesSnapshot is the customer-facing price (typically in CNY for display).
type SalesSnapshot struct {
	Currency        string            `json:"currency"`
	Amount          alpacadecimal.Decimal `json:"amount"`
	RateCardVersion string            `json:"rate_card_version"`
}

// RatingSnapshot captures the cost and sales price resolution for one line item.
type RatingSnapshot struct {
	ResourceCode  ResourceCode  `json:"resource_code"`
	CostSnapshot  CostSnapshot  `json:"cost_snapshot"`
	SalesSnapshot SalesSnapshot `json:"sales_snapshot"`

	// Credits is the integer Credit charge for this line item.
	Credits int64 `json:"credits"`
}

// LedgerEntryRef references a grant that was burned during settlement.
type LedgerEntryRef struct {
	GrantID  string  `json:"grant_id"`
	Amount   float64 `json:"amount"`
	Priority uint8   `json:"priority"`
}

// FundingSource identifies the origin of credits consumed during settlement.
// The burn order is: plan → promotional → paid_top_up → enterprise_receivable.
type FundingSource string

const (
	FundingSourcePlan                 FundingSource = "plan"
	FundingSourcePromotional          FundingSource = "promotional"
	FundingSourcePaidTopUp            FundingSource = "paid_top_up"
	FundingSourceEnterpriseReceivable FundingSource = "enterprise_receivable"
)

// BatchSettlementResult is returned after processing a batch.
type BatchSettlementResult struct {
	BatchID          string           `json:"batch_id"`
	Status           BatchStatus      `json:"status"`
	TotalCredits     int64            `json:"total_credits"`
	RatingSnapshots  []RatingSnapshot `json:"rating_snapshots"`
	LedgerEntries    []LedgerEntryRef `json:"ledger_entries"`
	CoveredTenantSeq int64            `json:"covered_tenant_seq"`
}

// SettlementScope controls whether a batch affects the formal ledger.
type SettlementScope string

const (
	// SettlementScopeShadow persists the batch for visibility without deducting grants.
	SettlementScopeShadow SettlementScope = "shadow"

	// SettlementScopeFormal deducts grants and emits ledger/outbox events.
	SettlementScopeFormal SettlementScope = "formal"
)

func (s SettlementScope) Validate() error {
	switch s {
	case SettlementScopeShadow, SettlementScopeFormal:
		return nil
	default:
		return fmt.Errorf("invalid settlement scope: %s", s)
	}
}

// Allocation records the Credit deduction from a single funding source (grant).
type Allocation struct {
	GrantID       string        `json:"grant_id"`
	Amount        float64       `json:"amount"`
	Priority      uint8         `json:"priority"`
	FundingSource FundingSource `json:"funding_source"`
}

// OutboxEvent is a transactional outbox record published after commit.
type OutboxEvent struct {
	EventType string                 `json:"event_type"`
	Payload   map[string]interface{} `json:"payload"`
}

// SettledBatch is the input to the persistence adapter: a fully rated and
// settled batch ready for atomic storage with its line items, rating
// snapshots, credit allocations, and outbox events.
type SettledBatch struct {
	Namespace       string          `json:"namespace"`
	CustomerID      string          `json:"customer_id"`
	SubjectID       string          `json:"subject_id"`
	UsageBatchID    string          `json:"usage_batch_id"`
	TenantSeq       int64           `json:"tenant_seq"`
	OccurredAt      time.Time       `json:"occurred_at"`
	ReservationID   *string         `json:"reservation_id,omitempty"`
	CeilingCredits  *int64          `json:"ceiling_credits,omitempty"`
	RateVersion     string          `json:"rate_version"`
	BillingMode     BillingMode     `json:"billing_mode"`
	PayloadHash     string          `json:"payload_hash"`
	SettlementScope SettlementScope `json:"settlement_scope"`
	Status          BatchStatus     `json:"status"`
	TotalCredits    int64           `json:"total_credits"`

	LineItems       []UsageLineItem  `json:"line_items"`
	RatingSnapshots []RatingSnapshot `json:"rating_snapshots"`
	Allocations     []Allocation     `json:"allocations"`
	OutboxEvents    []OutboxEvent    `json:"outbox_events"`
}

func (b SettledBatch) Validate() error {
	var errs []error

	if b.Namespace == "" {
		errs = append(errs, fmt.Errorf("namespace must not be empty"))
	}
	if b.CustomerID == "" {
		errs = append(errs, fmt.Errorf("customer_id must not be empty"))
	}
	if b.SubjectID == "" {
		errs = append(errs, fmt.Errorf("subject_id must not be empty"))
	}
	if b.UsageBatchID == "" {
		errs = append(errs, fmt.Errorf("usage_batch_id must not be empty"))
	}
	if b.TenantSeq <= 0 {
		errs = append(errs, fmt.Errorf("tenant_seq must be positive"))
	}
	if b.PayloadHash == "" {
		errs = append(errs, fmt.Errorf("payload_hash must not be empty"))
	}

	if err := b.BillingMode.Validate(); err != nil {
		errs = append(errs, err)
	}
	if err := b.SettlementScope.Validate(); err != nil {
		errs = append(errs, err)
	}
	if err := b.Status.Validate(); err != nil {
		errs = append(errs, err)
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// Batch is the domain entity returned by the persistence adapter.
type Batch struct {
	models.ManagedModelWithID `json:",inline"`

	Namespace       string          `json:"namespace"`
	CustomerID      string          `json:"customer_id"`
	SubjectID       string          `json:"subject_id"`
	UsageBatchID    string          `json:"usage_batch_id"`
	TenantSeq       int64           `json:"tenant_seq"`
	OccurredAt      time.Time       `json:"occurred_at"`
	ReservationID   *string         `json:"reservation_id,omitempty"`
	CeilingCredits  *int64          `json:"ceiling_credits,omitempty"`
	RateVersion     string          `json:"rate_version"`
	BillingMode     BillingMode     `json:"billing_mode"`
	PayloadHash     string          `json:"payload_hash"`
	SettlementScope SettlementScope `json:"settlement_scope"`
	Status          BatchStatus     `json:"status"`
	TotalCredits    int64           `json:"total_credits"`
}

// IngestBatchInput is the API input for submitting a Canonical Usage Batch.
type IngestBatchInput struct {
	ProviderManaged bool `json:"provider_managed"`

	Namespace      string           `json:"namespace"`
	CustomerID     string           `json:"customer_id"`
	SubjectID      string           `json:"subject_id"`
	UsageBatchID   string           `json:"usage_batch_id"`
	TenantSeq      int64            `json:"tenant_seq"`
	OccurredAt     time.Time        `json:"occurred_at"`
	ReservationID  *string          `json:"reservation_id,omitempty"`
	CeilingCredits *int64           `json:"ceiling_credits,omitempty"`
	RateVersion    string           `json:"rate_version"`
	BillingMode    BillingMode      `json:"billing_mode"`
	PayloadHash    string           `json:"payload_hash"`
	LineItems      []UsageLineItem  `json:"line_items"`
}

// UsageLineInput is a normalized usage line for pricing resolution.
// Unlike UsageLineItem, this is the input to the pricing service before
// merging and rating. It carries a CanonicalLineIndex for stable ceiling
// allocation when the total exceeds the reservation ceiling.
type UsageLineInput struct {
	// ResourceCode identifies the billable resource type.
	ResourceCode ResourceCode `json:"resource_code"`

	// Quantity is the raw consumption amount (tokens, calls, seconds, etc.).
	Quantity int64 `json:"quantity"`

	// Provider is the LLM vendor (e.g., "openai", "anthropic").
	Provider string `json:"provider"`

	// Model is the canonical model identifier.
	Model string `json:"model"`

	// Dimensions carries optional resource-specific attributes.
	Dimensions map[string]string `json:"dimensions,omitempty"`

	// CanonicalLineIndex is the stable position used for deterministic
	// ceiling allocation when the total exceeds the reservation.
	CanonicalLineIndex int `json:"canonical_line_index"`
}

func (i IngestBatchInput) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, fmt.Errorf("namespace must not be empty"))
	}
	if i.CustomerID == "" {
		errs = append(errs, fmt.Errorf("customer_id must not be empty"))
	}
	if i.SubjectID == "" {
		errs = append(errs, fmt.Errorf("subject_id must not be empty"))
	}
	if i.UsageBatchID == "" {
		errs = append(errs, fmt.Errorf("usage_batch_id must not be empty"))
	}
	if i.TenantSeq <= 0 {
		errs = append(errs, fmt.Errorf("tenant_seq must be positive"))
	}
	if i.PayloadHash == "" {
		errs = append(errs, fmt.Errorf("payload_hash must not be empty"))
	}

	if err := i.BillingMode.Validate(); err != nil {
		errs = append(errs, err)
	}

	switch i.BillingMode {
	case BillingModeComponent:
		if len(i.LineItems) == 0 {
			errs = append(errs, fmt.Errorf("line_items must not be empty for component billing mode"))
		}
		for idx, item := range i.LineItems {
			if err := item.Validate(); err != nil {
				errs = append(errs, fmt.Errorf("line_item[%d]: %w", idx, err))
			}
		}
	case BillingModeBundle:
		if i.CeilingCredits == nil {
			errs = append(errs, fmt.Errorf("ceiling_credits must be set for bundle billing mode"))
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}
