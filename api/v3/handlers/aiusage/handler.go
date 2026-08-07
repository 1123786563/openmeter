package aiusage

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/aiusage"
	"github.com/openmeterio/openmeter/openmeter/aiusage/runtimeauthorization"
	"github.com/openmeterio/openmeter/openmeter/aiusage/ratecard"
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

	// Rate card management
	CreateRateCardEntry() CreateRateCardEntryHandler
	GetRateCardEntry() GetRateCardEntryHandler
	ListRateCardEntries() ListRateCardEntriesHandler
	UpdateRateCardEntry() UpdateRateCardEntryHandler
	DeleteRateCardEntry() DeleteRateCardEntryHandler
}

type handler struct {
	resolveNamespace            func(ctx context.Context) (string, error)
	service                     aiusage.Service
	runtimeAuthorizationService runtimeauthorization.Service
	creditBalanceReader         CreditBalanceReader
	rateCardService             ratecard.Service
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
	rateCardService ratecard.Service,
	options ...httptransport.HandlerOption,
) Handler {
	return &handler{
		resolveNamespace:            resolveNamespace,
		service:                     service,
		runtimeAuthorizationService: runtimeAuthorizationService,
		creditBalanceReader:         creditBalanceReader,
		rateCardService:             rateCardService,
		options:                     options,
	}
}
