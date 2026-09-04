package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"math/big"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/pkg/models"
)

const testClientID = "openmeter-client"

// randomSecret returns a non-reusable secret so no credential literal ever
// appears in source control.
func randomSecret(t *testing.T) string {
	t.Helper()

	return randomHex()
}

func randomHex() string {
	buf := make([]byte, 32)
	_, _ = rand.Read(buf)

	return hex.EncodeToString(buf)
}

// fakeIDP is a minimal OIDC provider serving discovery, JWKS and token
// endpoints so the full login flow can run without a real Casdoor.
type fakeIDP struct {
	*httptest.Server

	// jwksKey is published at the JWKS endpoint; signingKey signs the ID
	// tokens. They are the same key unless a test simulates a forgery.
	jwksKey    *rsa.PrivateKey
	signingKey *rsa.PrivateKey

	// failTokenExchange makes the token endpoint answer 500, simulating a
	// provider outage during the code exchange.
	failTokenExchange bool

	// omitIDToken makes the token response omit the id_token field.
	omitIDToken bool

	// organization adds an "organization" claim alongside "owner" when set;
	// noOrganization strips the "owner" claim instead.
	organization   string
	noOrganization bool
}

func newFakeIDP(t *testing.T) *fakeIDP {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	idp := &fakeIDP{jwksKey: key, signingKey: key}
	idp.Server = httptest.NewServer(http.HandlerFunc(idp.serveHTTP))
	t.Cleanup(idp.Close)

	return idp
}

func (idp *fakeIDP) serveHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/.well-known/openid-configuration":
		writeJSON(w, map[string]any{
			"issuer":                                idp.URL,
			"authorization_endpoint":                idp.URL + "/login/oauth/authorize",
			"token_endpoint":                        idp.URL + "/api/login/oauth/access_token",
			"jwks_uri":                              idp.URL + "/api/certs",
			"id_token_signing_alg_values_supported": []string{"RS256"},
		})
	case "/api/certs":
		pub := idp.jwksKey.PublicKey
		writeJSON(w, map[string]any{
			"keys": []map[string]string{
				{
					"kty": "RSA",
					"alg": "RS256",
					"use": "sig",
					"kid": "test-key",
					"n":   base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
					"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pub.E)).Bytes()),
				},
			},
		})
	case "/api/login/oauth/access_token":
		if idp.failTokenExchange {
			http.Error(w, "upstream unavailable", http.StatusInternalServerError)

			return
		}

		now := time.Now()

		claims := jwt.MapClaims{
			"iss":   idp.URL,
			"aud":   testClientID,
			"sub":   "user-123",
			"iat":   now.Unix(),
			"exp":   now.Add(time.Hour).Unix(),
			"email": "user@example.com",
			"name":  "Test User",
			"owner": "built-in",
		}
		if idp.organization != "" {
			claims["organization"] = idp.organization
		}
		if idp.noOrganization {
			delete(claims, "owner")
		}

		idToken := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
		idToken.Header["kid"] = "test-key"

		if idp.omitIDToken {
			writeJSON(w, map[string]any{
				"access_token": randomHex(),
				"token_type":   "Bearer",
				"expires_in":   3600,
			})

			return
		}

		signed, err := idToken.SignedString(idp.signingKey)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)

			return
		}

		writeJSON(w, map[string]any{
			"access_token": randomHex(),
			"token_type":   "Bearer",
			"expires_in":   3600,
			"id_token":     signed,
		})
	default:
		http.NotFound(w, r)
	}
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}

func newTestHandler(t *testing.T, idp *fakeIDP) (*Handler, *SessionTokenIssuer) {
	t.Helper()

	tokens, err := NewSessionTokenIssuer(randomSecret(t), time.Hour)
	require.NoError(t, err)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	handler, err := NewHandler(t.Context(), HandlerConfig{
		Issuer:       idp.URL,
		ClientID:     testClientID,
		ClientSecret: randomSecret(t),
		RedirectURL:  "http://api.example.com/auth/oidc/callback",
		DashboardURL: "http://front.example.com/auth/callback",
		Tokens:       tokens,
	}, logger)
	require.NoError(t, err)

	return handler, tokens
}

