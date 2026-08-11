// Package refund implements the refundable Credit fence and original-route
// refund flow for the Phase 2 commerce domain. The lifecycle is:
//
//  1. Create the Refund in pending_fence.
//  2. Establish a whole-customer WeKnora fence (new reservations stopped,
//     in-flight calls terminal, pending usage covered, watermark reached).
//  3. Under the fence, lock the source allocation + previous refunds, recompute
//     unused paid Credit, validate the 10 Credit : 1 fen quantum, reserve the
//     exact amount, move to provider_processing, and write the Provider Outbox.
//  4. On verified Provider success: move to ledger_reversing, write the Credit
//     Note, reverse only fenced Credit, persist the Refund Fact, mark fulfilled,
//     and release the fence.
//  5. On definitive failure: mark failed and release the fence.
//  6. On unknown result: retain the fence and query the Provider.
//
// Credits are int64. Money uses minor currency units (fen for CNY). The refund
// quantum is fixed at 10 Credit : 1 fen. Sub-quantum Credit remains available
// for consumption rather than being silently reversed.
package refund

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"time"

	"github.com/openmeterio/openmeter/openmeter/commerce"
	"github.com/openmeterio/openmeter/openmeter/commerce/payment"
	"github.com/openmeterio/openmeter/pkg/clock"
)

// ---------------------------------------------------------------------------
// Quantum constants
// ---------------------------------------------------------------------------

// CreditQuantum is the number of Credits per fen (the minimum refundable money
// unit). 10 Credit : 1 fen.
const CreditQuantum int64 = 10

// RefundQuantumFen is the fen count per quantum. 1 fen per 10 Credits.
const RefundQuantumFen int64 = 1

// ---------------------------------------------------------------------------
// Status
// ---------------------------------------------------------------------------

// RefundStatus is the lifecycle status of a refund request.
type RefundStatus string

const (
	// RefundStatusPendingFence: created, fence not yet established.
	RefundStatusPendingFence RefundStatus = "pending_fence"
	// RefundStatusProviderProcessing: fence active, credits reserved, refund
	// submitted to the provider, awaiting verified result.
	RefundStatusProviderProcessing RefundStatus = "provider_processing"
	// RefundStatusLedgerReversing: provider confirmed success, reversing fenced
	// credit and writing the Credit Note.
	RefundStatusLedgerReversing RefundStatus = "ledger_reversing"
	// RefundStatusFulfilled: reversal complete, fence released.
	RefundStatusFulfilled RefundStatus = "fulfilled"
	// RefundStatusFailed: refund rejected or provider returned definitive
	// failure; fence released. No credit was reversed.
	RefundStatusFailed RefundStatus = "failed"
)

// IsTerminal returns true if the status is a terminal state.
func (s RefundStatus) IsTerminal() bool {
	return s == RefundStatusFulfilled || s == RefundStatusFailed
}

// ---------------------------------------------------------------------------
// Domain types
// ---------------------------------------------------------------------------

// RefundRequest is the persisted refund state. AmountCents is the customer-
// requested amount in minor currency units (fen); 0 means "refund all available".
// The quantum fields are persisted on every refund so the exact Credit-to-money
// ratio is auditable. ReservedCredits is the Credits fenced for this refund.
// RemainderCredits is the sub-quantum Credit left available for consumption.
type RefundRequest struct {
	ID              string
	Namespace       string
	CommerceOrderID string
	CustomerID      string
	AmountCents     int64
	Currency        string
	Status          RefundStatus
	Reason          string
	IdempotencyKey  string

	// Quantum persisted for every refund (10 Credit : 1 fen).
	CreditQuantum    int64
	RefundQuantumFen int64

	// Computed during the reserve step (provider_processing onward).
	ReservedCredits  int64
	RefundFen        int64
	RemainderCredits int64

	// Provider details (provider_processing onward).
	ProviderName     string
	ProviderRefundID string

	// Fence details (pending_fence onward).
	FenceSequence string

	// Failure detail.
	FailureReason *string

	CreatedAt time.Time
	UpdatedAt time.Time
}

// RefundFactRecord is the persisted form of a verified provider refund fact.
type RefundFactRecord struct {
	ID               string
	Namespace        string
	RefundRequestID  string
	Provider         payment.Provider
	ProviderRefundID string
	ProviderOrderID  string
	AmountMinor      int64
	Currency         string
	Success          bool
	RawHash          string
	Timestamp        time.Time
	SignedPayload    map[string]any
	CreatedAt        time.Time
}

