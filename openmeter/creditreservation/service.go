package creditreservation

import (
	"context"
	"time"

	"github.com/openmeterio/openmeter/pkg/models"
)

// Evidence proves that an external provider did not execute a request. It is
// deliberately separate from TransitionEvidence: it documents provider-side
// effects, while TransitionEvidence guards recovery from UNKNOWN.
type Evidence struct {
	Kind      EvidenceKind `json:"kind"`
	Reference string       `json:"reference,omitempty"`
}

type EvidenceKind string

const (
	EvidenceNotSent                      EvidenceKind = "not_sent"
	EvidenceProviderConfirmedNotExecuted EvidenceKind = "provider_confirmed_not_executed"
)

// ReserveInput captures the immutable call identity and pricing request for
// an authorization. AuthorizationExpiresAt is an absolute deadline so callers
// cannot accidentally extend an already issued authorization on retries.
type ReserveInput struct {
	ID                     models.NamespacedID
	CustomerID             string
	SubjectID              string
	ClientCallID           string
	Operation              string
	CommandIdentity        CommandIdentity
	Lines                  []ResourceLine
	AuthorizationExpiresAt time.Time
	Provider               string
	Model                  string
	RequestID              string
}

type ExecuteInput struct {
	ID                models.NamespacedID
	IdempotencyKey    string
	PayloadHash       string
	ExecutionDeadline time.Time
}

type ReleaseInput struct {
	ID             models.NamespacedID
	IdempotencyKey string
	PayloadHash    string
	Evidence       Evidence
}

type UnknownInput struct {
	ID             models.NamespacedID
	IdempotencyKey string
	PayloadHash    string
}

// SettleInput supplies observed resource usage for an executed reservation.
// OpenMeter rates the lines against the reservation's persisted rate snapshot;
// callers never submit an authoritative credit amount.
type SettleInput struct {
	ID              models.NamespacedID
	CommandIdentity CommandIdentity
	ActualLines     []ResourceLine
	SettledAt       time.Time
}

// ChargeInput is the no-reservation path. It still uses the same command
// identity, customer lock and collector boundary as Settle.
type ChargeInput struct {
	ID              models.NamespacedID
	CustomerID      string
	SubjectID       string
	Operation       string
	CommandIdentity CommandIdentity
	Lines           []ResourceLine
	BookedAt        time.Time
}

// ReverseChargeInput requests exactly one correction of a previously settled
// direct charge. The service must use the stored collector realization
// provenance; it must not infer a reversal from the current customer balance.
type ReverseChargeInput struct {
	ID              models.NamespacedID
	CommandIdentity CommandIdentity
	ReversedAt      time.Time
}

type SweepResult struct {
	Expired      int
	Unknown      int
	ManualReview int
}

type FenceResult struct {
	Sequence    string
	Established bool
}

// Service owns only the synchronous authorization lifecycle. Settlement and
// ledger posting are deliberately added by the settlement service, keeping a
// temporary hold from becoming a balance projection.
type Service interface {
	Reserve(context.Context, ReserveInput) (Reservation, error)
	Execute(context.Context, ExecuteInput) (Reservation, error)
	Get(context.Context, models.NamespacedID) (Reservation, error)
	Release(context.Context, ReleaseInput) (Reservation, error)
	MarkUnknown(context.Context, UnknownInput) (Reservation, error)
	SweepExpired(context.Context, time.Time, int) (SweepResult, error)
	Settle(context.Context, SettleInput) (Reservation, error)
	Charge(context.Context, ChargeInput) (Charge, error)
	ReverseCharge(context.Context, ReverseChargeInput) (Charge, error)
}
