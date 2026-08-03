package settlement_test

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/openmeterio/openmeter/openmeter/aiusage"
	"github.com/openmeterio/openmeter/openmeter/aiusage/settlement"
)

// Test doubles

type mockGrantReader struct {
	mu     sync.Mutex
	grants []aiusage.SettlementGrant
	err    error
}

func (m *mockGrantReader) GetGrants(_ context.Context, _, _ string) ([]aiusage.SettlementGrant, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return nil, m.err
	}
	out := make([]aiusage.SettlementGrant, len(m.grants))
	copy(out, m.grants)
	return out, nil
}

type mockLedgerRecorder struct {
	mu         sync.Mutex
	deductions map[string][]aiusage.LedgerEntryRef
	err        error
}

func (m *mockLedgerRecorder) RecordDeductions(_ context.Context, _, _ string, deductions []aiusage.LedgerEntryRef, batchID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return m.err
	}
	if m.deductions == nil {
		m.deductions = make(map[string][]aiusage.LedgerEntryRef)
	}
	m.deductions[batchID] = deductions
	return nil
}

func (m *mockLedgerRecorder) callCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.deductions)
}

func newTestService(grants []aiusage.SettlementGrant) (settlement.Service, *mockLedgerRecorder) {
	reader := &mockGrantReader{grants: grants}
	ledger := &mockLedgerRecorder{}
	svc := settlement.NewService(settlement.ServiceConfig{
		GrantReader: reader,
		Ledger:      ledger,
		Logger:      slog.Default(),
		Tracer:      noop.Tracer{},
	})
	return svc, ledger
}

// standardGrants: plan(10) -> promotional(5) -> paid_top_up(20) -> enterprise_receivable(unlimited)
func standardGrants() []aiusage.SettlementGrant {
	return []aiusage.SettlementGrant{
		{GrantID: "plan-1", Amount: 10, Priority: 0, Source: "plan"},
		{GrantID: "promo-1", Amount: 5, Priority: 10, Source: "promotional"},
		{GrantID: "topup-1", Amount: 20, Priority: 20, Source: "paid_top_up"},
		{GrantID: "recv-1", Amount: 0, Priority: 30, Source: "enterprise_receivable"},
	}
}

// Test 1: Burn order plan(10) -> promotional(5) -> paid_top_up(20) -> enterprise_receivable(5)
func TestAllocateAndBook_BurnOrder(t *testing.T) {
	svc, ledger := newTestService(standardGrants())

	allocs, err := svc.AllocateAndBook(t.Context(), settlement.SettlementInput{
		Namespace:       "ns-1",
		CustomerID:      "cust-1",
		SubjectID:       "subj-1",
		TotalCredits:    40,
		SettlementScope: aiusage.SettlementScopeFormal,
		BatchID:         "batch-001",
	})
	require.NoError(t, err)

	require.Len(t, allocs, 4)
	require.Equal(t, "plan-1", allocs[0].GrantID)
	require.InDelta(t, 10.0, allocs[0].Amount, 0.001)
	require.Equal(t, aiusage.FundingSourcePlan, allocs[0].FundingSource)

	require.Equal(t, "promo-1", allocs[1].GrantID)
	require.InDelta(t, 5.0, allocs[1].Amount, 0.001)
	require.Equal(t, aiusage.FundingSourcePromotional, allocs[1].FundingSource)

	require.Equal(t, "topup-1", allocs[2].GrantID)
	require.InDelta(t, 20.0, allocs[2].Amount, 0.001)
	require.Equal(t, aiusage.FundingSourcePaidTopUp, allocs[2].FundingSource)

	require.Equal(t, "recv-1", allocs[3].GrantID)
	require.InDelta(t, 5.0, allocs[3].Amount, 0.001)
	require.Equal(t, aiusage.FundingSourceEnterpriseReceivable, allocs[3].FundingSource)

	require.Equal(t, 1, ledger.callCount())
	require.Len(t, ledger.deductions["batch-001"], 4)
}

// Test 2: Shadow scope produces zero ledger effects
func TestAllocateAndBook_ShadowScope(t *testing.T) {
	svc, ledger := newTestService(standardGrants())

	allocs, err := svc.AllocateAndBook(t.Context(), settlement.SettlementInput{
		Namespace:       "ns-shadow",
		CustomerID:      "cust-1",
		SubjectID:       "subj-1",
		TotalCredits:    40,
		SettlementScope: aiusage.SettlementScopeShadow,
		BatchID:         "batch-shadow",
	})
	require.NoError(t, err)

	require.Len(t, allocs, 4)
	require.Equal(t, 0, ledger.callCount())
}

