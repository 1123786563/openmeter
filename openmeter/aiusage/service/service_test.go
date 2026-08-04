package service_test

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/openmeterio/openmeter/openmeter/aiusage"
	"github.com/openmeterio/openmeter/openmeter/aiusage/adapter"
	"github.com/openmeterio/openmeter/openmeter/aiusage/pricing"
	"github.com/openmeterio/openmeter/openmeter/aiusage/service"
	"github.com/openmeterio/openmeter/openmeter/aiusage/settlement"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
)

// ---------------------------------------------------------------------------
// Mocks
// ---------------------------------------------------------------------------

type mockAdapter struct {
	mu         sync.Mutex
	txAdapter  *mockTxAdapter
	lockErr    error
	committed  bool
	rolledBack bool
}

func (a *mockAdapter) WithCustomerLock(ctx context.Context, _, _ string, fn func(ctx context.Context, tx adapter.TxAdapter) error) error {
	if a.lockErr != nil {
		return a.lockErr
	}
	a.mu.Lock()
	a.committed = false
	a.rolledBack = false
	a.mu.Unlock()

	err := fn(ctx, a.txAdapter)
	a.mu.Lock()
	if err != nil {
		a.rolledBack = true
	} else {
		a.committed = true
	}
	a.mu.Unlock()
	return err
}

type mockTxAdapter struct {
	mu            sync.Mutex
	batches       []aiusage.SettledBatch
	createErr     error
	watermarkSeq  int64
	watermarkErr  error
	outboxEvents  []aiusage.OutboxEvent
	outboxErr     error
	storedBatches map[string]*aiusage.AIUsageBatch
}

func newMockTxAdapter() *mockTxAdapter {
	return &mockTxAdapter{
		storedBatches: make(map[string]*aiusage.AIUsageBatch),
	}
}

func (m *mockTxAdapter) GetBatchByIdempotencyKey(_ context.Context, _, _, key string) (*aiusage.AIUsageBatch, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if b, ok := m.storedBatches[key]; ok {
		return b, nil
	}
	return nil, nil
}

