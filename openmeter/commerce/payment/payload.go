package payment

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// ExtractSignedPayload parses a raw callback body (JSON) into a map of verified
// fields. This is the durable audit record stored in PaymentFact.SignedPayload.
// The raw body itself is never persisted — only its SHA-256 hash (RawHash).
func ExtractSignedPayload(body []byte) (map[string]any, error) {
	var payload map[string]any
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()
	if err := dec.Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode callback body: %w", err)
	}
	return payload, nil
}
