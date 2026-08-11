package payment

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/oklog/ulid/v2"

	"github.com/openmeterio/openmeter/openmeter/commerce"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
)

// ---------------------------------------------------------------------------
// Repository interfaces
// ---------------------------------------------------------------------------

// AttemptRepository manages the PaymentAttempt lifecycle.
type AttemptRepository interface {
	// CreateAttempt persists a new payment attempt. Returns a boolean indicating
	// fresh insert (true) or idempotent replay (false).
	CreateAttempt(ctx context.Context, attempt PaymentAttempt) (*PaymentAttempt, bool, error)

	// GetAttempt retrieves an attempt by namespace and ID.
	GetAttempt(ctx context.Context, namespace, id string) (*PaymentAttempt, error)

	// GetAttemptByIdempotencyKey looks up an attempt by its idempotency key.
	GetAttemptByIdempotencyKey(ctx context.Context, namespace, customerID, key string) (*PaymentAttempt, error)

	// GetAttemptByProviderOrder looks up an attempt by namespace + provider +
	// provider order ID. The namespace must always be scoped — never empty.
	GetAttemptByProviderOrder(ctx context.Context, namespace string, provider Provider, providerOrderID string) (*PaymentAttempt, error)

	// UpdateAttemptStatus atomically updates the attempt status with optimistic
	// concurrency (expectedFrom).
	UpdateAttemptStatus(ctx context.Context, namespace, id string, expectedFrom, to AttemptStatus) (*PaymentAttempt, error)

	// SetProviderIDs sets the provider order/payment IDs and session on an
	// attempt after checkout.
	SetProviderIDs(ctx context.Context, namespace, id string, providerOrderID, providerPaymentID, sessionID string) (*PaymentAttempt, error)
}

// FactRepository manages immutable PaymentFact records.
type FactRepository interface {
	// InsertFact persists a PaymentFact. It must deduplicate on RawHash — if the
	// same immutable structured fact already exists, return it with (false,
	// nil); if the persisted structured fact conflicts, return an error.
	InsertFact(ctx context.Context, fact PaymentFactRecord) (*PaymentFactRecord, bool, error)

	// GetFactByRawHash retrieves a fact by its raw body hash (for dedup). It
	// returns commerce.ErrPaymentFactNotFound when no fact exists.
	GetFactByRawHash(ctx context.Context, namespace string, rawHash string) (*PaymentFactRecord, error)

	// GetFactsByProviderOrder retrieves all facts for a provider order.
	GetFactsByProviderOrder(ctx context.Context, namespace string, provider Provider, providerOrderID string) ([]PaymentFactRecord, error)

	// GetFactByProviderEvent retrieves a fact by provider event ID (dedup). It
	// returns commerce.ErrPaymentFactNotFound when no fact exists.
	GetFactByProviderEvent(ctx context.Context, namespace string, provider Provider, providerEventID string) (*PaymentFactRecord, error)
}

// OrderStatusUpdater transitions the order to paid. This is a narrow port that
// avoids coupling the payment service to the full order repository.
type OrderStatusUpdater interface {
	UpdateOrderStatus(ctx context.Context, namespace, id string, expectedFrom, to commerce.OrderStatus) (*commerce.Order, error)
	GetOrder(ctx context.Context, namespace, id string) (*commerce.Order, error)
}

// PaidTransitionInput carries everything needed for the atomic paid transition:
// inserting the payment fact, moving the order to paid, creating the fulfillment
// request, and writing an outbox record — all within one database transaction.
type PaidTransitionInput struct {
	Namespace      string
	Attempt        *PaymentAttempt
	Fact           PaymentFactRecord
	FulfillmentReq FulfillmentRequestCreate
}

// FulfillmentRequestCreate is the fulfillment request to create inside the same
// transaction as the paid transition.
type FulfillmentRequestCreate struct {
	OrderID    string
	CustomerID string
}

// OutboxRecord is a durable record written within the paid transition
// transaction so a worker can pick up the fulfillment request even if the
// process crashes immediately after commit.
type OutboxRecord struct {
	AggregateType string // e.g. "commerce_order"
	AggregateID   string // e.g. the order ID
	EventType     string // e.g. "order.paid"
	Payload       map[string]any
}

