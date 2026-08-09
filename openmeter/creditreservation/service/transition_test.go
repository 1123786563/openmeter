package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/creditreservation"
)

func TestReleaseEvidenceForActiveDoesNotRequireProviderConfirmation(t *testing.T) {
	err := validateReleaseEvidence(creditreservation.ReservationStateActive, creditreservation.Evidence{
		Kind: creditreservation.EvidenceNotSent,
	})
	require.NoError(t, err)
}

func TestReleaseEvidenceForExecutingRequiresProviderConfirmation(t *testing.T) {
	err := validateReleaseEvidence(creditreservation.ReservationStateExecuting, creditreservation.Evidence{
		Kind: creditreservation.EvidenceNotSent,
	})
	require.ErrorIs(t, err, creditreservation.ErrTransitionEvidenceRequired)

	err = validateReleaseEvidence(creditreservation.ReservationStateExecuting, creditreservation.Evidence{
		Kind:      creditreservation.EvidenceProviderConfirmedNotExecuted,
		Reference: "provider-request-1",
	})
	require.NoError(t, err)
}

func TestSweepTransitionNeverReleasesUnknown(t *testing.T) {
	now := time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC)
	require.Equal(t, creditreservation.ReservationStateExpired, sweepTransition(creditreservation.ReservationStateActive, now.Add(-time.Second), now))
	require.Equal(t, creditreservation.ReservationStateUnknown, sweepTransition(creditreservation.ReservationStateExecuting, now.Add(-time.Second), now))
	require.Empty(t, sweepTransition(creditreservation.ReservationStateUnknown, now.Add(-time.Second), now))
	require.Empty(t, sweepTransition(creditreservation.ReservationStateActive, now.Add(time.Second), now))
}
