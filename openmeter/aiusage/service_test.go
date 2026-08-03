package aiusage

import (
	"context"
	"testing"
	"time"

	"log/slog"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"

)

// --- Mock implementations for service-level testing ---

type mockRateCardResolver struct {
	entries map[string]CustomerRateCardEntry
	err     error
}

func (m *mockRateCardResolver) Resolve(_ context.Context, namespace, customerID string, resource ResourceCode, provider, model string, _ time.Time) (CustomerRateCardEntry, error) {
	if m.err != nil {
		return CustomerRateCardEntry{}, m.err
	}
	// Try customer-specific first.
	key := namespace + ":" + customerID + ":" + string(resource)
	if entry, ok := m.entries[key]; ok {
		return entry, nil
	}
	// Fall back to namespace default.
	key = namespace + "::" + string(resource)
	if entry, ok := m.entries[key]; ok {
		return entry, nil
	}
	return CustomerRateCardEntry{}, ErrMissingRateCard
}

type mockCostResolver struct{}

func (m *mockCostResolver) ResolveCost(_ context.Context, _, provider, _ string, _ ResourceCode, quantity int64) (CostSnapshot, error) {
	// Simulate a cost of $0.000002 per token.
	cost := alpacadecimal.NewFromFloat(0.000002).Mul(alpacadecimal.NewFromInt(quantity))
	return CostSnapshot{
		Currency: "USD",
		Amount:   cost,
		Source:   "mock",
	}, nil
}

type mockRepo struct {
	batches  map[string]*AIUsageBatch
	results  map[string]*BatchSettlementResult
	coveredSeq int64
}

func newMockRepo() *mockRepo {
	return &mockRepo{
		batches: make(map[string]*AIUsageBatch),
		results: make(map[string]*BatchSettlementResult),
	}
}

func (r *mockRepo) CreateBatch(_ context.Context, batch AIUsageBatch, snapshots []RatingSnapshot) (*BatchSettlementResult, error) {
	key := batch.Namespace + ":" + batch.UsageBatchID
	if existing, ok := r.batches[key]; ok {
		if existing.PayloadHash == batch.PayloadHash {
			return r.results[key], nil
		}
		return nil, ErrBatchPayloadConflict
	}

	// Calculate total credits from snapshots or ceiling for bundle.
	var totalCredits int64
	for _, s := range snapshots {
		totalCredits += s.Credits
	}
	if batch.BillingMode == BillingModeBundle && batch.CeilingCredits != nil {
		totalCredits = *batch.CeilingCredits
	}

	r.batches[key] = &batch
	result := &BatchSettlementResult{
		BatchID:          batch.UsageBatchID,
		Status:           batch.Status,
		TotalCredits:     totalCredits,
		CoveredTenantSeq: batch.TenantSeq,
	}
	r.results[key] = result
	if batch.Status == BatchStatusSettled && batch.TenantSeq > r.coveredSeq {
		r.coveredSeq = batch.TenantSeq
	}
	return result, nil
}

func (r *mockRepo) GetBatchByBatchID(_ context.Context, namespace, usageBatchID string) (*AIUsageBatch, error) {
	return r.batches[namespace+":"+usageBatchID], nil
}

func (r *mockRepo) GetBatchResult(_ context.Context, namespace, usageBatchID string) (*BatchSettlementResult, error) {
	return r.results[namespace+":"+usageBatchID], nil
}

func (r *mockRepo) GetCoveredSeq(_ context.Context, _, _ string) (int64, error) {
	return r.coveredSeq, nil
}

func newTestService(repo Repository, resolver RateCardResolver) Service {
	return NewService(ServiceConfig{
		Repo:             repo,
		RateCardResolver: resolver,
		CostResolver:     &mockCostResolver{},
		SettlementEngine: NewSettlementEngine(SettlementEngineConfig{
			GrantReader: &mockGrantReader{grants: []SettlementGrant{
				{GrantID: "plan-1", Amount: 10000, Priority: 0, Source: "plan"},
			}},
			Ledger: &mockLedgerRecorder{},
			Logger: slog.Default(),
			Tracer: noop.Tracer{},
		}),
		Tracer: noop.Tracer{},
	})
}

// --- Service-level tests ---

func TestService_IngestBatch_HappyPath(t *testing.T) {
	repo := newMockRepo()
	resolver := &mockRateCardResolver{
		entries: map[string]CustomerRateCardEntry{
			"ns-1:cust-1:chat_input_token": {
				PricePerUnitCNY: alpacadecimal.NewFromFloat(0.000002),
				CreditRate:      1000,
			},
		},
	}
	svc := newTestService(repo, resolver)

	input := IngestBatchInput{
		Namespace:    "ns-1",
		CustomerID:   "cust-1",
		SubjectID:    "subj-1",
		UsageBatchID: "batch-001",
		TenantSeq:    1,
		OccurredAt:   time.Now(),
		RateVersion:  "v1",
		BillingMode:  BillingModeComponent,
		PayloadHash:  "hash1",
		LineItems: []UsageLineItem{
			{ResourceCode: ResourceChatInputToken, Quantity: 1000, Provider: "openai", Model: "gpt-4", ProviderManaged: true},
		},
	}

	result, err := svc.IngestBatch(t.Context(), input)
	require.NoError(t, err)
	require.Equal(t, BatchStatusSettled, result.Status)
	require.Equal(t, int64(2), result.TotalCredits) // ceil(1000 * 0.000002 * 1000) = ceil(2.0) = 2
}