// PaidTransitionResult holds the outputs of the atomic paid transition.
type PaidTransitionResult struct {
	Order       *commerce.Order
	Fact        *PaymentFactRecord
	AlreadyPaid bool
}

// PaidTxRunner executes the paid transition — insert fact, move order to paid,
// create fulfillment request, write outbox — within one database transaction.
// If any step fails the entire transition rolls back. The implementation joins
// a single Ent transaction and propagates it through context.
type PaidTxRunner interface {
	RunPaidTransition(ctx context.Context, in PaidTransitionInput) (PaidTransitionResult, error)
}

// ---------------------------------------------------------------------------
// Domain types
// ---------------------------------------------------------------------------

// AttemptStatus is the lifecycle status of a payment attempt.
type AttemptStatus string

const (
	AttemptStatusCreated   AttemptStatus = "created"
	AttemptStatusPending   AttemptStatus = "pending"
	AttemptStatusSucceeded AttemptStatus = "succeeded"
	AttemptStatusFailed    AttemptStatus = "failed"
	AttemptStatusClosed    AttemptStatus = "closed"
)

// PaymentAttempt tracks one attempt to pay an order through a provider. The
// MerchantID and ApplicationID fields capture the expected provider identity so
// callbacks can be validated against them (I2).
type PaymentAttempt struct {
	ID                    string
	Namespace             string
	OrderID               string
	CustomerID            string
	Provider              Provider
	ProviderOrderID       string
	ProviderPaymentID     string
	Status                AttemptStatus
	ProviderSessionID     string
	IdempotencyKey        string
	AmountMinor           int64
	Currency              string
	ExpectedMerchantID    string // expected merchant ID from provider config
	ExpectedApplicationID string // expected app ID from provider config
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

// PaymentFactRecord is the persisted form of a verified PaymentFact, linked to
// its payment attempt.
type PaymentFactRecord struct {
	ID                string
	Namespace         string
	AttemptID         string
	Provider          Provider
	ProviderOrderID   string
	ProviderPaymentID string
	ProviderEventID   string
	MerchantID        string
	ApplicationID     string
	AmountMinor       int64
	Currency          string
	Success           bool
	RawHash           string
	Timestamp         time.Time
	SignedPayload     map[string]any
	CreatedAt         time.Time
}

// CreateAttemptInput creates a new payment attempt.
type CreateAttemptInput struct {
	Namespace             string
	OrderID               string
	CustomerID            string
	Provider              Provider
	IdempotencyKey        string
	AmountMinor           int64
	Currency              string
	ExpectedMerchantID    string
	ExpectedApplicationID string
}

// ---------------------------------------------------------------------------
// Service
// ---------------------------------------------------------------------------

// Service is the payment fact verification and attempt lifecycle service.
// It implements the paid-versus-fulfilled separation: a verified fact moves the
// order to paid (not fulfilled). Fulfillment is a separate step.
type Service interface {
	CreateAttempt(ctx context.Context, in CreateAttemptInput) (*PaymentAttempt, bool, error)
	GetAttempt(ctx context.Context, namespace, id string) (*PaymentAttempt, error)
	InitiateCheckout(ctx context.Context, namespace, attemptID string) (CheckoutResult, error)

	// HandleCallback is the entry point for provider callbacks. The namespace
	// parameter scopes the attempt lookup — callbacks are always tenant-scoped
	// because the callback URL is registered per-tenant. It verifies the
	// signature, deduplicates, checks the fact against the attempt's order
	// (amount, currency, merchant, application identity), persists the fact,
	// and transitions the order to paid. Repeated callbacks return the original
	// result.
	HandleCallback(ctx context.Context, namespace string, providerName Provider, headers map[string][]string, body []byte) (CallbackResult, error)

	// ConfirmPayment queries the provider directly (callback-lost recovery).
	ConfirmPayment(ctx context.Context, namespace, attemptID string) (CallbackResult, error)
}

// CheckoutResult holds the provider QR code and updated attempt.
type CheckoutResult struct {
	Attempt *PaymentAttempt
	Fact    CheckoutFact
}

// CallbackResult describes the outcome of a callback or confirmation.
type CallbackResult struct {
	Attempt     *PaymentAttempt
	Fact        *PaymentFactRecord
	AlreadyPaid bool // true if the order was already paid (idempotent replay)
}

// Config wires the payment service.
type Config struct {
	Attempts  AttemptRepository
	Facts     FactRepository
	Orders    OrderStatusUpdater
	TxRunner  PaidTxRunner
	Providers map[Provider]ProviderAdapter
	Logger    *slog.Logger
}

type service struct {
	attempts  AttemptRepository
	facts     FactRepository
	orders    OrderStatusUpdater
	txRunner  PaidTxRunner
	providers map[Provider]ProviderAdapter
	logger    *slog.Logger
}

// New creates a payment Service.
func New(cfg Config) (Service, error) {
	if cfg.Attempts == nil {
		return nil, errors.New("payment service: attempts repository is required")
	}
	if cfg.Facts == nil {
		return nil, errors.New("payment service: facts repository is required")
	}
	if cfg.Orders == nil {
		return nil, errors.New("payment service: orders port is required")
	}
	if cfg.TxRunner == nil {
		return nil, errors.New("payment service: paid transaction runner is required")
	}
	if cfg.Providers == nil {
		return nil, errors.New("payment service: providers map is required")
	}
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}
	return &service{
		attempts:  cfg.Attempts,
		facts:     cfg.Facts,
		orders:    cfg.Orders,
		txRunner:  cfg.TxRunner,
		providers: cfg.Providers,
		logger:    logger,
	}, nil
}

