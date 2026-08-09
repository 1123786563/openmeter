// Package creditreservation defines the synchronous CREDIT authorization
// contract. It deliberately owns no ledger posting or price table.
package creditreservation

import (
	"context"
	"encoding/hex"
	"strings"
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

// CommandIdentity binds an idempotency key to the exact command payload. A
// caller must reuse both values on retries; a changed payload requires a new
// key, preventing a retry from becoming a second ledger instruction.
type CommandIdentity struct {
	IdempotencyKey string `json:"idempotencyKey"`
	PayloadHash    string `json:"payloadHash"`
}

func (c CommandIdentity) Validate() error {
	if strings.TrimSpace(c.IdempotencyKey) == "" || len(c.PayloadHash) != 64 {
		return ErrInvalidCommandIdentity
	}
	if _, err := hex.DecodeString(c.PayloadHash); err != nil {
		return ErrInvalidCommandIdentity
	}
	return nil
}

// Reservation is the temporary authorization hold for a CREDIT-denominated
// resource call. Ledger posting is intentionally outside this package.
type Reservation struct {
	ID                      string                       `json:"id"`
	Namespace               string                       `json:"namespace"`
	CustomerID              string                       `json:"customerId"`
	Currency                currencies.CurrencyReference `json:"currency"`
	State                   ReservationState             `json:"state"`
	RateVersion             string                       `json:"rateVersion"`
	Lines                   []RatedLine                  `json:"lines"`
	TotalCredits            int64                        `json:"totalCredits"`
	ExpiresAt               *time.Time                   `json:"expiresAt,omitempty"`
	ExecutionDeadline       *time.Time                   `json:"executionDeadline,omitempty"`
	CommandIdentity         CommandIdentity              `json:"commandIdentity"`
	SettledCredits          int64                        `json:"settledCredits"`
	PrepaidHold             int64                        `json:"prepaidHold"`
	EnterpriseHold          int64                        `json:"enterpriseHold"`
	SettlementLedgerGroupID string                       `json:"settlementLedgerGroupId,omitempty"`
	SettlementIdentity      CommandIdentity              `json:"settlementIdentity,omitempty"`
}

func (r Reservation) Validate() error {
	return r.CommandIdentity.Validate()
}

// Charge records a settlement instruction derived from a reservation. It does
// not represent a booked ledger entry.
type Charge struct {
	ID                      string                       `json:"id"`
	Namespace               string                       `json:"namespace"`
	CustomerID              string                       `json:"customerId"`
	ReservationID           string                       `json:"reservationId"`
	Currency                currencies.CurrencyReference `json:"currency"`
	RateVersion             string                       `json:"rateVersion"`
	Lines                   []RatedLine                  `json:"lines"`
	TotalCredits            int64                        `json:"totalCredits"`
	CommandIdentity         CommandIdentity              `json:"commandIdentity"`
	State                   string                       `json:"state"`
	SettlementLedgerGroupID string                       `json:"settlementLedgerGroupId,omitempty"`
	ReversalLedgerGroupID   string                       `json:"reversalLedgerGroupId,omitempty"`
	SettlementAllocations   []SettlementAllocation       `json:"settlementAllocations,omitempty"`
}

type SettlementAllocation struct {
	ID       string `json:"id"`
	GroupID  string `json:"groupId"`
	Amount   int64  `json:"amount"`
	SortHint int    `json:"sortHint"`
}

// UsageEvent is the standard event payload persisted by the reservation
// transactional outbox. It contains no balance or ledger projection data.
type UsageEvent struct {
	EventID       string         `json:"eventId"`
	AggregateType string         `json:"aggregateType"`
	AggregateID   string         `json:"aggregateId"`
	EventType     string         `json:"eventType"`
	Payload       map[string]any `json:"payload"`
}

func (c Charge) Validate() error {
	return c.CommandIdentity.Validate()
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
