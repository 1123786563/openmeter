package commerce

// This file implements Payment, Refund, Fulfillment repository methods on
// EntAdapter. PaidTransitionParams and RunPaidTransition live in
// paid_tx_runner.go — this file only adds the CRUD repo methods.

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/oklog/ulid/v2"

	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ent/db/externalinvoiceref"
	"github.com/openmeterio/openmeter/openmeter/ent/db/fulfillment"
	"github.com/openmeterio/openmeter/openmeter/ent/db/offlinepayment"
	"github.com/openmeterio/openmeter/openmeter/ent/db/paymentattempt"
	"github.com/openmeterio/openmeter/openmeter/ent/db/paymentfact"
	"github.com/openmeterio/openmeter/openmeter/ent/db/receivableperiod"
	"github.com/openmeterio/openmeter/openmeter/ent/db/refundfact"
	"github.com/openmeterio/openmeter/openmeter/ent/db/refundrequest"
	"github.com/openmeterio/openmeter/pkg/clock"
)

// ===========================================================================
// Errors
// ===========================================================================

var (
	ErrPaymentAttemptNotFound       = errors.New("payment attempt not found")
	ErrPaymentFactNotFound          = errors.New("payment fact not found")
	ErrInvalidAttemptTransition     = errors.New("invalid payment attempt transition")
	ErrFulfillmentNotFound          = errors.New("fulfillment not found")
	ErrFulfillmentAlreadyClaimed    = errors.New("fulfillment already claimed")
	ErrInvalidFulfillmentTransition = errors.New("invalid fulfillment transition")
	ErrRefundNotFound               = errors.New("refund not found")
	ErrReceivablePeriodNotFound     = errors.New("receivable period not found")
)

// ===========================================================================
// Wire types (domain-neutral transport structs)
// ===========================================================================

// PaymentProviderWire is the string-typed provider name.
type PaymentProviderWire = string

// AttemptStatusWire is the string-typed attempt status.
type AttemptStatusWire = string

// RefundStatusWire is the string-typed refund status.
type RefundStatusWire = string