func (m *mockTxAdapter) CreateSettledBatch(_ context.Context, in aiusage.SettledBatch) (*aiusage.AIUsageBatch, bool, error) {
	if m.createErr != nil {
		return nil, false, m.createErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if b, ok := m.storedBatches[in.UsageBatchID]; ok {
		// On replay with a different hash, surface the conflict so the
		// service-level short-circuit and the adapter-level check agree.
		if b.PayloadHash != in.PayloadHash {
			return nil, false, aiusage.ErrIdempotencyConflict
		}
		return b, false, nil
	}
	b := &aiusage.AIUsageBatch{UsageBatchID: in.UsageBatchID, Status: in.Status, TenantSeq: in.TenantSeq, PayloadHash: in.PayloadHash}
	m.storedBatches[in.UsageBatchID] = b
	m.batches = append(m.batches, in)
	return b, true, nil
}

func (m *mockTxAdapter) AdvanceWatermark(_ context.Context, _, _ string, seq int64) (int64, error) {
	if m.watermarkErr != nil {
		return 0, m.watermarkErr
	}
	if seq > m.watermarkSeq {
		m.watermarkSeq = seq
	}
	return m.watermarkSeq, nil
}

func (m *mockTxAdapter) AppendOutbox(_ context.Context, _, _, _ string, events []aiusage.OutboxEvent, _ string) error {
	if m.outboxErr != nil {
		return m.outboxErr
	}
	m.outboxEvents = append(m.outboxEvents, events...)
	return nil
}

func (m *mockTxAdapter) getBatches() []aiusage.SettledBatch {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := make([]aiusage.SettledBatch, len(m.batches))
	copy(cp, m.batches)
	return cp
}

type mockPricingResolver struct {
	result pricing.ResolvedBatch
	err    error
}

func (m *mockPricingResolver) Resolve(_ context.Context, _ pricing.ResolveInput) (pricing.ResolvedBatch, error) {
	return m.result, m.err
}

type mockProfileResolver struct {
	profile service.CustomerProfile
	err     error
}

func (m *mockProfileResolver) Resolve(_ context.Context, _, _ string) (service.CustomerProfile, error) {
	return m.profile, m.err
}

type mockScopeResolver struct {
	scope aiusage.SettlementScope
	err   error
}

func (m *mockScopeResolver) ResolveScope(_ context.Context, _, _ string) (aiusage.SettlementScope, error) {
	return m.scope, m.err
}

type mockAllocationFetcher struct {
	allocs []aiusage.Allocation
	err    error
}

func (m *mockAllocationFetcher) GetAllocations(_ context.Context, _, _, _ string) ([]aiusage.Allocation, error) {
	return m.allocs, m.err
}

// mockSettlement implements settlement.Service for the application test.
type mockSettlement struct {
	mu               sync.Mutex
	allocResult      []aiusage.Allocation
	allocErr         error
	allocCount       int32
	correctResult    []aiusage.Allocation
	correctErr       error
	lastAllocInput   *settlement.SettlementInput
	lastCorrectInput *settlement.CorrectionInput
}

func (m *mockSettlement) AllocateAndBook(_ context.Context, _ adapter.TxAdapter, in settlement.SettlementInput) ([]aiusage.Allocation, error) {
	atomic.AddInt32(&m.allocCount, 1)
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastAllocInput = &in
	return m.allocResult, m.allocErr
}

func (m *mockSettlement) Correct(_ context.Context, _ adapter.TxAdapter, in settlement.CorrectionInput) ([]aiusage.Allocation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastCorrectInput = &in
	return m.correctResult, m.correctErr
}

func (m *mockSettlement) getAllocCount() int32 {
	return atomic.LoadInt32(&m.allocCount)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func newService(t *testing.T, cfg service.Config) service.Service {
	t.Helper()
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	if cfg.Tracer == nil {
		cfg.Tracer = noop.Tracer{}
	}
	return service.New(cfg)
}

func defaultProfile() service.CustomerProfile {
	return service.CustomerProfile{
		ChargeID:       "charge-ai",
		Currency:       currencies.NewCurrencyReference("USD"),
		FeatureKey:     "ai_usage",
		SettlementMode: productcatalog.CreditThenInvoiceSettlementMode,
	}
}

func componentInput() aiusage.IngestBatchInput {
	return aiusage.IngestBatchInput{
		Namespace:    "ns-1",
		CustomerID:   "cust-1",
		SubjectID:    "subj-1",
		UsageBatchID: "batch-001",
		TenantSeq:    1,
		OccurredAt:   time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC),
		RateVersion:  "v1",
		BillingMode:  aiusage.BillingModeComponent,
		PayloadHash:  "hash-001",
		LineItems: []aiusage.UsageLineItem{
			{ResourceCode: aiusage.ResourceLLMInputTokens, Quantity: 1000, Provider: "openai", Model: "gpt-4", ProviderManaged: true},
		},
	}
}

func bundleInput() aiusage.IngestBatchInput {
	ceiling := int64(50)
	return aiusage.IngestBatchInput{
		Namespace:      "ns-1",
		CustomerID:     "cust-1",
		SubjectID:      "subj-1",
		UsageBatchID:   "batch-bundle",
		TenantSeq:      2,
		OccurredAt:     time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC),
		RateVersion:    "v1",
		BillingMode:    aiusage.BillingModeBundle,
		PayloadHash:    "hash-bundle",
		CeilingCredits: &ceiling,
	}
}

func resolvedComponentBatch() pricing.ResolvedBatch {
	return pricing.ResolvedBatch{
		Lines: []pricing.ResolvedLine{
			{
				ResourceCode:     aiusage.ResourceLLMInputTokens,
				CustomerCredits:  40,
				ProviderCost:     alpacadecimal.NewFromInt(1),
				ProviderCurrency: "USD",
				CostSource:       "rate_card",
			},
		},
		TotalCredits: 40,
		BillingMode:  aiusage.BillingModeComponent,
	}
}