// CreateRefundInput creates a new refund request.
type CreateRefundInput struct {
	Namespace      string
	OrderID        string
	CustomerID     string
	AmountCents    int64
	Currency       string
	Reason         string
	IdempotencyKey string
}

// ---------------------------------------------------------------------------
// Repository
// ---------------------------------------------------------------------------

// Repository manages RefundRequest records and provides the atomic credit
// reservation that enforces the refundable Credit fence.
type Repository interface {
	CreateRefund(ctx context.Context, req RefundRequest) (*RefundRequest, bool, error)
	GetRefund(ctx context.Context, namespace, id string) (*RefundRequest, error)
	GetRefundByIdempotencyKey(ctx context.Context, namespace, customerID, key string) (*RefundRequest, error)
	GetRefundByProviderRefundID(ctx context.Context, namespace, providerRefundID string) (*RefundRequest, error)
	TransitionStatus(ctx context.Context, namespace, id string, expectedFrom, to RefundStatus) (*RefundRequest, error)
	SaveQuantum(ctx context.Context, namespace, id string, q QuantumReservation) (*RefundRequest, error)
	SetProviderRefundID(ctx context.Context, namespace, id, providerName, providerRefundID string) (*RefundRequest, error)
	SetFence(ctx context.Context, namespace, id, fenceSequence string) (*RefundRequest, error)
	MarkFailed(ctx context.Context, namespace, id, reason string) (*RefundRequest, error)

	// ReserveCredits atomically locks the source allocation and previous refunds,
	// recomputes unused paid Credit, validates the quantum, and reserves the
	// exact amount. Returns Granted=false if insufficient refundable Credit
	// remains after accounting for previously reserved (non-failed) refunds.
	ReserveCredits(ctx context.Context, refundID string, in ReservationInput) (ReservationResult, error)

	AppendFact(ctx context.Context, fact RefundFactRecord) (*RefundFactRecord, bool, error)
	GetFacts(ctx context.Context, namespace, refundID string) ([]RefundFactRecord, error)
}

// QuantumReservation carries the computed quantum and reservation values.
type QuantumReservation struct {
	CreditQuantum    int64
	RefundQuantumFen int64
	ReservedCredits  int64
	RefundFen        int64
	RemainderCredits int64
}

// ReservationInput carries everything the atomic reserve step needs.
type ReservationInput struct {
	Namespace         string
	OrderID           string
	CustomerID        string
	RequestedFen      int64
	RefundableCredits int64
}

// ReservationResult describes the outcome of the atomic reserve.
type ReservationResult struct {
	Granted           bool
	RefundFen         int64
	ReservedCredits   int64
	RemainderCredits  int64
	RefundableCredits int64
	AlreadyReserved   int64
}

// ---------------------------------------------------------------------------
// Collaborator interfaces
// ---------------------------------------------------------------------------

// OrderReader reads orders to validate refund eligibility.
type OrderReader interface {
	GetOrder(ctx context.Context, namespace, id string) (*commerce.Order, error)
}

// ProviderResolver looks up which payment provider was used for an order by
// inspecting the order's payment attempts. This replaces non-deterministic
// map iteration (I5) with a deterministic, data-driven provider selection.
type ProviderResolver interface {
	// ResolveProviderForOrder returns the provider used for the order's
	// successful (or most recent) payment attempt. Returns "" if no attempt
	// is found.
	ResolveProviderForOrder(ctx context.Context, namespace, orderID string) (payment.Provider, error)
}

// WalletDataPort provides refundable credit grants from the Credit Ledger.
type WalletDataPort interface {
	GetGrants(ctx context.Context, namespace, customerID string) ([]commerce.AllocationGrant, error)
}

// FenceResult holds the fence establishment result.
type FenceResult struct {
	Sequence    string
	Established bool
}

// FenceClient establishes and releases whole-customer Credit fences via the
// service-authenticated WeKnora Fence API. The fence stops new reservations,
// drains in-flight calls, covers pending usage, and advances the watermark.
// Implementations serialize fence requests per customer so only one refund can
// hold the fence at a time.
type FenceClient interface {
	EstablishFence(ctx context.Context, namespace, customerID, refundID string) (FenceResult, error)
	ReleaseFence(ctx context.Context, namespace, customerID, refundID, fenceSequence string) error
}