// CreateAttempt creates a new payment attempt idempotently.
func (s *service) CreateAttempt(ctx context.Context, in CreateAttemptInput) (*PaymentAttempt, bool, error) {
	if err := validateCreateAttempt(in); err != nil {
		return nil, false, err
	}

	// Idempotency: check for existing attempt.
	existing, err := s.attempts.GetAttemptByIdempotencyKey(ctx, in.Namespace, in.CustomerID, in.IdempotencyKey)
	if err == nil && existing != nil {
		return existing, false, nil
	}

	identity, err := s.providerIdentity(ctx, in.Provider)
	if err != nil {
		return nil, false, err
	}

	now := clock.Now()
	attempt := PaymentAttempt{
		ID:                    ulid.Make().String(),
		Namespace:             in.Namespace,
		OrderID:               in.OrderID,
		CustomerID:            in.CustomerID,
		Provider:              in.Provider,
		Status:                AttemptStatusCreated,
		IdempotencyKey:        in.IdempotencyKey,
		AmountMinor:           in.AmountMinor,
		Currency:              in.Currency,
		ExpectedMerchantID:    identity.MerchantID,
		ExpectedApplicationID: identity.ApplicationID,
		CreatedAt:             now,
		UpdatedAt:             now,
	}

	return s.attempts.CreateAttempt(ctx, attempt)
}

func (s *service) providerIdentity(ctx context.Context, providerName Provider) (ProviderIdentity, error) {
	provider, ok := s.providers[providerName]
	if !ok || provider == nil {
		return ProviderIdentity{}, &ProviderError{
			Provider:  providerName,
			Operation: "identity",
			Kind:      ProviderErrorPermanent,
			Cause:     fmt.Errorf("%w: provider is disabled or not configured", ErrPermanentProviderProtocol),
		}
	}

	identity, err := provider.Identity(ctx)
	if err != nil {
		return ProviderIdentity{}, &ProviderError{
			Provider:  providerName,
			Operation: "identity",
			Kind:      ProviderErrorPermanent,
			Cause:     fmt.Errorf("%w: %v", ErrPermanentProviderProtocol, err),
		}
	}
	identity.MerchantID = strings.TrimSpace(identity.MerchantID)
	identity.ApplicationID = strings.TrimSpace(identity.ApplicationID)
	if identity.MerchantID == "" || identity.ApplicationID == "" {
		return ProviderIdentity{}, &ProviderError{
			Provider:  providerName,
			Operation: "identity",
			Kind:      ProviderErrorPermanent,
			Cause:     fmt.Errorf("%w: merchant and application identity are required", ErrPermanentProviderProtocol),
		}
	}

	return identity, nil
}