// startLogin performs the login request and returns the redirect target plus
// the cookies the callback must present. A non-empty loginState is sent as the
// front-end nonce and asserted to come back in the callback fragment.
func startLogin(t *testing.T, client *http.Client, srvURL, loginState string) (*url.URL, []*http.Cookie) {
	t.Helper()

	loginURL := srvURL + "/auth/oidc/login"
	if loginState != "" {
		loginURL += "?state=" + url.QueryEscape(loginState)
	}

	resp, err := client.Get(loginURL)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusFound, resp.StatusCode)

	location, err := url.Parse(resp.Header.Get("Location"))
	require.NoError(t, err)
	require.Equal(t, "/login/oauth/authorize", location.Path)
	require.Equal(t, testClientID, location.Query().Get("client_id"))

	state := location.Query().Get("state")
	require.NotEmpty(t, state)

	var stateCookie *http.Cookie

	var loginStateCookie *http.Cookie

	for _, cookie := range resp.Cookies() {
		switch cookie.Name {
		case stateCookieName:
			stateCookie = cookie
		case loginStateCookieName:
			loginStateCookie = cookie
		}
	}

	require.NotNil(t, stateCookie, "oidc state cookie is set")
	require.Equal(t, state, stateCookie.Value)

	if loginState != "" {
		require.NotNil(t, loginStateCookie, "front-end nonce cookie is set")
		require.Equal(t, loginState, loginStateCookie.Value)
	} else {
		require.Nil(t, loginStateCookie, "no front-end nonce cookie without a login state")
	}

	return location, resp.Cookies()
}

func TestHandlerCallbackIssuesSessionToken(t *testing.T) {
	// given an OIDC handler backed by a fake identity provider
	idp := newFakeIDP(t)
	handler, tokens := newTestHandler(t, idp)

	router := chi.NewRouter()
	require.NoError(t, handler.RegisterRoutes(router))

	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)

	client := &http.Client{
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}

	// when the browser completes the login flow, passing a front-end nonce
	location, cookies := startLogin(t, client, srv.URL, "front-end-nonce")

	req, err := http.NewRequest(http.MethodGet, srv.URL+"/auth/oidc/callback?code=test-code&state="+url.QueryEscape(location.Query().Get("state")), nil)
	require.NoError(t, err)
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// then the browser is sent to the dashboard with a valid session token
	require.Equal(t, http.StatusFound, resp.StatusCode)

	redirect, err := url.Parse(resp.Header.Get("Location"))
	require.NoError(t, err)
	require.Equal(t, "front.example.com", redirect.Host)
	require.Equal(t, "/auth/callback", redirect.Path)

	fragment, err := url.ParseQuery(redirect.Fragment)
	require.NoError(t, err)

	// The fragment follows the front-end SSO callback contract: the token,
	// the organization as tenant_id and the nonce this tab sent.
	require.Equal(t, "front-end-nonce", fragment.Get("state"))
	require.Equal(t, "built-in", fragment.Get("tenant_id"))

	claims, err := tokens.Verify(fragment.Get("token"))
	require.NoError(t, err)
	require.Equal(t, "user-123", claims.Subject)
	require.Equal(t, "user@example.com", claims.Email)
	require.Equal(t, "Test User", claims.Name)
	require.Equal(t, "built-in", claims.Organization)
	require.Equal(t, SessionTokenIssuerName, claims.Issuer)
}

func TestHandlerCallbackRejectsInvalidState(t *testing.T) {
	testCases := []struct {
		name string
		url  string
	}{
		{
			name: "state mismatch",
			url:  "/auth/oidc/callback?code=test-code&state=forged-state",
		},
		{
			name: "missing code",
			url:  "/auth/oidc/callback?state=anything",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// given an OIDC handler backed by a fake identity provider
			idp := newFakeIDP(t)
			handler, _ := newTestHandler(t, idp)

			router := chi.NewRouter()
			require.NoError(t, handler.RegisterRoutes(router))

			srv := httptest.NewServer(router)
			t.Cleanup(srv.Close)

			client := &http.Client{
				CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
			}

			// when the callback is called with a state cookie that does not match
			req, err := http.NewRequest(http.MethodGet, srv.URL+tc.url, nil)
			require.NoError(t, err)
			req.AddCookie(&http.Cookie{Name: stateCookieName, Value: "legitimate-state"})

			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			// then the request is rejected
			require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		})
	}
}

