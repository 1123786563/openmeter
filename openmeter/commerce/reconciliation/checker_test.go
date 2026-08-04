package reconciliation

import (
	"context"
	"testing"
	"time"
)

type mockProbe struct {
	stale       []StaleOrder
	fulfilled   []FulfilledOrder
	provGaps    []ProviderSuccessGap
	refundGaps  []RefundFactGap
	walletDrift []WalletMismatch
	receivable  []ReceivableRangeDrift
	unknown     []UnknownEvent
	mismatches  []EventIdMismatch
}

func (m *mockProbe) ListPaidOrdersWithoutFulfillment(context.Context, string, time.Duration) ([]StaleOrder, error) {
	return m.stale, nil
}
func (m *mockProbe) ListFulfilledOrdersWithoutGrant(context.Context, string) ([]FulfilledOrder, error) {
	return m.fulfilled, nil
}
func (m *mockProbe) ListProviderSuccessWithoutFact(context.Context, string) ([]ProviderSuccessGap, error) {
	return m.provGaps, nil
}
func (m *mockProbe) ListRefundFactsWithoutFence(context.Context, string) ([]RefundFactGap, error) {
	return m.refundGaps, nil
}
func (m *mockProbe) ListWalletLedgerMismatches(context.Context, string) ([]WalletMismatch, error) {
	return m.walletDrift, nil
}
func (m *mockProbe) ListClosedReceivableRangeChanges(context.Context, string) ([]ReceivableRangeDrift, error) {
	return m.receivable, nil
}
func (m *mockProbe) ListUnknownEventTypes(context.Context, string) ([]UnknownEvent, error) {
	return m.unknown, nil
}
func (m *mockProbe) ListEventIdMismatches(context.Context, string) ([]EventIdMismatch, error) {
	return m.mismatches, nil
}

func TestChecker_NoFindings(t *testing.T) {
	c := New(Config{Probe: &mockProbe{}})
	report := c.Run(context.Background(), "ns-1")
	if report.HasErrors() {
		t.Errorf("expected no errors, got %d findings", len(report.Findings))
	}
	if report.ChecksRun != 8 {
		t.Errorf("expected 8 checks, got %d", report.ChecksRun)
	}
}

func TestChecker_PaidWithoutFulfillment(t *testing.T) {
	c := New(Config{
		Probe: &mockProbe{
			stale: []StaleOrder{{OrderID: "ord-1", CustomerID: "cust-1"}},
		},
	})
	report := c.Run(context.Background(), "ns-1")
	if !report.HasErrors() {
		t.Error("expected errors for stale paid orders")
	}
	found := false
	for _, f := range report.Findings {
		if f.Check == CheckPaidWithoutFulfillment && f.Severity == SeverityError {
			found = true
		}
	}
	if !found {
		t.Error("expected a paid_without_fulfillment error finding")
	}
}

func TestChecker_WalletMismatch(t *testing.T) {
	c := New(Config{
		Probe: &mockProbe{
			walletDrift: []WalletMismatch{{
				CustomerID: "cust-1", WalletTotal: 1000, LedgerTotal: 900, Difference: 100,
			}},
		},
	})
	report := c.Run(context.Background(), "ns-1")
	if !report.HasErrors() {
		t.Error("expected errors for wallet mismatch")
	}
}

func TestChecker_UnknownEventTypes(t *testing.T) {
	c := New(Config{
		Probe: &mockProbe{
			unknown: []UnknownEvent{
				{OutboxID: "row-1", EventType: "order.created"}, // not in approved list
				{OutboxID: "row-2", EventType: "order.updated"}, // approved
			},
		},
	})
	report := c.Run(context.Background(), "ns-1")
	count := 0
	for _, f := range report.Findings {
		if f.Check == CheckUnknownEventTypes {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected 1 unknown event finding, got %d", count)
	}
}

func TestChecker_EventIdMismatch(t *testing.T) {
	c := New(Config{
		Probe: &mockProbe{
			mismatches: []EventIdMismatch{{OutboxID: "row-1", EventID: "evt-99"}},
		},
	})
	report := c.Run(context.Background(), "ns-1")
	if !report.HasErrors() {
		t.Error("expected errors for event ID mismatch")
	}
}
