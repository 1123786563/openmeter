package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/namespace/namespacedriver"
)

func TestNewAPICORSMiddleware(t *testing.T) {
	newRouter := func(t *testing.T, origins []string) http.Handler {
		t.Helper()

		middleware := NewAPICORSMiddleware(origins)
		require.NotNil(t, middleware)

		router := chi.NewRouter()
		router.Use(middleware)
		router.Get("/api/v3/openmeter/namespaces", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		return router
	}

	t.Run("empty origins yields no middleware", func(t *testing.T) {
		require.Nil(t, NewAPICORSMiddleware(nil))
		require.Nil(t, NewAPICORSMiddleware([]string{}))
	})

	t.Run("preflight from an allowed origin is answered successfully with headers", func(t *testing.T) {
		router := newRouter(t, []string{"https://admin.example.com"})

		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodOptions, "/api/v3/openmeter/namespaces", nil)
		request.Header.Set("Origin", "https://admin.example.com")
		request.Header.Set("Access-Control-Request-Method", http.MethodGet)
		request.Header.Set("Access-Control-Request-Headers", "authorization,"+namespacedriver.NamespaceHeader)

		router.ServeHTTP(recorder, request)

		// go-chi/cors answers preflight with 200 (any 2xx is spec-compliant).
		require.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, "https://admin.example.com", recorder.Header().Get("Access-Control-Allow-Origin"))
		assert.Contains(t, recorder.Header().Get("Access-Control-Allow-Headers"), "Authorization")
		assert.Contains(t, recorder.Header().Get("Access-Control-Allow-Headers"), namespacedriver.NamespaceHeader)
		// The API uses bearer tokens instead of cookies: never credentialed.
		assert.Empty(t, recorder.Header().Get("Access-Control-Allow-Credentials"))
	})

	t.Run("preflight from a disallowed origin carries no allow-origin header", func(t *testing.T) {
		router := newRouter(t, []string{"https://admin.example.com"})

		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodOptions, "/api/v3/openmeter/namespaces", nil)
		request.Header.Set("Origin", "https://evil.example.com")
		request.Header.Set("Access-Control-Request-Method", http.MethodGet)

		router.ServeHTTP(recorder, request)

		assert.Empty(t, recorder.Header().Get("Access-Control-Allow-Origin"))
	})

	t.Run("wildcard origin is allowed without credentials", func(t *testing.T) {
		router := newRouter(t, []string{"*"})

		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/api/v3/openmeter/namespaces", nil)
		request.Header.Set("Origin", "https://any.example.com")

		router.ServeHTTP(recorder, request)

		require.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, "*", recorder.Header().Get("Access-Control-Allow-Origin"))
		assert.Empty(t, recorder.Header().Get("Access-Control-Allow-Credentials"))
	})

	t.Run("portal routes are left to the portal CORS handler", func(t *testing.T) {
		middleware := NewAPICORSMiddleware([]string{"https://admin.example.com"})
		require.NotNil(t, middleware)

		var sawRequest bool
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sawRequest = true
		})

		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodOptions, "/api/v1/portal/meters", nil)
		request.Header.Set("Origin", "https://admin.example.com")
		request.Header.Set("Access-Control-Request-Method", http.MethodGet)

		middleware(next).ServeHTTP(recorder, request)

		// The preflight is not short-circuited by the API CORS middleware.
		require.True(t, sawRequest)
		assert.Empty(t, recorder.Header().Get("Access-Control-Allow-Origin"))
	})

	t.Run("actual request from an allowed origin carries the CORS header", func(t *testing.T) {
		router := newRouter(t, []string{"https://admin.example.com"})

		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/api/v3/openmeter/namespaces", nil)
		request.Header.Set("Origin", "https://admin.example.com")

		router.ServeHTTP(recorder, request)

		require.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, "https://admin.example.com", recorder.Header().Get("Access-Control-Allow-Origin"))
	})
}
