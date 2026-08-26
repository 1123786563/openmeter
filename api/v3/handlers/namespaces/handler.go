package namespaces

import (
	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type Handler interface {
	ListNamespaces() ListNamespacesHandler
}

type handler struct {
	namespaceConfig config.NamespaceConfiguration
	options         []httptransport.HandlerOption
}

// New returns a handler serving the deployment's namespace configuration.
// Unlike other handlers it resolves no service: the default namespace and the
// allowlist are static server configuration.
func New(
	namespaceConfig config.NamespaceConfiguration,
	options ...httptransport.HandlerOption,
) Handler {
	return &handler{
		namespaceConfig: namespaceConfig,
		options:         options,
	}
}
