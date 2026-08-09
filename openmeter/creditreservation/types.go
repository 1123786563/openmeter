// Package creditreservation defines the synchronous CREDIT authorization
// contract. It deliberately owns no ledger posting or price table.
package creditreservation

import (
	"context"
	"time"

	"github.com/openmeterio/openmeter/openmeter/currencies"
)

type ReservationState string

const (
	ReservationStateActive       ReservationState = "ACTIVE"
	ReservationStateExecuting    ReservationState = "EXECUTING"
	ReservationStateSettled      ReservationState = "SETTLED"
	ReservationStateReleased     ReservationState = "RELEASED"
	ReservationStateUnknown      ReservationState = "UNKNOWN"
	ReservationStateExpired      ReservationState = "EXPIRED"
	ReservationStateManualReview ReservationState = "MANUAL_REVIEW"
)

type ResourceLine struct {
	FeatureKey   string            `json:"featureKey"`
	ResourceCode string            `json:"resourceCode"`
	Quantity     int64             `json:"quantity"`
	Provider     string            `json:"provider,omitempty"`
	Model        string            `json:"model,omitempty"`
	Dimensions   map[string]string `json:"dimensions,omitempty"`
}

type RatedLine struct {
	ResourceLine
	RateCardKey string `json:"rateCardKey"`
	RateVersion string `json:"rateVersion"`
	Credits     int64  `json:"credits"`
}

// Reservation is the temporary authorization hold for a CREDIT-denominated
// resource call. Ledger posting is intentionally outside this package.
type Reservation struct {
	ID           string                       `json:"id"`
	Namespace    string                       `json:"namespace"`
	CustomerID   string                       `json:"customerId"`
	Currency     currencies.CurrencyReference `json:"currency"`
	State        ReservationState             `json:"state"`
	RateVersion  string                       `json:"rateVersion"`
	Lines        []RatedLine                  `json:"lines"`
	TotalCredits int64                        `json:"totalCredits"`
	ExpiresAt    *time.Time                   `json:"expiresAt,omitempty"`
}

// Charge records a settlement instruction derived from a reservation. It does
// not represent a booked ledger entry.
type Charge struct {
	ReservationID string                       `json:"reservationId"`
	Currency      currencies.CurrencyReference `json:"currency"`
	RateVersion   string                       `json:"rateVersion"`
	Lines         []RatedLine                  `json:"lines"`
	TotalCredits  int64                        `json:"totalCredits"`
}

type PriceResolver interface {
	Resolve(ctx context.Context, input ResolvePriceInput) (ResolvedPrice, error)
}

type ResolvePriceInput struct {
	Namespace  string         `json:"namespace"`
	CustomerID string         `json:"customerId"`
	At         time.Time      `json:"at"`
	Lines      []ResourceLine `json:"lines"`
}

type ResolvedPrice struct {
	Currency     currencies.CurrencyReference `json:"currency"`
	RateVersion  string                       `json:"rateVersion"`
	Lines        []RatedLine                  `json:"lines"`
	TotalCredits int64                        `json:"totalCredits"`
}