// GetAttempt retrieves a payment attempt.
func (s *service) GetAttempt(ctx context.Context, namespace, id string) (*PaymentAttempt, error) {
	return s.attempts.GetAttempt(ctx, namespace, id)
}

// InitiateCheckout calls the provider to create a QR code and stores the
// provider IDs on the attempt.
func (s *service) InitiateCheckout(ctx context.Context, namespace, attemptID string) (CheckoutResult, error) {
	attempt, err := s.attempts.GetAttempt(ctx, namespace, attemptID)
	if err != nil {
		return CheckoutResult{}, err
	}

	provider, ok := s.providers[attempt.Provider]
	if !ok {
		return CheckoutResult{}, fmt.Errorf("payment: provider %s not configured", attempt.Provider)
	}

	// Fetch order for the public ID and amount.
	order, err := s.orders.GetOrder(ctx, namespace, attempt.OrderID)
	if err != nil {
		return CheckoutResult{}, fmt.Errorf("payment: get order for checkout: %w", err)
	}

	fact, err := provider.CreateQRCode(ctx, CheckoutInput{
		Namespace:      namespace,
		OrderID:        attempt.OrderID,
		OrderPublicID:  order.PublicID,
		CustomerID:     attempt.CustomerID,
		AmountMinor:    attempt.AmountMinor,
		Currency:       attempt.Currency,
		IdempotencyKey: attempt.IdempotencyKey,
	})
	if err != nil {
		return CheckoutResult{}, fmt.Errorf("payment: provider create qr code: %w", err)
	}

	// Store provider IDs and transition to pending.
	_, err = s.attempts.SetProviderIDs(ctx, namespace, attemptID, fact.ProviderOrderID, fact.ProviderPaymentID, fact.QRCodeURL)
	if err != nil {
		return CheckoutResult{}, err
	}
	updated, err := s.attempts.UpdateAttemptStatus(ctx, namespace, attemptID, AttemptStatusCreated, AttemptStatusPending)
	if err != nil {
		return CheckoutResult{}, err
	}

	// Transition the order from created to awaiting_payment so the paid-tx
	// runner can later move it to paid when the callback arrives.
	if _, err := s.orders.UpdateOrderStatus(ctx, namespace, attempt.OrderID, commerce.OrderStatusCreated, commerce.OrderStatusAwaitingPayment); err != nil {
		// If already awaiting_payment (idempotent checkout), this is fine.
		s.logger.WarnContext(ctx, "payment: order status transition to awaiting_payment", "error", err, "orderID", attempt.OrderID)
	}

	return CheckoutResult{Attempt: updated, Fact: fact}, nil
}

// HandleCallback processes a provider callback: verify signature, deduplicate,
// match against the order, persist fact, transition to paid.
//
// The namespace parameter scopes the attempt lookup. Payment callback URLs are
// registered per-tenant, so the HTTP handler always knows the namespace.
func (s *service) HandleCallback(ctx context.Context, namespace string, providerName Provider, headers map[string][]string, body []byte) (CallbackResult, error) {
	provider, ok := s.providers[providerName]
	if !ok {
		return CallbackResult{}, &ProviderError{
			Provider:  providerName,
			Operation: "callback",
			Kind:      ProviderErrorPermanent,
			Cause:     fmt.Errorf("%w: provider is disabled or not configured", ErrPermanentProviderProtocol),
		}
	}

	// Verify the signature — the adapter does this and extracts verified fields.
	pf, err := provider.VerifyCallback(ctx, toHTTPHeader(headers), body)
	if err != nil {
		return CallbackResult{}, err
	}

	return s.applyPaymentFact(ctx, namespace, pf)
}