// PaymentAttemptWire is the wire format for a payment attempt.
type PaymentAttemptWire struct {
	ID                    string
	Namespace             string
	OrderID               string
	CustomerID            string
	Provider              PaymentProviderWire
	ProviderOrderID       string
	ProviderPaymentID     string
	ProviderSessionID     string
	ExpectedMerchantID    string
	ExpectedApplicationID string
	Status                AttemptStatusWire
	IdempotencyKey        string
	AmountMinor           int64
	Currency              string
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

// PaymentFactWire is the wire format for a payment fact.
type PaymentFactWire struct {
	ID                string
	Namespace         string
	AttemptID         string
	Provider          PaymentProviderWire
	ProviderOrderID   string
	ProviderPaymentID string
	ProviderEventID   string
	MerchantID        string
	ApplicationID     string
	AmountMinor       int64
	Currency          string
	Success           bool
	RawHash           string
	SignedPayload     map[string]any
	Timestamp         time.Time
	CreatedAt         time.Time
}

// FulfillmentCreateWire is the input for creating a fulfillment.
type FulfillmentCreateWire struct {
	Namespace  string
	OrderID    string
	CustomerID string
}

// FulfillmentWire is the wire format for a fulfillment record.
type FulfillmentWire struct {
	ID             string
	Namespace      string
	OrderID        string
	CustomerID     string
	Status         string
	ClaimedAt      *time.Time
	GrantID        string
	CreditsGranted int64
	FulfilledAt    *time.Time
	FailureReason  string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// RefundRequestWire is the wire format for a refund request.
type RefundRequestWire struct {
	ID               string
	Namespace        string
	OrderID          string
	CustomerID       string
	AmountMinor      int64
	Currency         string
	Status           RefundStatusWire
	Reason           string
	IdempotencyKey   string
	CreditQuantum    int64
	RefundQuantumFen int64
	ReservedCredits  int64
	RefundFen        int64
	RemainderCredits int64
	ProviderName     string
	ProviderRefundID string
	FenceSequence    string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// OfflinePaymentWire is the wire format for an offline payment.
type OfflinePaymentWire struct {
	ID          string
	Namespace   string
	AccountID   string
	AmountMinor int64
	Currency    string
	ConfirmedBy string
	ConfirmedAt time.Time
	Reference   string
	Note        string
}

// ReceivablePeriodWire is the wire format for a receivable period.
type ReceivablePeriodWire struct {
	ID          string
	Namespace   string
	AccountID   string
	Status      string
	PeriodStart time.Time
	PeriodEnd   time.Time
	TotalMinor  int64
	PaidMinor   int64
	Currency    string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// ExternalInvoiceRefWire is the input for updating an external invoice reference.
type ExternalInvoiceRefWire struct {
	PeriodID      string
	InvoiceNumber string
	InvoiceURL    string
	Issuer        string
	IssuedAt      *time.Time
}

// ===========================================================================
// PaymentAttempt Repository
// ===========================================================================

// CreatePaymentAttempt persists a new payment attempt idempotently.
func (a *EntAdapter) CreatePaymentAttempt(ctx context.Context, attempt PaymentAttemptWire) (*PaymentAttemptWire, bool, error) {
	existing, err := a.GetPaymentAttemptByIdempotencyKey(ctx, attempt.Namespace, attempt.CustomerID, attempt.IdempotencyKey)
	if err == nil && existing != nil {
		return existing, false, nil
	}
	if err != nil && !errors.Is(err, ErrPaymentAttemptNotFound) {
		return nil, false, fmt.Errorf("ent: payment attempt idempotency check: %w", err)
	}

	provider, err := mapAttemptProviderToEnt(attempt.Provider)
	if err != nil {
		return nil, false, err
	}

	saved, err := a.db.PaymentAttempt.Create().
		SetNamespace(attempt.Namespace).
		SetCommerceOrderID(attempt.OrderID).
		SetCustomerID(attempt.CustomerID).
		SetProvider(provider).
		SetStatus(paymentattempt.StatusCreated).
		SetNillableExpectedMerchantID(nonEmptyPtr(attempt.ExpectedMerchantID)).
		SetNillableExpectedApplicationID(nonEmptyPtr(attempt.ExpectedApplicationID)).
		SetIdempotencyKey(attempt.IdempotencyKey).
		SetAmountCents(attempt.AmountMinor).
		SetCurrency(attempt.Currency).
		SetCreatedAt(attempt.CreatedAt).
		SetUpdatedAt(attempt.UpdatedAt).
		Save(ctx)
	if err != nil {
		if entdb.IsConstraintError(err) {
			existing, gErr := a.GetPaymentAttemptByIdempotencyKey(ctx, attempt.Namespace, attempt.CustomerID, attempt.IdempotencyKey)
			if gErr != nil {
				return nil, false, fmt.Errorf("ent: concurrent insert recovery: %w", gErr)
			}
			return existing, false, nil
		}
		return nil, false, fmt.Errorf("ent: create payment attempt: %w", err)
	}
	return mapEntPaymentAttempt(saved), true, nil
}

func (a *EntAdapter) GetPaymentAttempt(ctx context.Context, namespace, id string) (*PaymentAttemptWire, error) {
	epa, err := a.db.PaymentAttempt.Get(ctx, id)
	if err != nil {
		if entdb.IsNotFound(err) {
			return nil, ErrPaymentAttemptNotFound
		}
		return nil, fmt.Errorf("ent: get payment attempt: %w", err)
	}
	if epa.Namespace != namespace {
		return nil, ErrPaymentAttemptNotFound
	}
	return mapEntPaymentAttempt(epa), nil
}

func (a *EntAdapter) GetPaymentAttemptByIdempotencyKey(ctx context.Context, namespace, customerID, key string) (*PaymentAttemptWire, error) {
	epa, err := a.db.PaymentAttempt.Query().
		Where(
			paymentattempt.NamespaceEQ(namespace),
			paymentattempt.CustomerIDEQ(customerID),
			paymentattempt.IdempotencyKeyEQ(key),
		).
		First(ctx)
	if err != nil {
		if entdb.IsNotFound(err) {
			return nil, ErrPaymentAttemptNotFound
		}
		return nil, fmt.Errorf("ent: get attempt by idempotency key: %w", err)
	}
	return mapEntPaymentAttempt(epa), nil
}

func (a *EntAdapter) GetPaymentAttemptByProviderOrder(ctx context.Context, namespace string, provider PaymentProviderWire, providerOrderID string) (*PaymentAttemptWire, error) {
	entProvider, err := mapAttemptProviderToEnt(provider)
	if err != nil {
		return nil, err
	}
	epa, err := a.db.PaymentAttempt.Query().
		Where(
			paymentattempt.NamespaceEQ(namespace),
			paymentattempt.ProviderEQ(entProvider),
			paymentattempt.ProviderOrderIDEQ(providerOrderID),
		).
		First(ctx)
	if err != nil {
		if entdb.IsNotFound(err) {
			return nil, ErrPaymentAttemptNotFound
		}
		return nil, fmt.Errorf("ent: get attempt by provider order: %w", err)
	}
	return mapEntPaymentAttempt(epa), nil
}

// ListStalePendingPaymentAttempts returns the callback-lost recovery batch in
// deterministic oldest-first order. The persistence boundary enforces the
// pending status and the production maximum batch size.
func (a *EntAdapter) ListStalePendingPaymentAttempts(ctx context.Context, namespace string, cutoff time.Time, limit int) ([]PaymentAttemptWire, error) {
	if limit <= 0 || limit > 100 {
		limit = 100
	}
	attempts, err := a.db.PaymentAttempt.Query().
		Where(
			paymentattempt.NamespaceEQ(namespace),
			paymentattempt.StatusEQ(paymentattempt.StatusPending),
			paymentattempt.UpdatedAtLTE(cutoff),
		).
		Order(paymentattempt.ByUpdatedAt(), paymentattempt.ByID()).
		Limit(limit).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("ent: list stale pending payment attempts: %w", err)
	}

	result := make([]PaymentAttemptWire, len(attempts))
	for i, attempt := range attempts {
		result[i] = *mapEntPaymentAttempt(attempt)
	}
	return result, nil
}

// ResolvePaymentAuthorityForOrder returns the successful payment attempt whose
// immutable provider identity, order, payment, and money fields are the
// persisted authority for refund callback ingestion.
func (a *EntAdapter) ResolvePaymentAuthorityForOrder(ctx context.Context, namespace, orderID string) (*PaymentAttemptWire, error) {
	attempts, err := a.listPaymentAttemptsForOrder(ctx, namespace, orderID)
	if err != nil {
		return nil, fmt.Errorf("ent: resolve payment authority for order: %w", err)
	}
	for _, attempt := range attempts {
		if attempt.Status == paymentattempt.StatusSucceeded {
			return mapEntPaymentAttempt(attempt), nil
		}
	}
	return nil, ErrPaymentAttemptNotFound
}

// ResolvePaymentProviderForOrder preserves the provider-only resolver contract
// for refund submission routing.
func (a *EntAdapter) ResolvePaymentProviderForOrder(ctx context.Context, namespace, orderID string) (PaymentProviderWire, error) {
	attempts, err := a.listPaymentAttemptsForOrder(ctx, namespace, orderID)
	if err != nil {
		return "", fmt.Errorf("ent: resolve payment provider for order: %w", err)
	}
	if len(attempts) == 0 {
		return "", ErrPaymentAttemptNotFound
	}
	for _, attempt := range attempts {
		if attempt.Status == paymentattempt.StatusSucceeded {
			return string(attempt.Provider), nil
		}
	}
	return string(attempts[0].Provider), nil
}

func (a *EntAdapter) listPaymentAttemptsForOrder(ctx context.Context, namespace, orderID string) ([]*entdb.PaymentAttempt, error) {
	return a.db.PaymentAttempt.Query().
		Where(
			paymentattempt.NamespaceEQ(namespace),
			paymentattempt.CommerceOrderIDEQ(orderID),
		).
		Order(paymentattempt.ByUpdatedAt(entsql.OrderDesc()), paymentattempt.ByID(entsql.OrderDesc())).
		All(ctx)
}

func (a *EntAdapter) UpdatePaymentAttemptStatus(ctx context.Context, namespace, id string, expectedFrom, to AttemptStatusWire) (*PaymentAttemptWire, error) {
	entFrom, err := mapAttemptStatusToEnt(expectedFrom)
	if err != nil {
		return nil, err
	}
	entTo, err := mapAttemptStatusToEnt(to)
	if err != nil {
		return nil, err
	}
	n, err := a.db.PaymentAttempt.Update().
		Where(
			paymentattempt.IDEQ(id),
			paymentattempt.NamespaceEQ(namespace),
			paymentattempt.StatusEQ(entFrom),
		).
		SetStatus(entTo).
		SetUpdatedAt(clock.Now()).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("ent: update attempt status: %w", err)
	}
	if n == 0 {
		return nil, ErrInvalidAttemptTransition
	}
	return a.GetPaymentAttempt(ctx, namespace, id)
}

func (a *EntAdapter) SetPaymentAttemptProviderIDs(ctx context.Context, namespace, id, providerOrderID, providerPaymentID, sessionID string) (*PaymentAttemptWire, error) {
	n, err := a.db.PaymentAttempt.Update().
		Where(paymentattempt.IDEQ(id), paymentattempt.NamespaceEQ(namespace)).
		SetNillableProviderOrderID(nonEmptyPtr(providerOrderID)).
		SetNillableProviderPaymentID(nonEmptyPtr(providerPaymentID)).
		SetNillableProviderSessionID(nonEmptyPtr(sessionID)).
		SetUpdatedAt(clock.Now()).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("ent: set provider ids: %w", err)
	}
	if n == 0 {
		return nil, ErrPaymentAttemptNotFound
	}
	return a.GetPaymentAttempt(ctx, namespace, id)
}

// ===========================================================================
// PaymentFact Repository
// ===========================================================================

func (a *EntAdapter) InsertPaymentFact(ctx context.Context, fact PaymentFactWire) (*PaymentFactWire, bool, error) {
	existing, err := a.GetPaymentFactByRawHash(ctx, fact.Namespace, fact.RawHash)
	if err == nil && existing != nil {
		if err := ensurePaymentFactReplayMatches(existing, fact); err != nil {
			return nil, false, err
		}
		return existing, false, nil
	}
	if err != nil && !errors.Is(err, ErrPaymentFactNotFound) {
		return nil, false, fmt.Errorf("ent: fact dedup check: %w", err)
	}

	provider, err := mapFactProviderToEnt(fact.Provider)
	if err != nil {
		return nil, false, err
	}

	create := a.db.PaymentFact.Create().
		SetID(fact.ID).
		SetNamespace(fact.Namespace).
		SetPaymentAttemptID(fact.AttemptID).
		SetRawHash(fact.RawHash).
		SetProvider(provider).
		SetProviderOrderID(fact.ProviderOrderID).
		SetNillableProviderPaymentID(nonEmptyPtr(fact.ProviderPaymentID)).
		SetNillableProviderEventID(nonEmptyPtr(fact.ProviderEventID)).
		SetNillableMerchantID(nonEmptyPtr(fact.MerchantID)).
		SetNillableApplicationID(nonEmptyPtr(fact.ApplicationID)).
		SetAmountMinor(fact.AmountMinor).
		SetCurrency(fact.Currency).
		SetSuccess(fact.Success).
		SetSignedPayload(fact.SignedPayload).
		SetTimestamp(fact.Timestamp).
		SetCreatedAt(fact.CreatedAt)
	saved, err := create.Save(ctx)
	if err != nil {
		if entdb.IsConstraintError(err) {
			existing, gErr := a.GetPaymentFactByRawHash(ctx, fact.Namespace, fact.RawHash)
			if gErr == nil {
				if err := ensurePaymentFactReplayMatches(existing, fact); err != nil {
					return nil, false, err
				}
				return existing, false, nil
			}
			if !errors.Is(gErr, ErrPaymentFactNotFound) {
				return nil, false, fmt.Errorf("ent: concurrent fact recovery: %w", gErr)
			}
			if fact.ProviderEventID != "" {
				existing, gErr = a.GetPaymentFactByProviderEvent(ctx, fact.Namespace, fact.Provider, fact.ProviderEventID)
				if gErr == nil {
					if err := ensurePaymentFactReplayMatches(existing, fact); err != nil {
						return nil, false, err
					}
					return existing, false, nil
				}
			}
			return nil, false, fmt.Errorf("ent: concurrent fact recovery: %w", gErr)
		}
		return nil, false, fmt.Errorf("ent: insert payment fact: %w", err)
	}
	return mapEntPaymentFact(saved), true, nil
}

// ensurePaymentFactReplayMatches prevents raw-hash or provider-event
// uniqueness conflicts from accepting a different verified structured fact as
// an idempotent replay. RawHash, SignedPayload, CreatedAt, and ID are excluded
// because they identify the transport observation or local record, not the
// provider fact itself.
func ensurePaymentFactReplayMatches(existing *PaymentFactWire, incoming PaymentFactWire) error {
	if existing == nil ||
		existing.Namespace != incoming.Namespace ||
		existing.AttemptID != incoming.AttemptID ||
		existing.Provider != incoming.Provider ||
		existing.ProviderOrderID != incoming.ProviderOrderID ||
		!paymentFactOptionalWireFieldMatches(existing.ProviderPaymentID, incoming.ProviderPaymentID) ||
		!paymentFactOptionalWireFieldMatches(existing.ProviderEventID, incoming.ProviderEventID) ||
		!paymentFactOptionalWireFieldMatches(existing.MerchantID, incoming.MerchantID) ||
		!paymentFactOptionalWireFieldMatches(existing.ApplicationID, incoming.ApplicationID) ||
		existing.AmountMinor != incoming.AmountMinor ||
		existing.Currency != incoming.Currency ||
		existing.Success != incoming.Success ||
		!existing.Timestamp.Equal(incoming.Timestamp) {
		return errors.New("ent: persisted payment fact does not match incoming verified fact")
	}
	return nil
}

func paymentFactOptionalWireFieldMatches(existing, incoming string) bool {
	existingValue := nonEmptyPtr(existing)
	incomingValue := nonEmptyPtr(incoming)
	if existingValue == nil || incomingValue == nil {
		return existingValue == nil && incomingValue == nil
	}
	return *existingValue == *incomingValue
}

func (a *EntAdapter) GetPaymentFactByRawHash(ctx context.Context, namespace, rawHash string) (*PaymentFactWire, error) {
	epf, err := a.db.PaymentFact.Query().
		Where(
			paymentfact.NamespaceEQ(namespace),
			paymentfact.RawHashEQ(rawHash),
		).
		First(ctx)
	if err != nil {
		if entdb.IsNotFound(err) {
			return nil, ErrPaymentFactNotFound
		}
		return nil, fmt.Errorf("ent: get fact by raw hash: %w", err)
	}
	return mapEntPaymentFact(epf), nil
}

func (a *EntAdapter) GetPaymentFactsByProviderOrder(ctx context.Context, namespace string, provider PaymentProviderWire, providerOrderID string) ([]PaymentFactWire, error) {
	entProvider, err := mapFactProviderToEnt(provider)
	if err != nil {
		return nil, err
	}
	facts, err := a.db.PaymentFact.Query().
		Where(
			paymentfact.NamespaceEQ(namespace),
			paymentfact.ProviderEQ(entProvider),
			paymentfact.ProviderOrderIDEQ(providerOrderID),
		).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("ent: get facts by provider order: %w", err)
	}
	result := make([]PaymentFactWire, len(facts))
	for i, f := range facts {
		result[i] = *mapEntPaymentFact(f)
	}
	return result, nil
}