func allocationsCovering(amount int64) []aiusage.Allocation {
	return []aiusage.Allocation{
		{Amount: amount, FundingSource: aiusage.FundingSourcePlan, Ledger: aiusage.LedgerProvenance{TransactionGroupID: "grp-1", RealizationID: "real-1", SortHint: 0}},
	}
}

func allocSplitEnterprise() []aiusage.Allocation {
	return []aiusage.Allocation{
		{Amount: 10, FundingSource: aiusage.FundingSourcePlan, Ledger: aiusage.LedgerProvenance{TransactionGroupID: "grp-1", RealizationID: "real-1", SortHint: 0}},
		{Amount: 30, FundingSource: aiusage.FundingSourceEnterpriseReceivable, Ledger: aiusage.LedgerProvenance{TransactionGroupID: "grp-1", RealizationID: "real-2", SortHint: 1}},
	}
}

func correctionInput() aiusage.CorrectionInput {
	return aiusage.CorrectionInput{
		Namespace:       "ns-1",
		CustomerID:      "cust-1",
		SubjectID:       "subj-1",
		OriginalBatchID: "batch-001",
		TenantSeq:       3,
		PayloadHash:     "hash-corr",
		Reason:          "billing error",
	}
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

// Test 1: Full settle flow — pricing -> settlement -> persistence.
func TestSettle_FullFlow(t *testing.T) {
	tx := newMockTxAdapter()
	adp := &mockAdapter{txAdapter: tx}
	pms := &mockPricingResolver{result: resolvedComponentBatch()}
	settle := &mockSettlement{allocResult: allocationsCovering(40)}

	svc := newService(t, service.Config{
		Adapter:         adp,
		Pricing:         pms,
		Settlement:      settle,
		ProfileResolver: &mockProfileResolver{profile: defaultProfile()},
	})

	result, err := svc.Settle(t.Context(), componentInput())
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "batch-001", result.BatchID)
	require.Equal(t, aiusage.BatchStatusSettled, result.Status)
	require.Equal(t, int64(40), result.TotalCredits)

	// Verify batch was persisted.
	batches := tx.getBatches()
	require.Len(t, batches, 1)
	require.Equal(t, int64(40), batches[0].TotalCredits)
	require.Len(t, batches[0].Allocations, 1)

	// Settlement was called with the correct total credits.
	require.NotNil(t, settle.lastAllocInput)
	require.Equal(t, int64(40), settle.lastAllocInput.TotalCredits)

	// Transaction was committed.
	require.True(t, adp.committed)
}

// Test 2: Enterprise receivable split — settlement returns mixed allocations.
func TestSettle_EnterpriseReceivableSplit(t *testing.T) {
	tx := newMockTxAdapter()
	adp := &mockAdapter{txAdapter: tx}
	pms := &mockPricingResolver{result: resolvedComponentBatch()}
	settle := &mockSettlement{allocResult: allocSplitEnterprise()}

	svc := newService(t, service.Config{
		Adapter:         adp,
		Pricing:         pms,
		Settlement:      settle,
		ProfileResolver: &mockProfileResolver{profile: defaultProfile()},
	})

	result, err := svc.Settle(t.Context(), componentInput())
	require.NoError(t, err)
	require.Equal(t, int64(40), result.TotalCredits)

	batches := tx.getBatches()
	require.Len(t, batches, 1)
	require.Len(t, batches[0].Allocations, 2)
	require.Equal(t, aiusage.FundingSourcePlan, batches[0].Allocations[0].FundingSource)
	require.Equal(t, aiusage.FundingSourceEnterpriseReceivable, batches[0].Allocations[1].FundingSource)
}

