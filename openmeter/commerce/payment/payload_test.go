package payment

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestExtractSignedPayload verifies that the JSON callback body is decoded
// into a map with json.Number (no float corruption) so integer amounts are
// preserved exactly.
func TestExtractSignedPayload(t *testing.T) {
	body := []byte(`{"amount":100,"currency":"CNY","merchant_id":"m1"}`)

	payload, err := ExtractSignedPayload(body)
	require.NoError(t, err)
	assert.Equal(t, "CNY", payload["currency"])
	assert.Equal(t, "m1", payload["merchant_id"])

	// json.Number preserves the original integer representation.
	num, ok := payload["amount"].(json.Number)
	require.True(t, ok, "amount must be json.Number, not float64")
	assert.Equal(t, "100", num.String())
}

// TestExtractSignedPayloadFloatNotCorrupted verifies that a decimal value is
// preserved as a json.Number string rather than a lossy float64.
func TestExtractSignedPayloadFloatNotCorrupted(t *testing.T) {
	body := []byte(`{"amount":0.001}`)

	payload, err := ExtractSignedPayload(body)
	require.NoError(t, err)

	num, ok := payload["amount"].(json.Number)
	require.True(t, ok)
	assert.Equal(t, "0.001", num.String())
}

// TestExtractSignedPayloadInvalidJSON returns an error for malformed input.
func TestExtractSignedPayloadInvalidJSON(t *testing.T) {
	_, err := ExtractSignedPayload([]byte(`{invalid`))
	require.Error(t, err)
}

// TestExtractSignedPayloadEmptyArray returns an error for non-object JSON.
func TestExtractSignedPayloadNonObject(t *testing.T) {
	_, err := ExtractSignedPayload([]byte(`[1,2,3]`))
	// [1,2,3] decodes into a map successfully only if it's actually an object.
	// json.Decode into map[string]any will fail for an array.
	require.Error(t, err)
}

// TestExtractSignedPayloadEmptyBody returns an error for empty input.
func TestExtractSignedPayloadEmptyBody(t *testing.T) {
	_, err := ExtractSignedPayload([]byte(``))
	require.Error(t, err)
}

// TestExtractSignedPayloadNested verifies nested objects are preserved.
func TestExtractSignedPayloadNested(t *testing.T) {
	body := []byte(`{"payer":{"id":"p1"},"amount":5000}`)

	payload, err := ExtractSignedPayload(body)
	require.NoError(t, err)

	payer, ok := payload["payer"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "p1", payer["id"])
}