// ===========================================================================
// Fulfillment Repository
// ===========================================================================

func (a *EntAdapter) CreateFulfillment(ctx context.Context, req FulfillmentCreateWire) (*FulfillmentWire, error) {
	saved, err := a.db.Fulfillment.Create().
		SetNamespace(req.Namespace).
		SetCommerceOrderID(req.OrderID).
		SetCustomerID(req.CustomerID).
		SetStatus(fulfillment.StatusPending).
		Save(ctx)
	if err != nil {
		if entdb.IsConstraintError(err) {
			existing, gErr := a.GetFulfillmentByOrder(ctx, req.Namespace, req.OrderID)
			if gErr != nil {
				return nil, fmt.Errorf("ent: fulfillment duplicate recovery: %w", gErr)
			}
			return existing, nil
		}
		return nil, fmt.Errorf("ent: create fulfillment: %w", err)
	}
	return mapEntFulfillment(saved), nil
}

func (a *EntAdapter) GetFulfillment(ctx context.Context, namespace, id string) (*FulfillmentWire, error) {
	ef, err := a.db.Fulfillment.Get(ctx, id)
	if err != nil {
		if entdb.IsNotFound(err) {
			return nil, ErrFulfillmentNotFound
		}
		return nil, fmt.Errorf("ent: get fulfillment: %w", err)
	}
	if ef.Namespace != namespace {
		return nil, ErrFulfillmentNotFound
	}
	return mapEntFulfillment(ef), nil
}

