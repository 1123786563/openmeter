package aiusage

import (
	"context"
	"testing"

	"log/slog"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/openmeterio/openmeter/pkg/models"
)

// mockGrantReader implements GrantBalanceReader for testing.
type mockGrantReader struct {
	grants []SettlementGrant
	err    error
}

func (m *mockGrantReader) GetGrants(_ context.Context, _, _ string) ([]SettlementGrant, error) {
	return m.grants, m.err
}

// mockLedgerRecorder implements LedgerRecorder for testing.
type mockLedgerRecorder struct {
	deductions map[string][]LedgerEntryRef
	err        error
}

func (m *mockLedgerRecorder) RecordDeductions(_ context.Context, _, _ string, deductions []LedgerEntryRef, batchID string) error {
	if m.err != nil {
		return m.err
	}
	if m.deductions == nil {
		m.deductions = make(map[string][]LedgerEntryRef)
	}
	m.deductions[batchID] = deductions
	return nil
}

func newTestEngine(grants []SettlementGrant) (*settlementEngine, *mockLedgerRecorder) {
	reader := &mockGrantReader{grants: grants}
	ledger := &mockLedgerRecorder{}
	return &settlementEngine{
		BalanceReader: reader,
		Ledger:        ledger,
		Logger:        slog.Default(),
		Tracer:        noop.Tracer{},
	}, ledger
}

func testBatch(ceiling *int64) AIUsageBatch {
	return AIUsageBatch{
		ManagedModel:   models.ManagedModel{},
		Namespace:      "ns-1",
		CustomerID:     "cust-1",
		SubjectID:      "subj-1",
		UsageBatchID:   "batch-001",
		TenantSeq:      1,
		BillingMode:    BillingModeComponent,
		PayloadHash:    "hash",
		LineItems:      []UsageLineItem{validLineItem()},
		CeilingCredits: ceiling,
	}
}

func TestBurnGrants_SingleGrantSufficient(t *testing.T) {
	grants := []SettlementGrant{
		{GrantID: "plan-1", Amount: 1000, Priority: 0, Source: "plan"},
	}

	deductions, remainder := burnGrants(grants, 500)
	require.Equal(t, int64(0), remainder)
	require.Len(t, deductions, 1)
	require.Equal(t, "plan-1", deductions[0].GrantID)
	require.InDelta(t, 500.0, deductions[0].Amount, 0.001)
}

func TestBurnGrants_PriorityOrder(t *testing.T) {
	grants := []SettlementGrant{
		{GrantID: "recharge-1", Amount: 500, Priority: 20, Source: "recharge"},
		{GrantID: "gift-1", Amount: 300, Priority: 10, Source: "gift"},
		{GrantID: "plan-1", Amount: 200, Priority: 0, Source: "plan"},
	}

	deductions, remainder := burnGrants(grants, 450)
	require.Equal(t, int64(0), remainder)
	require.Len(t, deductions, 2)
	require.Equal(t, "plan-1", deductions[0].GrantID)
	require.Equal(t, "gift-1", deductions[1].GrantID)
	require.InDelta(t, 200.0, deductions[0].Amount, 0.001) // plan fully consumed
	require.InDelta(t, 250.0, deductions[1].Amount, 0.001) // gift partially consumed
}

func TestBurnGrants_AllExhausted(t *testing.T) {
	grants := []SettlementGrant{
		{GrantID: "plan-1", Amount: 100, Priority: 0, Source: "plan"},
		{GrantID: "gift-1", Amount: 100, Priority: 10, Source: "gift"},
	}

	deductions, remainder := burnGrants(grants, 300)
	require.Equal(t, int64(100), remainder) // 100 remaining after consuming all
	require.Len(t, deductions, 2)
}

func TestBurnGrants_EnterpriseReceivable(t *testing.T) {
	grants := []SettlementGrant{
		{GrantID: "plan-1", Amount: 100, Priority: 0, Source: "plan"},
		{GrantID: "receivable-1", Amount: 0, Priority: 30, Source: "receivable"},
	}

	deductions, remainder := burnGrants(grants, 500)
	require.Equal(t, int64(0), remainder) // receivable absorbs everything
	require.Len(t, deductions, 2)
	require.Equal(t, "plan-1", deductions[0].GrantID)
	require.Equal(t, "receivable-1", deductions[1].GrantID)
	require.InDelta(t, 400.0, deductions[1].Amount, 0.001) // 500 - 100 = 400
}

