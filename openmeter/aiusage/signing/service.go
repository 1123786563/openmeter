package signing

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"
)

// DefaultTTL is the default authorization package validity duration.
const DefaultTTL = 5 * time.Minute

// MaxTTL is the maximum allowed TTL. Values above this fail config validation.
const MaxTTL = 15 * time.Minute

// KeyPair holds an Ed25519 key pair identified by a stable key_id.
type KeyPair struct {
	KeyID string
	Seed  []byte // 32-byte Ed25519 seed; the private key is derived from it.
}

// Config configures the Ed25519 signer with key rotation support.
type Config struct {
	// CurrentKey is the active signing key.
	CurrentKey KeyPair

	// PreviousKey is the key being rotated out. During the configured overlap
	// window both keys verify signatures, allowing consumers to refresh their
	// cached public keys without a hard cutover.
	PreviousKey *KeyPair

	// TTL is how long a signed authorization package remains valid. Defaults to
	// 5 minutes. Must not exceed 15 minutes.
	TTL time.Duration
}

func (c Config) validate() error {
	if c.CurrentKey.KeyID == "" {
		return errors.New("signing: current key_id must not be empty")
	}
	if len(c.CurrentKey.Seed) != ed25519.SeedSize {
		return fmt.Errorf("signing: current key seed must be %d bytes", ed25519.SeedSize)
	}
	if c.PreviousKey != nil {
		if c.PreviousKey.KeyID == "" {
			return errors.New("signing: previous key_id must not be empty")
		}
		if len(c.PreviousKey.Seed) != ed25519.SeedSize {
			return fmt.Errorf("signing: previous key seed must be %d bytes", ed25519.SeedSize)
		}
		if c.PreviousKey.KeyID == c.CurrentKey.KeyID {
			return errors.New("signing: previous and current key_id must differ")
		}
	}

	ttl := c.TTL
	if ttl == 0 {
		ttl = DefaultTTL
	}
	if ttl > MaxTTL {
		return fmt.Errorf("signing: TTL %s exceeds maximum %s", c.TTL, MaxTTL)
	}

	return nil
}

// resolvedTTL returns the effective TTL, applying the default when zero.
func (c Config) resolvedTTL() time.Duration {
	if c.TTL == 0 {
		return DefaultTTL
	}
	return c.TTL
}

// Signer signs authorization packages with Ed25519 and verifies them against
// the current (and optionally previous) key.
type Signer interface {
	Sign(pkg AuthorizationPackage) (AuthorizationPackage, error)
	Verify(pkg AuthorizationPackage) error
	KeyID() string
	TTL() time.Duration
}

type signer struct {
	current  KeyPair
	previous *KeyPair
	ttl      time.Duration
	now      func() time.Time
}

// New creates a Signer from Config.
func New(cfg Config) (Signer, error) {
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return &signer{
		current:  cfg.CurrentKey,
		previous: cfg.PreviousKey,
		ttl:      cfg.resolvedTTL(),
		now:      time.Now,
	}, nil
}

// KeyID returns the current signing key_id.
func (s *signer) KeyID() string { return s.current.KeyID }

// TTL returns the package validity duration.
func (s *signer) TTL() time.Duration { return s.ttl }

// Sign computes the Ed25519 signature over the canonical bytes and returns the
// package with key_id and signature populated.
func (s *signer) Sign(pkg AuthorizationPackage) (AuthorizationPackage, error) {
	pkg.KeyID = s.current.KeyID

	canonical, err := CanonicalBytes(pkg)
	if err != nil {
		return AuthorizationPackage{}, err
	}

	priv := ed25519.NewKeyFromSeed(s.current.Seed)
	sig := ed25519.Sign(priv, canonical)

	pkg.Signature = hex.EncodeToString(sig)
	return pkg, nil
}

// Verify checks the Ed25519 signature. During a rotation overlap the previous
// key also verifies successfully.
func (s *signer) Verify(pkg AuthorizationPackage) error {
	sigBytes, err := hex.DecodeString(pkg.Signature)
	if err != nil {
		return fmt.Errorf("%w: decode signature: %v", ErrInvalidSignature, err)
	}
	if len(sigBytes) != ed25519.SignatureSize {
		return fmt.Errorf("%w: expected %d bytes, got %d", ErrInvalidSignature, ed25519.SignatureSize, len(sigBytes))
	}

	canonical, err := CanonicalBytes(pkg)
	if err != nil {
		return err
	}

	// Try the key matching the package's key_id first.
	for _, kp := range s.verifiableKeys() {
		if pkg.KeyID != "" && kp.KeyID != pkg.KeyID {
			continue
		}
		pub := ed25519.NewKeyFromSeed(kp.Seed).Public()
		if ed25519.Verify(pub.(ed25519.PublicKey), canonical, sigBytes) {
			return nil
		}
	}

	return ErrInvalidSignature
}

// verifiableKeys returns the keys that may verify, in priority order: current
// first, then previous.
func (s *signer) verifiableKeys() []KeyPair {
	keys := []KeyPair{s.current}
	if s.previous != nil {
		keys = append(keys, *s.previous)
	}
	return keys
}

// ---- helpers for tests and callers ----

// GenerateKeyPair creates a random Ed25519 key pair with the given key_id.
func GenerateKeyPair(keyID string) (KeyPair, error) {
	seed := make([]byte, ed25519.SeedSize)
	if _, err := rand.Read(seed); err != nil {
		return KeyPair{}, fmt.Errorf("signing: generate seed: %w", err)
	}
	return KeyPair{KeyID: keyID, Seed: seed}, nil
}
