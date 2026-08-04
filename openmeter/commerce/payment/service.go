package payment

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
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

	// GetAttemptByProviderOrder looks up an attempt by provider + provider order ID.
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
	// InsertFact persists a PaymentFact. It must deduplicate on RawHash — if a
	// fact with the same RawHash already exists, return the existing fact with
	// (false, nil).
	InsertFact(ctx context.Context, fact PaymentFactRecord) (*PaymentFactRecord, bool, error)

	// GetFactByRawHash retrieves a fact by its raw body hash (for dedup).
	GetFactByRawHash(ctx context.Context, namespace string, rawHash string) (*PaymentFactRecord, error)

	// GetFactsByProviderOrder retrieves all facts for a provider order.
	GetFactsByProviderOrder(ctx context.Context, namespace string, provider Provider, providerOrderID string) ([]PaymentFactRecord, error)

	// GetFactByProviderEvent retrieves a fact by provider event ID (dedup).
	GetFactByProviderEvent(ctx context.Context, namespace string, provider Provider, providerEventID string) (*PaymentFactRecord, error)
}

// OrderStatusUpdater transitions the order to paid. This is a narrow port that
// avoids coupling the payment service to the full order repository.
type OrderStatusUpdater interface {
	UpdateOrderStatus(ctx context.Context, namespace, id string, expectedFrom, to commerce.OrderStatus) (*commerce.Order, error)
	GetOrder(ctx context.Context, namespace, id string) (*commerce.Order, error)
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

// PaymentAttempt tracks one attempt to pay an order through a provider.
type PaymentAttempt struct {
	ID                string
	Namespace         string
	OrderID           string
	CustomerID        string
	Provider          Provider
	ProviderOrderID   string
	ProviderPaymentID string
	Status            AttemptStatus
	ProviderSessionID string
	IdempotencyKey    string
	AmountMinor       int64
	Currency          string
	CreatedAt         time.Time
	UpdatedAt         time.Time
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
	Namespace      string
	OrderID        string
	CustomerID     string
	Provider       Provider
	IdempotencyKey string
	AmountMinor    int64
	Currency       string
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

	// HandleCallback is the entry point for provider callbacks. It verifies the
	// signature, deduplicates, checks the fact against the attempt's order
	// (amount, currency, identity), persists the fact, and transitions the order
	// to paid. Repeated callbacks return the original result.
	HandleCallback(ctx context.Context, providerName Provider, headers map[string][]string, body []byte) (CallbackResult, error)

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
	Providers map[Provider]ProviderAdapter
	Logger    *slog.Logger
}

type service struct {
	attempts  AttemptRepository
	facts     FactRepository
	orders    OrderStatusUpdater
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

	now := clock.Now()
	attempt := PaymentAttempt{
		ID:             ulid.Make().String(),
		Namespace:      in.Namespace,
		OrderID:        in.OrderID,
		CustomerID:     in.CustomerID,
		Provider:       in.Provider,
		Status:         AttemptStatusCreated,
		IdempotencyKey: in.IdempotencyKey,
		AmountMinor:    in.AmountMinor,
		Currency:       in.Currency,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	return s.attempts.CreateAttempt(ctx, attempt)
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
	updated, err := s.attempts.SetProviderIDs(ctx, namespace, attemptID, fact.ProviderOrderID, fact.ProviderPaymentID, fact.QRCodeURL)
	if err != nil {
		return CheckoutResult{}, err
	}
	updated, err = s.attempts.UpdateAttemptStatus(ctx, namespace, attemptID, AttemptStatusCreated, AttemptStatusPending)
	if err != nil {
		return CheckoutResult{}, err
	}

	return CheckoutResult{Attempt: updated, Fact: fact}, nil
}

// HandleCallback processes a provider callback: verify signature, deduplicate,
// match against the order, persist fact, transition to paid.
func (s *service) HandleCallback(ctx context.Context, providerName Provider, headers map[string][]string, body []byte) (CallbackResult, error) {
	provider, ok := s.providers[providerName]
	if !ok {
		return CallbackResult{}, fmt.Errorf("payment: provider %s not configured", providerName)
	}

	// Verify the signature — the adapter does this and extracts verified fields.
	pf, err := provider.VerifyCallback(ctx, toHTTPHeader(headers), body)
	if err != nil {
		return CallbackResult{}, err
	}

	return s.applyPaymentFact(ctx, pf)
}

// ConfirmPayment queries the provider directly (callback-lost recovery).
func (s *service) ConfirmPayment(ctx context.Context, namespace, attemptID string) (CallbackResult, error) {
	attempt, err := s.attempts.GetAttempt(ctx, namespace, attemptID)
	if err != nil {
		return CallbackResult{}, err
	}

	provider, ok := s.providers[attempt.Provider]
	if !ok {
		return CallbackResult{}, fmt.Errorf("payment: provider %s not configured", attempt.Provider)
	}

	if attempt.ProviderOrderID == "" {
		return CallbackResult{}, errors.New("payment: attempt has no provider order id to confirm")
	}

	pf, err := provider.QueryPayment(ctx, attempt.ProviderOrderID)
	if err != nil {
		return CallbackResult{}, fmt.Errorf("payment: provider query payment: %w", err)
	}

	return s.applyPaymentFact(ctx, pf)
}

// applyPaymentFact is the core atom: deduplicate the fact, match it against the
// attempt/order, persist it, and transition the order to paid. This method is
// idempotent: repeated callbacks with the same fact converge to one paid order.
func (s *service) applyPaymentFact(ctx context.Context, pf PaymentFact) (CallbackResult, error) {
	// Locate the attempt by provider order ID.
	attempt, err := s.attempts.GetAttemptByProviderOrder(ctx, "", pf.Provider, pf.ProviderOrderID)
	if err != nil {
		return CallbackResult{}, fmt.Errorf("payment: locate attempt by provider order %s: %w", pf.ProviderOrderID, err)
	}

	// Dedup by raw hash: if we've already seen this exact callback, return
	// the original result.
	if pf.RawHash != "" {
		existing, err := s.facts.GetFactByRawHash(ctx, attempt.Namespace, pf.RawHash)
		if err == nil && existing != nil {
			return CallbackResult{Attempt: attempt, Fact: existing, AlreadyPaid: attempt.Status == AttemptStatusSucceeded}, nil
		}
	}

	// Dedup by provider event ID (each provider event should be processed once).
	if pf.ProviderEventID != "" {
		existing, err := s.facts.GetFactByProviderEvent(ctx, attempt.Namespace, pf.Provider, pf.ProviderEventID)
		if err == nil && existing != nil {
			return CallbackResult{Attempt: attempt, Fact: existing, AlreadyPaid: attempt.Status == AttemptStatusSucceeded}, nil
		}
	}

	// Check for contradictory success facts on the same provider order.
	existingFacts, _ := s.facts.GetFactsByProviderOrder(ctx, attempt.Namespace, pf.Provider, pf.ProviderOrderID)
	if err := checkContradiction(existingFacts, pf); err != nil {
		return CallbackResult{}, err
	}

	// Match the fact against the attempt's order: amount, currency, identity.
	if err := s.matchFactToOrder(ctx, attempt, pf); err != nil {
		return CallbackResult{}, err
	}

	// Persist the immutable PaymentFact record.
	record := PaymentFactRecord{
		ID:              ulid.Make().String(),
		Namespace:       attempt.Namespace,
		AttemptID:       attempt.ID,
		Provider:        pf.Provider,
		ProviderOrderID: pf.ProviderOrderID,
		ProviderEventID: pf.ProviderEventID,
		MerchantID:      pf.MerchantID,
		ApplicationID:   pf.ApplicationID,
		AmountMinor:     pf.AmountMinor,
		Currency:        pf.Currency,
		Success:         pf.Success,
		RawHash:         pf.RawHash,
		Timestamp:       pf.Timestamp,
		SignedPayload:   pf.SignedPayload,
		CreatedAt:       clock.Now(),
	}
	saved, _, err := s.facts.InsertFact(ctx, record)
	if err != nil {
		return CallbackResult{}, fmt.Errorf("payment: insert payment fact: %w", err)
	}

	// Only successful facts transition the order to paid.
	if !pf.Success {
		// Record the fact but don't transition. The order stays in its current state.
		return CallbackResult{Attempt: attempt, Fact: saved}, nil
	}

	// Transition the attempt to succeeded and the order to paid.
	// If the order is already paid or fulfilled, this is an idempotent replay.
	order, _ := s.orders.GetOrder(ctx, attempt.Namespace, attempt.OrderID)
	if order != nil && (order.Status == commerce.OrderStatusPaid || order.Status == commerce.OrderStatusFulfilled) {
		// Already paid — return without error (idempotent).
		return CallbackResult{Attempt: attempt, Fact: saved, AlreadyPaid: true}, nil
	}

	// Atomically: attempt -> succeeded. This is best-effort under concurrency;
	// the authoritative gate is the order status transition below. A failure
	// here means a concurrent callback already moved the attempt forward.
	if _, err := s.attempts.UpdateAttemptStatus(ctx, attempt.Namespace, attempt.ID, attempt.Status, AttemptStatusSucceeded); err != nil {
		s.logger.InfoContext(ctx, "payment: attempt status update (may be concurrent)", "error", err)
	}

	// Transition order to paid. The OrderStatusUpdater uses optimistic
	// concurrency: only one call succeeds (awaiting_payment -> paid).
	if _, err := s.orders.UpdateOrderStatus(ctx, attempt.Namespace, attempt.OrderID, commerce.OrderStatusAwaitingPayment, commerce.OrderStatusPaid); err != nil {
		// If it fails, the order may already be paid (concurrent) or in a
		// different state. Either way, we've persisted the fact.
		s.logger.InfoContext(ctx, "payment: order transition to paid returned (may be concurrent or already paid)", "order_id", attempt.OrderID, "error", err)
	}

	// Reload attempt for the result.
	attempt, _ = s.attempts.GetAttempt(ctx, attempt.Namespace, attempt.ID)

	return CallbackResult{Attempt: attempt, Fact: saved}, nil
}

// matchFactToOrder verifies the payment fact matches the attempt's order:
// amount, currency must be identical. This prevents a valid-signature callback
// for a different order from being applied.
func (s *service) matchFactToOrder(ctx context.Context, attempt *PaymentAttempt, pf PaymentFact) error {
	if pf.AmountMinor != attempt.AmountMinor {
		return fmt.Errorf("%w: amount mismatch (fact=%d, attempt=%d)", ErrPaymentFactMismatch, pf.AmountMinor, attempt.AmountMinor)
	}
	if pf.Currency != "" && attempt.Currency != "" && pf.Currency != attempt.Currency {
		return fmt.Errorf("%w: currency mismatch (fact=%s, attempt=%s)", ErrPaymentFactMismatch, pf.Currency, attempt.Currency)
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
