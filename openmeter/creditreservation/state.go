package creditreservation

import (
	"fmt"
	"strings"
)

type TransitionEvidenceKind string

const (
	TransitionEvidenceSettlement TransitionEvidenceKind = "SETTLEMENT"
	TransitionEvidenceRelease    TransitionEvidenceKind = "RELEASE"
)

// TransitionEvidence is the durable external reference required before an
// UNKNOWN hold may settle or release. Manual review is explicitly not an
// external-effect transition and therefore does not require evidence.
type TransitionEvidence struct {
	Kind      TransitionEvidenceKind `json:"kind"`
	Reference string                 `json:"reference"`
}

// ValidateTransition permits only explicit lifecycle edges. Terminal states
// intentionally have no outgoing transitions. UNKNOWN can move only when an
// evidence-bearing settle/release command or a manual review action calls it.
func ValidateTransition(from, to ReservationState) error {
	return ValidateTransitionWithEvidence(from, to, TransitionEvidence{})
}

// ValidateTransitionWithEvidence permits explicit lifecycle edges and requires
// matching settlement/release evidence when recovering from UNKNOWN.
func ValidateTransitionWithEvidence(from, to ReservationState, evidence TransitionEvidence) error {
	allowed := false

	switch from {
	case ReservationStateActive:
		switch to {
		case ReservationStateExecuting, ReservationStateReleased, ReservationStateUnknown, ReservationStateExpired:
			allowed = true
		}
	case ReservationStateExecuting:
		switch to {
		case ReservationStateSettled, ReservationStateReleased, ReservationStateUnknown:
			allowed = true
		}
	case ReservationStateUnknown:
		switch to {
		case ReservationStateSettled, ReservationStateReleased, ReservationStateManualReview:
			allowed = true
		}
	}

	if !allowed {
		return fmt.Errorf("%w: %s -> %s", ErrStateConflict, from, to)
	}

	if from == ReservationStateUnknown {
		switch to {
		case ReservationStateSettled:
			if evidence.Kind != TransitionEvidenceSettlement || strings.TrimSpace(evidence.Reference) == "" {
				return fmt.Errorf("%w: settlement reference", ErrTransitionEvidenceRequired)
			}
		case ReservationStateReleased:
			if evidence.Kind != TransitionEvidenceRelease || strings.TrimSpace(evidence.Reference) == "" {
				return fmt.Errorf("%w: release reference", ErrTransitionEvidenceRequired)
			}
		}
	}

	return nil
}
