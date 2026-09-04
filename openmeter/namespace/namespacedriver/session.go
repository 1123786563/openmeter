package namespacedriver

import (
	"context"
	"strings"

	"github.com/openmeterio/openmeter/openmeter/session"
)

// SessionNamespaceDecoder resolves the namespace from the authenticated
// session's organization, falling back to the given namespace when the
// request carries no session (e.g. portal-token or anonymous requests).
type SessionNamespaceDecoder struct {
	Fallback string
}

// NewSessionNamespaceDecoder returns a decoder falling back to the given
// namespace for requests without an authenticated session.
func NewSessionNamespaceDecoder(fallback string) SessionNamespaceDecoder {
	return SessionNamespaceDecoder{Fallback: fallback}
}

// GetNamespace returns the session organization lower-cased as the namespace,
// or the fallback namespace when no session is attached to the context.
func (d SessionNamespaceDecoder) GetNamespace(ctx context.Context) (string, bool) {
	if session := session.GetActiveSession(ctx); session != nil && session.OrgSlug != "" {
		return strings.ToLower(session.OrgSlug), true
	}

	return d.Fallback, true
}
