package creditreservation

import "errors"

var (
	ErrStateConflict              = errors.New("credit reservation state conflict")
	ErrCreditCurrencyRequired     = errors.New("CREDIT managed custom currency is required")
	ErrAmbiguousRate              = errors.New("ambiguous product catalog rate")
	ErrRateNotFound               = errors.New("product catalog rate not found")
	ErrSubscriptionNotFound       = errors.New("active subscription not found")
	ErrAmbiguousSubscription      = errors.New("multiple active subscriptions found")
	ErrUnitPriceRequired          = errors.New("unit price is required for credit reservation")
	ErrInvalidQuantity            = errors.New("quantity must be non-negative")
	ErrCreditOverflow             = errors.New("credit calculation overflow")
	ErrResourceLinesRequired      = errors.New("at least one resource line is required")
	ErrInvalidCommandIdentity     = errors.New("valid idempotency key and payload hash are required")
	ErrIdempotencyConflict        = errors.New("credit reservation idempotency conflict")
	ErrTransitionEvidenceRequired = errors.New("transition evidence is required")
	ErrInsufficientFunds          = errors.New("insufficient credit funds")
	ErrCustomerFenced             = errors.New("customer credit reservations are fenced")
	ErrRefundFenceNotFound        = errors.New("active refund fence not found")
	ErrFenceSequenceConflict      = errors.New("refund fence sequence conflict")
	ErrSettlementNotConfigured    = errors.New("credit settlement collector is not configured")
	ErrSettlementProvenanceAbsent = errors.New("credit settlement provenance is not persisted")
)
