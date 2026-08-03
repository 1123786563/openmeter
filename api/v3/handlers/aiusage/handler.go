package aiusage

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/aiusage"
	"github.com/openmeterio/openmeter/openmeter/aiusage/runtimeauthorization"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

// Handler exposes the AI Usage operations defined by the generated
// api.ServerInterface.
type Handler interface {
	// Batch operations
	CreateAiUsageBatch() CreateAiUsageBatchHandler
	GetAiUsageBatch() GetAiUsageBatchHandler

	// Runtime authorization
	GetCustomerRuntimeAuthorization() GetCustomerRuntimeAuthorizationHandler

	// Credit balance / transactions
	GetAiUsageCreditBalance() GetAiUsageCreditBalanceHandler
	ListAiUsageCreditTransactions() ListAiUsageCreditTransactionsHandler
}

type handler struct {
	resolveNamespace            func(ctx context.Context) (string, error)
	service                     aiusage.Service
	runtimeAuthorizationService runtimeauthorization.Service
	creditBalanceReader         CreditBalanceReader
	options                     []httptransport.HandlerOption
}

// New creates a Handler from the AI Usage service and optional runtime
// authorization service. When runtimeAuthorizationService is nil the runtime
// authorization operation returns 501.
func New(
	resolveNamespace func(ctx context.Context) (string, error),
	service aiusage.Service,
	runtimeAuthorizationService runtimeauthorization.Service,
	creditBalanceReader CreditBalanceReader,
	options ...httptransport.HandlerOption,
) Handler {
	return &handler{
		resolveNamespace:            resolveNamespace,
		service:                     service,
		runtimeAuthorizationService: runtimeAuthorizationService,
		creditBalanceReader:         creditBalanceReader,
		options:                     options,
	}
}