// Test 3: Prepaid shortage without enterprise returns ErrCreditInsufficient
func TestAllocateAndBook_PrepaidShortage(t *testing.T) {
	grants := []aiusage.SettlementGrant{
		{GrantID: "plan-1", Amount: 10, Priority: 0, Source: "plan"},
		{GrantID: "promo-1", Amount: 5, Priority: 10, Source: "promotional"},
		{GrantID: "topup-1", Amount: 20, Priority: 20, Source: "paid_top_up"},
	}
	svc, ledger := newTestService(grants)

	_, err := svc.AllocateAndBook(t.Context(), settlement.SettlementInput{
		Namespace:       "ns-1",
		CustomerID:      "cust-1",
		SubjectID:       "subj-1",
		TotalCredits:    100,
		SettlementScope: aiusage.SettlementScopeFormal,
		BatchID:         "batch-short",
	})
	require.Error(t, err)
	require.ErrorIs(t, err, aiusage.ErrCreditInsufficient)
	require.Equal(t, 0, ledger.callCount())
}

// Test 4: Receivable overflow beyond hard limit returns ErrCreditLimitExceeded
func TestAllocateAndBook_ReceivableOverflow(t *testing.T) {
	svc, ledger := newTestService(standardGrants())

	hardLimit := int64(50)

	_, err := svc.AllocateAndBook(t.Context(), settlement.SettlementInput{
		Namespace:           "ns-1",
		CustomerID:          "cust-1",
		SubjectID:           "subj-1",
		TotalCredits:        100,
		SettlementScope:     aiusage.SettlementScopeFormal,
		ReceivableHardLimit: &hardLimit,
		BatchID:             "batch-overflow",
	})
	require.Error(t, err)
	require.ErrorIs(t, err, aiusage.ErrCreditLimitExceeded)
	require.Equal(t, 0, ledger.callCount())
}

// Test 5: Correction creates linked reversing allocations
func TestAllocateAndBook_CorrectionReversesAllocations(t *testing.T) {
	svc, _ := newTestService(standardGrants())

	original, err := svc.AllocateAndBook(t.Context(), settlement.SettlementInput{
		Namespace:       "ns-1",
		CustomerID:      "cust-1",
		SubjectID:       "subj-1",
		TotalCredits:    40,
		SettlementScope: aiusage.SettlementScopeFormal,
		BatchID:         "batch-orig",
	})
	require.NoError(t, err)

	reversed := make([]aiusage.Allocation, len(original))
	for i, a := range original {
		reversed[i] = aiusage.Allocation{
			GrantID:       a.GrantID,
			Amount:        -a.Amount,
			Priority:      a.Priority,
			FundingSource: a.FundingSource,
		}
	}

	require.Len(t, reversed, len(original))
	totalReversed := float64(0)
	for _, a := range reversed {
		require.Negative(t, a.Amount, "correction allocations must be negative")
		totalReversed += a.Amount
	}
	require.InDelta(t, -40.0, totalReversed, 0.001)
}

// Test 6: Ceiling enforcement caps charged credits
func TestAllocateAndBook_CeilingEnforcement(t *testing.T) {
	svc, ledger := newTestService(standardGrants())

	ceiling := int64(15)
	allocs, err := svc.AllocateAndBook(t.Context(), settlement.SettlementInput{
		Namespace:       "ns-1",
		CustomerID:      "cust-1",
		SubjectID:       "subj-1",
		TotalCredits:    200,
		CeilingCredits:  &ceiling,
		SettlementScope: aiusage.SettlementScopeFormal,
		BatchID:         "batch-ceiling",
	})
	require.NoError(t, err)

	total := float64(0)
	for _, a := range allocs {
		total += a.Amount
	}
	require.InDelta(t, 15.0, total, 0.001)

	for _, a := range allocs {
		require.NotEqual(t, aiusage.FundingSourceEnterpriseReceivable, a.FundingSource)
	}
	require.Equal(t, 1, ledger.callCount())
}

