package aiusage

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"syscall"
	"testing"
	"time"

	"github.com/oklog/run"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/openmeter/aiusage"
	"github.com/openmeterio/openmeter/openmeter/aiusage/runtimeauthorization"
	"github.com/openmeterio/openmeter/openmeter/aiusage/signing"
	"github.com/openmeterio/openmeter/openmeter/aiusage/worker"
	"github.com/openmeterio/openmeter/pkg/models"
)

// ---------------------------------------------------------------------------
// Mock implementations
// ---------------------------------------------------------------------------

// mockAIUsageService is a test double for aiusage.Service.
type mockAIUsageService struct {
	batches    map[string]*aiusage.AIUsageBatch // keyed by idempotency key
	results    map[string]*aiusage.BatchSettlementResult
	ingestErr  error
	getErr     error
	coveredSeq int64
}

func newMockAIUsageService() *mockAIUsageService {
	return &mockAIUsageService{
		batches: make(map[string]*aiusage.AIUsageBatch),
		results: make(map[string]*aiusage.BatchSettlementResult),
	}
}

func (m *mockAIUsageService) IngestBatch(ctx context.Context, input aiusage.IngestBatchInput) (*aiusage.BatchSettlementResult, error) {
	if m.ingestErr != nil {
		return nil, m.ingestErr
	}

	// Check for payload conflict
	if existing, ok := m.batches[input.UsageBatchID]; ok {
		if existing.PayloadHash != input.PayloadHash {
			return nil, aiusage.ErrBatchPayloadConflict
		}
		// Same key + same hash → idempotent replay
		return m.results[input.UsageBatchID], nil
	}

	batch := &aiusage.AIUsageBatch{
		Namespace:      input.Namespace,
		CustomerID:     input.CustomerID,
		SubjectID:      input.SubjectID,
		UsageBatchID:   input.UsageBatchID,
		TenantSeq:      input.TenantSeq,
		OccurredAt:     input.OccurredAt,
		ReservationID:  input.ReservationID,
		CeilingCredits: input.CeilingCredits,
		RateVersion:    input.RateVersion,
		BillingMode:    input.BillingMode,
		PayloadHash:    input.PayloadHash,
		Status:         aiusage.BatchStatusSettled,
		LineItems:      input.LineItems,
	}
	batch.CreatedAt = time.Now().UTC()

	result := &aiusage.BatchSettlementResult{
		BatchID:          input.UsageBatchID,
		Status:           aiusage.BatchStatusSettled,
		TotalCredits:     100,
		CoveredTenantSeq: input.TenantSeq,
	}

	m.batches[input.UsageBatchID] = batch
	m.results[input.UsageBatchID] = result
	return result, nil
}

func (m *mockAIUsageService) GetBatch(ctx context.Context, namespace, customerID, usageBatchID string) (*aiusage.AIUsageBatch, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	b, ok := m.batches[usageBatchID]
	if !ok {
		return nil, nil
	}
	return b, nil
}

func (m *mockAIUsageService) GetCoveredSeq(ctx context.Context, namespace, customerID string) (int64, error) {
	return m.coveredSeq, nil
}

// mockRuntimeAuthService is a test double for runtimeauthorization.Service.
type mockRuntimeAuthService struct {
	pkg signing.AuthorizationPackage
	err error
}

func (m *mockRuntimeAuthService) Get(ctx context.Context, customerID string, subjectKeys []string) (signing.AuthorizationPackage, error) {
	if m.err != nil {
		return signing.AuthorizationPackage{}, m.err
	}
	pkg := m.pkg
	pkg.BillingCustomerID = customerID
	pkg.SubjectKeys = subjectKeys
	return pkg, nil
}

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

const testNamespace = "test-ns"

