package creditreservation_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/creditreservation"
)

// This catches accidental relaxation of a terminal reservation state.
func TestReservationStateTransitions(t *testing.T) {
	valid := map[creditreservation.ReservationState][]creditreservation.ReservationState{
		creditreservation.ReservationStateActive: {
			creditreservation.ReservationStateExecuting,
			creditreservation.ReservationStateReleased,
			creditreservation.ReservationStateUnknown,
			creditreservation.ReservationStateExpired,
		},
		creditreservation.ReservationStateExecuting: {
			creditreservation.ReservationStateSettled,
			creditreservation.ReservationStateReleased,
			creditreservation.ReservationStateUnknown,
		},
		creditreservation.ReservationStateUnknown: {
			creditreservation.ReservationStateSettled,
			creditreservation.ReservationStateReleased,
			creditreservation.ReservationStateManualReview,
		},
	}

	for from, targets := range valid {
		for _, to := range targets {
			require.NoError(t, creditreservation.ValidateTransition(from, to))
		}
	}

	require.ErrorIs(t,
		creditreservation.ValidateTransition(creditreservation.ReservationStateSettled, creditreservation.ReservationStateReleased),
		creditreservation.ErrStateConflict,
	)
}