func TestHandlerCallbackRejectsMissingStateCookie(t *testing.T) {
	// given an OIDC handler backed by a fake identity provider
	idp := newFakeIDP(t)
	handler, _ := newTestHandler(t, idp)

	router := chi.NewRouter()
	require.NoError(t, handler.RegisterRoutes(router))

	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)

	client := &http.Client{
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}

	// when the callback is called without any state cookie
	resp, err := client.Get(srv.URL + "/auth/oidc/callback?code=test-code&state=anything")
	require.NoError(t, err)
	defer resp.Body.Close()

	// then the request is rejected
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestHandlerCallbackRejectsForgedIDToken(t *testing.T) {
	// given an identity provider whose token endpoint signs with a key that
	// is not published in its JWKS
	idp := newFakeIDP(t)

	rogueKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	idp.signingKey = rogueKey

	handler, _ := newTestHandler(t, idp)

	router := chi.NewRouter()
	require.NoError(t, handler.RegisterRoutes(router))

	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)

	client := &http.Client{
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}

	// when the browser completes the login flow
	location, cookies := startLogin(t, client, srv.URL, "front-end-nonce")

	req, err := http.NewRequest(http.MethodGet, srv.URL+"/auth/oidc/callback?code=test-code&state="+url.QueryEscape(location.Query().Get("state")), nil)
	require.NoError(t, err)
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// then the forged ID token is rejected
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestSessionTokenIssuer(t *testing.T) {
	t.Run("roundtrip", func(t *testing.T) {
		issuer, err := NewSessionTokenIssuer(randomSecret(t), time.Hour)
		require.NoError(t, err)

		token, err := issuer.Issue(IssueSessionTokenInput{
			UserID:       "user-123",
			Email:        "user@example.com",
			Organization: "built-in",
		})
		require.NoError(t, err)

		claims, err := issuer.Verify(token)
		require.NoError(t, err)
		require.Equal(t, "user-123", claims.Subject)
		require.Equal(t, "user@example.com", claims.Email)
		require.Equal(t, "built-in", claims.Organization)
	})

	t.Run("rejects token signed with a different secret", func(t *testing.T) {
		issuer, err := NewSessionTokenIssuer(randomSecret(t), time.Hour)
		require.NoError(t, err)

		other, err := NewSessionTokenIssuer(randomSecret(t), time.Hour)
		require.NoError(t, err)

		token, err := issuer.Issue(IssueSessionTokenInput{UserID: "user-123"})
		require.NoError(t, err)

		_, err = other.Verify(token)
		require.Error(t, err)
	})

	t.Run("rejects input without a user id", func(t *testing.T) {
		issuer, err := NewSessionTokenIssuer(randomSecret(t), time.Hour)
		require.NoError(t, err)

		_, err = issuer.Issue(IssueSessionTokenInput{})
		require.Error(t, err)

		var validationErr *models.GenericValidationError
		require.ErrorAs(t, err, &validationErr)
	})
}

func TestHandlerCallbackTokenExchangeFailure(t *testing.T) {
	// given a provider whose token endpoint is down
	idp := newFakeIDP(t)
	idp.failTokenExchange = true

	handler, _ := newTestHandler(t, idp)

	router := chi.NewRouter()
	require.NoError(t, handler.RegisterRoutes(router))

	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)

	client := &http.Client{
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}

	location, cookies := startLogin(t, client, srv.URL, "")

	req, err := http.NewRequest(http.MethodGet, srv.URL+"/auth/oidc/callback?code=test-code&state="+url.QueryEscape(location.Query().Get("state")), nil)
	require.NoError(t, err)
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// then the callback fails with a bad gateway, not a session token
	require.Equal(t, http.StatusBadGateway, resp.StatusCode)
}

