package aiusage

import (
	"context"
	"net/http"
	"time"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
)

// --- Get batch ---

type (
	GetAiUsageBatchParams struct {
		BatchID string
		CustomerID string
	}

	GetAiUsageBatchRequest struct {
		Namespace string
		CustomerID string
		BatchID   string
	}

	GetAiUsageBatchResponse = api.AIUsageUsageBatch

	GetAiUsageBatchHandler = httptransport.HandlerWithArgs[GetAiUsageBatchRequest, GetAiUsageBatchResponse, GetAiUsageBatchParams]
)

// GetAiUsageBatch implements GET /ai-usage-batches/{batchId}.
func (h *handler) GetAiUsageBatch() GetAiUsageBatchHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params GetAiUsageBatchParams) (GetAiUsageBatchRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return GetAiUsageBatchRequest{}, err
			}
			return GetAiUsageBatchRequest{Namespace: ns, BatchID: params.BatchID}, nil
			return GetAiUsageBatchRequest{Namespace: ns, CustomerID: params.CustomerID, BatchID: params.BatchID}, nil
		},
		func(ctx context.Context, req GetAiUsageBatchRequest) (GetAiUsageBatchResponse, error) {
			batch, err := h.service.GetBatch(ctx, req.Namespace, req.CustomerID, req.BatchID)
			if err != nil {
				return GetAiUsageBatchResponse{}, err
			}
			if batch == nil {
				return GetAiUsageBatchResponse{}, models.NewGenericNotFoundError(nil)
			}
			return toAPIBatchFromDomain(batch), nil
		},
		commonhttp.JSONResponseEncoder[GetAiUsageBatchResponse],
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("get-ai-usage-batch"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}

// --- Credit balance ---

type (
	GetAiUsageCreditBalanceParams struct {
		CustomerID string
		Timestamp  *api.DateTime
	}

	GetAiUsageCreditBalanceRequest struct {
		Namespace  string
		CustomerID string
		At         time.Time
	}

	GetAiUsageCreditBalanceResponse = api.AIUsageCreditBalance

	GetAiUsageCreditBalanceHandler = httptransport.HandlerWithArgs[GetAiUsageCreditBalanceRequest, GetAiUsageCreditBalanceResponse, GetAiUsageCreditBalanceParams]
)

// GetAiUsageCreditBalance implements GET /customers/{customerId}/credit-balance.
func (h *handler) GetAiUsageCreditBalance() GetAiUsageCreditBalanceHandler {
	reader := h.creditBalanceReader
	if reader == nil {
		reader = noopCreditBalanceReader{}
	}

	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params GetAiUsageCreditBalanceParams) (GetAiUsageCreditBalanceRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return GetAiUsageCreditBalanceRequest{}, err
			}

			at := time.Now()
			if params.Timestamp != nil {
				at = time.Time(*params.Timestamp)
			}

			return GetAiUsageCreditBalanceRequest{Namespace: ns, CustomerID: params.CustomerID, At: at}, nil
		},
		func(ctx context.Context, req GetAiUsageCreditBalanceRequest) (GetAiUsageCreditBalanceResponse, error) {
			view, err := reader.ReadBalance(ctx, req.Namespace, req.CustomerID, req.At)
			if err != nil {
				return GetAiUsageCreditBalanceResponse{}, err
			}

			return api.AIUsageCreditBalance{
				RetrievedAt:      view.RetrievedAt,
				AvailableCredits: view.AvailableCredits,
				SettledCredits:   view.SettledCredits,
				PendingCredits:   view.PendingCredits,
			}, nil
		},
		commonhttp.JSONResponseEncoder[GetAiUsageCreditBalanceResponse],
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("get-ai-usage-credit-balance"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}

// --- Credit transactions ---

type (
	ListAiUsageCreditTransactionsParams struct {
		CustomerID string
		Page       *api.CursorPaginationQuery
	}

	ListAiUsageCreditTransactionsRequest struct {
		Namespace  string
		CustomerID string
		Page       Pagination
	}

	ListAiUsageCreditTransactionsResponse = api.AICreditTransactionPaginatedResponse

	ListAiUsageCreditTransactionsHandler = httptransport.HandlerWithArgs[ListAiUsageCreditTransactionsRequest, ListAiUsageCreditTransactionsResponse, ListAiUsageCreditTransactionsParams]
)

// ListAiUsageCreditTransactions implements GET /customers/{customerId}/credit-transactions.
func (h *handler) ListAiUsageCreditTransactions() ListAiUsageCreditTransactionsHandler {
	txnReader := h.creditBalanceReader
	if txnReader == nil {
		txnReader = noopCreditBalanceReader{}
	}

	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params ListAiUsageCreditTransactionsParams) (ListAiUsageCreditTransactionsRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return ListAiUsageCreditTransactionsRequest{}, err
			}

			page := Pagination{Size: 20}
			if params.Page != nil {
				page.After = params.Page.After
				page.Before = params.Page.Before
				if params.Page.Size != nil {
					page.Size = *params.Page.Size
				}
			}

			return ListAiUsageCreditTransactionsRequest{Namespace: ns, CustomerID: params.CustomerID, Page: page}, nil
		},
		func(ctx context.Context, req ListAiUsageCreditTransactionsRequest) (ListAiUsageCreditTransactionsResponse, error) {
			views, err := txnReader.ListTransactions(ctx, req.Namespace, req.CustomerID, req.Page)
			if err != nil {
				return ListAiUsageCreditTransactionsResponse{}, err
			}

			items := make([]api.AIUsageCreditTransaction, len(views))
			for i, v := range views {
				items[i] = api.AIUsageCreditTransaction{
					Id:                     v.ID,
					BookedAt:               v.BookedAt,
					Type:                   api.AIUsageCreditTransactionType(v.Type),
					Amount:                 v.Amount,
					AvailableBalanceBefore: v.AvailableBalanceBefore,
					AvailableBalanceAfter:  v.AvailableBalanceAfter,
				}
			}

			return api.AICreditTransactionPaginatedResponse{
				Data: items,
				Meta: api.CursorMeta{
					Page: api.CursorMetaPage{
						Size: float32(req.Page.Size),
					},
				},
			}, nil
		},
		commonhttp.JSONResponseEncoder[ListAiUsageCreditTransactionsResponse],
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("list-ai-usage-credit-transactions"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