func (a *EntAdapter) GetFulfillmentByOrder(ctx context.Context, namespace, orderID string) (*FulfillmentWire, error) {
	ef, err := a.db.Fulfillment.Query().
		Where(
			fulfillment.NamespaceEQ(namespace),
			fulfillment.CommerceOrderIDEQ(orderID),
		).
		First(ctx)
	if err != nil {
		if entdb.IsNotFound(err) {
			return nil, ErrFulfillmentNotFound
		}
		return nil, fmt.Errorf("ent: get fulfillment by order: %w", err)
	}
	return mapEntFulfillment(ef), nil
}

func (a *EntAdapter) ListPendingFulfillments(ctx context.Context, namespace string, limit int) ([]FulfillmentWire, error) {
	if limit <= 0 {
		limit = 50
	}
	efs, err := a.db.Fulfillment.Query().
		Where(
			fulfillment.NamespaceEQ(namespace),
			fulfillment.StatusEQ(fulfillment.StatusPending),
		).
		Limit(limit).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("ent: list pending fulfillments: %w", err)
	}
	result := make([]FulfillmentWire, 0, len(efs))
	for _, ef := range efs {
		result = append(result, *mapEntFulfillment(ef))
	}
	return result, nil
}

func (a *EntAdapter) ClaimFulfillment(ctx context.Context, namespace, id string) (*FulfillmentWire, error) {
	now := clock.Now()
	n, err := a.db.Fulfillment.Update().
		Where(
			fulfillment.IDEQ(id),
			fulfillment.NamespaceEQ(namespace),
			fulfillment.StatusEQ(fulfillment.StatusPending),
		).
		SetStatus(fulfillment.StatusProcessing).
		SetClaimedAt(now).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("ent: claim fulfillment: %w", err)
	}
	if n == 0 {
		return nil, ErrFulfillmentAlreadyClaimed
	}
	return a.GetFulfillment(ctx, namespace, id)
}

func (a *EntAdapter) MarkFulfillmentFulfilled(ctx context.Context, namespace, id string, grantID string, creditsGranted int64) (*FulfillmentWire, error) {
	now := clock.Now()
	n, err := a.db.Fulfillment.Update().
		Where(
			fulfillment.IDEQ(id),
			fulfillment.NamespaceEQ(namespace),
			fulfillment.StatusEQ(fulfillment.StatusProcessing),
		).
		SetStatus(fulfillment.StatusFulfilled).
		SetGrantID(grantID).
		SetCreditsGranted(creditsGranted).
		SetFulfilledAt(now).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("ent: mark fulfillment fulfilled: %w", err)
	}
	if n == 0 {
		return nil, ErrInvalidFulfillmentTransition
	}
	return a.GetFulfillment(ctx, namespace, id)
}

func (a *EntAdapter) MarkFulfillmentFailed(ctx context.Context, namespace, id, reason string) (*FulfillmentWire, error) {
	n, err := a.db.Fulfillment.Update().
		Where(
			fulfillment.IDEQ(id),
			fulfillment.NamespaceEQ(namespace),
			fulfillment.StatusEQ(fulfillment.StatusProcessing),
		).
		SetStatus(fulfillment.StatusFailed).
		SetFailureReason(reason).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("ent: mark fulfillment failed: %w", err)
	}
	if n == 0 {
		return nil, ErrInvalidFulfillmentTransition
	}
	return a.GetFulfillment(ctx, namespace, id)
}

