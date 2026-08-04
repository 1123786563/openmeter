package signing_test

import (
	"testing"
	"time"

	"github.com/openmeterio/openmeter/openmeter/aiusage/signing"
)

// testNow is the fixed instant all test signers use as their clock. The
// fixture package in service_test.go sets ExpiresAt to 12:05 UTC, five minutes
// after this point, so packages are always freshly valid (never expired)
// regardless of the real wall clock.
var testNow = time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)

// fixedClock returns a Now func pinned to testNow for deterministic expiry.
func fixedClock() func() time.Time {
	return func() time.Time { return testNow }
}

// newTestSigner builds a deterministic signer for tests using a fixed key pair
// derived from the key_id. This avoids randomness in test assertions while
// still using real Ed25519.
func newTestSigner(t *testing.T, keyID string) signing.Signer {
	t.Helper()
	kp, err := deterministicKeyPair(keyID)
	if err != nil {
		t.Fatalf("derive key pair: %v", err)
	}
	s, err := signing.New(signing.Config{
		CurrentKey: kp,
		Now:        fixedClock(),
	})
	if err != nil {
		t.Fatalf("new signer: %v", err)
	}
	return s
}

// newTestSignerWithRotation builds a signer with a previous key for rotation tests.
func newTestSignerWithRotation(t *testing.T, currentID, previousID string) signing.Signer {
	t.Helper()
	curr, err := deterministicKeyPair(currentID)
	if err != nil {
		t.Fatalf("derive current key: %v", err)
	}
	prev, err := deterministicKeyPair(previousID)
	if err != nil {
		t.Fatalf("derive previous key: %v", err)
	}
	s, err := signing.New(signing.Config{
		CurrentKey:  curr,
		PreviousKey: &prev,
		Now:         fixedClock(),
	})
	if err != nil {
		t.Fatalf("new signer: %v", err)
	}
	return s
}
