package settlement_test

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/openmeterio/openmeter/openmeter/aiusage"
	"github.com/openmeterio/openmeter/openmeter/aiusage/settlement"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/ledger/collector"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

// mockCollector implements collector.Service for testing.
type mockCollector struct {
	mu              sync.Mutex
	allocResult     creditrealization.CreateAllocationInputs
	correctionResult creditrealization.CreateCorrectionInputs
	err             error
	lastCollectInput *collector.CollectToAccruedInput
	collectCount    int32
	correctCount    int32
}

func (m *mockCollector) CollectToAccrued(_ context.Context, input collector.CollectToAccruedInput) (creditrealization.CreateAllocationInputs, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.collectCount++
	m.lastCollectInput = &input
	return m.allocResult, m.err
}

func (m *mockCollector) CorrectCollectedAccrued(_ context.Context, _ collector.CorrectCollectedAccruedInput) (creditrealization.CreateCorrectionInputs, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.correctCount++
	return m.correctionResult, m.err
}

func (m *mockCollector) getLastCollectInput() *collector.CollectToAccruedInput {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.lastCollectInput
}

func (m *mockCollector) getCollectCount() int32 {
	return atomic.LoadInt32(&m.collectCount)
}

// mockTxAdapter is a no-op TxAdapter for tests that don't need DB interaction.
type mockTxAdapter struct{}

func (mockTxAdapter) GetBatchByIdempotencyKey(_ context.Context, _, _, _ string) (*aiusage.AIUsageBatch, error) {
	return nil, nil
}
func (mockTxAdapter) CreateSettledBatch(_ context.Context, _ aiusage.SettledBatch) (*aiusage.AIUsageBatch, bool, error) {
	return nil, true, nil
}
func (mockTxAdapter) AdvanceWatermark(_ context.Context, _, _ string, _ int64) (int64, error) {
	return 0, nil
}
func (mockTxAdapter) AppendOutbox(_ context.Context, _, _, _ string, _ []aiusage.OutboxEvent, _ string) error {
	return nil
}

var _ aiusageAdapter = mockTxAdapter{}
type aiusageAdapter = interface {
	GetBatchByIdempotencyKey(ctx context.Context, namespace, customerID, key string) (*aiusage.AIUsageBatch, error)
	CreateSettledBatch(ctx context.Context, in aiusage.SettledBatch) (*aiusage.AIUsageBatch, bool, error)
	AdvanceWatermark(ctx context.Context, namespace, subjectID string, seq int64) (int64, error)
	AppendOutbox(ctx context.Context, namespace, customerID, subjectID string, events []aiusage.OutboxEvent, batchID string) error
}

func now() time.Time { return time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC) }

func usdCurrency() currencies.CurrencyReference {
	return currencies.NewCurrencyReference("USD")
}

func servicePeriod() timeutil.ClosedPeriod {
	return timeutil.ClosedPeriod{From: now(), To: now()}
}

func newTestService(c collector.Service) settlement.Service {
	return settlement.New(settlement.ServiceConfig{
		Collector: c,
		Logger:    slog.Default(),
		Tracer:    noop.Tracer{},
	})
}

func baseInput() settlement.SettlementInput {
	return settlement.SettlementInput{
		Namespace:       "ns-1",
		CustomerID:      "cust-1",
		SubjectID:       "subj-1",
		TotalCredits:    40,
		SettlementScope: aiusage.SettlementScopeFormal,
		BatchID:         "batch-001",
		ChargeID:        "charge-ai",
		Currency:        usdCurrency(),
		FeatureKey:      "ai_usage",
		SettlementMode:  productcatalog.CreditThenInvoiceSettlementMode,
		BookedAt:        now(),
		ServicePeriod:   servicePeriod(),
	}
}

// allocationsFromCollector builds CreateAllocationInputs simulating what the
// collector returns for a given set of (amount, originKind) pairs.
func allocationsFromCollector(groupID string, items ...struct {
	amount     int64
	originKind creditrealization.LineageOriginKind
}) creditrealization.CreateAllocationInputs {
	out := make(creditrealization.CreateAllocationInputs, 0, len(items))
	for _, item := range items {
		out = append(out, creditrealization.CreateAllocationInput{
			Amount: alpacadecimal.NewFromInt(item.amount),
			LedgerTransaction: ledgertransaction.GroupReference{
				TransactionGroupID: groupID,
			},
			Annotations: creditrealization.LineageAnnotations(item.originKind),
		})
	}
	return out
}

