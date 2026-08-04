// Package enterprise implements the Phase 2 enterprise monthly receivables and
// limited offline authorization lifecycle.
//
// Enterprise customers consume Credit beyond their prepaid buckets against a
// credit line (receivable). The lifecycle is:
//
//  1. An enterprise customer gets a ReceivableAccount with a credit limit.
//  2. Each billing period opens a ReceivablePeriod that accrues settled Credit
//     usage (the usage that fell through prepaid buckets into the receivable).
//  3. Usage beyond prepaid buckets accrues only when an active receivable
//     account AND a valid (signed, unexpired, under-ceiling) offline
//     authorization exist.
//  4. Closing a period freezes the settlement range (period_start..period_end)
//     and the amount (total_minor). The frozen amount cannot be rewritten.
//  5. Offline payments (bank transfers) are recorded against a period. They
//     require finance-admin authorization, a bank reference, payer, amount,
//     currency, received time and an evidence hash. Each application writes an
//     immutable audit fact; corrections use reversing entries rather than
//     mutating or deleting prior facts.
//  6. The versioned collection policy drives grace/collection transitions:
//     overdue -> restricted -> suspended. These publish auditable events and do
//     NOT rewrite historical usage.
//  7. External invoice references expose only external_system,
//     external_invoice_id, status, issued_at and updated_at. No tax
//     calculations, invoice PDFs or fiscal numbers are generated.
//
// Design rules (from the phase 2 brief):
//   - Credits are int64 — no floats.
//   - Money uses int64 minor units + ISO 4217 currency.
//   - Enterprise authorization is signed with customer, subject, maximum
//     Credit, issued time, expiry and nonce.
//   - Default offline window is 24h; configured value may be lower but never
//     greater than 72h.
//   - Two Tenant (namespace) entities under one enterprise remain isolated.
//   - Usage detail remains attributable by Tenant, user, Agent and API key.
//   - Use github.com/openmeterio/openmeter/pkg/clock for time.
package enterprise

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/openmeterio/openmeter/pkg/clock"
)

// DefaultOfflineWindow is the default validity window for an offline
// authorization when none is configured.
const DefaultOfflineWindow = 24 * time.Hour

// MaxOfflineWindow is the hard ceiling on the offline authorization window. A
// configured value greater than this is clamped down. This is a safety
// invariant: offline authorizations must never remain valid for more than 72h.
const MaxOfflineWindow = 72 * time.Hour

// CollectionState is the grace/collection policy state of a receivable account.
// The policy advances monotonically: active -> overdue -> restricted ->
// suspended. Transitions never go backwards; they publish auditable events and
// never rewrite historical usage.
type CollectionState string

const (
	CollectionStateActive     CollectionState = "active"
	CollectionStateOverdue    CollectionState = "overdue"
	CollectionStateRestricted CollectionState = "restricted"
	CollectionStateSuspended  CollectionState = "suspended"
)

// IsTerminal reports whether the state is a terminal (most-severe) collection
// state.
func (s CollectionState) IsTerminal() bool {
	return s == CollectionStateSuspended
}

// collectionSeverity ranks collection states so transitions can only escalate.
func collectionSeverity(s CollectionState) int {
	switch s {
	case CollectionStateActive:
		return 0
	case CollectionStateOverdue:
		return 1
	case CollectionStateRestricted:
		return 2
	case CollectionStateSuspended:
		return 3
	default:
		return -1
	}
}

// PeriodStatus is the lifecycle status of a receivable period. It mirrors the
// Ent schema enum: open, closed, partially_paid, paid, overdue.
type PeriodStatus string

const (
	PeriodStatusOpen          PeriodStatus = "open"
	PeriodStatusClosed        PeriodStatus = "closed"
	PeriodStatusPartiallyPaid PeriodStatus = "partially_paid"
	PeriodStatusPaid          PeriodStatus = "paid"
	PeriodStatusOverdue       PeriodStatus = "overdue"
)

