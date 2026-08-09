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
	require.ErrorIs(t,
		creditreservation.ValidateTransition(creditreservation.ReservationStateUnknown, creditreservation.ReservationStateSettled),
		creditreservation.ErrTransitionEvidenceRequired,
	)
}

func TestUnknownTransitionRequiresMatchingEvidence(t *testing.T) {
	require.ErrorIs(t,
		creditreservation.ValidateTransitionWithEvidence(
			creditreservation.ReservationStateUnknown,
			creditreservation.ReservationStateSettled,
			creditreservation.TransitionEvidence{},
		),
		creditreservation.ErrTransitionEvidenceRequired,
	)
	require.ErrorIs(t,
		creditreservation.ValidateTransitionWithEvidence(
			creditreservation.ReservationStateUnknown,
			creditreservation.ReservationStateReleased,
			creditreservation.TransitionEvidence{Kind: creditreservation.TransitionEvidenceRelease, Reference: " "},
		),
		creditreservation.ErrTransitionEvidenceRequired,
	)

	require.NoError(t,
		creditreservation.ValidateTransitionWithEvidence(
			creditreservation.ReservationStateUnknown,
			creditreservation.ReservationStateSettled,
			creditreservation.TransitionEvidence{Kind: creditreservation.TransitionEvidenceSettlement, Reference: "ledger-entry-1"},
		),
	)
	require.NoError(t,
		creditreservation.ValidateTransitionWithEvidence(
			creditreservation.ReservationStateUnknown,
			creditreservation.ReservationStateReleased,
			creditreservation.TransitionEvidence{Kind: creditreservation.TransitionEvidenceRelease, Reference: "release-confirmation-1"},
		),
	)
}
