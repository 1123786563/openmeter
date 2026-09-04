package auth

import (
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
)

// mintSessionToken hand-builds a session-shaped JWT so tests can vary a single
// property (algorithm, issuer, expiry) while keeping everything else valid.
func mintSessionToken(t *testing.T, secret string, signingMethod jwt.SigningMethod, claims jwt.MapClaims, key any) string {
	t.Helper()

	token := jwt.NewWithClaims(signingMethod, claims)
	tokenString, err := token.SignedString(key)
	require.NoError(t, err)

	return tokenString
}

func TestSessionTokenIssuerVerifyRejects(t *testing.T) {
	secret := randomSecret(t)

	issuer, err := NewSessionTokenIssuer(secret, time.Hour)
	require.NoError(t, err)

	now := time.Now()

	t.Run("expired token", func(t *testing.T) {
		expired, err := NewSessionTokenIssuer(secret, -time.Minute)
		require.NoError(t, err)

		token, err := expired.Issue(IssueSessionTokenInput{UserID: "user-123", Organization: "built-in"})
		require.NoError(t, err)

		_, err = issuer.Verify(token)
		require.Error(t, err)
	})

	t.Run("wrong issuer", func(t *testing.T) {
		token := mintSessionToken(t, secret, jwt.SigningMethodHS256, jwt.MapClaims{
			"iss":          "not-openmeter",
			"sub":          "user-123",
			"iat":          now.Unix(),
			"exp":          now.Add(time.Hour).Unix(),
			"organization": "built-in",
		}, []byte(secret))

		_, err := issuer.Verify(token)
		require.Error(t, err)
	})

	t.Run("alg none", func(t *testing.T) {
		token := mintSessionToken(t, secret, jwt.SigningMethodNone, jwt.MapClaims{
			"iss": SessionTokenIssuerName,
			"sub": "user-123",
			"iat": now.Unix(),
			"exp": now.Add(time.Hour).Unix(),
		}, jwt.UnsafeAllowNoneSignatureType)

		_, err = issuer.Verify(token)
		require.Error(t, err)
	})

	t.Run("HS512 with the same secret", func(t *testing.T) {
		token := mintSessionToken(t, secret, jwt.SigningMethodHS512, jwt.MapClaims{
			"iss": SessionTokenIssuerName,
			"sub": "user-123",
			"iat": now.Unix(),
			"exp": now.Add(time.Hour).Unix(),
		}, []byte(secret))

		_, err := issuer.Verify(token)
		require.Error(t, err)
	})

	t.Run("missing expiry", func(t *testing.T) {
		token := mintSessionToken(t, secret, jwt.SigningMethodHS256, jwt.MapClaims{
			"iss": SessionTokenIssuerName,
			"sub": "user-123",
			"iat": now.Unix(),
		}, []byte(secret))

		_, err := issuer.Verify(token)
		require.Error(t, err)
	})

	t.Run("tampered payload", func(t *testing.T) {
		token, err := issuer.Issue(IssueSessionTokenInput{UserID: "user-123", Organization: "built-in"})
		require.NoError(t, err)

		parts := strings.SplitN(token, ".", 3)
		require.Len(t, parts, 3)

		head, payload := parts[0], parts[1]

		// Flip the first payload character so the signature no longer matches.
		flipped := []byte(payload)
		if flipped[0] == 'e' {
			flipped[0] = 'f'
		} else {
			flipped[0] = 'e'
		}

		_, err = issuer.Verify(head + "." + string(flipped) + "." + strings.Split(token, ".")[2])
		require.Error(t, err)
	})

	t.Run("unknown fields are accepted (claim set is NOT pinned)", func(t *testing.T) {
		// Documents a real gap: jwt/v5's WithStrictDecoding only validates
		// base64 padding (RFC 4648 §3.5); it does not reject unknown claim
		// fields, so the claim set of session tokens is not actually pinned.
		// Verified against jwt/v5 v5.3.1. Tracked in the auth test plan risk
		// register (R1) — flip this to require.Error if production gains a
		// DisallowUnknownFields-style check later.
		token := mintSessionToken(t, secret, jwt.SigningMethodHS256, jwt.MapClaims{
			"iss":      SessionTokenIssuerName,
			"sub":      "user-123",
			"iat":      now.Unix(),
			"exp":      now.Add(time.Hour).Unix(),
			"surprise": "future-claim",
		}, []byte(secret))

		claims, err := issuer.Verify(token)
		require.NoError(t, err)
		require.Equal(t, "user-123", claims.Subject)
	})

	t.Run("empty organization verifies (enforcement is the middleware's job)", func(t *testing.T) {
		token, err := issuer.Issue(IssueSessionTokenInput{UserID: "user-123"})
		require.NoError(t, err)

		claims, err := issuer.Verify(token)
		require.NoError(t, err)
		require.Empty(t, claims.Organization)
	})
}