// ReceivableAccount models an enterprise customer's credit account. One account
// per (namespace, customer_id). current_balance_minor is negative when the
// customer owes money, down to -credit_limit_minor.
type ReceivableAccount struct {
	ID                  string
	Namespace           string
	CustomerID          string
	CreditLimitMinor    int64 // money credit limit (schema: credit_limit_cents)
	CurrentBalanceMinor int64 // money balance; negative = owes (schema: current_balance_cents)
	CeilingCredits      int64 // Credit ceiling for the current open period
	UsedCredits         int64 // Credits drawn against the current open period
	Currency            string
	CollectionState     CollectionState
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// ReceivablePeriod is a billing period within a ReceivableAccount. period_start
// and period_end are immutable. The total is frozen when the period is closed.
type ReceivablePeriod struct {
	ID           string
	Namespace    string
	AccountID    string
	Status       PeriodStatus
	PeriodStart  time.Time
	PeriodEnd    time.Time
	TotalCredits int64 // settled Credit usage accrued (snapshot at close)
	TotalMinor   int64 // frozen money amount (schema: total_cents)
	PaidMinor    int64 // applied money amount (schema: paid_cents)
	Currency     string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// OutstandingMinor returns the unpaid money for the period. It is always
// non-negative once the period is closed (the total is frozen).
func (p ReceivablePeriod) OutstandingMinor() int64 {
	out := p.TotalMinor - p.PaidMinor
	if out < 0 {
		return 0
	}
	return out
}

// IsPaid reports whether the period is fully paid.
func (p ReceivablePeriod) IsPaid() bool {
	return p.TotalMinor > 0 && p.PaidMinor >= p.TotalMinor
}

// UsageAccrual is an immutable record of settled Credit usage that fell through
// prepaid buckets into the receivable. It carries full attribution by Tenant
// (namespace), user, Agent and API key.
type UsageAccrual struct {
	ID          string
	Namespace   string
	PeriodID    string
	CustomerID  string
	Credits     int64
	RateMinor   int64 // money per Credit (the rate snapshot)
	AmountMinor int64 // Credits * RateMinor
	UserID      string
	AgentID     string
	APIKeyID    string
	OccurredAt  time.Time
	CreatedAt   time.Time
}

// OfflineAuthorization is a signed authorization that permits receivable usage.
// It is signed with customer, subject, maximum Credit, issued time, expiry and
// nonce. The nonce makes each authorization unique; the signature makes it
// tamper-proof.
type OfflineAuthorization struct {
	Namespace  string
	CustomerID string
	Subject    string // the principal holding the authorization
	MaxCredit  int64  // Credit ceiling
	IssuedAt   time.Time
	Expiry     time.Time
	Nonce      string
	Signature  string
}

// AuthorizationPayload is the canonical signed content of an authorization.
type AuthorizationPayload struct {
	CustomerID string
	Subject    string
	MaxCredit  int64
	IssuedAt   time.Time
	Expiry     time.Time
	Nonce      string
}

// PaymentFactKind distinguishes an applied payment from a reversing entry.
type PaymentFactKind string

const (
	PaymentFactApplied  PaymentFactKind = "applied"
	PaymentFactReversed PaymentFactKind = "reversed"
)

// PaymentAuditFact is an immutable audit record of an offline payment or a
// reversing correction. Applied facts add to a period's paid amount; reversed
// facts subtract. Corrections never mutate or delete prior facts.
type PaymentAuditFact struct {
	ID                 string
	Namespace          string
	AccountID          string
	PeriodID           string
	Kind               PaymentFactKind
	AmountMinor        int64  // always non-negative; Kind determines sign
	ReferencePaymentID string // non-empty for reversals
	BankReference      string
	Payer              string
	Currency           string
	ReceivedAt         time.Time
	EvidenceHash       string
	ConfirmedBy        string
	Note               string
	CreatedAt          time.Time
}

// SignedAmountMinor returns the contribution of this fact to a period's paid
// amount: positive for applied facts, negative for reversals.
func (f PaymentAuditFact) SignedAmountMinor() int64 {
	if f.Kind == PaymentFactReversed {
		return -f.AmountMinor
	}
	return f.AmountMinor
}

// InvoiceRefStatus is the lifecycle status of an external invoice reference.
type InvoiceRefStatus string

const (
	InvoiceRefDraft  InvoiceRefStatus = "draft"
	InvoiceRefIssued InvoiceRefStatus = "issued"
	InvoiceRefVoid   InvoiceRefStatus = "void"
	InvoiceRefPaid   InvoiceRefStatus = "paid"
)

// ExternalInvoiceRef links a receivable period to an external invoicing system.
// It exposes only external_system, external_invoice_id, status, issued_at and
// updated_at. No tax calculations, invoice PDFs or fiscal numbers are stored.
type ExternalInvoiceRef struct {
	ID                string
	Namespace         string
	PeriodID          string
	ExternalSystem    string
	ExternalInvoiceID string
	Status            InvoiceRefStatus
	IssuedAt          *time.Time
	UpdatedAt         time.Time
}

// AuditEvent is an auditable event published by collection policy transitions.
type AuditEvent struct {
	Namespace  string
	AccountID  string
	CustomerID string
	Kind       string // e.g. "collection_state_changed", "period_closed"
	FromState  CollectionState
	ToState    CollectionState
	Reason     string
	OccurredAt time.Time
}

// CollectionPolicy is the versioned contract policy that controls grace length,
// collection notifications, restricted operations and suspension.
type CollectionPolicy struct {
	Version          string
	GracePeriod      time.Duration // time after period end before overdue
	RestrictionDelay time.Duration // time overdue before restricted
	SuspensionDelay  time.Duration // time restricted before suspended
}

// DefaultCollectionPolicy is a sane default policy.
var DefaultCollectionPolicy = CollectionPolicy{
	Version:          "collection.v1",
	GracePeriod:      7 * 24 * time.Hour,
	RestrictionDelay: 14 * 24 * time.Hour,
	SuspensionDelay:  30 * 24 * time.Hour,
}

// OpenAccountInput opens (or returns the existing) receivable account.
type OpenAccountInput struct {
	Namespace        string
	CustomerID       string
	CreditLimitMinor int64
	CeilingCredits   int64
	Currency         string
}

// IssueAuthorizationInput issues a signed offline authorization. The expiry is
// computed from the configured offline window (default 24h, clamped to 72h).
type IssueAuthorizationInput struct {
	Namespace  string
	CustomerID string
	Subject    string
	MaxCredit  int64
	Nonce      string
	IssuedAt   time.Time // optional; defaults to clock.Now
}

// AccrueUsageInput accrues settled Credit usage into the open receivable period.
type AccrueUsageInput struct {
	Namespace  string
	CustomerID string
	Nonce      string // the authorization nonce permitting this usage
	Credits    int64
	RateMinor  int64
	UserID     string
	AgentID    string
	APIKeyID   string
	OccurredAt time.Time
}

// ClosePeriodInput closes the open period, freezing the settlement range and
// amount.
type ClosePeriodInput struct {
	Namespace string
	AccountID string
}

// RecordPaymentInput records an offline payment. It requires finance-admin
// authorization, a bank reference, payer, amount, currency, received time and
// an evidence hash.
type RecordPaymentInput struct {
	Namespace         string
	AccountID         string
	PeriodID          string
	AmountMinor       int64
	Currency          string
	BankReference     string
	Payer             string
	ReceivedAt        time.Time
	EvidenceHash      string
	ConfirmedBy       string // the finance-admin actor
	FinanceAdminProof string // proof of finance-admin role
	Note              string
}

// ReversePaymentInput writes a reversing entry against a prior payment. Like a
// payment, it requires finance-admin authorization.
type ReversePaymentInput struct {
	Namespace         string
	PaymentFactID     string
	FinanceAdminProof string
	ConfirmedBy       string
	Reason            string
}

// RecordInvoiceRefInput records an external invoice reference.
type RecordInvoiceRefInput struct {
	Namespace         string
	PeriodID          string
	ExternalSystem    string
	ExternalInvoiceID string
	Status            InvoiceRefStatus
	InvoiceURL        string
}

// Repository manages all enterprise persistence. Implementations must be safe
// for concurrent use.
type Repository interface {
	// --- Accounts ---
	CreateAccount(ctx context.Context, account ReceivableAccount) (*ReceivableAccount, error)
	GetAccount(ctx context.Context, namespace, customerID string) (*ReceivableAccount, error)
	GetAccountByID(ctx context.Context, namespace, id string) (*ReceivableAccount, error)
	SetCollectionState(ctx context.Context, namespace, id string, state CollectionState) (*ReceivableAccount, error)

	// --- Periods ---
	CreatePeriod(ctx context.Context, period ReceivablePeriod) (*ReceivablePeriod, error)
	GetOpenPeriod(ctx context.Context, namespace, accountID string) (*ReceivablePeriod, error)
	GetPeriod(ctx context.Context, namespace, id string) (*ReceivablePeriod, error)
	ListPeriods(ctx context.Context, namespace, accountID string) ([]ReceivablePeriod, error)
	ClosePeriod(ctx context.Context, namespace, id string, totalCredits, totalMinor int64) (*ReceivablePeriod, error)
	SetPeriodPaid(ctx context.Context, namespace, id string, paidMinor int64, status PeriodStatus) (*ReceivablePeriod, error)

	// --- Usage accruals ---
	AppendUsage(ctx context.Context, accrual UsageAccrual) (*ReceivablePeriod, error)
	ListUsage(ctx context.Context, namespace, periodID string) ([]UsageAccrual, error)

	// --- Offline payments / audit facts ---
	AppendPaymentFact(ctx context.Context, fact PaymentAuditFact) (*PaymentAuditFact, *ReceivablePeriod, error)
	GetPaymentFact(ctx context.Context, namespace, id string) (*PaymentAuditFact, error)
	ListPaymentFacts(ctx context.Context, namespace, periodID string) ([]PaymentAuditFact, error)

	// --- Authorizations ---
	SaveAuthorization(ctx context.Context, auth OfflineAuthorization) (*OfflineAuthorization, error)
	GetAuthorization(ctx context.Context, namespace, nonce string) (*OfflineAuthorization, error)

	// --- External invoices ---
	AppendInvoiceRef(ctx context.Context, ref ExternalInvoiceRef) (*ExternalInvoiceRef, error)
	ListInvoiceRefs(ctx context.Context, namespace, periodID string) ([]ExternalInvoiceRef, error)
	UpdateInvoiceRefStatus(ctx context.Context, namespace, id string, status InvoiceRefStatus) (*ExternalInvoiceRef, error)

	// --- Audit events ---
	AppendEvent(ctx context.Context, event AuditEvent) error
}

// AuthorizationSigner signs and verifies offline authorizations. The signature
// covers customer, subject, maximum Credit, issued time, expiry and nonce.
type AuthorizationSigner interface {
	Sign(payload AuthorizationPayload) (string, error)
	Verify(payload AuthorizationPayload, signature string) error
}

// RoleAuthorizer checks that an actor holds a required role. Offline payment
// entry requires finance-admin authorization.
type RoleAuthorizer interface {
	IsFinanceAdmin(ctx context.Context, actor, proof string) (bool, error)
}

// EventPublisher publishes auditable collection/policy events.
type EventPublisher interface {
	Publish(ctx context.Context, event AuditEvent) error
}

// IDGenerator produces unique IDs for new records.
type IDGenerator interface {
	NewID() string
}

// HMACSigner is a default AuthorizationSigner using HMAC-SHA256. It is safe
// for concurrent use. The secret must come from a secret manager in production.
type HMACSigner struct {
	Secret []byte
}

// Sign produces an HMAC-SHA256 signature over the canonical payload.
func (h HMACSigner) Sign(p AuthorizationPayload) (string, error) {
	mac := hmac.New(sha256.New, h.Secret)
	mac.Write([]byte(canonicalPayload(p)))
	return hex.EncodeToString(mac.Sum(nil)), nil
}

// Verify checks the signature against the canonical payload.
func (h HMACSigner) Verify(p AuthorizationPayload, signature string) error {
	want, err := h.Sign(p)
	if err != nil {
		return err
	}
	if !hmac.Equal([]byte(want), []byte(signature)) {
		return errors.New("signature mismatch")
	}
	return nil
}

func canonicalPayload(p AuthorizationPayload) string {
	return fmt.Sprintf("%s|%s|%d|%d|%d|%s",
		p.CustomerID, p.Subject, p.MaxCredit,
		p.IssuedAt.UnixNano(), p.Expiry.UnixNano(), p.Nonce)
}

// Errors.
var (
	ErrAccountNotFound       = errors.New("enterprise: receivable account not found")
	ErrPeriodNotFound        = errors.New("enterprise: receivable period not found")
	ErrNoOpenPeriod          = errors.New("enterprise: no open period")
	ErrPeriodNotOpen         = errors.New("enterprise: period is not open")
	ErrAuthorizationNotFound = errors.New("enterprise: authorization not found")
	ErrAuthorizationExpired  = errors.New("enterprise: authorization expired")
	ErrAuthorizationInvalid  = errors.New("enterprise: authorization signature invalid")
	ErrAboveCeiling          = errors.New("enterprise: usage above credit ceiling")
	ErrCeilingExhausted      = errors.New("enterprise: credit ceiling exhausted")
	ErrAccountRestricted     = errors.New("enterprise: account is restricted or suspended")
	ErrFinanceAdminRequired  = errors.New("enterprise: finance-admin authorization required")
	ErrPaymentNotFound       = errors.New("enterprise: payment fact not found")
	ErrInvalidInput          = errors.New("enterprise: invalid input")
	ErrNonceReuse            = errors.New("enterprise: authorization nonce already issued")
)

// Service implements the enterprise receivables and offline authorization
// lifecycle.
type Service interface {
	OpenAccount(ctx context.Context, in OpenAccountInput) (*ReceivableAccount, error)
	GetAccount(ctx context.Context, namespace, customerID string) (*ReceivableAccount, error)
	EnsureOpenPeriod(ctx context.Context, namespace, accountID string) (*ReceivablePeriod, error)
	IssueAuthorization(ctx context.Context, in IssueAuthorizationInput) (*OfflineAuthorization, error)
	Authorize(ctx context.Context, namespace, customerID, nonce string, credits int64) error
	AccrueUsage(ctx context.Context, in AccrueUsageInput) (*ReceivablePeriod, error)
	ClosePeriod(ctx context.Context, in ClosePeriodInput) (*ReceivablePeriod, error)
	RecordPayment(ctx context.Context, in RecordPaymentInput) (*ReceivablePeriod, error)
	ReversePayment(ctx context.Context, in ReversePaymentInput) (*ReceivablePeriod, error)
	RecordInvoiceRef(ctx context.Context, in RecordInvoiceRefInput) (*ExternalInvoiceRef, error)
	UpdateInvoiceRefStatus(ctx context.Context, namespace, id string, status InvoiceRefStatus) (*ExternalInvoiceRef, error)
	EvaluateCollection(ctx context.Context, namespace, accountID string) (*ReceivableAccount, error)
}

// Config wires the enterprise service.
type Config struct {
	Repo          Repository
	Signer        AuthorizationSigner
	Roles         RoleAuthorizer
	Events        EventPublisher
	IDs           IDGenerator
	OfflineWindow time.Duration // default 24h if zero; clamped to MaxOfflineWindow (72h)
	Policy        CollectionPolicy
	Logger        *slog.Logger
}

type service struct {
	repo          Repository
	signer        AuthorizationSigner
	roles         RoleAuthorizer
	events        EventPublisher
	ids           IDGenerator
	offlineWindow time.Duration
	policy        CollectionPolicy
	logger        *slog.Logger
}

// New creates an enterprise Service.
func New(cfg Config) (Service, error) {
	if cfg.Repo == nil {
		return nil, errors.New("enterprise: repository is required")
	}
	if cfg.Signer == nil {
		return nil, errors.New("enterprise: authorization signer is required")
	}
	if cfg.Roles == nil {
		return nil, errors.New("enterprise: role authorizer is required")
	}

	window := cfg.OfflineWindow
	if window <= 0 {
		window = DefaultOfflineWindow
	}
	// Clamp: configured value may be lower than the default but never greater
	// than 72 hours.
	if window > MaxOfflineWindow {
		window = MaxOfflineWindow
	}

	policy := cfg.Policy
	if policy.Version == "" {
		policy = DefaultCollectionPolicy
	}

	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	ids := cfg.IDs
	if ids == nil {
		ids = randomIDGen{}
	}

	return &service{
		repo:          cfg.Repo,
		signer:        cfg.Signer,
		roles:         cfg.Roles,
		events:        cfg.Events,
		ids:           ids,
		offlineWindow: window,
		policy:        policy,
		logger:        logger,
	}, nil
}

// OfflineWindow returns the effective (clamped) offline window.
func (s *service) OfflineWindow() time.Duration {
	return s.offlineWindow
}

// OpenAccount opens (or returns the existing) receivable account for a customer.
func (s *service) OpenAccount(ctx context.Context, in OpenAccountInput) (*ReceivableAccount, error) {
	if err := validateOpenAccount(in); err != nil {
		return nil, err
	}

	// Idempotency: return existing account for this customer.
	if existing, err := s.repo.GetAccount(ctx, in.Namespace, in.CustomerID); err == nil && existing != nil {
		return existing, nil
	}

	now := clock.Now()
	acc := ReceivableAccount{
		ID:                  s.ids.NewID(),
		Namespace:           in.Namespace,
		CustomerID:          in.CustomerID,
		CreditLimitMinor:    in.CreditLimitMinor,
		CurrentBalanceMinor: 0,
		CeilingCredits:      in.CeilingCredits,
		UsedCredits:         0,
		Currency:            in.Currency,
		CollectionState:     CollectionStateActive,
		CreatedAt:           now,
		UpdatedAt:           now,
	}
	return s.repo.CreateAccount(ctx, acc)
}

// GetAccount retrieves the account by namespace and customer.
func (s *service) GetAccount(ctx context.Context, namespace, customerID string) (*ReceivableAccount, error) {
	return s.repo.GetAccount(ctx, namespace, customerID)
}

// EnsureOpenPeriod returns the open period for an account, creating one if none
// exists. A new period spans [now, now + 1 month) by default.
func (s *service) EnsureOpenPeriod(ctx context.Context, namespace, accountID string) (*ReceivablePeriod, error) {
	acc, err := s.repo.GetAccountByID(ctx, namespace, accountID)
	if err != nil {
		return nil, err
	}

	if existing, err := s.repo.GetOpenPeriod(ctx, namespace, accountID); err == nil && existing != nil {
		return existing, nil
	}

	now := clock.Now()
	period := ReceivablePeriod{
		ID:          s.ids.NewID(),
		Namespace:   namespace,
		AccountID:   accountID,
		Status:      PeriodStatusOpen,
		PeriodStart: now,
		PeriodEnd:   now.AddDate(0, 1, 0),
		Currency:    acc.Currency,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	return s.repo.CreatePeriod(ctx, period)
}

// IssueAuthorization issues a signed offline authorization.
func (s *service) IssueAuthorization(ctx context.Context, in IssueAuthorizationInput) (*OfflineAuthorization, error) {
	if err := validateIssueAuthorization(in); err != nil {
		return nil, err
	}

	if _, err := s.repo.GetAccount(ctx, in.Namespace, in.CustomerID); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrAccountNotFound, err)
	}

	// Nonce must be unique.
	if existing, err := s.repo.GetAuthorization(ctx, in.Namespace, in.Nonce); err == nil && existing != nil {
		return nil, ErrNonceReuse
	}

	issuedAt := in.IssuedAt
	if issuedAt.IsZero() {
		issuedAt = clock.Now()
	}
	expiry := issuedAt.Add(s.offlineWindow)

	payload := AuthorizationPayload{
		CustomerID: in.CustomerID,
		Subject:    in.Subject,
		MaxCredit:  in.MaxCredit,
		IssuedAt:   issuedAt,
		Expiry:     expiry,
		Nonce:      in.Nonce,
	}
	sig, err := s.signer.Sign(payload)
	if err != nil {
		return nil, fmt.Errorf("enterprise: sign authorization: %w", err)
	}

	auth := OfflineAuthorization{
		Namespace:  in.Namespace,
		CustomerID: in.CustomerID,
		Subject:    in.Subject,
		MaxCredit:  in.MaxCredit,
		IssuedAt:   issuedAt,
		Expiry:     expiry,
		Nonce:      in.Nonce,
		Signature:  sig,
	}
	return s.repo.SaveAuthorization(ctx, auth)
}

// Authorize checks that the authorization permits usage of the requested
// credits. It enforces the ceiling and expiry but does NOT consume credits.
func (s *service) Authorize(ctx context.Context, namespace, customerID, nonce string, credits int64) error {
	if credits < 0 {
		return fmt.Errorf("%w: credits must be non-negative", ErrInvalidInput)
	}

	acc, err := s.repo.GetAccount(ctx, namespace, customerID)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrAccountNotFound, err)
	}
	// Restricted or suspended accounts cannot draw new receivable usage.
	if collectionSeverity(acc.CollectionState) >= collectionSeverity(CollectionStateRestricted) {
		return fmt.Errorf("%w: state=%s", ErrAccountRestricted, acc.CollectionState)
	}

	auth, err := s.repo.GetAuthorization(ctx, namespace, nonce)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrAuthorizationNotFound, err)
	}

	payload := AuthorizationPayload{
		CustomerID: auth.CustomerID,
		Subject:    auth.Subject,
		MaxCredit:  auth.MaxCredit,
		IssuedAt:   auth.IssuedAt,
		Expiry:     auth.Expiry,
		Nonce:      auth.Nonce,
	}
	if err := s.signer.Verify(payload, auth.Signature); err != nil {
		return fmt.Errorf("%w: %v", ErrAuthorizationInvalid, err)
	}

	// Expiry: authorization rejects use after expiry.
	if !clock.Now().Before(auth.Expiry) {
		return fmt.Errorf("%w: expiry=%s", ErrAuthorizationExpired, auth.Expiry.Format(time.RFC3339))
	}

	// Ceiling: used + requested must not exceed the authorization's maximum.
	if auth.MaxCredit-acc.UsedCredits < credits {
		return fmt.Errorf("%w: used=%d requested=%d ceiling=%d", ErrAboveCeiling, acc.UsedCredits, credits, auth.MaxCredit)
	}
	return nil
}

