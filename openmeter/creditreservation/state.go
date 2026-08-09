package creditreservation

import "fmt"

// ValidateTransition permits only explicit lifecycle edges. Terminal states
// intentionally have no outgoing transitions. UNKNOWN can move only when an
// evidence-bearing settle/release command or a manual review action calls it.
func ValidateTransition(from, to ReservationState) error {
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

	return nil
}
