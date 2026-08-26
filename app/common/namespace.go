package common

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/google/wire"

	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/namespace"
	"github.com/openmeterio/openmeter/openmeter/namespace/namespacedriver"
)

var Namespace = wire.NewSet(
	NewNamespaceManager,
)

func NewNamespaceManager(
	conf config.NamespaceConfiguration,
) (*namespace.Manager, error) {
	manager, err := namespace.NewManager(namespace.ManagerConfig{
		DefaultNamespace:  conf.Default,
		DisableManagement: conf.DisableManagement,
	})
	if err != nil {
		return nil, fmt.Errorf("create namespace manager: %w", err)
	}

	return manager, nil
}

var NamespaceDecoder = wire.NewSet(
	NewNamespaceDecoder,
)

// NewNamespaceDecoder returns the namespace decoder for request handlers: with
// an empty allowlist it stays static, a non-empty allowlist opts into
// request-level namespace selection (request context first, static default as
// fallback). Pair with NewRequestNamespaceMiddleware, which populates the
// request context.
func NewNamespaceDecoder(
	conf config.NamespaceConfiguration,
) namespacedriver.NamespaceDecoder {
	if len(conf.Allowlist) == 0 {
		return namespacedriver.StaticNamespaceDecoder(conf.Default)
	}

	return namespacedriver.RequestNamespaceDecoder(conf.Default)
}

// NewRequestNamespaceMiddleware returns the middleware validating the
// X-Namespace header against the allowlist, or nil when request-level
// namespace selection is disabled (empty allowlist), so callers can mount the
// result unconditionally.
func NewRequestNamespaceMiddleware(
	conf config.NamespaceConfiguration,
	logger *slog.Logger,
) (func(http.Handler) http.Handler, error) {
	if len(conf.Allowlist) == 0 {
		return nil, nil
	}

	middleware, err := namespacedriver.NewRequestNamespaceMiddleware(namespacedriver.RequestNamespaceMiddlewareConfig{
		Logger:           logger,
		DefaultNamespace: conf.Default,
		Allowlist:        conf.Allowlist,
	})
	if err != nil {
		return nil, fmt.Errorf("create request namespace middleware: %w", err)
	}

	return middleware.Handle, nil
}