// AccrueUsage accrues settled Credit usage into the open receivable period.
func (s *service) AccrueUsage(ctx context.Context, in AccrueUsageInput) (*ReceivablePeriod, error) {
	if err := validateAccrueUsage(in); err != nil {
		return nil, err
	}

	acc, err := s.repo.GetAccount(ctx, in.Namespace, in.CustomerID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrAccountNotFound, err)
	}
	// Usage accrues only when an active account exists (active or overdue are
	// permitted; restricted/suspended block new usage).
	if collectionSeverity(acc.CollectionState) >= collectionSeverity(CollectionStateRestricted) {
		return nil, fmt.Errorf("%w: state=%s", ErrAccountRestricted, acc.CollectionState)
	}

	// A valid authorization must exist (ceiling + expiry enforced).
	if err := s.Authorize(ctx, in.Namespace, in.CustomerID, in.Nonce, in.Credits); err != nil {
		return nil, err
	}

	period, err := s.repo.GetOpenPeriod(ctx, in.Namespace, acc.ID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrNoOpenPeriod, err)
	}

	occurredAt := in.OccurredAt
	if occurredAt.IsZero() {
		occurredAt = clock.Now()
	}
	amount := in.Credits * in.RateMinor
	accrual := UsageAccrual{
		ID:          s.ids.NewID(),
		Namespace:   in.Namespace,
		PeriodID:    period.ID,
		CustomerID:  in.CustomerID,
		Credits:     in.Credits,
		RateMinor:   in.RateMinor,
		AmountMinor: amount,
		UserID:      in.UserID,
		AgentID:     in.AgentID,
		APIKeyID:    in.APIKeyID,
		OccurredAt:  occurredAt,
		CreatedAt:   clock.Now(),
	}
	return s.repo.AppendUsage(ctx, accrual)
}