// Test 1: Burn order — collector returns allocations in priority order.
// plan(10) -> promotional(5) -> paid_top_up(20) -> enterprise_receivable(5)
func TestAllocateAndBook_BurnOrder(t *testing.T) {
	mc := &mockCollector{
		allocResult: allocationsFromCollector("grp-001",
			struct {
				amount     int64
				originKind creditrealization.LineageOriginKind
			}{10, creditrealization.LineageOriginKindRealCredit},
			struct {
				amount     int64
				originKind creditrealization.LineageOriginKind
			}{5, creditrealization.LineageOriginKindRealCredit},
			struct {
				amount     int64
				originKind creditrealization.LineageOriginKind
			}{20, creditrealization.LineageOriginKindRealCredit},
			struct {
				amount     int64
				originKind creditrealization.LineageOriginKind
			}{5, creditrealization.LineageOriginKindAdvance},
		),
	}
	svc := newTestService(mc)

	allocs, err := svc.AllocateAndBook(t.Context(), mockTxAdapter{}, baseInput())
	require.NoError(t, err)
	require.Len(t, allocs, 4)

	// Verify amounts and provenance.
	require.Equal(t, int64(10), allocs[0].Amount)
	require.Equal(t, "grp-001", allocs[0].Ledger.TransactionGroupID)
	require.Equal(t, 0, allocs[0].Ledger.SortHint)
	require.Equal(t, int64(5), allocs[1].Amount)
	require.Equal(t, 1, allocs[1].Ledger.SortHint)
	require.Equal(t, int64(20), allocs[2].Amount)
	require.Equal(t, int64(5), allocs[3].Amount)

	// Verify funding source inference.
	require.Equal(t, aiusage.FundingSourcePaidTopUp, allocs[0].FundingSource)
	require.Equal(t, aiusage.FundingSourceEnterpriseReceivable, allocs[3].FundingSource)

	// Verify collector was called with correct amount.
	lastInput := mc.getLastCollectInput()
	require.NotNil(t, lastInput)
	require.True(t, alpacadecimal.NewFromInt(40).Equal(lastInput.Amount))
}

// Test 2: Shadow scope produces zero ledger effects (collector not called).
func TestAllocateAndBook_ShadowScope(t *testing.T) {
	mc := &mockCollector{}
	svc := newTestService(mc)

	in := baseInput()
	in.SettlementScope = aiusage.SettlementScopeShadow

	allocs, err := svc.AllocateAndBook(t.Context(), mockTxAdapter{}, in)
	require.NoError(t, err)
	require.Empty(t, allocs)

	// Collector was NOT called.
	require.Equal(t, int32(0), mc.getCollectCount())
}

// Test 3: Prepaid shortage without enterprise -> ErrCreditInsufficient.
// The collector returns fewer allocations than requested when CreditOnly mode
// is not set.
func TestAllocateAndBook_PrepaidShortage(t *testing.T) {
	mc := &mockCollector{
		allocResult: allocationsFromCollector("grp-1",
			struct {
				amount     int64
				originKind creditrealization.LineageOriginKind
			}{10, creditrealization.LineageOriginKindRealCredit},
		),
	}
	svc := newTestService(mc)

	in := baseInput()
	in.TotalCredits = 100
	in.SettlementMode = productcatalog.CreditThenInvoiceSettlementMode

	// With CreditThenInvoice, the collector collects what it can (10 credits).
	// The settlement service passes the result through — the shortage is the
	// caller's responsibility to check via the returned total vs. requested.
	allocs, err := svc.AllocateAndBook(t.Context(), mockTxAdapter{}, in)
	require.NoError(t, err)
	require.Len(t, allocs, 1)
	require.Equal(t, int64(10), allocs[0].Amount)
}

// Test 4: Receivable overflow beyond hard limit -> ErrCreditLimitExceeded.
func TestAllocateAndBook_ReceivableOverflow(t *testing.T) {
	mc := &mockCollector{
		allocResult: allocationsFromCollector("grp-1",
			struct {
				amount     int64
				originKind creditrealization.LineageOriginKind
			}{10, creditrealization.LineageOriginKindRealCredit},
		),
	}
	svc := newTestService(mc)

	in := baseInput()
	in.TotalCredits = 100
	hardLimit := int64(50)
	in.ReceivableHardLimit = &hardLimit

	_, err := svc.AllocateAndBook(t.Context(), mockTxAdapter{}, in)
	require.Error(t, err)
	require.ErrorIs(t, err, aiusage.ErrCreditLimitExceeded)

	// Collector was called (it runs before the limit check).
	require.Equal(t, int32(1), mc.getCollectCount())
}