// ConfirmPayment queries the provider directly (callback-lost recovery).
func (s *service) ConfirmPayment(ctx context.Context, namespace, attemptID string) (CallbackResult, error) {
	attempt, err := s.attempts.GetAttempt(ctx, namespace, attemptID)
	if err != nil {
		return CallbackResult{}, err
	}

	provider, ok := s.providers[attempt.Provider]
	if !ok {
		return CallbackResult{}, &ProviderError{
			Provider:  attempt.Provider,
			Operation: "confirm",
			Kind:      ProviderErrorPermanent,
			Cause:     fmt.Errorf("%w: provider is disabled or not configured", ErrPermanentProviderProtocol),
		}
	}

	if attempt.ProviderOrderID == "" {
		return CallbackResult{}, errors.New("payment: attempt has no provider order id to confirm")
	}

	pf, err := provider.QueryPayment(ctx, attempt.ProviderOrderID)
	if err != nil {
		return CallbackResult{}, fmt.Errorf("payment: provider query payment: %w", err)
	}

	result, err := s.applyPaymentFact(ctx, attempt.Namespace, pf)
	if err != nil {
		return CallbackResult{}, err
	}
	if !pf.Success && pf.Terminal {
		updated, err := s.attempts.UpdateAttemptStatus(ctx, attempt.Namespace, attempt.ID, AttemptStatusPending, AttemptStatusFailed)
		if err != nil {
			return CallbackResult{}, fmt.Errorf("payment: mark definitively failed attempt: %w", err)
		}
		result.Attempt = updated
	}
	return result, nil
}

// applyPaymentFact is the core atom: deduplicate the fact, match it against the
// attempt/order, persist it, and transition the order to paid. This method is
// idempotent: repeated callbacks with the same fact converge to one paid order.
//
// The namespace parameter scopes the attempt lookup — it is always provided by
// the caller and never empty.
func (s *service) applyPaymentFact(ctx context.Context, namespace string, pf PaymentFact) (CallbackResult, error) {
	// Locate the attempt by namespace + provider + provider order ID.
	attempt, err := s.attempts.GetAttemptByProviderOrder(ctx, namespace, pf.Provider, pf.ProviderOrderID)
	if err != nil {
		wrapped := fmt.Errorf("payment: locate attempt by provider order %s in namespace %s: %w", pf.ProviderOrderID, namespace, err)
		if errors.Is(err, commerce.ErrPaymentAttemptNotFound) || errors.Is(err, ErrPaymentAttemptNotFound) {
			return CallbackResult{}, wrapped
		}
		return CallbackResult{}, markRetryableCallback(wrapped)
	}

	// Dedup by raw hash: if we've already seen this exact callback, return
	// the original result.
	if pf.RawHash != "" {
		existing, err := s.facts.GetFactByRawHash(ctx, namespace, pf.RawHash)
		if err == nil && existing != nil {
			return s.resultForExistingFact(ctx, namespace, attempt, pf, existing)
		}
		if err != nil && !errors.Is(err, commerce.ErrPaymentFactNotFound) {
			return CallbackResult{}, markRetryableCallback(fmt.Errorf("payment: look up fact by raw hash: %w", err))
		}
	}

	// Dedup by provider event ID (each provider event should be processed once).
	if pf.ProviderEventID != "" {
		existing, err := s.facts.GetFactByProviderEvent(ctx, namespace, pf.Provider, pf.ProviderEventID)
		if err == nil && existing != nil {
			return s.resultForExistingFact(ctx, namespace, attempt, pf, existing)
		}
		if err != nil && !errors.Is(err, commerce.ErrPaymentFactNotFound) {
			return CallbackResult{}, markRetryableCallback(fmt.Errorf("payment: look up fact by provider event: %w", err))
		}
	}

	// Check for contradictory success facts on the same provider order.
	existingFacts, err := s.facts.GetFactsByProviderOrder(ctx, namespace, pf.Provider, pf.ProviderOrderID)
	if err != nil {
		return CallbackResult{}, markRetryableCallback(fmt.Errorf("payment: list facts by provider order: %w", err))
	}
	if err := checkContradiction(existingFacts, pf); err != nil {
		return CallbackResult{}, err
	}

	// Match the fact against the attempt: amount, currency, merchant, application.
	if err := matchFactToAttempt(attempt, pf); err != nil {
		return CallbackResult{}, err
	}

	// Build the immutable PaymentFact record.
	record := PaymentFactRecord{
		ID:                ulid.Make().String(),
		Namespace:         namespace,
		AttemptID:         attempt.ID,
		Provider:          pf.Provider,
		ProviderOrderID:   pf.ProviderOrderID,
		ProviderPaymentID: pf.ProviderPaymentID,
		ProviderEventID:   pf.ProviderEventID,
		MerchantID:        pf.MerchantID,
		ApplicationID:     pf.ApplicationID,
		AmountMinor:       pf.AmountMinor,
		Currency:          pf.Currency,
		Success:           pf.Success,
		RawHash:           pf.RawHash,
		Timestamp:         pf.Timestamp,
		SignedPayload:     pf.SignedPayload,
		CreatedAt:         clock.Now(),
	}

	// Only successful facts transition the order to paid.
	if !pf.Success {
		// Record the fact but don't transition. The order stays in its current state.
		saved, _, err := s.facts.InsertFact(ctx, record)
		if err != nil {
			return CallbackResult{}, markRetryableCallback(fmt.Errorf("payment: insert payment fact: %w", err))
		}
		return CallbackResult{Attempt: attempt, Fact: saved}, nil
	}

	result, err := s.txRunner.RunPaidTransition(ctx, PaidTransitionInput{
		Namespace: namespace,
		Attempt:   attempt,
		Fact:      record,
		FulfillmentReq: FulfillmentRequestCreate{
			OrderID:    attempt.OrderID,
			CustomerID: attempt.CustomerID,
		},
	})
	if err != nil {
		return CallbackResult{}, markRetryableCallback(fmt.Errorf("payment: paid transition: %w", err))
	}
	if result.Fact == nil {
		return CallbackResult{}, markRetryableCallback(errors.New("payment: paid transition returned no payment fact"))
	}
	attempt, err = s.attempts.GetAttempt(ctx, namespace, attempt.ID)
	if err != nil {
		return CallbackResult{}, markRetryableCallback(fmt.Errorf("payment: reload payment attempt after paid transition: %w", err))
	}

	return CallbackResult{Attempt: attempt, Fact: result.Fact, AlreadyPaid: result.AlreadyPaid}, nil
}

