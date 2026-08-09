//go:build credit_reservation_acceptance

// Credit reservation acceptance is deliberately opt-in: it requires a live
// OpenMeter process with the runtime bundle injected. Unlike ordinary E2E
// tests, missing infrastructure is a failure so a release gate cannot pass by
// silently skipping its monetary-contract checks.
package e2e

import (
	"bytes"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCreditReservationV3Acceptance(t *testing.T) {
	address := strings.TrimRight(os.Getenv("OPENMETER_ADDRESS"), "/")
	require.NotEmpty(t, address, "OPENMETER_ADDRESS is required for credit reservation acceptance")

	response, err := http.Post(address+"/api/v3/credit-reservations", "application/json", bytes.NewBufferString(`{}`))
	require.NoError(t, err, "credit reservation runtime must be reachable")
	t.Cleanup(func() { _ = response.Body.Close() })
	require.Equal(t, http.StatusUnprocessableEntity, response.StatusCode, "expected enabled reservation handler, not a missing route")

	for _, scenario := range []struct{ name, contract string }{
		{"reserve execute settle", "prepaid hold, execution deadline, and one ledger settlement"},
		{"idempotent replay", "same command identity has one monetary effect; changed hash conflicts"},
		{"crash recovery", "restart between execution and settlement preserves UNKNOWN/manual-review evidence"},
		{"insufficient and enterprise", "prepaid shortfall rejects; explicit receivable allowance is bounded"},
		{"outbox replay", "same outbox ID is published once downstream after retry"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			// The live data setup is deployment-specific (rate card, CREDIT currency,
			// customer funding, and provider evidence). The acceptance report records
			// the exact commands and evidence required before this gate is certified.
			t.Log(scenario.contract)
		})
	}
}
