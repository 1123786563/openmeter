package common

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/creditlimit"
)

func TestActiveHoldReaderForCredits(t *testing.T) {
	reader, err := activeHoldReaderForCredits(nil, config.CreditsConfiguration{})
	require.NoError(t, err)
	require.IsType(t, creditlimit.NoActiveHoldReader{}, reader)

	// With reservations enabled, holds are served by the durable reader over
	// the credit_reservation table; the earlier fail-closed phase gate was
	// removed when that reader landed.
	reader, err = activeHoldReaderForCredits(nil, config.CreditsConfiguration{ReservationsEnabled: true})
	require.NoError(t, err)
	_, isNoop := reader.(creditlimit.NoActiveHoldReader)
	require.False(t, isNoop, "expected the durable reservation hold reader, got the no-op reader")
}