func (s *service) resultForExistingFact(
	ctx context.Context,
	namespace string,
	attempt *PaymentAttempt,
	incoming PaymentFact,
	fact *PaymentFactRecord,
) (CallbackResult, error) {
	if err := matchFactToAttempt(attempt, incoming); err != nil {
		return CallbackResult{}, err
	}
	if !existingPaymentFactMatches(attempt, incoming, fact) {
		return CallbackResult{}, ErrContradictoryPaymentFact
	}
	current, err := s.attempts.GetAttempt(ctx, namespace, attempt.ID)
	if err != nil {
		return CallbackResult{}, markRetryableCallback(fmt.Errorf("payment: reload payment attempt for existing fact: %w", err))
	}
	return CallbackResult{
		Attempt:     current,
		Fact:        fact,
		AlreadyPaid: current.Status == AttemptStatusSucceeded,
	}, nil
}

func markRetryableCallback(err error) error {
	if err == nil || errors.Is(err, ErrRetryableCallback) {
		return err
	}
	return fmt.Errorf("%w: %w", ErrRetryableCallback, err)
}

// existingPaymentFactMatches prevents a raw-hash or provider-event collision
// from being treated as an idempotent replay for another attempt or for
// different verified provider fields. RawHash, SignedPayload, and CreatedAt
// are intentionally excluded: the same provider event can arrive through a
// callback and a status query with different transport representations.
func existingPaymentFactMatches(attempt *PaymentAttempt, incoming PaymentFact, existing *PaymentFactRecord) bool {
	return existing != nil &&
		existing.Namespace == attempt.Namespace &&
		existing.AttemptID == attempt.ID &&
		existing.Provider == incoming.Provider &&
		existing.ProviderOrderID == incoming.ProviderOrderID &&
		existing.ProviderPaymentID == incoming.ProviderPaymentID &&
		existing.ProviderEventID == incoming.ProviderEventID &&
		existing.MerchantID == incoming.MerchantID &&
		existing.ApplicationID == incoming.ApplicationID &&
		existing.AmountMinor == incoming.AmountMinor &&
		existing.Currency == incoming.Currency &&
		existing.Success == incoming.Success &&
		existing.Timestamp.Equal(incoming.Timestamp)
}

