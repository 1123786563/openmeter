package aiusage

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/request"
	"github.com/openmeterio/openmeter/openmeter/aiusage/ratecard"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

// --- List ---

type (
	listRateCardEntriesRequest struct {
		Namespace    string
		ResourceCode string
		Provider     string
		Model        string
		ActiveOnly   bool
	}

	listRateCardEntriesResponse struct {
		Body rateCardEntryListResponse
	}

	ListRateCardEntriesHandler = httptransport.Handler[listRateCardEntriesRequest, listRateCardEntriesResponse]
)

func (h *handler) ListRateCardEntries() ListRateCardEntriesHandler {
	return httptransport.NewHandler(
		func(ctx context.Context, r *http.Request) (listRateCardEntriesRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return listRateCardEntriesRequest{}, err
			}
			return listRateCardEntriesRequest{
				Namespace:    ns,
				ResourceCode: r.URL.Query().Get("resource_code"),
				Provider:     r.URL.Query().Get("provider"),
				Model:        r.URL.Query().Get("model"),
				ActiveOnly:   r.URL.Query().Get("active_only") == "true",
			}, nil
		},
		func(ctx context.Context, req listRateCardEntriesRequest) (listRateCardEntriesResponse, error) {
			entries, err := h.rateCardService.List(ctx, ratecard.ListParams{
				Namespace:    req.Namespace,
				ResourceCode: req.ResourceCode,
				Provider:     req.Provider,
				Model:        req.Model,
				ActiveOnly:   req.ActiveOnly,
			})
			if err != nil {
				return listRateCardEntriesResponse{}, err
			}
			resp := rateCardEntryListResponse{Entries: make([]rateCardEntryResponse, 0, len(entries))}
			for i := range entries {
				resp.Entries = append(resp.Entries, rateCardEntryToResponse(&entries[i]))
			}
			return listRateCardEntriesResponse{Body: resp}, nil
		},
		func(_ context.Context, w http.ResponseWriter, _ *http.Request, resp listRateCardEntriesResponse) error {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			return json.NewEncoder(w).Encode(resp.Body)
		},
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("list-rate-card-entries"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}

// --- Create ---

type (
	createRateCardEntryRequest struct {
		Body      rateCardEntryCreate
		Namespace string
	}

	createRateCardEntryResponse struct {
		Body       rateCardEntryResponse
		StatusCode int
	}

	CreateRateCardEntryHandler = httptransport.Handler[createRateCardEntryRequest, createRateCardEntryResponse]
)

func (h *handler) CreateRateCardEntry() CreateRateCardEntryHandler {
	return httptransport.NewHandler(
		func(ctx context.Context, r *http.Request) (createRateCardEntryRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return createRateCardEntryRequest{}, err
			}
			var body rateCardEntryCreate
			if err := request.ParseBody(r, &body); err != nil {
				return createRateCardEntryRequest{}, err
			}
			return createRateCardEntryRequest{Body: body, Namespace: ns}, nil
		},
		func(ctx context.Context, req createRateCardEntryRequest) (createRateCardEntryResponse, error) {
			input := rateCardCreateToInput(req.Namespace, req.Body)
			entry, err := h.rateCardService.Create(ctx, input)
			if err != nil {
				return createRateCardEntryResponse{}, err
			}
			return createRateCardEntryResponse{
				Body:       rateCardEntryToResponse(entry),
				StatusCode: http.StatusCreated,
			}, nil
		},
		func(_ context.Context, w http.ResponseWriter, _ *http.Request, resp createRateCardEntryResponse) error {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(resp.StatusCode)
			return json.NewEncoder(w).Encode(resp.Body)
		},
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("create-rate-card-entry"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}

// --- Get ---

type (
	getRateCardEntryRequest struct {
		ID        string
		Namespace string
	}

	getRateCardEntryResponse struct {
		Body rateCardEntryResponse
	}

	GetRateCardEntryHandler = httptransport.Handler[getRateCardEntryRequest, getRateCardEntryResponse]
)

func (h *handler) GetRateCardEntry() GetRateCardEntryHandler {
	return httptransport.NewHandler(
		func(ctx context.Context, r *http.Request) (getRateCardEntryRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return getRateCardEntryRequest{}, err
			}
			id := r.PathValue("id")
			if id == "" {
				id = r.URL.Query().Get("id")
			}
			return getRateCardEntryRequest{ID: id, Namespace: ns}, nil
		},
		func(ctx context.Context, req getRateCardEntryRequest) (getRateCardEntryResponse, error) {
			entry, err := h.rateCardService.Get(ctx, req.Namespace, req.ID)
			if err != nil {
				return getRateCardEntryResponse{}, err
			}
			if entry == nil {
				return getRateCardEntryResponse{}, apierrors.NewNotFoundError(ctx, nil, "rate card entry")
			}
			return getRateCardEntryResponse{Body: rateCardEntryToResponse(entry)}, nil
		},
		func(_ context.Context, w http.ResponseWriter, _ *http.Request, resp getRateCardEntryResponse) error {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			return json.NewEncoder(w).Encode(resp.Body)
		},
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("get-rate-card-entry"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}

// --- Update ---

type (
	updateRateCardEntryRequest struct {
		ID        string
		Body      rateCardEntryUpdate
		Namespace string
	}

	updateRateCardEntryResponse struct {
		Body rateCardEntryResponse
	}

	UpdateRateCardEntryHandler = httptransport.Handler[updateRateCardEntryRequest, updateRateCardEntryResponse]
)

func (h *handler) UpdateRateCardEntry() UpdateRateCardEntryHandler {
	return httptransport.NewHandler(
		func(ctx context.Context, r *http.Request) (updateRateCardEntryRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return updateRateCardEntryRequest{}, err
			}
			var body rateCardEntryUpdate
			if err := request.ParseBody(r, &body); err != nil {
				return updateRateCardEntryRequest{}, err
			}
			id := r.PathValue("id")
			if id == "" {
				id = r.URL.Query().Get("id")
			}
			return updateRateCardEntryRequest{ID: id, Body: body, Namespace: ns}, nil
		},
		func(ctx context.Context, req updateRateCardEntryRequest) (updateRateCardEntryResponse, error) {
			input := rateCardCreateToInput(req.Namespace, req.Body)
			entry, err := h.rateCardService.Update(ctx, req.Namespace, req.ID, input)
			if err != nil {
				return updateRateCardEntryResponse{}, err
			}
			return updateRateCardEntryResponse{Body: rateCardEntryToResponse(entry)}, nil
		},
		func(_ context.Context, w http.ResponseWriter, _ *http.Request, resp updateRateCardEntryResponse) error {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			return json.NewEncoder(w).Encode(resp.Body)
		},
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("update-rate-card-entry"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}

// --- Delete ---

type (
	deleteRateCardEntryRequest struct {
		ID        string
		Namespace string
	}

	deleteRateCardEntryResponse struct{}

	DeleteRateCardEntryHandler = httptransport.Handler[deleteRateCardEntryRequest, deleteRateCardEntryResponse]
)

func (h *handler) DeleteRateCardEntry() DeleteRateCardEntryHandler {
	return httptransport.NewHandler(
		func(ctx context.Context, r *http.Request) (deleteRateCardEntryRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return deleteRateCardEntryRequest{}, err
			}
			id := r.PathValue("id")
			if id == "" {
				id = r.URL.Query().Get("id")
			}
			return deleteRateCardEntryRequest{ID: id, Namespace: ns}, nil
		},
		func(ctx context.Context, req deleteRateCardEntryRequest) (deleteRateCardEntryResponse, error) {
			if err := h.rateCardService.Delete(ctx, req.Namespace, req.ID); err != nil {
				return deleteRateCardEntryResponse{}, err
			}
			return deleteRateCardEntryResponse{}, nil
		},
		func(_ context.Context, w http.ResponseWriter, _ *http.Request, _ deleteRateCardEntryResponse) error {
			w.WriteHeader(http.StatusNoContent)
			return nil
		},
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("delete-rate-card-entry"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
