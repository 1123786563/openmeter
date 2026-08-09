package common

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/creditlimit"
)

func TestActiveHoldReaderForCreditsPhaseGate(t *testing.T) {
	reader, err := activeHoldReaderForCredits(config.CreditsConfiguration{})
	require.NoError(t, err)
	require.IsType(t, creditlimit.NoActiveHoldReader{}, reader)

	reader, err = activeHoldReaderForCredits(config.CreditsConfiguration{ReservationsEnabled: true})
	require.Nil(t, reader)
	require.ErrorContains(t, err, "durable active hold reader")
}
