package namespacedriver

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/session"
)

func TestSessionNamespaceDecoderGetNamespace(t *testing.T) {
	t.Run("session organization resolves to the namespace", func(t *testing.T) {
		// given a request context with an authenticated session
		authSession, err := session.NewAuthenticationSession("acme", "Acme-Corp", "admin", "user-123", nil)
		require.NoError(t, err)

		ctx := session.WithAuthenticationSession(t.Context(), authSession)

		// when the namespace is decoded
		decoder := NewSessionNamespaceDecoder("default")

		// then the lower-cased organization is the namespace
		namespace, ok := decoder.GetNamespace(ctx)
		require.True(t, ok)
		require.Equal(t, "acme-corp", namespace)
	})

	t.Run("requests without a session fall back", func(t *testing.T) {
		// given a request context without a session
		decoder := NewSessionNamespaceDecoder("default")

		// when the namespace is decoded
		namespace, ok := decoder.GetNamespace(t.Context())

		// then the fallback namespace is used
		require.True(t, ok)
		require.Equal(t, "default", namespace)
	})
}