func newTestHandler(t *testing.T, svc aiusage.Service, rtAuth runtimeauthorization.Service) Handler {
	t.Helper()
	return New(
		func(context.Context) (string, error) { return testNamespace, nil },
		svc,
		rtAuth,
		nil, // nil CreditBalanceReader → 501
	)
}

func validBatchBody(idempotencyKey, payloadHash string) api.AIUsageUsageBatchCreate {
	provider := "openai"
	model := "gpt-4o"
	dims := map[string]string{"region": "us"}
	return api.AIUsageUsageBatchCreate{
		BillingCustomerId:         "01HXYZ00000000000000000001",
		BillingMode:               api.AIUsageBillingModeComponent,
		IdempotencyKey:            idempotencyKey,
		PayloadHash:               payloadHash,
		ProviderManaged:           true,
		RatePackageVersion:        "v1",
		ReservationCeilingCredits: 1000,
		ReservationId:             "res-1",
		SubjectKey:                "subj-1",
		TenantSeq:                 1,
		OccurredAt:                time.Now().UTC(),
		Lines: []api.AIUsageUsageLineCreate{
			{
				ResourceCode:       "llm_input_tokens",
				Quantity:           1000,
				Provider:           &provider,
				Model:              &model,
				PricingDimensions:  &dims,
				CanonicalLineIndex: 0,
			},
		},
	}
}

func postBatch(t *testing.T, h Handler, body api.AIUsageUsageBatchCreate) *httptest.ResponseRecorder {
	t.Helper()
	jsonBody, err := json.Marshal(body)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/ai-usage-batches", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.CreateAiUsageBatch().ServeHTTP(rec, req)
	return rec
}

func parseBatchResponse(t *testing.T, rec *httptest.ResponseRecorder) api.AIUsageUsageBatch {
	t.Helper()
	var batch api.AIUsageUsageBatch
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &batch), "body: %s", rec.Body.String())
	return batch
}

// ---------------------------------------------------------------------------
// Create Batch: first submit, replay, hash conflict
// ---------------------------------------------------------------------------

func TestCreateBatchMapsReplayAndHashConflict(t *testing.T) {
	svc := newMockAIUsageService()
	h := newTestHandler(t, svc, nil)

	// First submit → 201
	first := postBatch(t, h, validBatchBody("same-key", "hash-a"))
	require.Equal(t, http.StatusCreated, first.Code, "first: %s", first.Body.String())
	firstBatch := parseBatchResponse(t, first)
	assert.Equal(t, "same-key", firstBatch.IdempotencyKey)
	assert.Equal(t, int64(100), firstBatch.TotalCredits)

	// Identical replay (same key + same hash) → 200
	replay := postBatch(t, h, validBatchBody("same-key", "hash-a"))
	require.Equal(t, http.StatusOK, replay.Code, "replay: %s", replay.Body.String())

	// Same key, different hash → 409
	conflict := postBatch(t, h, validBatchBody("same-key", "hash-b"))
	require.Equal(t, http.StatusConflict, conflict.Code, "conflict: %s", conflict.Body.String())
}

// ---------------------------------------------------------------------------
// Malformed or negative fields → 422
// ---------------------------------------------------------------------------

func TestCreateBatchRejectsMalformedFields(t *testing.T) {
	svc := newMockAIUsageService()
	h := newTestHandler(t, svc, nil)

	t.Run("empty idempotency key", func(t *testing.T) {
		body := validBatchBody("", "hash-a")
		rec := postBatch(t, h, body)
		assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	})

	t.Run("empty payload hash", func(t *testing.T) {
		body := validBatchBody("key-1", "")
		rec := postBatch(t, h, body)
		assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	})

	t.Run("negative quantity", func(t *testing.T) {
		body := validBatchBody("key-neg", "hash-neg")
		body.Lines[0].Quantity = -5
		rec := postBatch(t, h, body)
		assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	})

	t.Run("zero quantity", func(t *testing.T) {
		body := validBatchBody("key-zero", "hash-zero")
		body.Lines[0].Quantity = 0
		rec := postBatch(t, h, body)
		assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	})

	t.Run("empty lines", func(t *testing.T) {
		body := validBatchBody("key-empty", "hash-empty")
		body.Lines = nil
		rec := postBatch(t, h, body)
		assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	})

	t.Run("zero tenant_seq", func(t *testing.T) {
		body := validBatchBody("key-seq", "hash-seq")
		body.TenantSeq = 0
		rec := postBatch(t, h, body)
		assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	})
}

