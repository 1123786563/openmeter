package payment

import (
	"encoding/json"
	"fmt"
)

// ExtractSignedPayload parses a raw callback body (JSON) into a map of verified
// fields. This is the durable audit record stored in PaymentFact.SignedPayload.
// The raw body itself is never persisted — only its SHA-256 hash (RawHash).
func ExtractSignedPayload(body []byte) (map[string]any, error) {
	var payload map[string]any
	dec := json.NewDecoder(newByteReader(body))
	dec.UseNumber()
	if err := dec.Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode callback body: %w", err)
	}
	return payload, nil
}

// byteReader wraps a []byte to implement io.Reader for json.NewDecoder.
type byteReader struct {
	data []byte
	off  int
}

func newByteReader(b []byte) *byteReader { return &byteReader{data: b} }

func (r *byteReader) Read(p []byte) (int, error) {
	if r.off >= len(r.data) {
		return 0, errEOF
	}
	n := copy(p, r.data[r.off:])
	r.off += n
	return n, nil
}

var errEOF = fmt.Errorf("EOF")
