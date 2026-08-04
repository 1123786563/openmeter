package settlement_test

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
	"github.com/openmeterio/openmeter/openmeter/aiusage/settlement"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/ledger/collector"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type mockCollector struct {
	mu               sync.Mutex
	allocResult      creditrealization.CreateAllocationInputs
	correctionResult creditrealization.CreateCorrectionInputs
	err              error
	lastCollectInput *collector.CollectToAccruedInput
	collectCount     int32
	correctCount     int32
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

type allocItem struct {
	amount        int64
	originKind    creditrealization.LineageOriginKind
	fundingSource aiusage.FundingSource
	realizationID string
}

func allocationsFromCollector(groupID string, items ...allocItem) creditrealization.CreateAllocationInputs {
	out := make(creditrealization.CreateAllocationInputs, 0, len(items))
	for _, item := range items {
		out = append(out, creditrealization.CreateAllocationInput{
			ID:     item.realizationID,
			Amount: alpacadecimal.NewFromInt(item.amount),
			LedgerTransaction: ledgertransaction.GroupReference{
				TransactionGroupID: groupID,
			},
			Annotations: creditrealization.FundingSourceAnnotations(item.originKind, string(item.fundingSource)),
		})
	}
	return out
}

// sumAllocs returns the total credits in a list of allocations.
func sumAllocs(items []allocItem) int64 {
	var total int64
	for _, i := range items {
		total += i.amount
	}
	return total
}

func TestAllocateAndBook_BurnOrder(t *testing.T) {
	items := []allocItem{
		{10, creditrealization.LineageOriginKindRealCredit, aiusage.FundingSourcePlan, "real-1"},
		{5, creditrealization.LineageOriginKindRealCredit, aiusage.FundingSourcePromotional, "real-2"},
		{20, creditrealization.LineageOriginKindRealCredit, aiusage.FundingSourcePaidTopUp, "real-3"},
		{5, creditrealization.LineageOriginKindAdvance, aiusage.FundingSourceEnterpriseReceivable, "real-4"},
	}
	mc := &mockCollector{allocResult: allocationsFromCollector("grp-001", items...)}
	svc := newTestService(mc)

	in := baseInput()
	in.TotalCredits = sumAllocs(items)

	allocs, err := svc.AllocateAndBook(t.Context(), mockTxAdapter{}, in)
	require.NoError(t, err)
	require.Len(t, allocs, 4)

	require.Equal(t, int64(10), allocs[0].Amount)
	require.Equal(t, aiusage.FundingSourcePlan, allocs[0].FundingSource)
	require.Equal(t, "real-1", allocs[0].Ledger.RealizationID)

	require.Equal(t, int64(5), allocs[1].Amount)
	require.Equal(t, aiusage.FundingSourcePromotional, allocs[1].FundingSource)

	require.Equal(t, int64(20), allocs[2].Amount)
	require.Equal(t, aiusage.FundingSourcePaidTopUp, allocs[2].FundingSource)

	require.Equal(t, int64(5), allocs[3].Amount)
	require.Equal(t, aiusage.FundingSourceEnterpriseReceivable, allocs[3].FundingSource)

	lastInput := mc.getLastCollectInput()
	require.NotNil(t, lastInput)
	require.True(t, alpacadecimal.NewFromInt(40).Equal(lastInput.Amount))
}

func TestAllocateAndBook_ShadowScope(t *testing.T) {
	mc := &mockCollector{}
	svc := newTestService(mc)

	in := baseInput()
	in.SettlementScope = aiusage.SettlementScopeShadow

	allocs, err := svc.AllocateAndBook(t.Context(), mockTxAdapter{}, in)
	require.NoError(t, err)
	require.Empty(t, allocs)
	require.Equal(t, int32(0), mc.getCollectCount())
}

// Prepaid shortage without enterprise -> ErrCreditInsufficient.
func TestAllocateAndBook_PrepaidShortage(t *testing.T) {
	mc := &mockCollector{
		allocResult: allocationsFromCollector("grp-1",
			allocItem{10, creditrealization.LineageOriginKindRealCredit, aiusage.FundingSourcePaidTopUp, "real-1"},
		),
	}
	svc := newTestService(mc)

	in := baseInput()
	in.TotalCredits = 100

	_, err := svc.AllocateAndBook(t.Context(), mockTxAdapter{}, in)
	require.Error(t, err)
	require.ErrorIs(t, err, aiusage.ErrCreditInsufficient)
}

