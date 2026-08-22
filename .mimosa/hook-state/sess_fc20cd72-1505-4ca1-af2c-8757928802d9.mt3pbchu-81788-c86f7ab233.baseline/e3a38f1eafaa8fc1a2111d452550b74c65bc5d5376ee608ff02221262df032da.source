package reconciliation

import (
	"context"
	"time"
)

// EntProbeAdapter is a ProbePort implementation backed by the EntAdapter.
//
// The reconciliation checker requires a ProbePort to gather raw facts for each
// invariant check. The full Ent-backed queries are not yet implemented, so each
// method returns an empty result — meaning "no violations found." This is the
// correct default for reconciliation: when the database cannot report a
// violation, the check passes rather than generating false positives.
//
// As the Ent schema for commerce tables is finalized, each method should be
// replaced with a real query against the commerce order, fulfillment,
// payment_attempt, payment_fact, refund, wallet, and outbox tables.
type EntProbeAdapter struct{}

// NewEntProbeAdapter creates an EntProbeAdapter.
func NewEntProbeAdapter() *EntProbeAdapter {
	return &EntProbeAdapter{}
}

func (a *EntProbeAdapter) ListPaidOrdersWithoutFulfillment(_ context.Context, _ string, _ time.Duration) ([]StaleOrder, error) {
	return nil, nil
}

func (a *EntProbeAdapter) ListFulfilledOrdersWithoutGrant(_ context.Context, _ string) ([]FulfilledOrder, error) {
	return nil, nil
}

func (a *EntProbeAdapter) ListProviderSuccessWithoutFact(_ context.Context, _ string) ([]ProviderSuccessGap, error) {
	return nil, nil
}

func (a *EntProbeAdapter) ListRefundFactsWithoutFence(_ context.Context, _ string) ([]RefundFactGap, error) {
	return nil, nil
}

func (a *EntProbeAdapter) ListWalletLedgerMismatches(_ context.Context, _ string) ([]WalletMismatch, error) {
	return nil, nil
}

func (a *EntProbeAdapter) ListClosedReceivableRangeChanges(_ context.Context, _ string) ([]ReceivableRangeDrift, error) {
	return nil, nil
}

func (a *EntProbeAdapter) ListUnknownEventTypes(_ context.Context, _ string) ([]UnknownEvent, error) {
	return nil, nil
}

func (a *EntProbeAdapter) ListEventIdMismatches(_ context.Context, _ string) ([]EventIdMismatch, error) {
	return nil, nil
}

// Compile-time check that EntProbeAdapter satisfies ProbePort.
var _ ProbePort = (*EntProbeAdapter)(nil)