// ClosePeriod closes the open period, freezing the settlement range and amount.
func (s *service) ClosePeriod(ctx context.Context, in ClosePeriodInput) (*ReceivablePeriod, error) {
	if in.Namespace == "" || in.AccountID == "" {
		return nil, fmt.Errorf("%w: namespace and account_id are required", ErrInvalidInput)
	}

	period, err := s.repo.GetOpenPeriod(ctx, in.Namespace, in.AccountID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrNoOpenPeriod, err)
	}
	if period.Status != PeriodStatusOpen {
		return nil, fmt.Errorf("%w: status=%s", ErrPeriodNotOpen, period.Status)
	}

	// Close freezes the settlement range and amount. The accrued usage total is
	// snapshotted into total_credits/total_minor and can no longer change.
	usage, err := s.repo.ListUsage(ctx, in.Namespace, period.ID)
	if err != nil {
		return nil, fmt.Errorf("enterprise: list usage: %w", err)
	}
	var totalCredits, totalMinor int64
	for _, u := range usage {
		totalCredits += u.Credits
		totalMinor += u.AmountMinor
	}

	closed, err := s.repo.ClosePeriod(ctx, in.Namespace, period.ID, totalCredits, totalMinor)
	if err != nil {
		return nil, fmt.Errorf("enterprise: close period: %w", err)
	}

	s.publishEvent(ctx, AuditEvent{
		Namespace:  in.Namespace,
		AccountID:  in.AccountID,
		Kind:       "period_closed",
		ToState:    CollectionStateActive,
		Reason:     fmt.Sprintf("period %s closed: %d credits, %d minor", closed.ID, totalCredits, totalMinor),
		OccurredAt: clock.Now(),
	})

	return closed, nil
}