// CreditReverser reverses fenced Credit from the customer's wallet.
type CreditReverser interface {
	ReverseCredits(ctx context.Context, in ReverseCreditsInput) (ReverseCreditsResult, error)
}

// ReverseCreditsInput carries the context for a credit reversal.
type ReverseCreditsInput struct {
	Namespace      string
	CustomerID     string
	RefundID       string
	OrderID        string
	Credits        int64
	IdempotencyKey string
}

// ReverseCreditsResult holds the result of a credit reversal.
type ReverseCreditsResult struct {
	LedgerEntryID string
	Credits       int64
}

// ProviderRefunder submits and queries refunds at the payment provider.
type ProviderRefunder interface {
	Refund(ctx context.Context, input payment.RefundInput) (payment.RefundSubmission, error)
	QueryRefund(ctx context.Context, input payment.RefundQueryInput) (payment.RefundFact, error)
	Name() payment.Provider
}

// RefundCallbackVerifier verifies provider refund callbacks.
type RefundCallbackVerifier interface {
	VerifyRefundCallback(ctx context.Context, headers http.Header, body []byte) (payment.RefundFact, error)
}

// ---------------------------------------------------------------------------
// Errors
// ---------------------------------------------------------------------------

var (
	ErrInvalidRefundTransition = errors.New("refund: invalid status transition")
	ErrRefundNotFound          = errors.New("refund: not found")
	ErrOrderNotRefundable      = errors.New("refund: order is not refundable")
	ErrInsufficientRefundable  = errors.New("refund: insufficient refundable credit")
	ErrFenceTimeout            = errors.New("refund: fence establishment timed out")
	ErrProviderRefundFailed    = errors.New("refund: provider returned failure")
)

// ---------------------------------------------------------------------------
// Service
// ---------------------------------------------------------------------------

// Service implements the refund lifecycle.
type Service interface {
	CreateRefund(ctx context.Context, in CreateRefundInput) (*RefundRequest, bool, error)
	ProcessOne(ctx context.Context, namespace, refundID string) (*RefundRequest, error)
	GetRefund(ctx context.Context, namespace, id string) (*RefundRequest, error)

	// ApplyRefundCallback ingests a verified provider refund callback fact and
	// drives the refund to completion. Idempotent on RawHash. The refund is
	// located by its provider refund ID (set during submission).
	ApplyRefundCallback(ctx context.Context, namespace string, fact payment.RefundFact) (*RefundRequest, error)
}

// Config wires the refund service.
type Config struct {
	Repo      Repository
	Orders    OrderReader
	Wallet    WalletDataPort
	Fence     FenceClient
	Reverser  CreditReverser
	Providers map[payment.Provider]ProviderRefunder
	// ProviderResolver determines which provider was used for an order's payment
	// (I5). If nil, lookupProvider falls back to the refund's ProviderName or,
	// failing that, a deterministic first-key selection (no map iteration).
	ProviderResolver ProviderResolver
	Logger           *slog.Logger
}

type service struct {
	repo      Repository
	orders    OrderReader
	wallet    WalletDataPort
	fence     FenceClient
	reverser  CreditReverser
	providers map[payment.Provider]ProviderRefunder
	pResolver ProviderResolver
	logger    *slog.Logger
}

// New creates a refund Service.
func New(cfg Config) (Service, error) {
	if cfg.Repo == nil {
		return nil, errors.New("refund service: repository is required")
	}
	if cfg.Orders == nil {
		return nil, errors.New("refund service: orders reader is required")
	}
	if cfg.Wallet == nil {
		return nil, errors.New("refund service: wallet data port is required")
	}
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}
	return &service{
		repo:      cfg.Repo,
		orders:    cfg.Orders,
		wallet:    cfg.Wallet,
		fence:     cfg.Fence,
		reverser:  cfg.Reverser,
		providers: cfg.Providers,
		pResolver: cfg.ProviderResolver,
		logger:    logger,
	}, nil
}