func (a *EntAdapter) RecoverExpiredFulfillmentLeases(ctx context.Context, namespace string, leaseDuration time.Duration) (int, error) {
	cutoff := clock.Now().Add(-leaseDuration)
	n, err := a.db.Fulfillment.Update().
		Where(
			fulfillment.NamespaceEQ(namespace),
			fulfillment.StatusEQ(fulfillment.StatusProcessing),
			fulfillment.ClaimedAtLTE(cutoff),
		).
		SetStatus(fulfillment.StatusPending).
		ClearClaimedAt().
		Save(ctx)
	if err != nil {
		return 0, fmt.Errorf("ent: recover expired leases: %w", err)
	}
	return n, nil
}

// ===========================================================================
// RefundRequest Repository
// ===========================================================================

func (a *EntAdapter) CreateRefundRequest(ctx context.Context, req RefundRequestWire) (*RefundRequestWire, bool, error) {
	existing, err := a.GetRefundRequestByIdempotencyKey(ctx, req.Namespace, req.CustomerID, req.IdempotencyKey)
	if err == nil && existing != nil {
		return existing, false, nil
	}
	if err != nil && !errors.Is(err, ErrRefundNotFound) {
		return nil, false, fmt.Errorf("ent: refund idempotency check: %w", err)
	}

	saved, err := a.db.RefundRequest.Create().
		SetNamespace(req.Namespace).
		SetCommerceOrderID(req.OrderID).
		SetCustomerID(req.CustomerID).
		SetAmountCents(req.AmountMinor).
		SetCurrency(req.Currency).
		SetStatus(refundrequest.StatusPendingFence).
		SetIdempotencyKey(req.IdempotencyKey).
		SetCreditQuantum(req.CreditQuantum).
		SetRefundQuantumFen(req.RefundQuantumFen).
		SetReason(req.Reason).
		Save(ctx)
	if err != nil {
		if entdb.IsConstraintError(err) {
			existing, gErr := a.GetRefundRequestByIdempotencyKey(ctx, req.Namespace, req.CustomerID, req.IdempotencyKey)
			if gErr != nil {
				return nil, false, fmt.Errorf("ent: concurrent refund recovery: %w", gErr)
			}
			return existing, false, nil
		}
		return nil, false, fmt.Errorf("ent: create refund request: %w", err)
	}
	return mapEntRefundRequest(saved), true, nil
}

func (a *EntAdapter) GetRefundRequest(ctx context.Context, namespace, id string) (*RefundRequestWire, error) {
	er, err := a.db.RefundRequest.Get(ctx, id)
	if err != nil {
		if entdb.IsNotFound(err) {
			return nil, ErrRefundNotFound
		}
		return nil, fmt.Errorf("ent: get refund request: %w", err)
	}
	if er.Namespace != namespace {
		return nil, ErrRefundNotFound
	}
	return mapEntRefundRequest(er), nil
}

func (a *EntAdapter) GetRefundRequestByIdempotencyKey(ctx context.Context, namespace, customerID, key string) (*RefundRequestWire, error) {
	er, err := a.db.RefundRequest.Query().
		Where(
			refundrequest.NamespaceEQ(namespace),
			refundrequest.CustomerIDEQ(customerID),
			refundrequest.IdempotencyKeyEQ(key),
		).
		First(ctx)
	if err != nil {
		if entdb.IsNotFound(err) {
			return nil, ErrRefundNotFound
		}
		return nil, fmt.Errorf("ent: get refund by idempotency key: %w", err)
	}
	return mapEntRefundRequest(er), nil
}

// ListProcessableRefundRequests returns every non-terminal refund state that
// ProcessOne can advance. This includes initial submissions and crash recovery
// after a provider success, not only provider-status polling.
func (a *EntAdapter) ListProcessableRefundRequests(ctx context.Context, namespace string, limit int) ([]RefundRequestWire, error) {
	if limit <= 0 || limit > 100 {
		limit = 100
	}
	requests, err := a.db.RefundRequest.Query().
		Where(
			refundrequest.NamespaceEQ(namespace),
			refundrequest.StatusIn(
				refundrequest.StatusPendingFence,
				refundrequest.StatusProviderProcessing,
				refundrequest.StatusLedgerReversing,
			),
		).
		Order(refundrequest.ByUpdatedAt(), refundrequest.ByID()).
		Limit(limit).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("ent: list processable refunds: %w", err)
	}
	result := make([]RefundRequestWire, len(requests))
	for i, request := range requests {
		result[i] = *mapEntRefundRequest(request)
	}
	return result, nil
}

func (a *EntAdapter) UpdateRefundStatus(ctx context.Context, namespace, id string, to RefundStatusWire) (*RefundRequestWire, error) {
	entTo, err := mapRefundStatusToEnt(to)
	if err != nil {
		return nil, err
	}
	n, err := a.db.RefundRequest.Update().
		Where(refundrequest.IDEQ(id), refundrequest.NamespaceEQ(namespace)).
		SetStatus(entTo).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("ent: update refund status: %w", err)
	}
	if n == 0 {
		return nil, ErrRefundNotFound
	}
	return a.GetRefundRequest(ctx, namespace, id)
}

func (a *EntAdapter) SetRefundProviderDetails(ctx context.Context, namespace, id, providerName, providerRefundID string, reservedCredits, refundFen, remainderCredits int64) (*RefundRequestWire, error) {
	n, err := a.db.RefundRequest.Update().
		Where(refundrequest.IDEQ(id), refundrequest.NamespaceEQ(namespace)).
		SetProviderName(providerName).
		SetProviderRefundID(providerRefundID).
		SetReservedCredits(reservedCredits).
		SetRefundFen(refundFen).
		SetRemainderCredits(remainderCredits).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("ent: set refund provider details: %w", err)
	}
	if n == 0 {
		return nil, ErrRefundNotFound
	}
	return a.GetRefundRequest(ctx, namespace, id)
}

