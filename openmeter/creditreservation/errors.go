package creditreservation

import "errors"

var (
	ErrStateConflict          = errors.New("credit reservation state conflict")
	ErrCreditCurrencyRequired = errors.New("CREDIT managed custom currency is required")
	ErrAmbiguousRate          = errors.New("ambiguous product catalog rate")
	ErrRateNotFound           = errors.New("product catalog rate not found")
	ErrSubscriptionNotFound   = errors.New("active subscription not found")
	ErrAmbiguousSubscription  = errors.New("multiple active subscriptions found")
	ErrUnitPriceRequired      = errors.New("unit price is required for credit reservation")
	ErrInvalidQuantity        = errors.New("quantity must be non-negative")
	ErrCreditOverflow         = errors.New("credit calculation overflow")
)