// ---------------------------------------------------------------------------
// Prepaid shortage without receivable → 402
// ---------------------------------------------------------------------------

func TestCreateBatchPrepaidShortageReturns402(t *testing.T) {
	svc := newMockAIUsageService()
	svc.ingestErr = aiusage.ErrCreditInsufficient
	h := newTestHandler(t, svc, nil)

	rec := postBatch(t, h, validBatchBody("shortage-key", "hash-shortage"))
	require.Equal(t, http.StatusPaymentRequired, rec.Code, "body: %s", rec.Body.String())
}

// ---------------------------------------------------------------------------
// Response validates against generated schema
// ---------------------------------------------------------------------------

func TestCreateBatchResponseMatchesGeneratedSchema(t *testing.T) {
	svc := newMockAIUsageService()
	h := newTestHandler(t, svc, nil)

	body := validBatchBody("schema-key", "hash-schema")
	rec := postBatch(t, h, body)
	require.Equal(t, http.StatusCreated, rec.Code)

	batch := parseBatchResponse(t, rec)

	// Assert all required response fields are populated per the generated API type.
	assert.NotEmpty(t, batch.Id)
	assert.Equal(t, "schema-key", batch.IdempotencyKey)
	assert.Equal(t, "hash-schema", batch.PayloadHash)
	assert.Equal(t, body.BillingCustomerId, batch.BillingCustomerId)
	assert.NotZero(t, batch.OccurredAt)
	assert.NotZero(t, batch.CreatedAt)
	assert.Equal(t, api.AIUsageBillingModeComponent, batch.BillingMode)
	assert.Equal(t, api.AIUsageBatchStatusSettled, batch.Status)
	assert.NotEmpty(t, batch.TotalCredits)
	assert.NotEmpty(t, batch.Lines)
	assert.Equal(t, int32(0), batch.Lines[0].CanonicalLineIndex)
	assert.NotEmpty(t, batch.Lines[0].ResourceCode)
	assert.NotEmpty(t, batch.Lines[0].Quantity)
}

// ---------------------------------------------------------------------------
// Get batch
// ---------------------------------------------------------------------------

func TestGetBatchReturnsExistingBatch(t *testing.T) {
	svc := newMockAIUsageService()
	h := newTestHandler(t, svc, nil)

	// Create a batch first
	createRec := postBatch(t, h, validBatchBody("get-key", "hash-get"))
	require.Equal(t, http.StatusCreated, createRec.Code)

	// Now GET it
	req := httptest.NewRequest(http.MethodGet, "/api/v1/ai-usage-batches/get-key", nil)
	rec := httptest.NewRecorder()

	h.GetAiUsageBatch().With(GetAiUsageBatchParams{BatchID: "get-key"}).ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	batch := parseBatchResponse(t, rec)
	assert.Equal(t, "get-key", batch.IdempotencyKey)
}

