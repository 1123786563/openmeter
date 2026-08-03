package signing_test

import (
	"testing"

	"github.com/openmeterio/openmeter/openmeter/aiusage/signing"
)

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
	})
	if err != nil {
		t.Fatalf("new signer: %v", err)
	}
	return s
}