func TestService_IngestBatch_Idempotency(t *testing.T) {
	repo := newMockRepo()
	resolver := &mockRateCardResolver{
		entries: map[string]CustomerRateCardEntry{
			"ns-1:cust-1:chat_input_token": {
				PricePerUnitCNY: alpacadecimal.NewFromFloat(0.000002),
				CreditRate:      1000,
			},
		},
	}
	svc := newTestService(repo, resolver)

	input := IngestBatchInput{
		Namespace:    "ns-1",
		CustomerID:   "cust-1",
		SubjectID:    "subj-1",
		UsageBatchID: "batch-001",
		TenantSeq:    1,
		PayloadHash:  "hash1",
		BillingMode:  BillingModeComponent,
		LineItems:    []UsageLineItem{{ResourceCode: ResourceChatInputToken, Quantity: 1000, Provider: "openai", Model: "gpt-4", ProviderManaged: true}},
	}

	// First call.
	result1, err := svc.IngestBatch(t.Context(), input)
	require.NoError(t, err)

	// Second call with same batch_id and hash should return same result.
	result2, err := svc.IngestBatch(t.Context(), input)
	require.NoError(t, err)
	require.Equal(t, result1.TotalCredits, result2.TotalCredits)
}

func TestService_IngestBatch_Conflict(t *testing.T) {
	repo := newMockRepo()
	resolver := &mockRateCardResolver{
		entries: map[string]CustomerRateCardEntry{
			"ns-1:cust-1:chat_input_token": {
				PricePerUnitCNY: alpacadecimal.NewFromFloat(0.000002),
				CreditRate:      1000,
			},
		},
	}
	svc := newTestService(repo, resolver)

	input1 := IngestBatchInput{
		Namespace:    "ns-1",
		CustomerID:   "cust-1",
		SubjectID:    "subj-1",
		UsageBatchID: "batch-001",
		TenantSeq:    1,
		PayloadHash:  "hash1",
		BillingMode:  BillingModeComponent,
		LineItems:    []UsageLineItem{{ResourceCode: ResourceChatInputToken, Quantity: 1000, Provider: "openai", Model: "gpt-4", ProviderManaged: true}},
	}

	_, err := svc.IngestBatch(t.Context(), input1)
	require.NoError(t, err)

	// Same batch_id, different hash -> conflict.
	input2 := input1
	input2.PayloadHash = "different-hash"
	_, err = svc.IngestBatch(t.Context(), input2)
	require.Error(t, err)
}

func TestService_IngestBatch_BYOK(t *testing.T) {
	repo := newMockRepo()
	resolver := &mockRateCardResolver{
		entries: map[string]CustomerRateCardEntry{
			"ns-1:cust-1:rag_retrieval": {
				PricePerUnitCNY: alpacadecimal.NewFromFloat(0.01),
				CreditRate:      1000,
			},
		},
	}
	svc := newTestService(repo, resolver)

	input := IngestBatchInput{
		Namespace:    "ns-1",
		CustomerID:   "cust-1",
		SubjectID:    "subj-1",
		UsageBatchID: "batch-byok",
		TenantSeq:    1,
		PayloadHash:  "hash",
		BillingMode:  BillingModeComponent,
		LineItems: []UsageLineItem{
			{ResourceCode: ResourceChatInputToken, Quantity: 1000, Provider: "openai", Model: "gpt-4", ProviderManaged: false},
			{ResourceCode: ResourceRAGRetrieval, Quantity: 5, ProviderManaged: false},
		},
	}

	result, err := svc.IngestBatch(t.Context(), input)
	require.NoError(t, err)
	// RAG: ceil(5 * 0.01 * 1000) = 50. Model tokens: 0 (BYOK).
	require.Equal(t, int64(50), result.TotalCredits)
}

func TestService_IngestBatch_BundleMode(t *testing.T) {
	repo := newMockRepo()
	svc := newTestService(repo, &mockRateCardResolver{entries: map[string]CustomerRateCardEntry{}})

	ceiling := int64(100)
	input := IngestBatchInput{
		Namespace:      "ns-1",
		CustomerID:     "cust-1",
		SubjectID:      "subj-1",
		UsageBatchID:   "batch-bundle",
		TenantSeq:      1,
		PayloadHash:    "hash",
		BillingMode:    BillingModeBundle,
		CeilingCredits: &ceiling,
	}

	result, err := svc.IngestBatch(t.Context(), input)
	require.NoError(t, err)
	require.Equal(t, int64(100), result.TotalCredits) // ceiling is the total
}

func TestService_IngestBatch_MissingRate(t *testing.T) {
	repo := newMockRepo()
	svc := newTestService(repo, &mockRateCardResolver{entries: map[string]CustomerRateCardEntry{}})

	input := IngestBatchInput{
		Namespace:    "ns-1",
		CustomerID:   "cust-1",
		SubjectID:    "subj-1",
		UsageBatchID: "batch-norate",
		TenantSeq:    1,
		PayloadHash:  "hash",
		BillingMode:  BillingModeComponent,
		LineItems:    []UsageLineItem{{ResourceCode: ResourceChatInputToken, Quantity: 1000, Provider: "openai", Model: "gpt-4", ProviderManaged: true}},
	}

	_, err := svc.IngestBatch(t.Context(), input)
	require.Error(t, err)
}

func TestService_GetCoveredSeq(t *testing.T) {
	repo := newMockRepo()
	repo.coveredSeq = 42
	svc := newTestService(repo, &mockRateCardResolver{})

	seq, err := svc.GetCoveredSeq(t.Context(), "ns-1", "cust-1")
	require.NoError(t, err)
	require.Equal(t, int64(42), seq)
}

// Compile-time check that mockRepo implements Repository.
var _ Repository = (*mockRepo)(nil)