// ===========================================================================
// Offline Payment Repository
// ===========================================================================

func (a *EntAdapter) CreateOfflinePayment(ctx context.Context, payment OfflinePaymentWire) (*OfflinePaymentWire, error) {
	saved, err := a.db.OfflinePayment.Create().
		SetNamespace(payment.Namespace).
		SetReceivableAccountID(payment.AccountID).
		SetAmountCents(payment.AmountMinor).
		SetCurrency(payment.Currency).
		SetConfirmedBy(payment.ConfirmedBy).
		SetConfirmedAt(payment.ConfirmedAt).
		SetReference(payment.Reference).
		SetNote(payment.Note).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("ent: create offline payment: %w", err)
	}
	return mapEntOfflinePayment(saved), nil
}

func (a *EntAdapter) ListOfflinePaymentsByAccount(ctx context.Context, namespace, accountID string) ([]OfflinePaymentWire, error) {
	eops, err := a.db.OfflinePayment.Query().
		Where(
			offlinepayment.NamespaceEQ(namespace),
			offlinepayment.ReceivableAccountIDEQ(accountID),
		).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("ent: list offline payments: %w", err)
	}
	result := make([]OfflinePaymentWire, 0, len(eops))
	for _, eop := range eops {
		result = append(result, *mapEntOfflinePayment(eop))
	}
	return result, nil
}

// ===========================================================================
// Receivable Period Repository
// ===========================================================================

func (a *EntAdapter) ListReceivablePeriodsByAccount(ctx context.Context, namespace, accountID string) ([]ReceivablePeriodWire, error) {
	eps, err := a.db.ReceivablePeriod.Query().
		Where(
			receivableperiod.NamespaceEQ(namespace),
			receivableperiod.ReceivableAccountIDEQ(accountID),
		).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("ent: list receivable periods: %w", err)
	}
	result := make([]ReceivablePeriodWire, 0, len(eps))
	for _, ep := range eps {
		result = append(result, *mapEntReceivablePeriod(ep))
	}
	return result, nil
}

func (a *EntAdapter) UpdateExternalInvoiceRef(ctx context.Context, namespace string, ref ExternalInvoiceRefWire) error {
	// Create or update an external invoice ref linked to the receivable period.
	_, err := a.db.ExternalInvoiceRef.Create().
		SetNamespace(namespace).
		SetReceivablePeriodID(ref.PeriodID).
		SetInvoiceNumber(ref.InvoiceNumber).
		SetNillableInvoiceURL(&ref.InvoiceURL).
		Save(ctx)
	if err != nil {
		if entdb.IsConstraintError(err) {
			// Update existing.
			_, uErr := a.db.ExternalInvoiceRef.Update().
				Where(
					externalinvoiceref.NamespaceEQ(namespace),
					externalinvoiceref.ReceivablePeriodIDEQ(ref.PeriodID),
				).
				SetNillableInvoiceURL(&ref.InvoiceURL).
				Save(ctx)
			if uErr != nil {
				return fmt.Errorf("ent: update external invoice ref: %w", uErr)
			}
			return nil
		}
		return fmt.Errorf("ent: create external invoice ref: %w", err)
	}
	return nil
}

// ===========================================================================
// Ent mappers
// ===========================================================================

func mapEntPaymentAttempt(epa *entdb.PaymentAttempt) *PaymentAttemptWire {
	w := &PaymentAttemptWire{
		ID:             epa.ID,
		Namespace:      epa.Namespace,
		OrderID:        epa.CommerceOrderID,
		CustomerID:     epa.CustomerID,
		Provider:       string(epa.Provider),
		Status:         string(epa.Status),
		IdempotencyKey: epa.IdempotencyKey,
		AmountMinor:    epa.AmountCents,
		Currency:       epa.Currency,
		CreatedAt:      epa.CreatedAt,
		UpdatedAt:      epa.UpdatedAt,
	}
	if epa.ProviderOrderID != nil {
		w.ProviderOrderID = *epa.ProviderOrderID
	}
	if epa.ProviderPaymentID != nil {
		w.ProviderPaymentID = *epa.ProviderPaymentID
	}
	if epa.ProviderSessionID != nil {
		w.ProviderSessionID = *epa.ProviderSessionID
	}
	if epa.ExpectedMerchantID != nil {
		w.ExpectedMerchantID = *epa.ExpectedMerchantID
	}
	if epa.ExpectedApplicationID != nil {
		w.ExpectedApplicationID = *epa.ExpectedApplicationID
	}
	return w
}

func mapEntPaymentFact(epf *entdb.PaymentFact) *PaymentFactWire {
	w := &PaymentFactWire{
		ID:              epf.ID,
		Namespace:       epf.Namespace,
		AttemptID:       epf.PaymentAttemptID,
		Provider:        string(epf.Provider),
		ProviderOrderID: epf.ProviderOrderID,
		AmountMinor:     epf.AmountMinor,
		Currency:        epf.Currency,
		Success:         epf.Success,
		RawHash:         epf.RawHash,
		SignedPayload:   epf.SignedPayload,
		Timestamp:       epf.Timestamp,
		CreatedAt:       epf.CreatedAt,
	}
	if epf.ProviderPaymentID != nil {
		w.ProviderPaymentID = *epf.ProviderPaymentID
	}
	if epf.ProviderEventID != nil {
		w.ProviderEventID = *epf.ProviderEventID
	}
	if epf.MerchantID != nil {
		w.MerchantID = *epf.MerchantID
	}
	if epf.ApplicationID != nil {
		w.ApplicationID = *epf.ApplicationID
	}
	return w
}

func mapEntFulfillment(ef *entdb.Fulfillment) *FulfillmentWire {
	w := &FulfillmentWire{
		ID:             ef.ID,
		Namespace:      ef.Namespace,
		OrderID:        ef.CommerceOrderID,
		CustomerID:     ef.CustomerID,
		Status:         string(ef.Status),
		CreditsGranted: ef.CreditsGranted,
		CreatedAt:      ef.CreatedAt,
		UpdatedAt:      ef.UpdatedAt,
	}
	if ef.GrantID != nil {
		w.GrantID = *ef.GrantID
	}
	if ef.FailureReason != nil {
		w.FailureReason = *ef.FailureReason
	}
	if ef.ClaimedAt != nil {
		w.ClaimedAt = ef.ClaimedAt
	}
	if ef.FulfilledAt != nil {
		w.FulfilledAt = ef.FulfilledAt
	}
	return w
}