// CreateRefund creates a refund request in pending_fence.
func (s *service) CreateRefund(ctx context.Context, in CreateRefundInput) (*RefundRequest, bool, error) {
	if err := validateCreateRefund(in); err != nil {
		return nil, false, err
	}

	existing, err := s.repo.GetRefundByIdempotencyKey(ctx, in.Namespace, in.CustomerID, in.IdempotencyKey)
	if err == nil && existing != nil {
		return existing, false, nil
	}

	order, err := s.orders.GetOrder(ctx, in.Namespace, in.OrderID)
	if err != nil {
		return nil, false, fmt.Errorf("refund: get order: %w", err)
	}
	if err := validateRefundable(order); err != nil {
		return nil, false, err
	}

	now := clock.Now()
	req := RefundRequest{
		// ID is assigned by the repository (avoids collisions in concurrent creates).
		Namespace:       in.Namespace,
		CommerceOrderID: in.OrderID,
		CustomerID:      in.CustomerID,
		AmountCents:     in.AmountCents,
		Currency:        in.Currency,
		Status:          RefundStatusPendingFence,
		Reason:          in.Reason,
		IdempotencyKey:  in.IdempotencyKey,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	return s.repo.CreateRefund(ctx, req)
}

// ProcessOne drives a refund request through the state machine.
func (s *service) ProcessOne(ctx context.Context, namespace, refundID string) (*RefundRequest, error) {
	rec, err := s.repo.GetRefund(ctx, namespace, refundID)
	if err != nil {
		return nil, fmt.Errorf("refund: get refund: %w", err)
	}

	if rec.Status.IsTerminal() {
		return rec, nil
	}

	switch rec.Status {
	case RefundStatusPendingFence:
		return s.processPendingFence(ctx, rec)
	case RefundStatusProviderProcessing:
		return s.processProviderProcessing(ctx, rec)
	case RefundStatusLedgerReversing:
		return s.processLedgerReversing(ctx, rec)
	default:
		return rec, nil
	}
}

// processPendingFence establishes the fence, reserves credits, and submits.
func (s *service) processPendingFence(ctx context.Context, rec *RefundRequest) (*RefundRequest, error) {
	if s.fence == nil {
		return nil, errors.New("refund: fence client is required for processing")
	}

	// Step 2: Establish the whole-customer fence.
	fenceRes, err := s.fence.EstablishFence(ctx, rec.Namespace, rec.CustomerID, rec.ID)
	if err != nil {
		s.logger.InfoContext(ctx, "refund: fence failed (stays in pending_fence)", "id", rec.ID, "error", err)
		return rec, err
	}
	if !fenceRes.Established {
		return rec, ErrFenceTimeout
	}

	// EstablishFence persisted the stable sequence under the shared customer
	// lock. Avoid a second, unlocked write that could race a Reserve command.
	rec.FenceSequence = fenceRes.Sequence

	// Step 3: Under the fence, recompute unused paid Credit and reserve.
	rec, err = s.reserveAndSubmit(ctx, rec)
	if err != nil {
		// Once the refund has entered provider_processing, reserved credits are
		// fenced and must NOT be released on transient errors — only failRefund
		// (definitive failure) or fulfilled release the fence.
		// For pre-provider_processing failures (reserve failures), the fence is
		// still safe to release.
		if rec != nil && rec.Status == RefundStatusPendingFence {
			s.releaseFenceQuietly(ctx, rec.Namespace, rec.CustomerID, rec.ID, fenceRes.Sequence)
		}
		return rec, err
	}
	return rec, nil
}

// reserveAndSubmit recomputes refundable Credit, validates the quantum, reserves,
// and submits to the provider.
func (s *service) reserveAndSubmit(ctx context.Context, rec *RefundRequest) (*RefundRequest, error) {
	grants, err := s.wallet.GetGrants(ctx, rec.Namespace, rec.CustomerID)
	if err != nil {
		return rec, fmt.Errorf("refund: get grants: %w", err)
	}

	refundableCredits := sumRefundableCredits(grants)
	if refundableCredits <= 0 {
		reason := "no refundable credit remaining"
		rec = s.failRefund(ctx, rec, reason)
		return rec, fmt.Errorf("%w: %s", ErrInsufficientRefundable, reason)
	}

	// Atomic reserve: lock source allocation + previous refunds, compute
	// available, reserve the exact amount.
	res, err := s.repo.ReserveCredits(ctx, rec.ID, ReservationInput{
		Namespace:         rec.Namespace,
		OrderID:           rec.CommerceOrderID,
		CustomerID:        rec.CustomerID,
		RequestedFen:      rec.AmountCents,
		RefundableCredits: refundableCredits,
	})
	if err != nil {
		return rec, fmt.Errorf("refund: reserve credits: %w", err)
	}
	if !res.Granted {
		reason := fmt.Sprintf("insufficient refundable credit: refundable=%d, already_reserved=%d", res.RefundableCredits, res.AlreadyReserved)
		rec = s.failRefund(ctx, rec, reason)
		return rec, fmt.Errorf("%w: %s", ErrInsufficientRefundable, reason)
	}

	// Persist the quantum and reservation.
	rec, err = s.repo.SaveQuantum(ctx, rec.Namespace, rec.ID, QuantumReservation{
		CreditQuantum:    CreditQuantum,
		RefundQuantumFen: RefundQuantumFen,
		ReservedCredits:  res.ReservedCredits,
		RefundFen:        res.RefundFen,
		RemainderCredits: res.RemainderCredits,
	})
	if err != nil {
		return rec, fmt.Errorf("refund: save quantum: %w", err)
	}

	// Move to provider_processing.
	rec, err = s.repo.TransitionStatus(ctx, rec.Namespace, rec.ID, RefundStatusPendingFence, RefundStatusProviderProcessing)
	if err != nil {
		return rec, fmt.Errorf("refund: transition to provider_processing: %w", err)
	}

	// Submit to the provider.
	return s.submitToProvider(ctx, rec)
}

// submitToProvider submits the refund to the payment provider.
func (s *service) submitToProvider(ctx context.Context, rec *RefundRequest) (*RefundRequest, error) {
	provider, err := s.lookupProvider(ctx, rec)
	if err != nil {
		return rec, err
	}
	if provider == nil {
		reason := "no provider configured for refund"
		rec = s.failRefund(ctx, rec, reason)
		return rec, errors.New("refund: " + reason)
	}

	order, err := s.orders.GetOrder(ctx, rec.Namespace, rec.CommerceOrderID)
	if err != nil {
		return rec, fmt.Errorf("refund: get order for provider submit: %w", err)
	}

	sub, err := provider.Refund(ctx, payment.RefundInput{
		Namespace:       rec.Namespace,
		OrderID:         rec.CommerceOrderID,
		ProviderOrderID: order.PublicID,
		AmountMinor:     rec.RefundFen,
		Currency:        rec.Currency,
		Reason:          rec.Reason,
		IdempotencyKey:  rec.IdempotencyKey,
	})
	if err != nil {
		reason := "provider refund submission failed: " + err.Error()
		rec = s.failRefund(ctx, rec, reason)
		return rec, fmt.Errorf("refund: provider refund: %w", err)
	}

	// Record the provider refund ID.
	rec, err = s.repo.SetProviderRefundID(ctx, rec.Namespace, rec.ID, string(provider.Name()), sub.ProviderRefundID)
	if err != nil {
		return rec, fmt.Errorf("refund: set provider refund id: %w", err)
	}

	// Handle the submission status.
	switch sub.Status {
	case "success", "succeeded":
		// The submission confirmed success — synthesize a fact from the submission.
		// AmountMinor is 0 here; money validation only applies when the fact
		// reports a positive amount (from a query/callback with verified data).
		return s.handleProviderSuccess(ctx, rec, payment.RefundFact{
			Provider:         provider.Name(),
			ProviderRefundID: sub.ProviderRefundID,
			Success:          true,
			AmountMinor:      0,
		})
	case "failed", "rejected":
		reason := "provider rejected the refund"
		rec = s.failRefund(ctx, rec, reason)
		return rec, fmt.Errorf("%w: %s", ErrProviderRefundFailed, reason)
	default:
		// "processing" or unknown — stay in provider_processing. Never infer
		// success from the submission result.
		return rec, nil
	}
}

// processProviderProcessing queries the provider for the current status.
func (s *service) processProviderProcessing(ctx context.Context, rec *RefundRequest) (*RefundRequest, error) {
	provider, err := s.lookupProvider(ctx, rec)
	if err != nil {
		return rec, err
	}
	if provider == nil {
		return rec, errors.New("refund: no provider configured")
	}

	if rec.ProviderRefundID == "" {
		return s.submitToProvider(ctx, rec)
	}

	order, err := s.orders.GetOrder(ctx, rec.Namespace, rec.CommerceOrderID)
	if err != nil {
		return rec, fmt.Errorf("refund: get order for provider query: %w", err)
	}
	fact, err := provider.QueryRefund(ctx, payment.RefundQueryInput{
		ProviderRefundID: rec.ProviderRefundID,
		ProviderOrderID:  order.PublicID,
		AmountMinor:      rec.RefundFen,
		Currency:         rec.Currency,
	})
	if err != nil {
		return rec, fmt.Errorf("refund: provider query: %w", err)
	}

	if err := s.persistRefundFact(ctx, rec, fact); err != nil {
		return rec, err
	}

	if fact.Success {
		return s.handleProviderSuccess(ctx, rec, fact)
	}

	// Definitive failure (verified fact with non-empty hash).
	if fact.RawHash != "" && !fact.Success {
		reason := "provider returned definitive failure"
		rec = s.failRefund(ctx, rec, reason)
		return rec, fmt.Errorf("%w: %s", ErrProviderRefundFailed, reason)
	}

	// Unknown result — stay in provider_processing and retain the fence.
	return rec, nil
}

// handleProviderSuccess transitions to ledger_reversing and performs reversal.
func (s *service) handleProviderSuccess(ctx context.Context, rec *RefundRequest, fact payment.RefundFact) (*RefundRequest, error) {
	// Validate provider-refunded money covers the reserved amount. Money cannot
	// exceed the remaining Provider-refunded money (global constraint). If the
	// provider reports an amount and it doesn't cover rec.RefundFen, the refund
	// is rejected.
	if fact.AmountMinor > 0 && fact.AmountMinor < rec.RefundFen {
		reason := fmt.Sprintf("provider refunded %d fen but %d was reserved", fact.AmountMinor, rec.RefundFen)
		rec = s.failRefund(ctx, rec, reason)
		return rec, fmt.Errorf("refund: %s", reason)
	}

	rec, err := s.repo.TransitionStatus(ctx, rec.Namespace, rec.ID, RefundStatusProviderProcessing, RefundStatusLedgerReversing)
	if err != nil {
		return rec, fmt.Errorf("refund: transition to ledger_reversing: %w", err)
	}
	return s.processLedgerReversing(ctx, rec)
}

// processLedgerReversing reverses fenced credit, marks fulfilled, and releases
// the fence.
func (s *service) processLedgerReversing(ctx context.Context, rec *RefundRequest) (*RefundRequest, error) {
	// Step 4: Reverse only fenced Credit.
	if s.reverser != nil {
		_, err := s.reverser.ReverseCredits(ctx, ReverseCreditsInput{
			Namespace:      rec.Namespace,
			CustomerID:     rec.CustomerID,
			RefundID:       rec.ID,
			OrderID:        rec.CommerceOrderID,
			Credits:        rec.ReservedCredits,
			IdempotencyKey: "refund-reverse:" + rec.ID,
		})
		if err != nil {
			return rec, fmt.Errorf("refund: reverse credits: %w", err)
		}
	}

	// Mark fulfilled.
	rec, err := s.repo.TransitionStatus(ctx, rec.Namespace, rec.ID, RefundStatusLedgerReversing, RefundStatusFulfilled)
	if err != nil {
		return rec, fmt.Errorf("refund: transition to fulfilled: %w", err)
	}

	// Release the fence.
	s.releaseFenceQuietly(ctx, rec.Namespace, rec.CustomerID, rec.ID, rec.FenceSequence)

	return rec, nil
}

// GetRefund retrieves a refund by ID.
func (s *service) GetRefund(ctx context.Context, namespace, id string) (*RefundRequest, error) {
	return s.repo.GetRefund(ctx, namespace, id)
}

// ApplyRefundCallback ingests a verified provider refund callback fact and drives
// the refund to completion. The callback is located by its provider refund ID
// (set during submission). This is the async callback entry point — Brief Step 4.
//
// The fact must already be signature-verified by the provider adapter. This
// method:
//  1. Locates the refund by provider refund ID.
//  2. Persists the fact (dedup on RawHash).
//  3. If success: validates money, transitions to ledger_reversing, reverses
//     credit, marks fulfilled, releases fence.
//  4. If definitive failure: marks failed, releases fence.
//  5. If unknown: retains fence and returns current state.
func (s *service) ApplyRefundCallback(ctx context.Context, namespace string, fact payment.RefundFact) (*RefundRequest, error) {
	rec, err := s.repo.GetRefundByProviderRefundID(ctx, namespace, fact.ProviderRefundID)
	if err != nil {
		return nil, fmt.Errorf("refund: locate refund by provider refund id %s: %w", fact.ProviderRefundID, err)
	}

	// Dedup: persist the fact. If we've already seen it, return the current state.
	_, inserted, err := s.repo.AppendFact(ctx, RefundFactRecord{
		ID:               rec.ID + "-fact-" + fact.ProviderRefundID,
		Namespace:        namespace,
		RefundRequestID:  rec.ID,
		Provider:         fact.Provider,
		ProviderRefundID: fact.ProviderRefundID,
		ProviderOrderID:  fact.ProviderOrderID,
		AmountMinor:      fact.AmountMinor,
		Currency:         fact.Currency,
		Success:          fact.Success,
		RawHash:          fact.RawHash,
		Timestamp:        fact.Timestamp,
		SignedPayload:    fact.SignedPayload,
		CreatedAt:        clock.Now(),
	})
	if err != nil {
		return rec, fmt.Errorf("refund: persist callback fact: %w", err)
	}

	// Terminal states are idempotent — return current state.
	if rec.Status.IsTerminal() {
		return rec, nil
	}

	// If this fact was already seen (dedup), the refund was already driven by
	// this callback in a prior call. Return current state.
	if !inserted && rec.Status == RefundStatusLedgerReversing {
		// Already processing this callback — retry the ledger reversal.
		return s.processLedgerReversing(ctx, rec)
	}

	if fact.Success {
		// Only proceed from provider_processing. If still in pending_fence, the
		// callback arrived before the submission was recorded — persist and wait.
		if rec.Status != RefundStatusProviderProcessing {
			return rec, nil
		}
		return s.handleProviderSuccess(ctx, rec, fact)
	}

	// Definitive failure (verified fact with non-empty hash).
	if fact.RawHash != "" && !fact.Success {
		reason := "provider callback returned definitive failure"
		rec = s.failRefund(ctx, rec, reason)
		return rec, fmt.Errorf("%w: %s", ErrProviderRefundFailed, reason)
	}

	// Unknown result — retain the fence.
	return rec, nil
}

// ---------------------------------------------------------------------------
// Arithmetic helpers
// ---------------------------------------------------------------------------

// ComputeRefundable computes the exact refundable Credit and fen from the
// refundable Credits. It applies the fixed 10 Credit : 1 fen quantum and caps
// at the requested fen (if > 0). Sub-quantum Credit remains available for
// consumption (RemainderCredits) rather than being silently reversed.
//
// Money is calculated only in the persisted quantum and cannot exceed the
// available refundable Credit.
func ComputeRefundable(refundableCredits, requestedFen int64) (refundFen, reservedCredits, remainderCredits int64) {
	if refundableCredits <= 0 {
		return 0, 0, 0
	}

	totalFen := refundableCredits / CreditQuantum

	actualFen := totalFen
	if requestedFen > 0 && requestedFen < actualFen {
		actualFen = requestedFen
	}
	if actualFen < 0 {
		actualFen = 0
	}

	reservedCredits = actualFen * CreditQuantum
	remainderCredits = refundableCredits - reservedCredits

	return actualFen, reservedCredits, remainderCredits
}

// sumRefundableCredits sums the Refundable field across all recharge-source
// grants. Only BucketSourceRecharge credits are refundable.
func sumRefundableCredits(grants []commerce.AllocationGrant) int64 {
	total := int64(0)
	for _, g := range grants {
		if g.Source == commerce.BucketSourceRecharge && g.Refundable > 0 {
			total += g.Refundable
		}
	}
	return total
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

// failRefund marks the refund as failed and releases the fence (if held). This
// is the single point of fence release for failure paths, preventing
// double-release when processPendingFence's error handler also runs.
func (s *service) failRefund(ctx context.Context, rec *RefundRequest, reason string) *RefundRequest {
	updated, err := s.repo.MarkFailed(ctx, rec.Namespace, rec.ID, reason)
	if err != nil {
		s.logger.ErrorContext(ctx, "refund: MarkFailed itself failed", "id", rec.ID, "error", err)
		return rec
	}
	s.releaseFenceQuietly(ctx, updated.Namespace, updated.CustomerID, updated.ID, updated.FenceSequence)
	return updated
}

// lookupProvider resolves the payment provider for a refund request. The
// resolution order is deterministic (I5):
//  1. If the refund already has a ProviderName (set during a prior submission),
//     use it.
//  2. If a ProviderResolver is configured, look up the provider from the
//     order's payment attempt data.
//  3. Only when no resolver is configured, deterministically fall back to the
//     lexicographically smallest provider key. A configured resolver is
//     authoritative: its errors and invalid results must not switch channels.
func (s *service) lookupProvider(ctx context.Context, rec *RefundRequest) (ProviderRefunder, error) {
	if rec.ProviderName != "" {
		if p, ok := s.providers[payment.Provider(rec.ProviderName)]; ok {
			return p, nil
		}
		return nil, fmt.Errorf("refund: persisted provider %q is not configured", rec.ProviderName)
	}
	// I5: resolve from the order's payment attempt if a resolver is wired.
	if s.pResolver != nil {
		prov, err := s.pResolver.ResolveProviderForOrder(ctx, rec.Namespace, rec.CommerceOrderID)
		if err != nil {
			return nil, fmt.Errorf("refund: resolve payment provider: %w", err)
		}
		if prov == "" {
			return nil, errors.New("refund: resolver returned an empty payment provider")
		}
		provider, ok := s.providers[prov]
		if !ok {
			return nil, fmt.Errorf("refund: resolved provider %q is not configured", prov)
		}
		return provider, nil
	}
	// Deterministic fallback: sorted first key — never iterate the map.
	if len(s.providers) > 0 {
		keys := make([]string, 0, len(s.providers))
		for k := range s.providers {
			keys = append(keys, string(k))
		}
		sort.Strings(keys)
		return s.providers[payment.Provider(keys[0])], nil
	}
	return nil, nil
}

func (s *service) persistRefundFact(ctx context.Context, rec *RefundRequest, fact payment.RefundFact) error {
	if fact.RawHash == "" {
		return nil
	}
	_, _, err := s.repo.AppendFact(ctx, RefundFactRecord{
		ID:               rec.ID + "-fact-" + fact.ProviderRefundID,
		Namespace:        rec.Namespace,
		RefundRequestID:  rec.ID,
		Provider:         fact.Provider,
		ProviderRefundID: fact.ProviderRefundID,
		ProviderOrderID:  fact.ProviderOrderID,
		AmountMinor:      fact.AmountMinor,
		Currency:         fact.Currency,
		Success:          fact.Success,
		RawHash:          fact.RawHash,
		Timestamp:        fact.Timestamp,
		SignedPayload:    fact.SignedPayload,
		CreatedAt:        clock.Now(),
	})
	if err != nil {
		return fmt.Errorf("refund: persist provider fact: %w", err)
	}
	return nil
}

func (s *service) releaseFenceQuietly(ctx context.Context, namespace, customerID, refundID, fenceSeq string) {
	if s.fence == nil || fenceSeq == "" {
		return
	}
	if err := s.fence.ReleaseFence(ctx, namespace, customerID, refundID, fenceSeq); err != nil {
		s.logger.WarnContext(ctx, "refund: release fence failed", "fence", fenceSeq, "error", err)
	}
}

func validateCreateRefund(in CreateRefundInput) error {
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
	if in.AmountCents < 0 {
		errs = append(errs, errors.New("amount_cents must be non-negative"))
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

func validateRefundable(order *commerce.Order) error {
	if order.Kind != commerce.OrderKindWalletTopUp {
		return fmt.Errorf("%w: order kind %s is not a wallet top-up", ErrOrderNotRefundable, order.Kind)
	}
	// I6: only fulfilled orders may be refunded. A paid-but-not-fulfilled order
	// has not granted credits yet, so there is nothing to fence or reverse.
	if order.Status != commerce.OrderStatusFulfilled {
		return fmt.Errorf("%w: order status %s is not fulfilled", ErrOrderNotRefundable, order.Status)
	}
	return nil
}

// Compile-time check.
var _ Service = (*service)(nil)
