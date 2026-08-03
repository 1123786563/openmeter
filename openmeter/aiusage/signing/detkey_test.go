package signing_test

import (
	"crypto/sha256"

	"github.com/openmeterio/openmeter/openmeter/aiusage/signing"
)

// deterministicKeyPair derives an Ed25519 seed from the SHA-256 of keyID. This
// gives a stable, reproducible key for tests without importing crypto/rand in
// assertions.
func deterministicKeyPair(keyID string) (signing.KeyPair, error) {
	h := sha256.Sum256([]byte(keyID))
	seed := h[:32]
	return signing.KeyPair{KeyID: keyID, Seed: seed}, nil
}
