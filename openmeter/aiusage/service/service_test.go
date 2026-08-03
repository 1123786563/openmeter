package service_test

import (
	"context"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/openmeterio/openmeter/openmeter/aiusage"
	"github.com/openmeterio/openmeter/openmeter/aiusage/pricing"
	"github.com/openmeterio/openmeter/openmeter/aiusage/service"
	"github.com/openmeterio/openmeter/openmeter/aiusage/settlement"
)

type staticRateProvider struct {
	entries []pricing.RateEntry
}

func (p *staticRateProvider) GetEntries(_ context.Context, _ string) ([]pricing.RateEntry, error) {
	return p.entries, nil
}

type mockGrantReader struct {
	grants []aiusage.SettlementGrant
}

func (m *mockGrantReader) GetGrants(_ context.Context, _, _ string) ([]aiusage.SettlementGrant, error) {
	out := make([]aiusage.SettlementGrant, len(m.grants))
	copy(out, m.grants)
	return out, nil
}

type mockLedgerRecorder struct {
	deductions map[string][]aiusage.LedgerEntryRef
}

func (m *mockLedgerRecorder) RecordDeductions(_ context.Context, _, _ string, deductions []aiusage.LedgerEntryRef, batchID string) error {
	if m.deductions == nil {
		m.deductions = make(map[string][]aiusage.LedgerEntryRef)
	}
	m.deductions[batchID] = deductions
	return nil
}

type mockStore struct {
	mu         sync.Mutex
	batches    map[string]*aiusage.SettledBatch
	watermark  int64
	storeCount int
}

func newMockStore() *mockStore {
	return &mockStore{batches: make(map[string]*aiusage.SettledBatch)}
}

func (s *mockStore) Store(_ context.Context, in aiusage.SettledBatch) (*aiusage.BatchSettlementResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := in.Namespace + ":" + in.UsageBatchID
	if existing, ok := s.batches[key]; ok {
		if existing.PayloadHash == in.PayloadHash {
			return &aiusage.BatchSettlementResult{
				BatchID:          existing.UsageBatchID,
				Status:           existing.Status,
				TotalCredits:     existing.TotalCredits,
				CoveredTenantSeq: existing.TenantSeq,
			}, nil
		}
		return nil, aiusage.ErrBatchPayloadConflict
	}

	copied := in
	s.batches[key] = &copied
	s.storeCount++

	if in.TenantSeq == s.watermark+1 {
		s.watermark = in.TenantSeq
	}

	return &aiusage.BatchSettlementResult{
		BatchID:          in.UsageBatchID,
		Status:           in.Status,
		TotalCredits:     in.TotalCredits,
		CoveredTenantSeq: s.watermark,
	}, nil
}

func (s *mockStore) getBatch(namespace, batchID string) *aiusage.SettledBatch {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.batches[namespace+":"+batchID]
}

func (s *mockStore) getWatermark() int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.watermark
}

func newTestService(t *testing.T, grants []aiusage.SettlementGrant) (service.Service, *mockStore, *mockLedgerRecorder) {
	t.Helper()

	rateProvider := &staticRateProvider{
		entries: []pricing.RateEntry{
			{
				ResourceCode:   aiusage.ResourceLLMInputTokens,
				Provider:       "openai",
				Model:          "gpt-4",
				CreditsPerUnit: 2,
				UnitSize:       1000,
				EffectiveFrom:  time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			{
				ResourceCode:   aiusage.ResourceRAGQueries,
				CreditsPerUnit: 10,
				UnitSize:       1,
				EffectiveFrom:  time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			},
		},
	}

	pricingSvc := pricing.NewService(rateProvider)
	ledger := &mockLedgerRecorder{}
	settleSvc := settlement.NewService(settlement.ServiceConfig{
		GrantReader: &mockGrantReader{grants: grants},
		Ledger:      ledger,
		Logger:      slog.Default(),
		Tracer:      noop.Tracer{},
	})
	store := newMockStore()

	appSvc := service.New(service.Config{
		Pricing:    pricingSvc,
		Settlement: settleSvc,
		Store:      store,
		Logger:     slog.Default(),
		Tracer:     noop.Tracer{},
	})

	return appSvc, store, ledger
}

func standardGrants() []aiusage.SettlementGrant {
	return []aiusage.SettlementGrant{
		{GrantID: "plan-1", Amount: 1000, Priority: 0, Source: "plan"},
		{GrantID: "promo-1", Amount: 500, Priority: 10, Source: "promotional"},
		{GrantID: "topup-1", Amount: 2000, Priority: 20, Source: "paid_top_up"},
		{GrantID: "recv-1", Amount: 0, Priority: 30, Source: "enterprise_receivable"},
	}
}

func validInput(seq int64) aiusage.IngestBatchInput {
	return aiusage.IngestBatchInput{
		ProviderManaged: true,
		Namespace:    "ns-1",
		CustomerID:   "cust-1",
		SubjectID:    "subj-1",
		UsageBatchID: "batch-" + string(rune('A'-1+seq)),
		TenantSeq:    seq,
		OccurredAt:   time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC),
		RateVersion:  "pro-v1",
		BillingMode:  aiusage.BillingModeComponent,
		PayloadHash:  "hash-" + string(rune('A'-1+seq)),
		LineItems: []aiusage.UsageLineItem{
			{
				ResourceCode:    aiusage.ResourceLLMInputTokens,
				Quantity:        5000,
				Provider:        "openai",
				Model:           "gpt-4",
				ProviderManaged: true,
			},
		},
	}
}