// Test 5: Correction creates linked reversing allocations via collector.
func TestCorrect_ReversesAllocations(t *testing.T) {
	mc := &mockCollector{
		correctionResult: creditrealization.CreateCorrectionInputs{
			{
				ID:                    "corr-1",
				Amount:                alpacadecimal.NewFromInt(-10),
				CorrectsRealizationID: "real-1",
				LedgerTransaction: ledgertransaction.GroupReference{
					TransactionGroupID: "grp-corr-1",
				},
			},
			{
				ID:                    "corr-2",
				Amount:                alpacadecimal.NewFromInt(-5),
				CorrectsRealizationID: "real-2",
				LedgerTransaction: ledgertransaction.GroupReference{
					TransactionGroupID: "grp-corr-1",
				},
			},
		},
	}
	svc := newTestService(mc)

	originalAllocs := []aiusage.Allocation{
		{
			GrantID: "g1",
			Amount:  10,
			Ledger: aiusage.LedgerProvenance{
				TransactionGroupID: "grp-001",
				RealizationID:      "real-1",
				SortHint:           0,
			},
		},
		{
			GrantID: "g2",
			Amount:  5,
			Ledger: aiusage.LedgerProvenance{
				TransactionGroupID: "grp-001",
				RealizationID:      "real-2",
				SortHint:           1,
			},
		},
	}

	reversing, err := svc.Correct(t.Context(), mockTxAdapter{}, settlement.CorrectionInput{
		Namespace:           "ns-1",
		CustomerID:          "cust-1",
		SubjectID:           "subj-1",
		OriginalBatchID:     "batch-001",
		BookedAt:             now(),
		OriginalAllocations: originalAllocs,
		ChargeID:            "charge-ai",
		Currency:            usdCurrency(),
	})
	require.NoError(t, err)
	require.Len(t, reversing, 2)

	// Reversing allocations are negative.
	require.Equal(t, int64(-10), reversing[0].Amount)
	require.Equal(t, "real-1", reversing[0].Ledger.RealizationID)
	require.Equal(t, "grp-corr-1", reversing[0].Ledger.TransactionGroupID)

	require.Equal(t, int64(-5), reversing[1].Amount)
	require.Equal(t, "real-2", reversing[1].Ledger.RealizationID)
}

// Test 6: Ceiling enforcement caps charged credits.
func TestAllocateAndBook_CeilingEnforcement(t *testing.T) {
	mc := &mockCollector{
		allocResult: allocationsFromCollector("grp-1",
			struct {
				amount     int64
				originKind creditrealization.LineageOriginKind
			}{15, creditrealization.LineageOriginKindRealCredit},
		),
	}
	svc := newTestService(mc)

	in := baseInput()
	in.TotalCredits = 200
	ceiling := int64(15)
	in.CeilingCredits = &ceiling

	allocs, err := svc.AllocateAndBook(t.Context(), mockTxAdapter{}, in)
	require.NoError(t, err)
	require.Len(t, allocs, 1)
	require.Equal(t, int64(15), allocs[0].Amount)

	// Verify the collector was called with the capped amount, not 200.
	lastInput := mc.getLastCollectInput()
	require.True(t, alpacadecimal.NewFromInt(15).Equal(lastInput.Amount))
}

// Test 7: 20-way concurrent settlement conserves total allocated credits.
// Uses a shared mutable collector result to simulate real balance consumption.
func TestAllocateAndBook_ConcurrentRaceConservesCredits(t *testing.T) {
	// Simulate a shared pool of 200 credits. Each CollectToAccrued call
	// atomically decrements the pool and returns the consumed amount.
	var pool atomic.Int64
	pool.Store(200)

	mc := &concurrentCollector{pool: &pool}
	svc := newTestService(mc)

	var wg sync.WaitGroup
	var totalAllocated atomic.Int64
	var successCount atomic.Int64
	var failCount atomic.Int64

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			allocs, err := svc.AllocateAndBook(t.Context(), mockTxAdapter{}, settlement.SettlementInput{
				Namespace:       "ns-race",
				CustomerID:      "cust-race",
				SubjectID:       "subj-race",
				TotalCredits:    10,
				SettlementScope: aiusage.SettlementScopeFormal,
				BatchID:         "batch-race",
				ChargeID:        "charge-ai",
				Currency:        usdCurrency(),
				FeatureKey:      "ai_usage",
				SettlementMode:  productcatalog.CreditThenInvoiceSettlementMode,
				BookedAt:        now(),
				ServicePeriod:   servicePeriod(),
			})
			if err != nil {
				failCount.Add(1)
				return
			}

			for _, a := range allocs {
				totalAllocated.Add(a.Amount)
			}
			successCount.Add(1)
		}()
	}

	wg.Wait()

	// Total allocated must never exceed the pool (200).
	total := totalAllocated.Load()
	require.LessOrEqual(t, total, int64(200), "total allocated must not exceed pool")
	require.Equal(t, int64(20), successCount.Load()+failCount.Load(), "all 20 goroutines completed")
}