// RecordPayment records an offline payment against a period.
func (s *service) RecordPayment(ctx context.Context, in RecordPaymentInput) (*ReceivablePeriod, error) {
	if err := validateRecordPayment(in); err != nil {
		return nil, err
	}

	// Finance-admin authorization is required.
	ok, err := s.roles.IsFinanceAdmin(ctx, in.ConfirmedBy, in.FinanceAdminProof)
	if err != nil {
		return nil, fmt.Errorf("enterprise: check finance-admin: %w", err)
	}
	if !ok {
		return nil, ErrFinanceAdminRequired
	}

	period, err := s.repo.GetPeriod(ctx, in.Namespace, in.PeriodID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPeriodNotFound, err)
	}
	// Payments may only be applied to closed periods (the amount must be frozen
	// before money is collected against it).
	if period.Status == PeriodStatusOpen {
		return nil, fmt.Errorf("%w: close the period before recording payments", ErrPeriodNotOpen)
	}

	fact := PaymentAuditFact{
		ID:            s.ids.NewID(),
		Namespace:     in.Namespace,
		AccountID:     period.AccountID,
		PeriodID:      in.PeriodID,
		Kind:          PaymentFactApplied,
		AmountMinor:   in.AmountMinor,
		BankReference: in.BankReference,
		Payer:         in.Payer,
		Currency:      orDefault(in.Currency, period.Currency),
		ReceivedAt:    in.ReceivedAt,
		EvidenceHash:  in.EvidenceHash,
		ConfirmedBy:   in.ConfirmedBy,
		Note:          in.Note,
		CreatedAt:     clock.Now(),
	}
	_, updated, err := s.repo.AppendPaymentFact(ctx, fact)
	if err != nil {
		return nil, fmt.Errorf("enterprise: append payment fact: %w", err)
	}
	return updated, nil
}