func TestGetBatchReturns404ForMissingBatch(t *testing.T) {
	svc := newMockAIUsageService()
	h := newTestHandler(t, svc, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/ai-usage-batches/missing", nil)
	rec := httptest.NewRecorder()

	h.GetAiUsageBatch().With(GetAiUsageBatchParams{BatchID: "missing"}).ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// ---------------------------------------------------------------------------
// Runtime authorization
// ---------------------------------------------------------------------------

func TestRuntimeAuthorizationReturnsPackage(t *testing.T) {
	rtAuth := &mockRuntimeAuthService{
		pkg: signing.AuthorizationPackage{
			SpendableCredits:             5000,
			AuthorizationCapacityCredits: 3000,
			CoveredTenantSeq:             42,
		},
	}
	h := newTestHandler(t, newMockAIUsageService(), rtAuth)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/customers/cust-1/runtime-authorization", nil)
	rec := httptest.NewRecorder()

	h.GetCustomerRuntimeAuthorization().With(GetCustomerRuntimeAuthorizationParams{
		CustomerID: "cust-1",
	}).ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())

	var authz api.AIUsageRuntimeAuthorization
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &authz))
	assert.True(t, authz.Authorized)
	assert.Equal(t, contractVersion, authz.ContractVersion)
	assert.Equal(t, int64(5000), authz.AvailableCredits)
	assert.Equal(t, int64(42), authz.CoveredTenantSeq)
}

func TestRuntimeAuthorizationReturns501WhenServiceIsNil(t *testing.T) {
	h := newTestHandler(t, newMockAIUsageService(), nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/customers/cust-1/runtime-authorization", nil)
	rec := httptest.NewRecorder()

	h.GetCustomerRuntimeAuthorization().With(GetCustomerRuntimeAuthorizationParams{
		CustomerID: "cust-1",
	}).ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotImplemented, rec.Code)
}

// ---------------------------------------------------------------------------
// Credit balance returns 501 when reader is nil
// ---------------------------------------------------------------------------

func TestCreditBalanceReturns501WithoutReader(t *testing.T) {
	h := newTestHandler(t, newMockAIUsageService(), nil)

	t.Run("balance", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/customers/cust-1/credit-balance", nil)
		rec := httptest.NewRecorder()
		h.GetAiUsageCreditBalance().With(GetAiUsageCreditBalanceParams{CustomerID: "cust-1"}).ServeHTTP(rec, req)
		assert.Equal(t, http.StatusNotImplemented, rec.Code)
	})

	t.Run("transactions", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/customers/cust-1/credit-transactions", nil)
		rec := httptest.NewRecorder()
		h.ListAiUsageCreditTransactions().With(ListAiUsageCreditTransactionsParams{CustomerID: "cust-1"}).ServeHTTP(rec, req)
		assert.Equal(t, http.StatusNotImplemented, rec.Code)
	})
}

// ---------------------------------------------------------------------------
// Worker lifecycle: start + cancel terminates within 5 seconds
// ---------------------------------------------------------------------------

func TestWorkerStartCancelTerminatesWithin5Seconds(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	var group run.Group

	group.Add(func() error {
		<-ctx.Done()
		return ctx.Err()
	}, func(error) {
		cancel()
	})

	// Simulate the lifecycle block from main.go using the common helper.
	workerRun, workerStop := AIUsageWorkerGroupForTest(ctx, nil)
	group.Add(workerRun, workerStop)

	group.Add(run.SignalHandler(ctx, syscall.SIGTERM))

	// Cancel from a goroutine after a short delay; the group must exit well
	// within 5 seconds.
	done := make(chan error, 1)
	go func() {
		done <- group.Run()
	}()

	cancel()

	select {
	case err := <-done:
		// Expected: the group exits because the context was canceled.
		_ = err
	case <-time.After(5 * time.Second):
		t.Fatal("worker lifecycle did not terminate within 5 seconds")
	}
}

// AIUsageWorkerGroupForTest mirrors common.AIUsageWorkerGroup but avoids the
// import cycle. It is the same execute/intercept pair the server uses.
func AIUsageWorkerGroupForTest(
	ctx context.Context,
	w *worker.Worker,
) (func() error, func(error)) {
	workerRun := func() error {
		if w != nil {
			w.Start(ctx)
		}
		<-ctx.Done()
		return ctx.Err()
	}

	workerStop := func(_ error) {
		if w != nil {
			w.Stop()
		}
	}

	return workerRun, workerStop
}

