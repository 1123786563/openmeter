package runtime

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/app/config"
)

func TestNewFailsClosedWithoutRuntimeDependencies(t *testing.T) {
	_, err := New(Config{Configuration: config.CreditReservationConfiguration{
		Enabled:                  true,
		AuthorizationTTL:         "5m",
		ExecutionDeadline:        "10m",
		UnknownManualReviewAfter: "1h",
		Worker: config.CreditReservationWorkerConfiguration{
			PollInterval: "5s", LeaseDuration: "30s", BatchSize: 50, MaxClaimCount: 3,
		},
	}})

	require.ErrorContains(t, err, "credit reservation runtime requires")
}
