package namespaces

import (
	"context"
	"net/http"

	v3 "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type (
	ListNamespacesRequest  = struct{}
	ListNamespacesResponse = v3.NamespaceList
	ListNamespacesHandler  = httptransport.Handler[ListNamespacesRequest, ListNamespacesResponse]
)

func (h *handler) ListNamespaces() ListNamespacesHandler {
	return httptransport.NewHandler(
		func(ctx context.Context, r *http.Request) (ListNamespacesRequest, error) {
			return ListNamespacesRequest{}, nil
		},
		func(ctx context.Context, _ ListNamespacesRequest) (ListNamespacesResponse, error) {
			// The namespaces listing is derived from the server configuration:
			// the default namespace plus the allowlist, deduplicated and sorted
			// (see config.NamespaceConfiguration.Namespaces).
			return ListNamespacesResponse{
				Default:    h.namespaceConfig.Default,
				Namespaces: h.namespaceConfig.Namespaces(),
			}, nil
		},
		commonhttp.JSONResponseEncoderWithStatus[ListNamespacesResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("list-namespaces"),
		)...,
	)
}