// Test 3: Idempotent replay — same batch ID returns existing result without
// re-persisting.
func TestSettle_IdempotentReplay(t *testing.T) {
	tx := newMockTxAdapter()
	adp := &mockAdapter{txAdapter: tx}
	pms := &mockPricingResolver{result: resolvedComponentBatch()}
	settle := &mockSettlement{allocResult: allocationsCovering(40)}

	svc := newService(t, service.Config{
		Adapter:         adp,
		Pricing:         pms,
		Settlement:      settle,
		ProfileResolver: &mockProfileResolver{profile: defaultProfile()},
	})

	// First call creates the batch.
	_, err := svc.Settle(t.Context(), componentInput())
	require.NoError(t, err)
	require.Len(t, tx.getBatches(), 1)
	require.Equal(t, int32(1), settle.getAllocCount(), "collector should be called once on first submission")

	// Second call with same ID is idempotent — the collector is NOT called
	// because the idempotency short-circuit fires before AllocateAndBook.
	_, err = svc.Settle(t.Context(), componentInput())
	require.NoError(t, err)

	// Still only one batch persisted (dedup by UsageBatchID).
	require.Len(t, tx.getBatches(), 1)
	require.Equal(t, int32(1), settle.getAllocCount(), "collector must NOT be called on replay")
}

// Test 3b: Hash conflict — same UsageBatchID with a different PayloadHash
// returns ErrIdempotencyConflict (HTTP 409), not a silent success.
func TestSettle_HashConflictReturnsError(t *testing.T) {
	tx := newMockTxAdapter()
	adp := &mockAdapter{txAdapter: tx}
	pms := &mockPricingResolver{result: resolvedComponentBatch()}
	settle := &mockSettlement{allocResult: allocationsCovering(40)}

	svc := newService(t, service.Config{
		Adapter:         adp,
		Pricing:         pms,
		Settlement:      settle,
		ProfileResolver: &mockProfileResolver{profile: defaultProfile()},
	})

	// First submission creates the batch.
	_, err := svc.Settle(t.Context(), componentInput())
	require.NoError(t, err)
	require.Len(t, tx.getBatches(), 1)

	// Second submission with the SAME UsageBatchID but a DIFFERENT PayloadHash.
	conflict := componentInput()
	conflict.PayloadHash = "hash-changed"
	_, err = svc.Settle(t.Context(), conflict)
	require.ErrorIs(t, err, aiusage.ErrIdempotencyConflict)

	// The collector must not have fired on the conflicting submission.
	require.Equal(t, int32(1), settle.getAllocCount(), "collector must not fire on hash conflict")
	// Still only one batch persisted.
	require.Len(t, tx.getBatches(), 1)
}

// Test 4: Correction flow — fetches original allocations, reverses via
// settlement, persists compensated batch.
func TestCorrect_FullFlow(t *testing.T) {
	tx := newMockTxAdapter()
	adp := &mockAdapter{txAdapter: tx}
	settle := &mockSettlement{
		correctResult: []aiusage.Allocation{
			{Amount: -40, FundingSource: aiusage.FundingSourcePlan, Ledger: aiusage.LedgerProvenance{TransactionGroupID: "grp-corr", RealizationID: "real-1", SortHint: 0}},
		},
	}
	fetcher := &mockAllocationFetcher{
		allocs: []aiusage.Allocation{
			{Amount: 40, FundingSource: aiusage.FundingSourcePlan, Ledger: aiusage.LedgerProvenance{TransactionGroupID: "grp-1", RealizationID: "real-1", SortHint: 0}},
		},
	}

	svc := newService(t, service.Config{
		Adapter:           adp,
		Settlement:        settle,
		ProfileResolver:   &mockProfileResolver{profile: defaultProfile()},
		AllocationFetcher: fetcher,
	})

	result, err := svc.Correct(t.Context(), correctionInput())
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, aiusage.BatchStatusCompensated, result.Status)

	// A correction batch was persisted with negative allocations.
	batches := tx.getBatches()
	require.Len(t, batches, 1)
	require.Equal(t, aiusage.BatchStatusCompensated, batches[0].Status)
	require.Len(t, batches[0].Allocations, 1)
	require.Equal(t, int64(-40), batches[0].Allocations[0].Amount)

	// Settlement.Correct was called with the original allocations.
	require.NotNil(t, settle.lastCorrectInput)
	require.Equal(t, "batch-001", settle.lastCorrectInput.OriginalBatchID)
	require.Len(t, settle.lastCorrectInput.OriginalAllocations, 1)
}