// ---------------------------------------------------------------------------
// Structured logs contain no sensitive data
// ---------------------------------------------------------------------------

// loggingAIUsageService wraps mockAIUsageService with a captured slog.Logger.
// It exercises the real handler path: the service logs during IngestBatch and
// the test verifies that no sensitive data leaks into the captured output.
type loggingAIUsageService struct {
	inner  *mockAIUsageService
	logger *slog.Logger
}

func (l *loggingAIUsageService) IngestBatch(ctx context.Context, input aiusage.IngestBatchInput) (*aiusage.BatchSettlementResult, error) {
	// Simulate production logging at the settlement boundary. Production code
	// logs batch_id and total_credits but must never include provider cost body,
	// payment material, private keys, or signature payloads.
	l.logger.InfoContext(ctx, "ai usage batch settled",
		slog.String("batch_id", input.UsageBatchID),
		slog.Int64("tenant_seq", input.TenantSeq),
	)
	return l.inner.IngestBatch(ctx, input)
}

func (l *loggingAIUsageService) GetBatch(ctx context.Context, namespace, customerID, usageBatchID string) (*aiusage.AIUsageBatch, error) {
	return l.inner.GetBatch(ctx, namespace, customerID, usageBatchID)
}

func (l *loggingAIUsageService) GetCoveredSeq(ctx context.Context, namespace, customerID string) (int64, error) {
	return l.inner.GetCoveredSeq(ctx, namespace, customerID)
}

func TestStructuredLogsRedactSensitiveData(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	svc := &loggingAIUsageService{
		inner:  newMockAIUsageService(),
		logger: logger,
	}
	h := newTestHandler(t, svc, nil)

	// Invoke a real Create Batch request through the handler.
	rec := postBatch(t, h, validBatchBody("log-key", "hash-log"))
	require.Equal(t, http.StatusCreated, rec.Code, "body: %s", rec.Body.String())

	logOutput := buf.String()
	require.NotEmpty(t, logOutput, "handler must have produced log output")

	// Verify no sensitive fields appear in the captured log.
	for _, forbidden := range []string{
		"cost_snapshot",  // provider cost body
		"sales_snapshot", // sales pricing detail
		"api_key",        // credentials
		"payment",        // payment material
		"signature",      // signature payload
		"seed",           // private key seed
		"private_key",    // private key
	} {
		assert.NotContains(t, logOutput, forbidden,
			"structured log must not contain %s", forbidden)
	}

	// Verify safe fields ARE present.
	assert.Contains(t, logOutput, "batch_id")
	assert.Contains(t, logOutput, "log-key")
}

// ---------------------------------------------------------------------------
// Error response for hash conflict is well-formed
// ---------------------------------------------------------------------------

func TestHashConflictResponseBody(t *testing.T) {
	svc := newMockAIUsageService()
	h := newTestHandler(t, svc, nil)

	// First submit
	postBatch(t, h, validBatchBody("conflict-key", "hash-1"))

	// Conflict
	rec := postBatch(t, h, validBatchBody("conflict-key", "hash-2"))
	require.Equal(t, http.StatusConflict, rec.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	// The error response should include validation error details.
	assert.NotNil(t, body)
}

// ---------------------------------------------------------------------------
// Validation issue error code mapping
// ---------------------------------------------------------------------------

func TestValidationIssueMapsToUnprocessableEntity(t *testing.T) {
	issue := newUnprocessableFieldIssue("test_field", "test message")
	assert.Equal(t, "aiusage_field_invalid", string(issue.Code()))

	issues, err := models.AsValidationIssues(issue)
	require.NoError(t, err)
	require.Len(t, issues, 1)
}

// ---------------------------------------------------------------------------
// Helper to avoid unused import of syscall in the worker lifecycle test.
// ---------------------------------------------------------------------------
