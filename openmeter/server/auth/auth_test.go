package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"log/slog"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testIssuer     = "https://casdoor.test"
	testAudience   = "openmeter-api"
	testJwksPath   = "/api/certs"
	testOperatorKV = "/api/v1/customers"
)

// testCasdoor is a minimal Casdoor stand-in: an RSA signing key whose
// certificate endpoint is served by an in-process HTTP test server. Rotating
// the key field simulates Casdoor-side key rotation.
type testCasdoor struct {
	key    *rsa.PrivateKey
	kid    string
	server *httptest.Server
}

func newTestCasdoor(t testing.TB, kid string) *testCasdoor {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	casdoor := &testCasdoor{key: key, kid: kid}

	mux := http.NewServeMux()
	mux.HandleFunc(testJwksPath, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"keys": []jsonWebKey{{
				Kid: casdoor.kid,
				Kty: "RSA",
				N:   base64.RawURLEncoding.EncodeToString(casdoor.key.PublicKey.N.Bytes()),
				E:   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(casdoor.key.PublicKey.E)).Bytes()),
			}},
		})
	})

	casdoor.server = httptest.NewServer(mux)
	t.Cleanup(casdoor.server.Close)

	return casdoor
}

func (c *testCasdoor) jwksURL() string {
	return c.server.URL + testJwksPath
}

// token mints an RS256 token signed with the issuer's current key.
func (c *testCasdoor) token(t testing.TB, mutate func(jwt.MapClaims)) string {
	t.Helper()

	return signToken(t, c.key, c.kid, baseClaims(mutate))
}

// signToken signs the given claims, allowing tokens from foreign keys.
func signToken(t testing.TB, key *rsa.PrivateKey, kid string, claims jwt.MapClaims) string {
	t.Helper()

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = kid

	signed, err := token.SignedString(key)
	require.NoError(t, err)

	return signed
}

func baseClaims(mutate func(jwt.MapClaims)) jwt.MapClaims {
	claims := jwt.MapClaims{
		"iss":   testIssuer,
		"sub":   "alice",
		"aud":   testAudience,
		"exp":   time.Now().Add(5 * time.Minute).Unix(),
		"iat":   time.Now().Add(time.Minute).Unix(),
		"owner": "acme",
		"roles": []any{"admin"},
	}

	if mutate != nil {
		mutate(claims)
	}

	return claims
}

func testLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

// baseConfig matches baseClaims: organization acme, admin as Operator,
// viewer as Viewer, audience enforced.
func baseConfig(casdoor *testCasdoor) Config {
	return Config{
		Logger:               testLogger(),
		Issuer:               testIssuer,
		JwksURL:              casdoor.jwksURL(),
		Audience:             testAudience,
		AllowedOrganizations: []string{"acme"},
		OrganizationClaim:    "owner",
		ViewerRoles:          []string{"viewer"},
		OperatorRoles:        []string{"admin"},
		RoleClaim:            "roles",
	}
}