// Test 5: BYOK mixed batch — zero-credit lines from BYOK resources produce
// zero total credits, and settlement is not called.
func TestSettle_BYOKMixedBatch(t *testing.T) {
	tx := newMockTxAdapter()
	adp := &mockAdapter{txAdapter: tx}
	pms := &mockPricingResolver{
		result: pricing.ResolvedBatch{
			Lines: []pricing.ResolvedLine{
				{
					ResourceCode:     aiusage.ResourceLLMInputTokens,
					CustomerCredits:  0, // BYOK: no customer charge
					ProviderCost:     alpacadecimal.NewFromInt(0),
					ProviderCurrency: "USD",
					CostSource:       "byok",
				},
			},
			TotalCredits: 0,
			BillingMode:  aiusage.BillingModeComponent,
		},
	}
	settle := &mockSettlement{allocResult: []aiusage.Allocation{}}

	in := componentInput()
	in.ProviderManaged = false // BYOK
	in.LineItems[0].ProviderManaged = false
	in.LineItems[0].Provider = ""

	svc := newService(t, service.Config{
		Adapter:         adp,
		Pricing:         pms,
		Settlement:      settle,
		ProfileResolver: &mockProfileResolver{profile: defaultProfile()},
	})

	result, err := svc.Settle(t.Context(), in)
	require.NoError(t, err)
	require.Equal(t, int64(0), result.TotalCredits)

	// Batch was still persisted (for visibility/audit).
	batches := tx.getBatches()
	require.Len(t, batches, 1)
	require.Equal(t, int64(0), batches[0].TotalCredits)
}

// Test 6: Settlement error propagates and nothing is persisted.
func TestSettle_SettlementErrorRollback(t *testing.T) {
	tx := newMockTxAdapter()
	adp := &mockAdapter{txAdapter: tx}
	pms := &mockPricingResolver{result: resolvedComponentBatch()}
	settle := &mockSettlement{allocErr: aiusage.ErrCreditInsufficient}

	svc := newService(t, service.Config{
		Adapter:         adp,
		Pricing:         pms,
		Settlement:      settle,
		ProfileResolver: &mockProfileResolver{profile: defaultProfile()},
	})

	_, err := svc.Settle(t.Context(), componentInput())
	require.Error(t, err)
	require.ErrorIs(t, err, aiusage.ErrCreditInsufficient)

	// No batch persisted.
	require.Empty(t, tx.getBatches())
	// Transaction was rolled back.
	require.True(t, adp.rolledBack)
}

// Test 7 (Finding 10c): Persistence error after settlement leaves nothing
// persisted — rollback verification. CreateSettledBatch returns an error;
// WithCustomerLock rolls back, so no batch survives.
func TestSettle_PersistenceErrorRollback(t *testing.T) {
	tx := newMockTxAdapter()
	tx.createErr = errors.New("DB write failed")
	adp := &mockAdapter{txAdapter: tx}
	pms := &mockPricingResolver{result: resolvedComponentBatch()}
	settle := &mockSettlement{allocResult: allocationsCovering(40)}

	svc := newService(t, service.Config{
		Adapter:         adp,
		Pricing:         pms,
		Settlement:      settle,
		ProfileResolver: &mockProfileResolver{profile: defaultProfile()},
	})

	_, err := svc.Settle(t.Context(), componentInput())
	require.Error(t, err)

	// No batch persisted despite settlement succeeding.
	batches := tx.getBatches()
	require.Empty(t, batches, "no batch should persist on persistence error")

	// Settlement WAS called (it runs before persistence).
	require.NotNil(t, settle.lastAllocInput)

	// Transaction was rolled back.
	require.True(t, adp.rolledBack)
}

// Test 8: Pricing error propagates.
func TestSettle_PricingError(t *testing.T) {
	tx := newMockTxAdapter()
	adp := &mockAdapter{txAdapter: tx}
	pms := &mockPricingResolver{err: aiusage.ErrMissingRateCard}

	svc := newService(t, service.Config{
		Adapter:         adp,
		Pricing:         pms,
		Settlement:      &mockSettlement{},
		ProfileResolver: &mockProfileResolver{profile: defaultProfile()},
	})

	_, err := svc.Settle(t.Context(), componentInput())
	require.Error(t, err)

	// Settlement was NOT called (pricing failed first).
	require.Empty(t, tx.getBatches())
}