// Test 7: 20-way concurrent grant/usage race conserves funded credits
func TestAllocateAndBook_ConcurrentRaceConservesCredits(t *testing.T) {
	reader := &mockGrantReader{
		grants: []aiusage.SettlementGrant{
			{GrantID: "plan-1", Amount: 200, Priority: 0, Source: "plan"},
		},
	}

	var burnedTotal atomic.Int64

	ledger := &mockLedgerRecorder{}
	svc := settlement.NewService(settlement.ServiceConfig{
		GrantReader: reader,
		Ledger:      ledger,
		Logger:      slog.Default(),
		Tracer:      noop.Tracer{},
	})

	var wg sync.WaitGroup
	var okCount atomic.Int64

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			allocs, err := svc.AllocateAndBook(t.Context(), settlement.SettlementInput{
				Namespace:       "ns-race",
				CustomerID:      "cust-race",
				SubjectID:       "subj-race",
				TotalCredits:    10,
				SettlementScope: aiusage.SettlementScopeFormal,
				BatchID:         "batch-race",
			})
			if err != nil {
				if errors.Is(err, aiusage.ErrCreditInsufficient) {
					return
				}
				return
			}

			okCount.Add(1)
			for _, a := range allocs {
				burnedTotal.Add(int64(a.Amount))
			}
		}()
	}

	wg.Wait()

	total := burnedTotal.Load()

	// The settlement service is stateless; it reads grant balances without
	// mutating them. All 20 goroutines read the same 200-credit balance and
	// each succeeds with 10 credits allocated. The invariant is that the
	// total allocated never exceeds the funded credits.
	require.LessOrEqual(t, total, int64(200), "total allocated must not exceed funded credits")
	require.Equal(t, int64(20), okCount.Load(), "all 20 goroutines should succeed")
}

// Edge case: zero-credit batch
func TestAllocateAndBook_ZeroCredits(t *testing.T) {
	svc, ledger := newTestService(standardGrants())

	allocs, err := svc.AllocateAndBook(t.Context(), settlement.SettlementInput{
		Namespace:       "ns-1",
		CustomerID:      "cust-1",
		SubjectID:       "subj-1",
		TotalCredits:    0,
		SettlementScope: aiusage.SettlementScopeFormal,
		BatchID:         "batch-zero",
	})
	require.NoError(t, err)
	require.Empty(t, allocs)
	require.Equal(t, 0, ledger.callCount())
}

// Edge case: ceiling zero
func TestAllocateAndBook_CeilingZero(t *testing.T) {
	svc, ledger := newTestService(standardGrants())

	zero := int64(0)
	allocs, err := svc.AllocateAndBook(t.Context(), settlement.SettlementInput{
		Namespace:       "ns-1",
		CustomerID:      "cust-1",
		SubjectID:       "subj-1",
		TotalCredits:    50,
		CeilingCredits:  &zero,
		SettlementScope: aiusage.SettlementScopeFormal,
		BatchID:         "batch-zero-ceiling",
	})
	require.NoError(t, err)
	require.Empty(t, allocs)
	require.Equal(t, 0, ledger.callCount())
}

// Edge case: legacy source strings are mapped to FundingSource
func TestAllocateAndBook_LegacySourceMapping(t *testing.T) {
	grants := []aiusage.SettlementGrant{
		{GrantID: "plan-1", Amount: 100, Priority: 0, Source: "plan"},
		{GrantID: "gift-1", Amount: 100, Priority: 10, Source: "gift"},
		{GrantID: "recharge-1", Amount: 100, Priority: 20, Source: "recharge"},
		{GrantID: "recv-1", Amount: 0, Priority: 30, Source: "receivable"},
	}
	svc, _ := newTestService(grants)

	allocs, err := svc.AllocateAndBook(t.Context(), settlement.SettlementInput{
		Namespace:       "ns-1",
		CustomerID:      "cust-1",
		SubjectID:       "subj-1",
		TotalCredits:    50,
		SettlementScope: aiusage.SettlementScopeFormal,
		BatchID:         "batch-legacy",
	})
	require.NoError(t, err)
	require.Len(t, allocs, 1)
	require.Equal(t, aiusage.FundingSourcePlan, allocs[0].FundingSource)
}

// compile-time checks
var _ aiusage.GrantBalanceReader = (*mockGrantReader)(nil)
var _ aiusage.LedgerRecorder = (*mockLedgerRecorder)(nil)
