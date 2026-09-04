package auth

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/session"
)

func newSessionMiddleware(t *testing.T, exemptPrefixes ...string) (func(http.Handler) http.Handler, *SessionTokenIssuer) {
	t.Helper()

	tokens, err := NewSessionTokenIssuer(randomSecret(t), time.Hour)
	require.NoError(t, err)

	middleware, err := NewSessionMiddleware(SessionMiddlewareConfig{
		Tokens:             tokens,
		Logger:             slog.New(slog.NewTextHandler(io.Discard, nil)),
		ExemptPathPrefixes: exemptPrefixes,
	})
	require.NoError(t, err)

	return middleware, tokens
}

// sessionEchoHandler asserts the session attached by the middleware.
func sessionEchoHandler(t *testing.T) http.HandlerFunc {
	t.Helper()

	return func(w http.ResponseWriter, r *http.Request) {
		active := session.GetActiveSession(r.Context())
		require.NotNil(t, active)
		require.Equal(t, "user-123", active.UserID)
		require.Equal(t, "built-in", active.OrgID)
		require.Equal(t, "built-in", active.OrgSlug)
		require.Equal(t, SessionOrganizationRoleAdmin, active.OrgRole)

		w.WriteHeader(http.StatusOK)
	}
}

func TestSessionMiddleware(t *testing.T) {
	t.Run("missing bearer token is rejected", func(t *testing.T) {
		// given session enforcement
		middleware, _ := newSessionMiddleware(t)

		// when a request arrives without credentials
		w := httptest.NewRecorder()
		middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			t.Error("next handler must not be called")
		})).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/meters", nil))

		// then the request is rejected
		require.Equal(t, http.StatusUnauthorized, w.Code)
		require.Equal(t, `Bearer realm="openmeter"`, w.Header().Get("WWW-Authenticate"))
	})

	t.Run("non-bearer authorization is rejected", func(t *testing.T) {
		// given session enforcement
		middleware, _ := newSessionMiddleware(t)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/meters", nil)
		req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")

		w := httptest.NewRecorder()
		middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			t.Error("next handler must not be called")
		})).ServeHTTP(w, req)

		// then the request is rejected
		require.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("invalid token is rejected", func(t *testing.T) {
		// given session enforcement
		middleware, _ := newSessionMiddleware(t)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/meters", nil)
		req.Header.Set("Authorization", "Bearer not-a-session-token")

		w := httptest.NewRecorder()
		middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			t.Error("next handler must not be called")
		})).ServeHTTP(w, req)

		// then the request is rejected
		require.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("token without organization is rejected", func(t *testing.T) {
		// given session enforcement
		middleware, tokens := newSessionMiddleware(t)

		token, err := tokens.Issue(IssueSessionTokenInput{UserID: "user-123"})
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/meters", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		w := httptest.NewRecorder()
		middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			t.Error("next handler must not be called")
		})).ServeHTTP(w, req)

		// then the request is rejected because the tenant cannot be resolved
		require.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("valid token attaches the session", func(t *testing.T) {
		// given session enforcement
		middleware, tokens := newSessionMiddleware(t)

		token, err := tokens.Issue(IssueSessionTokenInput{
			UserID:       "user-123",
			Organization: "built-in",
		})
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/meters", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		w := httptest.NewRecorder()
		middleware(sessionEchoHandler(t)).ServeHTTP(w, req)

		// then the request passes with the session attached
		require.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("exempt path prefix passes through without a token", func(t *testing.T) {
		// given session enforcement with an exempt prefix
		middleware, _ := newSessionMiddleware(t, "/api/v1/portal/")

		w := httptest.NewRecorder()
		middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		})).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/portal/tokens", nil))

		// then the request passes through
		require.Equal(t, http.StatusOK, w.Code)
	})
}

func TestSessionMiddlewareExemptPrefixBoundaries(t *testing.T) {
	// given the production exemption list shape (trailing-slash portal prefix)
	tokens, err := NewSessionTokenIssuer(randomSecret(t), time.Hour)
	require.NoError(t, err)

	middleware, err := NewSessionMiddleware(SessionMiddlewareConfig{
		Tokens:             tokens,
		Logger:             slog.New(slog.NewTextHandler(io.Discard, nil)),
		ExemptPathPrefixes: []string{"/api/v1/portal/", "/api/swagger.json"},
	})
	require.NoError(t, err)

	reached := false
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		reached = true

		w.WriteHeader(http.StatusOK)
	}))

	t.Run("portal without trailing slash is not exempt", func(t *testing.T) {
		reached = false

		req := httptest.NewRequest(http.MethodGet, "/api/v1/portal", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		require.Equal(t, http.StatusUnauthorized, rec.Code)
		require.False(t, reached)
	})

	t.Run("portal sibling path is not exempt", func(t *testing.T) {
		reached = false

		req := httptest.NewRequest(http.MethodGet, "/api/v1/portalx/tokens", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		require.Equal(t, http.StatusUnauthorized, rec.Code)
		require.False(t, reached)
	})

	t.Run("portal subtree is exempt", func(t *testing.T) {
		reached = false

		req := httptest.NewRequest(http.MethodGet, "/api/v1/portal/apikeys", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		require.True(t, reached)
	})

	t.Run("swagger spec is exempt", func(t *testing.T) {
		reached = false

		req := httptest.NewRequest(http.MethodGet, "/api/swagger.json", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		require.True(t, reached)
	})
}