func TestBurnGrants_ZeroAmount(t *testing.T) {
	deductions, remainder := burnGrants(nil, 0)
	require.Equal(t, int64(0), remainder)
	require.Nil(t, deductions)
}

func TestSettlementEngine_Settle_SingleGrant(t *testing.T) {
	engine, ledger := newTestEngine([]SettlementGrant{
		{GrantID: "plan-1", Amount: 1000, Priority: 0, Source: "plan"},
	})

	batch := testBatch(nil)
	snapshots := []RatingSnapshot{
		{Credits: 100},
		{Credits: 50},
	}

	result, err := engine.Settle(t.Context(), batch, snapshots, nil)
	require.NoError(t, err)
	require.Equal(t, BatchStatusSettled, result.Status)
	require.Equal(t, int64(150), result.TotalCredits)
	require.Len(t, result.LedgerEntries, 1)
	require.Len(t, ledger.deductions, 1)
}

func TestSettlementEngine_Settle_CeilingEnforcement(t *testing.T) {
	engine, _ := newTestEngine([]SettlementGrant{
		{GrantID: "plan-1", Amount: 10000, Priority: 0, Source: "plan"},
	})

	ceiling := int64(100)
	batch := testBatch(&ceiling)
	snapshots := []RatingSnapshot{
		{Credits: 200},
	}

	result, err := engine.Settle(t.Context(), batch, snapshots, &ceiling)
	require.NoError(t, err)
	require.Equal(t, int64(100), result.TotalCredits) // capped at ceiling
}

func TestSettlementEngine_Settle_ZeroCredits(t *testing.T) {
	engine, _ := newTestEngine(nil)

	batch := testBatch(nil)
	snapshots := []RatingSnapshot{
		{Credits: 0},
	}

	result, err := engine.Settle(t.Context(), batch, snapshots, nil)
	require.NoError(t, err)
	require.Equal(t, int64(0), result.TotalCredits)
	require.Empty(t, result.LedgerEntries)
}

func TestSettlementEngine_Settle_EnterpriseReceivable(t *testing.T) {
	engine, ledger := newTestEngine([]SettlementGrant{
		{GrantID: "plan-1", Amount: 100, Priority: 0, Source: "plan"},
		{GrantID: "recv-1", Amount: 0, Priority: 30, Source: "receivable"},
	})

	batch := testBatch(nil)
	snapshots := []RatingSnapshot{
		{Credits: 500},
	}

	result, err := engine.Settle(t.Context(), batch, snapshots, nil)
	require.NoError(t, err)
	require.Equal(t, int64(500), result.TotalCredits)
	require.Len(t, result.LedgerEntries, 2)
	require.Len(t, ledger.deductions, 1)
}

func TestSettlementEngine_Settle_NonEnterprise_Insufficient(t *testing.T) {
	engine, _ := newTestEngine([]SettlementGrant{
		{GrantID: "plan-1", Amount: 100, Priority: 0, Source: "plan"},
		{GrantID: "gift-1", Amount: 50, Priority: 10, Source: "gift"},
	})

	batch := testBatch(nil)
	snapshots := []RatingSnapshot{
		{Credits: 500},
	}

	result, err := engine.Settle(t.Context(), batch, snapshots, nil)
	require.Error(t, err)
	require.Nil(t, result)
}

func TestSettlementEngine_Settle_PriorityOrder(t *testing.T) {
	engine, _ := newTestEngine([]SettlementGrant{
		{GrantID: "recharge-1", Amount: 1000, Priority: 20, Source: "recharge"},
		{GrantID: "gift-1", Amount: 1000, Priority: 10, Source: "gift"},
		{GrantID: "plan-1", Amount: 100, Priority: 0, Source: "plan"},
	})

	batch := testBatch(nil)
	snapshots := []RatingSnapshot{
		{Credits: 250},
	}

	result, err := engine.Settle(t.Context(), batch, snapshots, nil)
	require.NoError(t, err)
	require.Equal(t, int64(250), result.TotalCredits)

	// Plan should be fully consumed (100), gift covers the rest (150).
	require.Len(t, result.LedgerEntries, 2)
	require.Equal(t, "plan-1", result.LedgerEntries[0].GrantID)
	require.InDelta(t, 100.0, result.LedgerEntries[0].Amount, 0.001)
	require.Equal(t, "gift-1", result.LedgerEntries[1].GrantID)
	require.InDelta(t, 150.0, result.LedgerEntries[1].Amount, 0.001)
}