// ReversePayment writes a reversing entry against a prior payment.
func (s *service) ReversePayment(ctx context.Context, in ReversePaymentInput) (*ReceivablePeriod, error) {
	if in.Namespace == "" || in.PaymentFactID == "" {
		return nil, fmt.Errorf("%w: namespace and payment_fact_id are required", ErrInvalidInput)
	}

	ok, err := s.roles.IsFinanceAdmin(ctx, in.ConfirmedBy, in.FinanceAdminProof)
	if err != nil {
		return nil, fmt.Errorf("enterprise: check finance-admin: %w", err)
	}
	if !ok {
		return nil, ErrFinanceAdminRequired
	}

	orig, err := s.repo.GetPaymentFact(ctx, in.Namespace, in.PaymentFactID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPaymentNotFound, err)
	}

	reversal := PaymentAuditFact{
		ID:                 s.ids.NewID(),
		Namespace:          in.Namespace,
		AccountID:          orig.AccountID,
		PeriodID:           orig.PeriodID,
		Kind:               PaymentFactReversed,
		AmountMinor:        orig.AmountMinor,
		ReferencePaymentID: orig.ID,
		BankReference:      orig.BankReference,
		Payer:              orig.Payer,
		Currency:           orig.Currency,
		ReceivedAt:         orig.ReceivedAt,
		EvidenceHash:       orig.EvidenceHash,
		ConfirmedBy:        in.ConfirmedBy,
		Note:               orDefault(in.Reason, "reversing entry"),
		CreatedAt:          clock.Now(),
	}
	_, updated, err := s.repo.AppendPaymentFact(ctx, reversal)
	if err != nil {
		return nil, fmt.Errorf("enterprise: append reversal fact: %w", err)
	}
	return updated, nil
}

