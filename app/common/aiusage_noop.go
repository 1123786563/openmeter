package common

import (
	"context"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/aiusage"
	"github.com/openmeterio/openmeter/openmeter/aiusage/runtimeauthorization"
	"github.com/openmeterio/openmeter/openmeter/aiusage/signing"
)

// ---------------------------------------------------------------------------
// Noop implementations for aiusage.Service dependencies.
//
// These are Phase 1 placeholders. They let the process start successfully when
// ai_usage.enabled=true without the full pricing/cost/settlement DI graph.
// Production wiring will replace them with real implementations.
// ---------------------------------------------------------------------------

// noopRateCardResolver returns a default rate card entry (1 credit per unit).
type noopRateCardResolver struct{}

func (noopRateCardResolver) Resolve(_ context.Context, namespace, customerID string, resource aiusage.ResourceCode, provider, model string, _ time.Time) (aiusage.CustomerRateCardEntry, error) {
	cust := customerID
	p := provider
	m := model
	return aiusage.CustomerRateCardEntry{
		Namespace:       namespace,
		CustomerID:      &cust,
		ResourceCode:    resource,
		Provider:        &p,
		Model:           &m,
		PricePerUnitCNY: alpacadecimal.NewFromInt(1),
		CreditRate:      1,
		EffectiveFrom:   time.Now().Add(-24 * time.Hour),
	}, nil
}

// noopCostResolver returns a zero-cost snapshot.
type noopCostResolver struct{}

func (noopCostResolver) ResolveCost(_ context.Context, _, _, _ string, _ aiusage.ResourceCode, _ int64) (aiusage.CostSnapshot, error) {
	return aiusage.CostSnapshot{
		Currency: "USD",
		Amount:   alpacadecimal.NewFromInt(0),
		Source:   "noop",
	}, nil
}

// noopSettlementEngine settles batches without recording any grant deductions
// or ledger entries. It returns the batch as settled so the pipeline proceeds.
type noopSettlementEngine struct {
	logger *slog.Logger
}

func (e *noopSettlementEngine) Settle(_ context.Context, batch aiusage.AIUsageBatch, snapshots []aiusage.RatingSnapshot, ceiling *int64) (*aiusage.BatchSettlementResult, error) {
	totalCredits := int64(0)
	for _, s := range snapshots {
		totalCredits += s.Credits
	}
	if batch.BillingMode == aiusage.BillingModeBundle && ceiling != nil {
		totalCredits = *ceiling
	}
	if ceiling != nil && totalCredits > *ceiling {
		totalCredits = *ceiling
	}
	return &aiusage.BatchSettlementResult{
		BatchID:          batch.UsageBatchID,
		Status:           aiusage.BatchStatusSettled,
		TotalCredits:     totalCredits,
		RatingSnapshots:  snapshots,
		LedgerEntries:    []aiusage.LedgerEntryRef{},
		CoveredTenantSeq: batch.TenantSeq,
	}, nil
}

// ---------------------------------------------------------------------------
// Noop implementations for runtimeauthorization readers.
// ---------------------------------------------------------------------------

// noopBalanceReader returns a zero credit balance.
type noopBalanceReader struct{}

func (noopBalanceReader) ReadCreditBalance(_ context.Context, _, _ string) (runtimeauthorization.CreditBalance, error) {
	return runtimeauthorization.CreditBalance{}, nil
}

// noopSubscriptionReader returns empty subscription info with active status.
type noopSubscriptionReader struct{}

func (noopSubscriptionReader) ReadSubscription(_ context.Context, _, _ string) (runtimeauthorization.SubscriptionInfo, error) {
	return runtimeauthorization.SubscriptionInfo{
		SubscriptionStatus: "active",
		EntitlementCodes:   []string{},
	}, nil
}

// noopRatePackageReader returns an empty rate package.
type noopRatePackageReader struct{}

func (noopRatePackageReader) ReadRatePackage(_ context.Context, _, _ string) (runtimeauthorization.RatePackageSnapshot, error) {
	return runtimeauthorization.RatePackageSnapshot{
		Version: "noop-v1",
		Entries: []signing.SignedRateEntry{},
	}, nil
}

// noopCoveredSeqReader returns zero.
type noopCoveredSeqReader struct{}

func (noopCoveredSeqReader) ReadCoveredSeq(_ context.Context, _, _ string) (int64, error) {
	return 0, nil
}

// atomicSnapshotVersionProvider generates strictly increasing snapshot versions
// using an in-memory atomic counter. Production wiring should use a DB-backed
// sequence for cross-restart monotonicity.
type atomicSnapshotVersionProvider struct {
	counter atomic.Int64
}

func (p *atomicSnapshotVersionProvider) Next(_ context.Context) (int64, error) {
	return p.counter.Add(1), nil
}
