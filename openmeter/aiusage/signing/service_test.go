package signing_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/aiusage/signing"
)

// validAuthorizationPackage returns a structurally valid but unsigned package.
func validAuthorizationPackage() signing.AuthorizationPackage {
	return signing.AuthorizationPackage{
		BillingCustomerID:            "cust-001",
		SubjectKeys:                  []string{"subj-b", "subj-a"},
		PlanCode:                     "enterprise",
		SubscriptionCode:             "sub-2026-001",
		SubscriptionStatus:           "active",
		EntitlementCodes:             []string{"ent-x", "ent-y"},
		SpendableCredits:             5000,
		EnterpriseAvailableCredits:   10000,
		AuthorizationCapacityCredits: 15000,
		CurrentPeriodStart:           time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
		CurrentPeriodEnd:             time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
		SnapshotVersion:              42,
		CoveredTenantSeq:             99,
		RatePackageVersion:           "rp-v3",
		RatePackage: []signing.SignedRateEntry{
			{ResourceCode: "llm.input_tokens", Provider: "openai", Model: "gpt-4", CreditsPerUnit: 7, UnitSize: 1000},
			{ResourceCode: "llm.output_tokens", Provider: "anthropic", Model: "claude-3", CreditsPerUnit: 21, UnitSize: 1000},
		},
		ExpiresAt: time.Date(2026, 8, 4, 12, 5, 0, 0, time.UTC),
	}
}

func TestAuthorizationSignatureCoversEveryField(t *testing.T) {
	signer := newTestSigner(t, "key-2026-08")

	pkg := validAuthorizationPackage()
	signed, err := signer.Sign(pkg)
	require.NoError(t, err)
	require.NotEmpty(t, signed.Signature)
	require.Equal(t, "key-2026-08", signed.KeyID)

	// Tampering with any signed field must break verification.
	require.NoError(t, signer.Verify(signed))
	signed.CoveredTenantSeq++
	require.ErrorIs(t, signer.Verify(signed), signing.ErrInvalidSignature)
}

func TestSignatureBrokenOnFieldTamper(t *testing.T) {
	signer := newTestSigner(t, "key-tamper")
	base := validAuthorizationPackage()

	signed, err := signer.Sign(base)
	require.NoError(t, err)

	// Each mutation must invalidate the signature.
	mutations := []func(*signing.AuthorizationPackage){
		func(p *signing.AuthorizationPackage) { p.SpendableCredits++ },
		func(p *signing.AuthorizationPackage) { p.AuthorizationCapacityCredits-- },
		func(p *signing.AuthorizationPackage) { p.SnapshotVersion++ },
		func(p *signing.AuthorizationPackage) { p.BillingCustomerID = "other" },
		func(p *signing.AuthorizationPackage) { p.SubjectKeys = []string{"changed"} },
		func(p *signing.AuthorizationPackage) { p.PlanCode = "starter" },
		func(p *signing.AuthorizationPackage) { p.RatePackageVersion = "rp-v4" },
		func(p *signing.AuthorizationPackage) { p.ExpiresAt = p.ExpiresAt.Add(time.Second) },
	}

	for i, mut := range mutations {
		tampered := signed
		mut(&tampered)
		require.ErrorIsf(t, signer.Verify(tampered), signing.ErrInvalidSignature,
			"mutation %d did not invalidate signature", i)
	}
}

func TestCanonicalDeterminism(t *testing.T) {
	a := validAuthorizationPackage()
	b := validAuthorizationPackage()

	// Reorder slices — canonical bytes must be identical after sorting.
	b.SubjectKeys = []string{"subj-a", "subj-b"}
	b.EntitlementCodes = []string{"ent-y", "ent-x"}
	b.RatePackage = []signing.SignedRateEntry{
		{ResourceCode: "llm.output_tokens", Provider: "anthropic", Model: "claude-3", CreditsPerUnit: 21, UnitSize: 1000},
		{ResourceCode: "llm.input_tokens", Provider: "openai", Model: "gpt-4", CreditsPerUnit: 7, UnitSize: 1000},
	}

	bytesA, err := signing.CanonicalBytes(a)
	require.NoError(t, err)
	bytesB, err := signing.CanonicalBytes(b)
	require.NoError(t, err)
	require.Equal(t, bytesA, bytesB, "canonical bytes must be identical regardless of slice order")
}

