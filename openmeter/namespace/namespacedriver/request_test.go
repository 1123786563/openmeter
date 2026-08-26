package namespacedriver

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequestNamespaceDecoder(t *testing.T) {
	decoder := RequestNamespaceDecoder("default")

	t.Run("request namespace takes precedence", func(t *testing.T) {
		ctx := WithRequestNamespace(t.Context(), "tenant-a")

		namespace, ok := decoder.GetNamespace(ctx)
		require.True(t, ok)
		require.Equal(t, "tenant-a", namespace)
	})

	t.Run("falls back to the default without a request namespace", func(t *testing.T) {
		namespace, ok := decoder.GetNamespace(t.Context())
		require.True(t, ok)
		require.Equal(t, "default", namespace)
	})
}

func TestRequestNamespaceContextRoundTrip(t *testing.T) {
	_, ok := RequestNamespaceFromContext(t.Context())
	require.False(t, ok)

	namespace, ok := RequestNamespaceFromContext(WithRequestNamespace(t.Context(), "tenant-a"))
	require.True(t, ok)
	require.Equal(t, "tenant-a", namespace)
}

func TestRequestNamespaceMiddlewareConfigValidate(t *testing.T) {
	base := RequestNamespaceMiddlewareConfig{
		Logger:           slog.Default(),
		DefaultNamespace: "default",
	}

	t.Run("valid", func(t *testing.T) {
		config := base
		config.Allowlist = []string{"tenant-a", "tenant-b"}
		require.NoError(t, config.Validate())
	})

	t.Run("missing logger", func(t *testing.T) {
		config := base
		config.Logger = nil
		require.Error(t, config.Validate())
	})

	t.Run("missing default", func(t *testing.T) {
		config := base
		config.DefaultNamespace = ""
		require.Error(t, config.Validate())
	})

	t.Run("empty allowlist entry", func(t *testing.T) {
		config := base
		config.Allowlist = []string{"tenant-a", ""}
		require.Error(t, config.Validate())
	})
}

func TestRequestNamespaceMiddleware(t *testing.T) {
	newMiddleware := func(t *testing.T, allowlist []string) *RequestNamespaceMiddleware {
		t.Helper()

		middleware, err := NewRequestNamespaceMiddleware(RequestNamespaceMiddlewareConfig{
			Logger:           slog.Default(),
			DefaultNamespace: "default",
			Allowlist:        allowlist,
		})
		require.NoError(t, err)

		return middleware
	}

	// observedNamespace records the namespace the downstream handler resolves.
	observedNamespace := func(decoder RequestNamespaceDecoder) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			namespace, ok := decoder.GetNamespace(r.Context())
			require.True(t, ok)

			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(namespace))
		}
	}

	t.Run("empty allowlist ignores the header entirely", func(t *testing.T) {
		middleware := newMiddleware(t, nil)
		decoder := RequestNamespaceDecoder("default")

		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/", nil)
		request.Header.Set(NamespaceHeader, "tenant-a")

		middleware.Handle(observedNamespace(decoder)).ServeHTTP(recorder, request)

		require.Equal(t, http.StatusOK, recorder.Code)
		require.Equal(t, "default", recorder.Body.String())
	})

	t.Run("allowed namespace is selected", func(t *testing.T) {
		middleware := newMiddleware(t, []string{"tenant-a", "tenant-b"})
		decoder := RequestNamespaceDecoder("default")

		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/", nil)
		request.Header.Set(NamespaceHeader, "tenant-b")

		middleware.Handle(observedNamespace(decoder)).ServeHTTP(recorder, request)

		require.Equal(t, http.StatusOK, recorder.Code)
		require.Equal(t, "tenant-b", recorder.Body.String())
	})

	t.Run("missing header falls back to the default", func(t *testing.T) {
		middleware := newMiddleware(t, []string{"tenant-a"})
		decoder := RequestNamespaceDecoder("default")

		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/", nil)

		middleware.Handle(observedNamespace(decoder)).ServeHTTP(recorder, request)

		require.Equal(t, http.StatusOK, recorder.Code)
		require.Equal(t, "default", recorder.Body.String())
	})

	t.Run("default namespace is always allowed even outside the allowlist", func(t *testing.T) {
		middleware := newMiddleware(t, []string{"tenant-a"})
		decoder := RequestNamespaceDecoder("default")

		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/", nil)
		request.Header.Set(NamespaceHeader, "default")

		middleware.Handle(observedNamespace(decoder)).ServeHTTP(recorder, request)

		require.Equal(t, http.StatusOK, recorder.Code)
		require.Equal(t, "default", recorder.Body.String())
	})

	t.Run("namespace outside the allowlist is rejected", func(t *testing.T) {
		middleware := newMiddleware(t, []string{"tenant-a"})
		decoder := RequestNamespaceDecoder("default")

		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/", nil)
		request.Header.Set(NamespaceHeader, "tenant-z")

		middleware.Handle(observedNamespace(decoder)).ServeHTTP(recorder, request)

		require.Equal(t, http.StatusForbidden, recorder.Code)
		assert.Contains(t, recorder.Body.String(), "tenant-z")
		// RFC 7807 problem responses are JSON
		assert.Contains(t, recorder.Header().Get("Content-Type"), "application/problem+json")
	})
}