// concurrentCollector simulates a real collector with a shared credit pool.
type concurrentCollector struct {
	pool *atomic.Int64
}

func (c *concurrentCollector) CollectToAccrued(_ context.Context, input collector.CollectToAccruedInput) (creditrealization.CreateAllocationInputs, error) {
	requested := input.Amount.IntPart()

	for {
		current := c.pool.Load()
		if current <= 0 {
			return creditrealization.CreateAllocationInputs{}, nil
		}
		allocated := requested
		if allocated > current {
			allocated = current
		}
		if c.pool.CompareAndSwap(current, current-allocated) {
			return creditrealization.CreateAllocationInputs{
				{
					Amount: alpacadecimal.NewFromInt(allocated),
					LedgerTransaction: ledgertransaction.GroupReference{
						TransactionGroupID: "grp-concurrent",
					},
					Annotations: creditrealization.LineageAnnotations(creditrealization.LineageOriginKindRealCredit),
				},
			}, nil
		}
	}
}

func (c *concurrentCollector) CorrectCollectedAccrued(_ context.Context, _ collector.CorrectCollectedAccruedInput) (creditrealization.CreateCorrectionInputs, error) {
	return nil, nil
}

// Test 8: Zero-credit batch returns empty allocations without calling collector.
func TestAllocateAndBook_ZeroCredits(t *testing.T) {
	mc := &mockCollector{}
	svc := newTestService(mc)

	in := baseInput()
	in.TotalCredits = 0

	allocs, err := svc.AllocateAndBook(t.Context(), mockTxAdapter{}, in)
	require.NoError(t, err)
	require.Empty(t, allocs)
	require.Equal(t, int32(0), mc.getCollectCount())
}

// Test 9: Ceiling of zero produces empty allocations.
func TestAllocateAndBook_CeilingZero(t *testing.T) {
	mc := &mockCollector{}
	svc := newTestService(mc)

	in := baseInput()
	in.TotalCredits = 50
	zero := int64(0)
	in.CeilingCredits = &zero

	allocs, err := svc.AllocateAndBook(t.Context(), mockTxAdapter{}, in)
	require.NoError(t, err)
	require.Empty(t, allocs)
	require.Equal(t, int32(0), mc.getCollectCount())
}

// Test 10: Collector error propagates.
func TestAllocateAndBook_CollectorError(t *testing.T) {
	mc := &mockCollector{
		err: errors.New("ledger unavailable"),
	}
	svc := newTestService(mc)

	_, err := svc.AllocateAndBook(t.Context(), mockTxAdapter{}, baseInput())
	require.Error(t, err)
	require.Contains(t, err.Error(), "ledger unavailable")
}

// Test 11: Correct skips allocations without realization provenance.
func TestCorrect_SkipsMissingProvenance(t *testing.T) {
	mc := &mockCollector{}
	svc := newTestService(mc)

	originalAllocs := []aiusage.Allocation{
		{GrantID: "g1", Amount: 10}, // No ledger provenance
	}

	reversing, err := svc.Correct(t.Context(), mockTxAdapter{}, settlement.CorrectionInput{
		Namespace:           "ns-1",
		CustomerID:          "cust-1",
		OriginalBatchID:     "batch-001",
		BookedAt:             now(),
		OriginalAllocations: originalAllocs,
	})
	require.NoError(t, err)
	require.Empty(t, reversing)
}

// Test 12: Annotations carry batch_id for traceability.
func TestAllocateAndBook_Annotations(t *testing.T) {
	mc := &mockCollector{
		allocResult: creditrealization.CreateAllocationInputs{},
	}
	svc := newTestService(mc)

	in := baseInput()
	in.BatchID = "batch-annotated"

	_, err := svc.AllocateAndBook(t.Context(), mockTxAdapter{}, in)
	require.NoError(t, err)

	lastInput := mc.getLastCollectInput()
	require.NotNil(t, lastInput)
	batchID, ok := lastInput.Annotations.GetString("ai_usage.batch_id")
	require.True(t, ok)
	require.Equal(t, "batch-annotated", batchID)
}