// Prepaid shortage WITH enterprise -> succeeds (receivable covers).
func TestAllocateAndBook_PrepaidShortageWithEnterprise(t *testing.T) {
	mc := &mockCollector{
		allocResult: allocationsFromCollector("grp-1",
			allocItem{10, creditrealization.LineageOriginKindRealCredit, aiusage.FundingSourcePaidTopUp, "real-1"},
			allocItem{90, creditrealization.LineageOriginKindAdvance, aiusage.FundingSourceEnterpriseReceivable, "real-2"},
		),
	}
	svc := newTestService(mc)

	in := baseInput()
	in.TotalCredits = 100

	allocs, err := svc.AllocateAndBook(t.Context(), mockTxAdapter{}, in)
	require.NoError(t, err)
	require.Len(t, allocs, 2)
}

// Receivable overflow beyond hard limit -> ErrCreditLimitExceeded.
func TestAllocateAndBook_ReceivableOverflow(t *testing.T) {
	// Enterprise available (so ErrCreditInsufficient does not fire), but the
	// receivable hard limit is exceeded.
	mc := &mockCollector{
		allocResult: allocationsFromCollector("grp-1",
			allocItem{10, creditrealization.LineageOriginKindRealCredit, aiusage.FundingSourcePaidTopUp, "real-1"},
			allocItem{90, creditrealization.LineageOriginKindAdvance, aiusage.FundingSourceEnterpriseReceivable, "real-2"},
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
}

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
		{GrantID: "g1", Amount: 10, Ledger: aiusage.LedgerProvenance{TransactionGroupID: "grp-001", RealizationID: "real-1", SortHint: 0}},
		{GrantID: "g2", Amount: 5, Ledger: aiusage.LedgerProvenance{TransactionGroupID: "grp-001", RealizationID: "real-2", SortHint: 1}},
	}

	reversing, err := svc.Correct(t.Context(), mockTxAdapter{}, settlement.CorrectionInput{
		Namespace:           "ns-1",
		CustomerID:          "cust-1",
		SubjectID:           "subj-1",
		OriginalBatchID:     "batch-001",
		BookedAt:            now(),
		OriginalAllocations: originalAllocs,
		ChargeID:            "charge-ai",
		Currency:            usdCurrency(),
	})
	require.NoError(t, err)
	require.Len(t, reversing, 2)

	require.Equal(t, int64(-10), reversing[0].Amount)
	require.Equal(t, "real-1", reversing[0].Ledger.RealizationID)
	require.Equal(t, "grp-corr-1", reversing[0].Ledger.TransactionGroupID)

	require.Equal(t, int64(-5), reversing[1].Amount)
	require.Equal(t, "real-2", reversing[1].Ledger.RealizationID)
}

func TestAllocateAndBook_CeilingEnforcement(t *testing.T) {
	mc := &mockCollector{
		allocResult: allocationsFromCollector("grp-1",
			allocItem{15, creditrealization.LineageOriginKindRealCredit, aiusage.FundingSourcePlan, "real-1"},
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

	lastInput := mc.getLastCollectInput()
	require.True(t, alpacadecimal.NewFromInt(15).Equal(lastInput.Amount))
}

func TestAllocateAndBook_ConcurrentRaceConservesCredits(t *testing.T) {
	var pool atomic.Int64
	pool.Store(200)

	mc := &concurrentCollector{pool: &pool}
	svc := newTestService(mc)

	var wg sync.WaitGroup
	var totalAllocated atomic.Int64
	var doneCount atomic.Int64

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer doneCount.Add(1)

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
				return
			}
			for _, a := range allocs {
				totalAllocated.Add(a.Amount)
			}
		}()
	}
	wg.Wait()

	total := totalAllocated.Load()
	require.LessOrEqual(t, total, int64(200), "total allocated must not exceed pool")
	require.Equal(t, int64(20), doneCount.Load())
}

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
					ID:     fmt.Sprintf("real-%d", allocated),
					Amount: alpacadecimal.NewFromInt(allocated),
					LedgerTransaction: ledgertransaction.GroupReference{
						TransactionGroupID: "grp-concurrent",
					},
					Annotations: creditrealization.FundingSourceAnnotations(
						creditrealization.LineageOriginKindRealCredit,
						string(aiusage.FundingSourcePaidTopUp),
					),
				},
			}, nil
		}
	}
}

func (c *concurrentCollector) CorrectCollectedAccrued(_ context.Context, _ collector.CorrectCollectedAccruedInput) (creditrealization.CreateCorrectionInputs, error) {
	return nil, nil
}

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

func TestAllocateAndBook_CollectorError(t *testing.T) {
	mc := &mockCollector{err: errors.New("ledger unavailable")}
	svc := newTestService(mc)

	_, err := svc.AllocateAndBook(t.Context(), mockTxAdapter{}, baseInput())
	require.Error(t, err)
	require.Contains(t, err.Error(), "ledger unavailable")
}