func TestSettle_FullFlow(t *testing.T) {
	svc, store, ledger := newTestService(t, standardGrants())

	input := validInput(1)

	result, err := svc.Settle(t.Context(), input)
	require.NoError(t, err)

	require.Equal(t, aiusage.BatchStatusSettled, result.Status)
	require.Equal(t, int64(10), result.TotalCredits)
	require.Equal(t, int64(1), result.CoveredTenantSeq)

	batch := store.getBatch("ns-1", input.UsageBatchID)
	require.NotNil(t, batch)
	require.Equal(t, int64(10), batch.TotalCredits)
	require.Len(t, batch.Allocations, 1)
	require.Equal(t, aiusage.FundingSourcePlan, batch.Allocations[0].FundingSource)
	require.Equal(t, aiusage.SettlementScopeFormal, batch.SettlementScope)

	require.Len(t, batch.OutboxEvents, 1)
	require.Equal(t, "ai_usage.batch.settled", batch.OutboxEvents[0].EventType)

	require.Len(t, ledger.deductions, 1)
	require.Len(t, ledger.deductions[input.UsageBatchID], 1)
}

func TestSettle_WatermarkAdvancement(t *testing.T) {
	svc, store, _ := newTestService(t, standardGrants())

	_, err := svc.Settle(t.Context(), validInput(1))
	require.NoError(t, err)
	require.Equal(t, int64(1), store.getWatermark())

	_, err = svc.Settle(t.Context(), validInput(2))
	require.NoError(t, err)
	require.Equal(t, int64(2), store.getWatermark())

	result3, err := svc.Settle(t.Context(), validInput(3))
	require.NoError(t, err)
	require.Equal(t, int64(3), store.getWatermark())
	require.Equal(t, int64(3), result3.CoveredTenantSeq)
}

func TestSettle_WatermarkGap(t *testing.T) {
	svc, store, _ := newTestService(t, standardGrants())

	_, err := svc.Settle(t.Context(), validInput(1))
	require.NoError(t, err)
	require.Equal(t, int64(1), store.getWatermark())

	_, err = svc.Settle(t.Context(), validInput(3))
	require.NoError(t, err)
	require.Equal(t, int64(1), store.getWatermark())
}

func TestSettle_InsufficientCredits(t *testing.T) {
	grants := []aiusage.SettlementGrant{
		{GrantID: "plan-1", Amount: 5, Priority: 0, Source: "plan"},
	}
	svc, store, _ := newTestService(t, grants)

	_, err := svc.Settle(t.Context(), validInput(1))
	require.Error(t, err)
	require.ErrorIs(t, err, aiusage.ErrCreditInsufficient)
	require.Equal(t, 0, store.storeCount)
}

func TestSettle_EnterpriseReceivable(t *testing.T) {
	grants := []aiusage.SettlementGrant{
		{GrantID: "plan-1", Amount: 3, Priority: 0, Source: "plan"},
		{GrantID: "recv-1", Amount: 0, Priority: 30, Source: "enterprise_receivable"},
	}
	svc, store, _ := newTestService(t, grants)

	result, err := svc.Settle(t.Context(), validInput(1))
	require.NoError(t, err)
	require.Equal(t, int64(10), result.TotalCredits)

	batch := store.getBatch("ns-1", validInput(1).UsageBatchID)
	require.Len(t, batch.Allocations, 2)
	require.Equal(t, aiusage.FundingSourcePlan, batch.Allocations[0].FundingSource)
	require.InDelta(t, 3.0, batch.Allocations[0].Amount, 0.001)
	require.Equal(t, aiusage.FundingSourceEnterpriseReceivable, batch.Allocations[1].FundingSource)
	require.InDelta(t, 7.0, batch.Allocations[1].Amount, 0.001)
}

