//go:build credit_reservation_acceptance

package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type creditReservationFixture struct{ address, customer, subject, feature, resource, provider, model string }
type reservationEnvelope struct {
	ID    string `json:"id"`
	State string `json:"state"`
}

func TestCreditReservationV3Acceptance(t *testing.T) {
	f := creditReservationFixture{address: strings.TrimRight(os.Getenv("OPENMETER_ADDRESS"), "/"), customer: os.Getenv("OPENMETER_CR_CUSTOMER_ID"), subject: os.Getenv("OPENMETER_CR_SUBJECT_ID"), feature: os.Getenv("OPENMETER_CR_FEATURE_KEY"), resource: os.Getenv("OPENMETER_CR_RESOURCE_CODE"), provider: os.Getenv("OPENMETER_CR_PROVIDER"), model: os.Getenv("OPENMETER_CR_MODEL")}
	require.NotEmpty(t, f.address, "OPENMETER_ADDRESS is required")
	require.NotEmpty(t, f.customer, "OPENMETER_CR_CUSTOMER_ID is required")
	require.NotEmpty(t, f.subject, "OPENMETER_CR_SUBJECT_ID is required")
	require.NotEmpty(t, f.feature, "OPENMETER_CR_FEATURE_KEY is required")
	require.NotEmpty(t, f.resource, "OPENMETER_CR_RESOURCE_CODE is required")

	t.Run("reserve execute settle get and replay", func(t *testing.T) {
		id, key := acceptanceID("reserve"), acceptanceID("key")
		reserve := f.reserve(id, key, strings.Repeat("a", 64))
		created := postReservation(t, f.address+"/api/v3/credit-reservations", reserve, http.StatusCreated)
		require.Equal(t, id, created.ID)
		postReservation(t, f.address+"/api/v3/credit-reservations/"+id+"/execute", map[string]any{"idempotency_key": key, "payload_hash": strings.Repeat("a", 64), "execution_deadline": time.Now().UTC().Add(5 * time.Minute)}, http.StatusOK)
		settled := postReservation(t, f.address+"/api/v3/credit-reservations/"+id+"/settle", map[string]any{"idempotency_key": acceptanceID("settle"), "payload_hash": strings.Repeat("b", 64), "actual_credits": 1, "settled_at": time.Now().UTC()}, http.StatusOK)
		require.Equal(t, "settled", strings.ToLower(settled.State))
		getReservation(t, f.address+"/api/v3/credit-reservations/"+id, http.StatusOK)
		postReservation(t, f.address+"/api/v3/credit-reservations", reserve, http.StatusCreated)
		postReservation(t, f.address+"/api/v3/credit-reservations", f.reserve(id, key, strings.Repeat("c", 64)), http.StatusConflict)
	})

	t.Run("unknown release has provider evidence", func(t *testing.T) {
		id, key := acceptanceID("unknown"), acceptanceID("key")
		postReservation(t, f.address+"/api/v3/credit-reservations", f.reserve(id, key, strings.Repeat("d", 64)), http.StatusCreated)
		unknown := postReservation(t, f.address+"/api/v3/credit-reservations/"+id+"/unknown", map[string]any{"idempotency_key": key, "payload_hash": strings.Repeat("d", 64)}, http.StatusOK)
		require.Equal(t, "unknown", strings.ToLower(unknown.State))
		released := postReservation(t, f.address+"/api/v3/credit-reservations/"+id+"/release", map[string]any{"idempotency_key": key, "payload_hash": strings.Repeat("d", 64), "evidence_kind": "provider_confirmed_not_executed", "evidence_reference": acceptanceID("provider")}, http.StatusOK)
		require.Equal(t, "released", strings.ToLower(released.State))
	})

	for _, env := range []string{"OPENMETER_CR_CRASH_EVIDENCE_URL", "OPENMETER_CR_OUTBOX_EVIDENCE_URL"} {
		env := env
		t.Run(env, func(t *testing.T) {
			url := os.Getenv(env)
			require.NotEmpty(t, url, "%s is required; acceptance must include deployment evidence", env)
			response, err := http.Get(url)
			require.NoError(t, err)
			t.Cleanup(func() { _ = response.Body.Close() })
			require.Equal(t, http.StatusOK, response.StatusCode, "%s must expose accepted crash/outbox evidence", env)
		})
	}
}

func (f creditReservationFixture) reserve(id, key, hash string) map[string]any {
	line := map[string]any{"feature_key": f.feature, "resource_code": f.resource, "quantity": 1}
	if f.provider != "" {
		line["provider"] = f.provider
	}
	if f.model != "" {
		line["model"] = f.model
	}
	return map[string]any{"id": id, "customer_id": f.customer, "subject_id": f.subject, "client_call_id": id, "operation": "credit_reservation_acceptance", "idempotency_key": key, "payload_hash": hash, "authorization_expires_at": time.Now().UTC().Add(5 * time.Minute), "lines": []any{line}}
}

func postReservation(t *testing.T, url string, body any, want int) reservationEnvelope {
	t.Helper()
	b, err := json.Marshal(body)
	require.NoError(t, err)
	r, err := http.Post(url, "application/json", bytes.NewReader(b))
	require.NoError(t, err)
	t.Cleanup(func() { _ = r.Body.Close() })
	require.Equal(t, want, r.StatusCode, "POST %s", url)
	var out reservationEnvelope
	if want < 300 {
		require.NoError(t, json.NewDecoder(r.Body).Decode(&out))
	}
	return out
}
func getReservation(t *testing.T, url string, want int) reservationEnvelope {
	t.Helper()
	r, err := http.Get(url)
	require.NoError(t, err)
	t.Cleanup(func() { _ = r.Body.Close() })
	require.Equal(t, want, r.StatusCode)
	var out reservationEnvelope
	require.NoError(t, json.NewDecoder(r.Body).Decode(&out))
	return out
}
func acceptanceID(prefix string) string {
	return fmt.Sprintf("cr_acceptance_%s_%d", prefix, time.Now().UnixNano())
}