// RecordInvoiceRef records an external invoice reference. No tax document
// content is stored.
func (s *service) RecordInvoiceRef(ctx context.Context, in RecordInvoiceRefInput) (*ExternalInvoiceRef, error) {
	if err := validateRecordInvoiceRef(in); err != nil {
		return nil, err
	}

	now := clock.Now()
	ref := ExternalInvoiceRef{
		ID:                s.ids.NewID(),
		Namespace:         in.Namespace,
		PeriodID:          in.PeriodID,
		ExternalSystem:    in.ExternalSystem,
		ExternalInvoiceID: in.ExternalInvoiceID,
		Status:            in.Status,
		UpdatedAt:         now,
	}
	if in.Status == InvoiceRefIssued || in.Status == InvoiceRefPaid {
		t := now
		ref.IssuedAt = &t
	}
	return s.repo.AppendInvoiceRef(ctx, ref)
}

// UpdateInvoiceRefStatus updates the status of an external invoice reference.
func (s *service) UpdateInvoiceRefStatus(ctx context.Context, namespace, id string, status InvoiceRefStatus) (*ExternalInvoiceRef, error) {
	if namespace == "" || id == "" {
		return nil, fmt.Errorf("%w: namespace and id are required", ErrInvalidInput)
	}
	return s.repo.UpdateInvoiceRefStatus(ctx, namespace, id, status)
}

// EvaluateCollection drives the versioned collection policy.
//
// The policy advances monotonically: active -> overdue -> restricted ->
// suspended. It is time-based:
//   - active -> overdue when a closed period's end is older than GracePeriod and
//     the period is not fully paid.
//   - overdue -> restricted after RestrictionDelay in the overdue state.
//   - restricted -> suspended after SuspensionDelay in the restricted state.
//
// The repository records UpdatedAt on each transition, which serves as the
// entry time for the current state. State changes publish auditable events and
// never rewrite historical usage.
func (s *service) EvaluateCollection(ctx context.Context, namespace, accountID string) (*ReceivableAccount, error) {
	acc, err := s.repo.GetAccountByID(ctx, namespace, accountID)
	if err != nil {
		return nil, err
	}

	// Suspended is terminal — no further transitions.
	if acc.CollectionState.IsTerminal() {
		return acc, nil
	}

	now := clock.Now()
	target := acc.CollectionState

	switch acc.CollectionState {
	case CollectionStateActive:
		if s.hasUnpaidPastGrace(ctx, namespace, accountID, now) {
			target = CollectionStateOverdue
		}
	case CollectionStateOverdue:
		if now.Sub(acc.UpdatedAt) >= s.policy.RestrictionDelay {
			target = CollectionStateRestricted
		}
	case CollectionStateRestricted:
		if now.Sub(acc.UpdatedAt) >= s.policy.SuspensionDelay {
			target = CollectionStateSuspended
		}
	}

	if collectionSeverity(target) <= collectionSeverity(acc.CollectionState) {
		return acc, nil
	}

	from := acc.CollectionState
	updated, err := s.repo.SetCollectionState(ctx, namespace, accountID, target)
	if err != nil {
		return nil, fmt.Errorf("enterprise: set collection state: %w", err)
	}

	s.publishEvent(ctx, AuditEvent{
		Namespace:  namespace,
		AccountID:  accountID,
		CustomerID: acc.CustomerID,
		Kind:       "collection_state_changed",
		FromState:  from,
		ToState:    target,
		Reason:     fmt.Sprintf("policy %s transition %s -> %s", s.policy.Version, from, target),
		OccurredAt: now,
	})

	return updated, nil
}