func TestSettle_BundleMode(t *testing.T) {
	svc, store, _ := newTestService(t, standardGrants())

	ceiling := int64(42)
	input := aiusage.IngestBatchInput{
		Namespace:      "ns-1",
		CustomerID:     "cust-1",
		SubjectID:      "subj-1",
		UsageBatchID:   "batch-bundle",
		TenantSeq:      1,
		OccurredAt:     time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC),
		RateVersion:    "pro-v1",
		BillingMode:    aiusage.BillingModeBundle,
		PayloadHash:    "hash-bundle",
		CeilingCredits: &ceiling,
	}

	result, err := svc.Settle(t.Context(), input)
	require.NoError(t, err)
	require.Equal(t, int64(42), result.TotalCredits)

	batch := store.getBatch("ns-1", "batch-bundle")
	require.Equal(t, int64(42), batch.TotalCredits)
}

func TestSettle_BYOKMixed(t *testing.T) {
	svc, store, _ := newTestService(t, standardGrants())

	input := aiusage.IngestBatchInput{
		ProviderManaged: true,
		Namespace:    "ns-1",
		CustomerID:   "cust-1",
		SubjectID:    "subj-1",
		UsageBatchID: "batch-byok",
		TenantSeq:    1,
		OccurredAt:   time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC),
		RateVersion:  "pro-v1",
		BillingMode:  aiusage.BillingModeComponent,
		PayloadHash:  "hash-byok",
		LineItems: []aiusage.UsageLineItem{
			{
				ResourceCode:    aiusage.ResourceLLMInputTokens,
				Quantity:        5000,
				Provider:        "openai",
				Model:           "gpt-4",
				ProviderManaged: true,
			},
			{
				ResourceCode: aiusage.ResourceRAGQueries,
				Quantity:     3,
			},
		},
	}

	result, err := svc.Settle(t.Context(), input)
	require.NoError(t, err)

	// LLM: 5000 * 2/1000 = 10 credits. RAG: 3 * 10 = 30 credits. Total = 40.
	require.Equal(t, int64(40), result.TotalCredits)

	batch := store.getBatch("ns-1", "batch-byok")
	require.Len(t, batch.RatingSnapshots, 2)
}

func TestCorrect(t *testing.T) {
	svc, store, _ := newTestService(t, standardGrants())

	_, err := svc.Settle(t.Context(), validInput(1))
	require.NoError(t, err)

	result, err := svc.Correct(t.Context(), aiusage.CorrectionInput{
		Namespace:       "ns-1",
		CustomerID:      "cust-1",
		SubjectID:       "subj-1",
		OriginalBatchID: validInput(1).UsageBatchID,
		TenantSeq:       2,
		PayloadHash:     "hash-corr",
		Reason:          "billing error",
	})
	require.NoError(t, err)
	require.Equal(t, aiusage.BatchStatusCompensated, result.Status)

	batch := store.getBatch("ns-1", "corr-"+validInput(1).UsageBatchID)
	require.NotNil(t, batch)
	require.Equal(t, aiusage.BatchStatusCompensated, batch.Status)
	require.Len(t, batch.OutboxEvents, 1)
	require.Equal(t, "ai_usage.batch.corrected", batch.OutboxEvents[0].EventType)
}

func TestSettle_Idempotent(t *testing.T) {
	svc, store, _ := newTestService(t, standardGrants())

	input := validInput(1)

	r1, err := svc.Settle(t.Context(), input)
	require.NoError(t, err)

	r2, err := svc.Settle(t.Context(), input)
	require.NoError(t, err)

	require.Equal(t, r1.TotalCredits, r2.TotalCredits)
	require.Equal(t, 1, store.storeCount)
}

func TestSettle_ValidationFailure(t *testing.T) {
	svc, store, _ := newTestService(t, standardGrants())

	_, err := svc.Settle(t.Context(), aiusage.IngestBatchInput{})
	require.Error(t, err)
	require.Equal(t, 0, store.storeCount)
}

var _ aiusage.GrantBalanceReader = (*mockGrantReader)(nil)
var _ aiusage.LedgerRecorder = (*mockLedgerRecorder)(nil)
var _ service.BatchStore = (*mockStore)(nil)
var _ pricing.RateEntryProvider = (*staticRateProvider)(nil)