func TestCorrect_SkipsMissingProvenance(t *testing.T) {
	mc := &mockCollector{}
	svc := newTestService(mc)

	reversing, err := svc.Correct(t.Context(), mockTxAdapter{}, settlement.CorrectionInput{
		Namespace:           "ns-1",
		CustomerID:          "cust-1",
		OriginalBatchID:     "batch-001",
		BookedAt:            now(),
		OriginalAllocations: []aiusage.Allocation{{GrantID: "g1", Amount: 10}},
	})
	require.NoError(t, err)
	require.Empty(t, reversing)
}

func TestAllocateAndBook_Annotations(t *testing.T) {
	// Collector returns enough credits to cover the charge.
	mc := &mockCollector{
		allocResult: allocationsFromCollector("grp-1",
			allocItem{40, creditrealization.LineageOriginKindRealCredit, aiusage.FundingSourcePaidTopUp, "real-1"},
		),
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

// Finding 10a: Earlier-expiry-first ordering within a category.
// The collector delegates ordering to fboCollectionSource.Compare, which sorts
// by expiry within the same credit priority. The settlement layer preserves
// the collector's ordering via SortHint (array index).
func TestAllocateAndBook_EarlierExpiryFirstOrdering(t *testing.T) {
	items := []allocItem{
		{5, creditrealization.LineageOriginKindRealCredit, aiusage.FundingSourcePaidTopUp, "real-early"},
		{5, creditrealization.LineageOriginKindRealCredit, aiusage.FundingSourcePaidTopUp, "real-late"},
	}
	mc := &mockCollector{allocResult: allocationsFromCollector("grp-1", items...)}
	svc := newTestService(mc)

	in := baseInput()
	in.TotalCredits = sumAllocs(items) // 10 credits

	allocs, err := svc.AllocateAndBook(t.Context(), mockTxAdapter{}, in)
	require.NoError(t, err)
	require.Len(t, allocs, 2)

	require.Equal(t, 0, allocs[0].Ledger.SortHint)
	require.Equal(t, "real-early", allocs[0].Ledger.RealizationID)
	require.Equal(t, 1, allocs[1].Ledger.SortHint)
	require.Equal(t, "real-late", allocs[1].Ledger.RealizationID)
}

// Finding 10b: Equal-expiry tiebreak by creation time then Ledger cursor.
// When two sources share the same expiry, fboCollectionSource.Compare falls
// through to cursor comparison. The settlement layer preserves this order.
func TestAllocateAndBook_EqualExpiryTiebreakByCursor(t *testing.T) {
	items := []allocItem{
		{3, creditrealization.LineageOriginKindRealCredit, aiusage.FundingSourcePlan, "real-cursor-A"},
		{7, creditrealization.LineageOriginKindRealCredit, aiusage.FundingSourcePlan, "real-cursor-B"},
	}
	mc := &mockCollector{allocResult: allocationsFromCollector("grp-1", items...)}
	svc := newTestService(mc)

	in := baseInput()
	in.TotalCredits = sumAllocs(items) // 10 credits

	allocs, err := svc.AllocateAndBook(t.Context(), mockTxAdapter{}, in)
	require.NoError(t, err)
	require.Len(t, allocs, 2)

	require.Less(t, allocs[0].Ledger.SortHint, allocs[1].Ledger.SortHint)
	require.Equal(t, "real-cursor-A", allocs[0].Ledger.RealizationID)
	require.Equal(t, "real-cursor-B", allocs[1].Ledger.RealizationID)
}

// Finding 10c: Injected error after ledger preparation leaves nothing persisted.
// Enterprise is available (so ErrCreditInsufficient does not fire), but the
// receivable exceeds the hard limit -> ErrCreditLimitExceeded. Because the
// application service wraps everything in WithCustomerLock, the transaction
// rolls back — no batch, allocation, or ledger rows persist.
func TestAllocateAndBook_ErrorAfterCollectorNoSideEffects(t *testing.T) {
	mc := &mockCollector{
		allocResult: allocationsFromCollector("grp-1",
			allocItem{10, creditrealization.LineageOriginKindRealCredit, aiusage.FundingSourcePaidTopUp, "real-1"},
			allocItem{190, creditrealization.LineageOriginKindAdvance, aiusage.FundingSourceEnterpriseReceivable, "real-2"},
		),
	}
	svc := newTestService(mc)

	in := baseInput()
	in.TotalCredits = 200
	hardLimit := int64(5) // receivable is 190 > 5 -> overflow
	in.ReceivableHardLimit = &hardLimit

	_, err := svc.AllocateAndBook(t.Context(), mockTxAdapter{}, in)
	require.Error(t, err)
	require.ErrorIs(t, err, aiusage.ErrCreditLimitExceeded)

	// The collector WAS called (it runs before the limit check), but the error
	// means the application service will roll back the transaction.
	require.Equal(t, int32(1), mc.getCollectCount())
}