// matchFactToAttempt verifies the payment fact matches the attempt:
// amount, currency, and provider identity (merchant + application). This
// prevents a valid-signature callback for a different merchant or application
// from being applied to this attempt's order.
func matchFactToAttempt(attempt *PaymentAttempt, pf PaymentFact) error {
	if pf.AmountMinor != attempt.AmountMinor {
		return fmt.Errorf("%w: amount mismatch (fact=%d, attempt=%d)", ErrPaymentFactMismatch, pf.AmountMinor, attempt.AmountMinor)
	}
	if pf.Currency != "" && attempt.Currency != "" && pf.Currency != attempt.Currency {
		return fmt.Errorf("%w: currency mismatch (fact=%s, attempt=%s)", ErrPaymentFactMismatch, pf.Currency, attempt.Currency)
	}
	// Validate merchant identity (I2): if the attempt has an expected merchant
	// ID, the fact must match it.
	if attempt.ExpectedMerchantID != "" && attempt.ExpectedMerchantID != pf.MerchantID {
		return fmt.Errorf("%w: merchant mismatch (fact=%s, attempt=%s)", ErrPaymentFactMismatch, pf.MerchantID, attempt.ExpectedMerchantID)
	}
	// Validate application identity (I2): if the attempt has an expected app ID,
	// the fact must match it.
	if attempt.ExpectedApplicationID != "" && attempt.ExpectedApplicationID != pf.ApplicationID {
		return fmt.Errorf("%w: application mismatch (fact=%s, attempt=%s)", ErrPaymentFactMismatch, pf.ApplicationID, attempt.ExpectedApplicationID)
	}
	return nil
}

// checkContradiction rejects a new success fact that contradicts an existing
// success fact for the same provider order (e.g. different amount or payment ID).
func checkContradiction(existing []PaymentFactRecord, pf PaymentFact) error {
	for _, e := range existing {
		if !e.Success || !pf.Success {
			continue
		}
		// Same provider order, same success state — must agree on amount.
		if e.AmountMinor != pf.AmountMinor {
			return ErrContradictoryPaymentFact
		}
		if e.ProviderPaymentID != "" && pf.ProviderPaymentID != "" && e.ProviderPaymentID != pf.ProviderPaymentID {
			return ErrContradictoryPaymentFact
		}
	}
	return nil
}

// toHTTPHeader converts a map[string][]string to http.Header.
func toHTTPHeader(m map[string][]string) http.Header {
	return http.Header(m)
}

// validateCreateAttempt validates the create-attempt input.
func validateCreateAttempt(in CreateAttemptInput) error {
	var errs []error
	if in.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}
	if in.OrderID == "" {
		errs = append(errs, errors.New("order_id is required"))
	}
	if in.CustomerID == "" {
		errs = append(errs, errors.New("customer_id is required"))
	}
	if in.IdempotencyKey == "" {
		errs = append(errs, errors.New("idempotency_key is required"))
	}
	if in.AmountMinor < 0 {
		errs = append(errs, errors.New("amount_minor must be non-negative"))
	}
	switch in.Provider {
	case ProviderWeChat, ProviderAlipay, ProviderOffline:
	default:
		errs = append(errs, fmt.Errorf("invalid provider: %s", in.Provider))
	}
	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// HashRawBody computes the SHA-256 hex hash of a raw callback body for dedup.
func HashRawBody(body []byte) string {
	h := sha256.Sum256(body)
	return fmt.Sprintf("%x", h[:])
}

// Compile-time check.
var _ Service = (*service)(nil)