func mapEntRefundRequest(er *entdb.RefundRequest) *RefundRequestWire {
	w := &RefundRequestWire{
		ID:               er.ID,
		Namespace:        er.Namespace,
		OrderID:          er.CommerceOrderID,
		CustomerID:       er.CustomerID,
		AmountMinor:      er.AmountCents,
		Currency:         er.Currency,
		Status:           string(er.Status),
		IdempotencyKey:   er.IdempotencyKey,
		CreditQuantum:    er.CreditQuantum,
		RefundQuantumFen: er.RefundQuantumFen,
		ReservedCredits:  er.ReservedCredits,
		RefundFen:        er.RefundFen,
		RemainderCredits: er.RemainderCredits,
		ProviderName:     er.ProviderName,
		ProviderRefundID: er.ProviderRefundID,
		FenceSequence:    er.FenceSequence,
		CreatedAt:        er.CreatedAt,
		UpdatedAt:        er.UpdatedAt,
	}
	if er.Reason != nil {
		w.Reason = *er.Reason
	}
	return w
}

func mapEntOfflinePayment(eop *entdb.OfflinePayment) *OfflinePaymentWire {
	w := &OfflinePaymentWire{
		ID:          eop.ID,
		Namespace:   eop.Namespace,
		AccountID:   eop.ReceivableAccountID,
		AmountMinor: eop.AmountCents,
		Currency:    eop.Currency,
		ConfirmedBy: eop.ConfirmedBy,
		ConfirmedAt: eop.ConfirmedAt,
	}
	if eop.Reference != nil {
		w.Reference = *eop.Reference
	}
	if eop.Note != nil {
		w.Note = *eop.Note
	}
	return w
}

func mapEntReceivablePeriod(ep *entdb.ReceivablePeriod) *ReceivablePeriodWire {
	return &ReceivablePeriodWire{
		ID:          ep.ID,
		Namespace:   ep.Namespace,
		AccountID:   ep.ReceivableAccountID,
		Status:      string(ep.Status),
		PeriodStart: ep.PeriodStart,
		PeriodEnd:   ep.PeriodEnd,
		TotalMinor:  ep.TotalCents,
		PaidMinor:   ep.PaidCents,
		Currency:    ep.Currency,
		CreatedAt:   ep.CreatedAt,
		UpdatedAt:   ep.UpdatedAt,
	}
}

// ===========================================================================
// Enum mappers
// ===========================================================================

func mapAttemptProviderToEnt(p PaymentProviderWire) (paymentattempt.Provider, error) {
	switch p {
	case "wechat":
		return paymentattempt.ProviderWechat, nil
	case "alipay":
		return paymentattempt.ProviderAlipay, nil
	case "offline":
		return paymentattempt.ProviderOffline, nil
	default:
		return "", fmt.Errorf("unknown payment provider: %s", p)
	}
}

func mapFactProviderToEnt(p PaymentProviderWire) (paymentfact.Provider, error) {
	switch p {
	case "wechat":
		return paymentfact.ProviderWechat, nil
	case "alipay":
		return paymentfact.ProviderAlipay, nil
	case "offline":
		return paymentfact.ProviderOffline, nil
	default:
		return "", fmt.Errorf("unknown payment provider: %s", p)
	}
}

func mapAttemptStatusToEnt(s AttemptStatusWire) (paymentattempt.Status, error) {
	switch s {
	case "created":
		return paymentattempt.StatusCreated, nil
	case "pending":
		return paymentattempt.StatusPending, nil
	case "succeeded":
		return paymentattempt.StatusSucceeded, nil
	case "failed":
		return paymentattempt.StatusFailed, nil
	case "closed":
		return paymentattempt.StatusClosed, nil
	default:
		return "", fmt.Errorf("unknown attempt status: %s", s)
	}
}

func mapRefundStatusToEnt(s RefundStatusWire) (refundrequest.Status, error) {
	switch s {
	case "pending_fence":
		return refundrequest.StatusPendingFence, nil
	case "provider_processing":
		return refundrequest.StatusProviderProcessing, nil
	case "ledger_reversing":
		return refundrequest.StatusLedgerReversing, nil
	case "fulfilled":
		return refundrequest.StatusFulfilled, nil
	case "failed":
		return refundrequest.StatusFailed, nil
	default:
		return "", fmt.Errorf("unknown refund status: %s", s)
	}
}

func newULID() string {
	return ulid.Make().String()
}

// ===========================================================================
// Refund Repository — missing methods
// ===========================================================================

// GetRefundRequestByProviderRefundID looks up a refund by its provider refund ID.
func (a *EntAdapter) GetRefundRequestByProviderRefundID(ctx context.Context, namespace, providerRefundID string) (*RefundRequestWire, error) {
	er, err := a.db.RefundRequest.Query().
		Where(
			refundrequest.NamespaceEQ(namespace),
			refundrequest.ProviderRefundIDEQ(providerRefundID),
		).
		Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("ent: get refund by provider refund id: %w", err)
	}
	return mapEntRefundRequest(er), nil
}

// TransitionRefundStatus atomically transitions the refund status with
// optimistic concurrency (expectedFrom must match current status).
func (a *EntAdapter) TransitionRefundStatus(ctx context.Context, namespace, id string, expectedFrom, to RefundStatusWire) (*RefundRequestWire, error) {
	entExpected, err := mapRefundStatusToEnt(expectedFrom)
	if err != nil {
		return nil, err
	}
	entTo, err := mapRefundStatusToEnt(to)
	if err != nil {
		return nil, err
	}
	n, err := a.db.RefundRequest.Update().
		Where(
			refundrequest.NamespaceEQ(namespace),
			refundrequest.IDEQ(id),
			refundrequest.StatusEQ(entExpected),
		).
		SetStatus(entTo).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("ent: transition refund status: %w", err)
	}
	if n == 0 {
		return nil, fmt.Errorf("ent: refund status transition conflict (expected %s)", expectedFrom)
	}
	return a.GetRefundRequest(ctx, namespace, id)
}