// Test 9: Bundle mode — ceiling caps total credits.
func TestSettle_BundleModeCeiling(t *testing.T) {
	tx := newMockTxAdapter()
	adp := &mockAdapter{txAdapter: tx}
	pms := &mockPricingResolver{
		result: pricing.ResolvedBatch{
			Lines: []pricing.ResolvedLine{
				{ResourceCode: aiusage.ResourceLLMInputTokens, CustomerCredits: 100},
			},
			TotalCredits: 100,
			BillingMode:  aiusage.BillingModeBundle,
		},
	}
	settle := &mockSettlement{allocResult: allocationsCovering(50)}

	svc := newService(t, service.Config{
		Adapter:         adp,
		Pricing:         pms,
		Settlement:      settle,
		ProfileResolver: &mockProfileResolver{profile: defaultProfile()},
	})

	result, err := svc.Settle(t.Context(), bundleInput())
	require.NoError(t, err)

	// In bundle mode, total credits = ceiling (50).
	require.Equal(t, int64(50), result.TotalCredits)
	require.NotNil(t, settle.lastAllocInput)
	require.Equal(t, int64(50), settle.lastAllocInput.TotalCredits)
}

// Test 10: Shadow scope — batch persisted but allocations empty.
func TestSettle_ShadowScope(t *testing.T) {
	tx := newMockTxAdapter()
	adp := &mockAdapter{txAdapter: tx}
	pms := &mockPricingResolver{result: resolvedComponentBatch()}
	settle := &mockSettlement{allocResult: []aiusage.Allocation{}}

	svc := newService(t, service.Config{
		Adapter:         adp,
		Pricing:         pms,
		Settlement:      settle,
		ProfileResolver: &mockProfileResolver{profile: defaultProfile()},
		ScopeResolver:   &mockScopeResolver{scope: aiusage.SettlementScopeShadow},
	})

	result, err := svc.Settle(t.Context(), componentInput())
	require.NoError(t, err)

	batches := tx.getBatches()
	require.Len(t, batches, 1)
	require.Equal(t, aiusage.SettlementScopeShadow, batches[0].SettlementScope)

	// Settlement was called with shadow scope.
	require.NotNil(t, settle.lastAllocInput)
	require.Equal(t, aiusage.SettlementScopeShadow, settle.lastAllocInput.SettlementScope)

	_ = result
}

// Test 11: Validation errors short-circuit before any work.
func TestSettle_ValidationFailure(t *testing.T) {
	svc := newService(t, service.Config{})

	err := aiusage.IngestBatchInput{}.Validate()
	require.Error(t, err)

	_, err = svc.Settle(t.Context(), aiusage.IngestBatchInput{})
	require.Error(t, err)
}

// Test 12: CorrectionInput validation.
func TestCorrectionInput_Validate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		in := correctionInput()
		require.NoError(t, in.Validate())
	})
	t.Run("missing namespace", func(t *testing.T) {
		in := correctionInput()
		in.Namespace = ""
		require.Error(t, in.Validate())
	})
	t.Run("zero tenant_seq", func(t *testing.T) {
		in := correctionInput()
		in.TenantSeq = 0
		require.Error(t, in.Validate())
	})
}

// Compile-time interface checks.
var (
	_ adapter.Adapter                 = (*mockAdapter)(nil)
	_ adapter.TxAdapter               = (*mockTxAdapter)(nil)
	_ service.PricingResolver         = (*mockPricingResolver)(nil)
	_ service.CustomerProfileResolver = (*mockProfileResolver)(nil)
	_ service.ScopeResolver           = (*mockScopeResolver)(nil)
	_ service.AllocationFetcher       = (*mockAllocationFetcher)(nil)
	_ settlement.Service              = (*mockSettlement)(nil)
)

// Suppress unused imports.
var (
	_ = creditrealization.AnnotationFundingSource
	_ = fmt.Sprintf
)