func TestCanonicalExcludesSignature(t *testing.T) {
	pkg := validAuthorizationPackage()
	pkg.Signature = "deadbeef"

	bytes, err := signing.CanonicalBytes(pkg)
	require.NoError(t, err)
	require.NotContains(t, string(bytes), "signature")
}

func TestKeyRotationOverlap(t *testing.T) {
	// Signer with rotation: current = "key-v2", previous = "key-v1".
	signerV2 := newTestSignerWithRotation(t, "key-v2", "key-v1")
	signerV1 := newTestSigner(t, "key-v1")

	// A package signed with the previous key must verify under the rotated signer.
	pkg := validAuthorizationPackage()
	signedByV1, err := signerV1.Sign(pkg)
	require.NoError(t, err)
	require.NoError(t, signerV2.Verify(signedByV1), "previous key must verify during overlap")

	// A package signed with the current key must also verify.
	signedByV2, err := signerV2.Sign(pkg)
	require.NoError(t, err)
	require.NoError(t, signerV2.Verify(signedByV2), "current key must verify")
}

func TestDefaultTTL(t *testing.T) {
	kp, err := deterministicKeyPair("ttl-test")
	require.NoError(t, err)

	s, err := signing.New(signing.Config{CurrentKey: kp})
	require.NoError(t, err)
	require.Equal(t, 5*time.Minute, s.TTL(), "default TTL must be 5 minutes")
}

func TestTTLExceedsMaxFails(t *testing.T) {
	kp, err := deterministicKeyPair("max-ttl")
	require.NoError(t, err)

	_, err = signing.New(signing.Config{
		CurrentKey: kp,
		TTL:        16 * time.Minute,
	})
	require.Error(t, err, "TTL over 15 minutes must fail validation")

	// Exactly 15 minutes is allowed.
	_, err = signing.New(signing.Config{
		CurrentKey: kp,
		TTL:        15 * time.Minute,
	})
	require.NoError(t, err)
}

func TestGenerateKeyPair(t *testing.T) {
	kp, err := signing.GenerateKeyPair("gen-key")
	require.NoError(t, err)
	require.Equal(t, "gen-key", kp.KeyID)
	require.Len(t, kp.Seed, 32)

	s, err := signing.New(signing.Config{CurrentKey: kp})
	require.NoError(t, err)

	pkg := validAuthorizationPackage()
	signed, err := s.Sign(pkg)
	require.NoError(t, err)
	require.NoError(t, s.Verify(signed))
}

func TestVerifyRejectsExpiredPackage(t *testing.T) {
	signer := newTestSigner(t, "key-expiry")

	pkg := validAuthorizationPackage()
	// Set ExpiresAt to the past.
	pkg.ExpiresAt = time.Now().Add(-1 * time.Minute)
	signed, err := signer.Sign(pkg)
	require.NoError(t, err)

	require.ErrorIs(t, signer.Verify(signed), signing.ErrPackageExpired)
}

func TestNegativeTTLFails(t *testing.T) {
	kp, err := deterministicKeyPair("neg-ttl")
	require.NoError(t, err)

	_, err = signing.New(signing.Config{
		CurrentKey: kp,
		TTL:        -1 * time.Minute,
	})
	require.Error(t, err, "negative TTL must fail validation")
}

func TestVerifyUnknownKeyIDReturnsNoMatchingKey(t *testing.T) {
	signer := newTestSigner(t, "key-known")

	pkg := validAuthorizationPackage()
	signed, err := signer.Sign(pkg)
	require.NoError(t, err)

	// Tamper the key_id to one the signer does not hold.
	signed.KeyID = "key-nonexistent"
	require.ErrorIs(t, signer.Verify(signed), signing.ErrNoMatchingKey)
}