// hasUnpaidPastGrace reports whether the account has a closed period that is not
// fully paid and whose end is older than the grace period. This is the trigger
// for the active -> overdue transition.
func (s *service) hasUnpaidPastGrace(ctx context.Context, namespace, accountID string, now time.Time) bool {
	periods, err := s.repo.ListPeriods(ctx, namespace, accountID)
	if err != nil {
		return false
	}
	for _, p := range periods {
		// Only closed periods with an outstanding balance can trigger overdue.
		if p.Status == PeriodStatusOpen {
			continue
		}
		if p.IsPaid() {
			continue
		}
		if now.Sub(p.PeriodEnd) >= s.policy.GracePeriod {
			return true
		}
	}
	return false
}

// publishEvent publishes an auditable event, logging but never failing the
// caller on publish errors.
func (s *service) publishEvent(ctx context.Context, event AuditEvent) {
	if s.events != nil {
		if err := s.events.Publish(ctx, event); err != nil {
			s.logger.WarnContext(ctx, "enterprise: publish event failed", "kind", event.Kind, "error", err)
		}
	}
}

// Validation helpers.

func validateOpenAccount(in OpenAccountInput) error {
	var errs []error
	if in.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}
	if in.CustomerID == "" {
		errs = append(errs, errors.New("customer_id is required"))
	}
	if in.CreditLimitMinor < 0 {
		errs = append(errs, errors.New("credit_limit_minor must be non-negative"))
	}
	if in.CeilingCredits < 0 {
		errs = append(errs, errors.New("ceiling_credits must be non-negative"))
	}
	if in.Currency == "" {
		errs = append(errs, errors.New("currency is required"))
	}
	if len(errs) > 0 {
		return fmt.Errorf("%w: %v", ErrInvalidInput, errors.Join(errs...))
	}
	return nil
}

func validateIssueAuthorization(in IssueAuthorizationInput) error {
	var errs []error
	if in.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}
	if in.CustomerID == "" {
		errs = append(errs, errors.New("customer_id is required"))
	}
	if in.Subject == "" {
		errs = append(errs, errors.New("subject is required"))
	}
	if in.MaxCredit < 0 {
		errs = append(errs, errors.New("max_credit must be non-negative"))
	}
	if in.Nonce == "" {
		errs = append(errs, errors.New("nonce is required"))
	}
	if len(errs) > 0 {
		return fmt.Errorf("%w: %v", ErrInvalidInput, errors.Join(errs...))
	}
	return nil
}

func validateAccrueUsage(in AccrueUsageInput) error {
	var errs []error
	if in.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}
	if in.CustomerID == "" {
		errs = append(errs, errors.New("customer_id is required"))
	}
	if in.Nonce == "" {
		errs = append(errs, errors.New("nonce is required"))
	}
	if in.Credits < 0 {
		errs = append(errs, errors.New("credits must be non-negative"))
	}
	if in.RateMinor < 0 {
		errs = append(errs, errors.New("rate_minor must be non-negative"))
	}
	if len(errs) > 0 {
		return fmt.Errorf("%w: %v", ErrInvalidInput, errors.Join(errs...))
	}
	return nil
}

func validateRecordPayment(in RecordPaymentInput) error {
	var errs []error
	if in.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}
	if in.PeriodID == "" {
		errs = append(errs, errors.New("period_id is required"))
	}
	if in.AmountMinor <= 0 {
		errs = append(errs, errors.New("amount_minor must be positive"))
	}
	if in.Currency == "" {
		errs = append(errs, errors.New("currency is required"))
	}
	if in.BankReference == "" {
		errs = append(errs, errors.New("bank_reference is required"))
	}
	if in.Payer == "" {
		errs = append(errs, errors.New("payer is required"))
	}
	if in.ReceivedAt.IsZero() {
		errs = append(errs, errors.New("received_at is required"))
	}
	if in.EvidenceHash == "" {
		errs = append(errs, errors.New("evidence_hash is required"))
	}
	if in.ConfirmedBy == "" {
		errs = append(errs, errors.New("confirmed_by is required"))
	}
	if in.FinanceAdminProof == "" {
		errs = append(errs, errors.New("finance_admin_proof is required"))
	}
	if len(errs) > 0 {
		return fmt.Errorf("%w: %v", ErrInvalidInput, errors.Join(errs...))
	}
	return nil
}

func validateRecordInvoiceRef(in RecordInvoiceRefInput) error {
	var errs []error
	if in.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}
	if in.PeriodID == "" {
		errs = append(errs, errors.New("period_id is required"))
	}
	if in.ExternalSystem == "" {
		errs = append(errs, errors.New("external_system is required"))
	}
	if in.ExternalInvoiceID == "" {
		errs = append(errs, errors.New("external_invoice_id is required"))
	}
	switch in.Status {
	case InvoiceRefDraft, InvoiceRefIssued, InvoiceRefVoid, InvoiceRefPaid:
	default:
		errs = append(errs, errors.New("status is invalid"))
	}
	if len(errs) > 0 {
		return fmt.Errorf("%w: %v", ErrInvalidInput, errors.Join(errs...))
	}
	return nil
}

func orDefault(v, def string) string {
	if v == "" {
		return def
	}
	return v
}

// randomIDGen is the default IDGenerator, producing hex-encoded random IDs.
type randomIDGen struct{}

func (randomIDGen) NewID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// Compile-time check.
var _ Service = (*service)(nil)
