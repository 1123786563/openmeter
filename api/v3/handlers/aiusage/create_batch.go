package aiusage

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/request"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
)

type (
	// createAiUsageBatchRequest carries the API body plus the resolved
	// namespace from the decoder to the operation.
	createAiUsageBatchRequest struct {
		Body      api.AIUsageUsageBatchCreate
		Namespace string
	}

	// createAiUsageBatchResponse wraps the settled batch with the HTTP status
	// code that distinguishes first submit (201) from identical replay (200).
	createAiUsageBatchResponse struct {
		Batch      api.AIUsageUsageBatch
		StatusCode int
	}

	CreateAiUsageBatchHandler = httptransport.Handler[createAiUsageBatchRequest, createAiUsageBatchResponse]
)

// CreateAiUsageBatch implements the POST /ai-usage-batches operation.
//
// First submit returns HTTP 201; an identical replay (same idempotency_key and
// payload_hash) returns HTTP 200; a replay with the same key but a different
// payload_hash returns HTTP 409 (ErrBatchPayloadConflict from the service).
func (h *handler) CreateAiUsageBatch() CreateAiUsageBatchHandler {
	return httptransport.NewHandler(
		func(ctx context.Context, r *http.Request) (createAiUsageBatchRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return createAiUsageBatchRequest{}, err
			}

			var body api.AIUsageUsageBatchCreate
			if err := request.ParseBody(r, &body); err != nil {
				return createAiUsageBatchRequest{}, err
			}

			// Validate request — negative quantities and missing fields map
			// to 422 via the ValidationIssue HTTP status attribute.
			if vErr := validateCreateBatch(body); vErr != nil {
				return createAiUsageBatchRequest{}, vErr
			}

			return createAiUsageBatchRequest{Body: body, Namespace: ns}, nil
		},
		func(ctx context.Context, req createAiUsageBatchRequest) (createAiUsageBatchResponse, error) {
			input := fromAPIBatchCreate(req.Namespace, req.Body)

			// Pre-check: was this batch already submitted? Determines the
			// response status code (201 new vs 200 replay). The service's
			// IngestBatch is idempotent regardless, so this race is safe —
			// the worst case is a 201 when another request created the batch
			// concurrently; the data is still correct.
			preExisting, _ := h.service.GetBatch(ctx, req.Namespace, req.Body.BillingCustomerId, req.Body.IdempotencyKey)
			wasReplay := preExisting != nil && preExisting.PayloadHash == req.Body.PayloadHash

			result, err := h.service.IngestBatch(ctx, input)
			if err != nil {
				return createAiUsageBatchResponse{}, err
			}

			createdAt := time.Now().UTC()
			if preExisting != nil {
				createdAt = preExisting.CreatedAt
			}

			statusCode := http.StatusCreated
			if wasReplay {
				statusCode = http.StatusOK
			}

			return createAiUsageBatchResponse{
				Batch:      toAPIBatch(input, result, createdAt),
				StatusCode: statusCode,
			}, nil
		},
		createAiUsageBatchResponseEncoder,
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("create-ai-usage-batch"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}

// createAiUsageBatchResponseEncoder writes the settled batch JSON with the
// dynamic status code (201 or 200).
func createAiUsageBatchResponseEncoder(_ context.Context, w http.ResponseWriter, _ *http.Request, resp createAiUsageBatchResponse) error {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(resp.StatusCode)
	return json.NewEncoder(w).Encode(resp.Batch)
}

// validateCreateBatch checks the API request body for malformed or negative
// fields. Returns a ValidationIssue carrying HTTP 422 so the generic error
// encoder maps it correctly.
func validateCreateBatch(body api.AIUsageUsageBatchCreate) error {
	var issues []models.ValidationIssue

	if body.IdempotencyKey == "" {
		issues = append(issues, newUnprocessableFieldIssue("idempotency_key", "must not be empty"))
	}
	if body.PayloadHash == "" {
		issues = append(issues, newUnprocessableFieldIssue("payload_hash", "must not be empty"))
	}
	if body.BillingCustomerId == "" {
		issues = append(issues, newUnprocessableFieldIssue("billing_customer_id", "must not be empty"))
	}
	if body.SubjectKey == "" {
		issues = append(issues, newUnprocessableFieldIssue("subject_key", "must not be empty"))
	}
	if body.TenantSeq <= 0 {
		issues = append(issues, newUnprocessableFieldIssue("tenant_seq", "must be positive"))
	}
	if len(body.Lines) == 0 {
		issues = append(issues, newUnprocessableFieldIssue("lines", "must contain at least one line"))
	}

	for i, line := range body.Lines {
		if line.Quantity <= 0 {
			issues = append(issues, newUnprocessableFieldIssue(
				fmt.Sprintf("lines[%d].quantity", i), "must be positive"))
		}
		if line.ResourceCode == "" {
			issues = append(issues, newUnprocessableFieldIssue(
				fmt.Sprintf("lines[%d].resource_code", i), "must not be empty"))
		}
	}

	if len(issues) == 0 {
		return nil
	}

	return models.NewGenericValidationError(issues[0])
}

// newUnprocessableFieldIssue creates a validation issue mapped to HTTP 422.
func newUnprocessableFieldIssue(field, msg string) models.ValidationIssue {
	return models.NewValidationIssue(
		"aiusage_field_invalid",
		msg,
		models.WithFieldString(field),
		models.WithCriticalSeverity(),
		commonhttp.WithHTTPStatusCodeAttribute(http.StatusUnprocessableEntity),
	)
}