func TestMiddlewareHandle(t *testing.T) {
	casdoor := newTestCasdoor(t, "test-key-1")

	rogueKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	tests := []struct {
		name          string
		config        func(*Config)
		method        string
		path          string
		authorization func() string
		wantStatus    int
		wantIdentity  *Identity
	}{
		{
			name:          "valid operator token",
			method:        http.MethodGet,
			path:          testOperatorKV,
			authorization: func() string { return "Bearer " + casdoor.token(t, nil) },
			wantStatus:    http.StatusOK,
			wantIdentity:  &Identity{Subject: "alice", Organization: "acme", Role: RoleOperator},
		},
		{
			name:          "operator can write",
			method:        http.MethodPost,
			path:          testOperatorKV,
			authorization: func() string { return "Bearer " + casdoor.token(t, nil) },
			wantStatus:    http.StatusOK,
			wantIdentity:  &Identity{Subject: "alice", Organization: "acme", Role: RoleOperator},
		},
		{
			name:   "viewer can read",
			method: http.MethodGet,
			path:   testOperatorKV,
			authorization: func() string {
				return "Bearer " + casdoor.token(t, func(claims jwt.MapClaims) { claims["roles"] = []any{"viewer"} })
			},
			wantStatus:   http.StatusOK,
			wantIdentity: &Identity{Subject: "alice", Organization: "acme", Role: RoleViewer},
		},
		{
			name:   "viewer cannot write",
			method: http.MethodPost,
			path:   testOperatorKV,
			authorization: func() string {
				return "Bearer " + casdoor.token(t, func(claims jwt.MapClaims) { claims["roles"] = []any{"viewer"} })
			},
			wantStatus: http.StatusForbidden,
		},
		{
			name:   "viewer cannot delete",
			method: http.MethodDelete,
			path:   testOperatorKV,
			authorization: func() string {
				return "Bearer " + casdoor.token(t, func(claims jwt.MapClaims) { claims["roles"] = "viewer" })
			},
			wantStatus: http.StatusForbidden,
		},
		{
			name:   "expired token",
			method: http.MethodGet,
			path:   testOperatorKV,
			authorization: func() string {
				return "Bearer " + casdoor.token(t, func(claims jwt.MapClaims) { claims["exp"] = time.Now().Add(-time.Minute).Unix() })
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:   "not yet valid token",
			method: http.MethodGet,
			path:   testOperatorKV,
			authorization: func() string {
				return "Bearer " + casdoor.token(t, func(claims jwt.MapClaims) { claims["nbf"] = time.Now().Add(time.Hour).Unix() })
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:   "wrong issuer",
			method: http.MethodGet,
			path:   testOperatorKV,
			authorization: func() string {
				return "Bearer " + casdoor.token(t, func(claims jwt.MapClaims) { claims["iss"] = "https://evil.test" })
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:   "wrong audience",
			method: http.MethodGet,
			path:   testOperatorKV,
			authorization: func() string {
				return "Bearer " + casdoor.token(t, func(claims jwt.MapClaims) { claims["aud"] = "other-api" })
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:   "wrong signature",
			method: http.MethodGet,
			path:   testOperatorKV,
			authorization: func() string {
				return "Bearer " + signToken(t, rogueKey, "test-key-1", baseClaims(nil))
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:   "unknown key id",
			method: http.MethodGet,
			path:   testOperatorKV,
			authorization: func() string {
				return "Bearer " + signToken(t, rogueKey, "unknown-kid", baseClaims(nil))
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:   "organization not allowed",
			method: http.MethodGet,
			path:   testOperatorKV,
			authorization: func() string {
				return "Bearer " + casdoor.token(t, func(claims jwt.MapClaims) { claims["owner"] = "contoso" })
			},
			wantStatus: http.StatusForbidden,
		},
		{
			name:   "missing organization claim",
			method: http.MethodGet,
			path:   testOperatorKV,
			authorization: func() string {
				return "Bearer " + casdoor.token(t, func(claims jwt.MapClaims) { delete(claims, "owner") })
			},
			wantStatus: http.StatusForbidden,
		},
		{
			name:   "no matching role",
			method: http.MethodGet,
			path:   testOperatorKV,
			authorization: func() string {
				return "Bearer " + casdoor.token(t, func(claims jwt.MapClaims) { claims["roles"] = []any{"nobody"} })
			},
			wantStatus: http.StatusForbidden,
		},
		{
			name:   "empty role lists grant operator",
			config: func(c *Config) { c.ViewerRoles, c.OperatorRoles = nil, nil },
			method: http.MethodPost,
			path:   testOperatorKV,
			authorization: func() string {
				return "Bearer " + casdoor.token(t, func(claims jwt.MapClaims) { delete(claims, "roles") })
			},
			wantStatus:   http.StatusOK,
			wantIdentity: &Identity{Subject: "alice", Organization: "acme", Role: RoleOperator},
		},
		{
			name:   "audience check skipped when not configured",
			config: func(c *Config) { c.Audience = "" },
			method: http.MethodGet,
			path:   testOperatorKV,
			authorization: func() string {
				return "Bearer " + casdoor.token(t, func(claims jwt.MapClaims) { claims["aud"] = "other-api" })
			},
			wantStatus:   http.StatusOK,
			wantIdentity: &Identity{Subject: "alice", Organization: "acme", Role: RoleOperator},
		},
		{
			name:          "missing authorization header",
			method:        http.MethodGet,
			path:          testOperatorKV,
			authorization: func() string { return "" },
			wantStatus:    http.StatusUnauthorized,
		},
		{
			name:          "non bearer authorization header",
			method:        http.MethodGet,
			path:          testOperatorKV,
			authorization: func() string { return "Basic dXNlcjpwYXNz" },
			wantStatus:    http.StatusUnauthorized,
		},
		{
			name:          "malformed token",
			method:        http.MethodGet,
			path:          testOperatorKV,
			authorization: func() string { return "Bearer not-a-jwt" },
			wantStatus:    http.StatusUnauthorized,
		},
		{
			name:          "portal path bypasses authentication",
			method:        http.MethodGet,
			path:          "/api/v1/portal/meters/api_requests_total/values",
			authorization: func() string { return "" },
			wantStatus:    http.StatusOK,
		},
		{
			name:          "v3 openapi json bypasses authentication",
			method:        http.MethodGet,
			path:          "/api/v3/openapi.json",
			authorization: func() string { return "" },
			wantStatus:    http.StatusOK,
		},
		{
			name:          "v3 openapi yaml bypasses authentication",
			method:        http.MethodGet,
			path:          "/api/v3/openapi.yaml",
			authorization: func() string { return "" },
			wantStatus:    http.StatusOK,
		},
		{
			name:          "v1 swagger json bypasses authentication",
			method:        http.MethodGet,
			path:          "/api/swagger.json",
			authorization: func() string { return "" },
			wantStatus:    http.StatusOK,
		},
		{
			name:          "debug metrics bypasses authentication",
			method:        http.MethodGet,
			path:          "/api/v1/debug/metrics",
			authorization: func() string { return "" },
			wantStatus:    http.StatusOK,
		},
		{
			name:          "portal token management requires authentication",
			method:        http.MethodPost,
			path:          "/api/v1/portal/tokens",
			authorization: func() string { return "" },
			wantStatus:    http.StatusUnauthorized,
		},
		{
			name:          "preflight options bypasses authentication",
			method:        http.MethodOptions,
			path:          testOperatorKV,
			authorization: func() string { return "" },
			wantStatus:    http.StatusOK,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := baseConfig(casdoor)
			if test.config != nil {
				test.config(&config)
			}

			middleware, err := New(config)
			require.NoError(t, err)

			var gotIdentity Identity
			var gotOK bool
			handler := middleware.Handle(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotIdentity, gotOK = FromContext(r.Context())
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest(test.method, test.path, nil).WithContext(t.Context())
			if authorization := test.authorization(); authorization != "" {
				req.Header.Set("Authorization", authorization)
			}
			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, req)

			assert.Equal(t, test.wantStatus, recorder.Code)

			if test.wantIdentity != nil {
				require.True(t, gotOK, "expected an identity in the request context")
				assert.Equal(t, *test.wantIdentity, gotIdentity)
			} else {
				assert.False(t, gotOK, "expected no identity in the request context")
			}
		})
	}
}

func TestMiddlewareJwksKeyRotation(t *testing.T) {
	casdoor := newTestCasdoor(t, "test-key-1")

	middleware, err := New(baseConfig(casdoor))
	require.NoError(t, err)

	do := func(token string) int {
		t.Helper()

		handler := middleware.Handle(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest(http.MethodGet, testOperatorKV, nil).WithContext(t.Context())
		req.Header.Set("Authorization", "Bearer "+token)
		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, req)

		return recorder.Code
	}

	// given the initial key is served, its tokens pass
	assert.Equal(t, http.StatusOK, do(casdoor.token(t, nil)))

	// when Casdoor rotates the key, cached verification keeps rejecting the
	// new tokens until the lazy refresh interval elapses
	rotatedKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	casdoor.key = rotatedKey

	assert.Equal(t, http.StatusUnauthorized, do(casdoor.token(t, nil)))

	// then a refresh past the throttle window picks the rotated key up
	middleware.jwks.mu.Lock()
	middleware.jwks.lastRefresh = time.Now().Add(-2 * jwksMinRefreshInterval)
	middleware.jwks.mu.Unlock()

	assert.Equal(t, http.StatusOK, do(casdoor.token(t, nil)))
}

func TestNewOptional(t *testing.T) {
	t.Run("disabled bypasses authentication without validating the config", func(t *testing.T) {
		middleware, err := NewOptional(false, Config{})
		require.NoError(t, err)
		assert.Nil(t, middleware)
	})

	t.Run("enabled with invalid config returns an error", func(t *testing.T) {
		middleware, err := NewOptional(true, Config{Logger: testLogger()})
		require.Error(t, err)
		assert.Nil(t, middleware)
	})

	t.Run("enabled with valid config returns the middleware", func(t *testing.T) {
		casdoor := newTestCasdoor(t, "test-key-1")

		middleware, err := NewOptional(true, baseConfig(casdoor))
		require.NoError(t, err)
		require.NotNil(t, middleware)
	})
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name   string
		config Config
		valid  bool
	}{
		{
			name:   "empty config is invalid",
			config: Config{},
		},
		{
			name:   "missing issuer",
			config: Config{Logger: testLogger(), JwksURL: "https://casdoor.test/api/certs"},
		},
		{
			name:   "missing jwks url",
			config: Config{Logger: testLogger(), Issuer: testIssuer},
		},
		{
			name:   "missing logger",
			config: Config{Issuer: testIssuer, JwksURL: "https://casdoor.test/api/certs"},
		},
		{
			name: "complete config",
			config: Config{
				Logger:  testLogger(),
				Issuer:  testIssuer,
				JwksURL: "https://casdoor.test/api/certs",
			},
			valid: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.valid {
				require.NoError(t, test.config.Validate())
			} else {
				require.Error(t, test.config.Validate())
			}
		})
	}
}

func TestJSONWebKeyPublicKeyFromX5c(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "casdoor-test"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)

	// The middleware parses the base64 DER form from the JWKS x5c array.
	pemCert := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	block, _ := pem.Decode(pemCert)
	require.NotNil(t, block)

	jwk := jsonWebKey{Kid: "x5c-key", Kty: "RSA", X5c: []string{base64.StdEncoding.EncodeToString(block.Bytes)}}

	publicKey, err := jwk.publicKey()
	require.NoError(t, err)
	assert.Equal(t, &key.PublicKey, publicKey)
}

// TestJWKSCacheThrottlesFailedRefresh guards the refresh rate limit: while a
// JWKS endpoint is failing, an unknown kid must not trigger a fetch per request.
func TestJWKSCacheThrottlesFailedRefresh(t *testing.T) {
	var requests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests++
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	t.Cleanup(server.Close)

	cache := newJWKSCache(server.URL, server.Client())

	for range 3 {
		_, err := cache.publicKey(t.Context(), "test-key-1")
		require.Error(t, err)
	}

	assert.Equal(t, 1, requests, "expected a single refresh attempt inside the throttle window")
}