// SaveRefundQuantum persists the quantum reservation values.
func (a *EntAdapter) SaveRefundQuantum(ctx context.Context, namespace, id string, creditQuantum, refundQuantumFen, reservedCredits, refundFen, remainderCredits int64) (*RefundRequestWire, error) {
	n, err := a.db.RefundRequest.Update().
		Where(
			refundrequest.NamespaceEQ(namespace),
			refundrequest.IDEQ(id),
		).
		SetCreditQuantum(creditQuantum).
		SetRefundQuantumFen(refundQuantumFen).
		SetReservedCredits(reservedCredits).
		SetRefundFen(refundFen).
		SetRemainderCredits(remainderCredits).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("ent: save refund quantum: %w", err)
	}
	if n == 0 {
		return nil, fmt.Errorf("ent: refund not found: %s", id)
	}
	return a.GetRefundRequest(ctx, namespace, id)
}

// SetRefundFence sets the fence sequence on a refund.
func (a *EntAdapter) SetRefundFence(ctx context.Context, namespace, id, fenceSequence string) (*RefundRequestWire, error) {
	n, err := a.db.RefundRequest.Update().
		Where(
			refundrequest.NamespaceEQ(namespace),
			refundrequest.IDEQ(id),
		).
		SetFenceSequence(fenceSequence).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("ent: set refund fence: %w", err)
	}
	if n == 0 {
		return nil, fmt.Errorf("ent: refund not found: %s", id)
	}
	return a.GetRefundRequest(ctx, namespace, id)
}

// MarkRefundFailed marks a refund as failed with a reason.
func (a *EntAdapter) MarkRefundFailed(ctx context.Context, namespace, id, reason string) (*RefundRequestWire, error) {
	entFailed, _ := mapRefundStatusToEnt("failed")
	n, err := a.db.RefundRequest.Update().
		Where(
			refundrequest.NamespaceEQ(namespace),
			refundrequest.IDEQ(id),
		).
		SetStatus(entFailed).
		SetFailureReason(reason).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("ent: mark refund failed: %w", err)
	}
	if n == 0 {
		return nil, fmt.Errorf("ent: refund not found: %s", id)
	}
	return a.GetRefundRequest(ctx, namespace, id)
}

// RefundFactWire is the wire format for a refund fact record.
type RefundFactWire struct {
	ID              string
	Namespace       string
	RefundRequestID string
	Provider        PaymentProviderWire
	RawHash         string
	SignedPayload   map[string]any
	Timestamp       time.Time
	CreatedAt       time.Time
}

// AppendRefundFact persists an immutable refund fact record (dedup on RawHash).
func (a *EntAdapter) AppendRefundFact(ctx context.Context, fact RefundFactWire) (*RefundFactWire, bool, error) {
	var entProvider refundfact.Provider
	switch fact.Provider {
	case "wechat":
		entProvider = refundfact.ProviderWechat
	case "alipay":
		entProvider = refundfact.ProviderAlipay
	case "offline":
		entProvider = refundfact.ProviderOffline
	default:
		return nil, false, fmt.Errorf("unknown provider: %s", fact.Provider)
	}
	created, err := a.db.RefundFact.Create().
		SetNamespace(fact.Namespace).
		SetRefundRequestID(fact.RefundRequestID).
		SetRawHash(fact.RawHash).
		SetProvider(entProvider).
		SetSignedPayload(fact.SignedPayload).
		SetTimestamp(fact.Timestamp).
		Save(ctx)
	if err != nil {
		if entdb.IsConstraintError(err) {
			existing, gErr := a.db.RefundFact.Query().
				Where(
					refundfact.NamespaceEQ(fact.Namespace),
					refundfact.RawHashEQ(fact.RawHash),
				).
				Only(ctx)
			if gErr != nil {
				return nil, false, fmt.Errorf("ent: get existing refund fact: %w", gErr)
			}
			return mapEntRefundFact(existing), false, nil
		}
		return nil, false, fmt.Errorf("ent: create refund fact: %w", err)
	}
	return mapEntRefundFact(created), true, nil
}

// GetRefundFacts retrieves all facts for a refund request.
func (a *EntAdapter) GetRefundFacts(ctx context.Context, namespace, refundID string) ([]RefundFactWire, error) {
	facts, err := a.db.RefundFact.Query().
		Where(
			refundfact.NamespaceEQ(namespace),
			refundfact.RefundRequestIDEQ(refundID),
		).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("ent: get refund facts: %w", err)
	}
	result := make([]RefundFactWire, len(facts))
	for i, f := range facts {
		result[i] = *mapEntRefundFact(f)
	}
	return result, nil
}

func mapEntRefundFact(ef *entdb.RefundFact) *RefundFactWire {
	return &RefundFactWire{
		ID:              ef.ID,
		Namespace:       ef.Namespace,
		RefundRequestID: ef.RefundRequestID,
		Provider:        string(ef.Provider),
		RawHash:         ef.RawHash,
		SignedPayload:   ef.SignedPayload,
		Timestamp:       ef.Timestamp,
		CreatedAt:       ef.CreatedAt,
	}
}

func (a *EntAdapter) GetPaymentFactByProviderEvent(ctx context.Context, namespace string, provider PaymentProviderWire, providerEventID string) (*PaymentFactWire, error) {
	entProvider, err := mapFactProviderToEnt(provider)
	if err != nil {
		return nil, err
	}
	ep, err := a.db.PaymentFact.Query().
		Where(
			paymentfact.NamespaceEQ(namespace),
			paymentfact.ProviderEQ(entProvider),
			paymentfact.ProviderEventIDEQ(providerEventID),
		).
		First(ctx)
	if err != nil {
		if entdb.IsNotFound(err) {
			return nil, ErrPaymentFactNotFound
		}
		return nil, fmt.Errorf("ent: get fact by provider event: %w", err)
	}
	return mapEntPaymentFact(ep), nil
}

func nonEmptyPtr(v string) *string {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	return &v
}