func TestHandlerCallbackTokenResponseWithoutIDToken(t *testing.T) {
	// given a provider whose token response carries no id_token
	idp := newFakeIDP(t)
	idp.omitIDToken = true

	handler, _ := newTestHandler(t, idp)

	router := chi.NewRouter()
	require.NoError(t, handler.RegisterRoutes(router))

	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)

	client := &http.Client{
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}

	location, cookies := startLogin(t, client, srv.URL, "")

	req, err := http.NewRequest(http.MethodGet, srv.URL+"/auth/oidc/callback?code=test-code&state="+url.QueryEscape(location.Query().Get("state")), nil)
	require.NoError(t, err)
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusBadGateway, resp.StatusCode)
}

func TestHandlerCallbackStateIsSingleUse(t *testing.T) {
	// given a browser that keeps cookies in a jar (honouring MaxAge=-1 deletes)
	idp := newFakeIDP(t)
	handler, _ := newTestHandler(t, idp)

	router := chi.NewRouter()
	require.NoError(t, handler.RegisterRoutes(router))

	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)

	jar, err := cookiejar.New(nil)
	require.NoError(t, err)

	client := &http.Client{
		Jar:           jar,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}

	location, _ := startLogin(t, client, srv.URL, "")

	callbackURL := srv.URL + "/auth/oidc/callback?code=test-code&state=" + url.QueryEscape(location.Query().Get("state"))

	// when the callback is completed, then replayed with the same URL
	first, err := client.Get(callbackURL)
	require.NoError(t, err)
	defer first.Body.Close()
	require.Equal(t, http.StatusFound, first.StatusCode)

	second, err := client.Get(callbackURL)
	require.NoError(t, err)
	defer second.Body.Close()

	// then the replay is rejected: the browser dropped the single-use state cookie
	require.Equal(t, http.StatusBadRequest, second.StatusCode)
}

func TestHandlerCallbackOrganizationClaims(t *testing.T) {
	t.Run("organization claim wins over owner", func(t *testing.T) {
		idp := newFakeIDP(t)
		idp.organization = "acme-org"

		handler, tokens := newTestHandler(t, idp)

		router := chi.NewRouter()
		require.NoError(t, handler.RegisterRoutes(router))

		srv := httptest.NewServer(router)
		t.Cleanup(srv.Close)

		client := &http.Client{
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		}

		location, cookies := startLogin(t, client, srv.URL, "")

		req, err := http.NewRequest(http.MethodGet, srv.URL+"/auth/oidc/callback?code=test-code&state="+url.QueryEscape(location.Query().Get("state")), nil)
		require.NoError(t, err)
		for _, cookie := range cookies {
			req.AddCookie(cookie)
		}

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		redirect, err := url.Parse(resp.Header.Get("Location"))
		require.NoError(t, err)

		fragment, err := url.ParseQuery(redirect.Fragment)
		require.NoError(t, err)
		require.Equal(t, "acme-org", fragment.Get("tenant_id"))

		claims, err := tokens.Verify(fragment.Get("token"))
		require.NoError(t, err)
		require.Equal(t, "acme-org", claims.Organization)
	})

	t.Run("no organization issues a token without tenant_id", func(t *testing.T) {
		// Documents the current dead-end: the login "succeeds" but the
		// session middleware will 401 every API call. Tracked in the risk
		// register (R3) — if fixed to reject at the callback, update this.
		idp := newFakeIDP(t)
		idp.noOrganization = true

		handler, tokens := newTestHandler(t, idp)

		router := chi.NewRouter()
		require.NoError(t, handler.RegisterRoutes(router))

		srv := httptest.NewServer(router)
		t.Cleanup(srv.Close)

		client := &http.Client{
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		}

		location, cookies := startLogin(t, client, srv.URL, "")

		req, err := http.NewRequest(http.MethodGet, srv.URL+"/auth/oidc/callback?code=test-code&state="+url.QueryEscape(location.Query().Get("state")), nil)
		require.NoError(t, err)
		for _, cookie := range cookies {
			req.AddCookie(cookie)
		}

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusFound, resp.StatusCode)

		redirect, err := url.Parse(resp.Header.Get("Location"))
		require.NoError(t, err)

		fragment, err := url.ParseQuery(redirect.Fragment)
		require.NoError(t, err)
		require.Empty(t, fragment.Get("tenant_id"))

		claims, err := tokens.Verify(fragment.Get("token"))
		require.NoError(t, err)
		require.Empty(t, claims.Organization)
	})
}
