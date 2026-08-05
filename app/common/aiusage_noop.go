package common

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/samber/mo"

	"github.com/openmeterio/openmeter/openmeter/aiusage/runtimeauthorization"
	"github.com/openmeterio/openmeter/openmeter/aiusage/signing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger/customerbalance"
)

// ---------------------------------------------------------------------------
// Noop implementations for runtimeauthorization readers.
//
// These remain Phase 1 placeholders: they let the process start when
// ai_usage.enabled=true without requiring the full balance/subscription/rate-
// package services in the DI graph. Production wiring will replace them with
// real implementations backed by the billing, subscription, and credit-grant
// stacks.
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

// ---------------------------------------------------------------------------
// Real balance reader backed by CustomerBalanceFacade.
// ---------------------------------------------------------------------------

// ledgerBalanceReader queries the ledger-backed customer balance facade to
// return the actual spendable credit balance for a billing customer.
type ledgerBalanceReader struct {
	facade *customerbalance.Facade
}

func (r ledgerBalanceReader) ReadCreditBalance(ctx context.Context, namespace, customerID string) (runtimeauthorization.CreditBalance, error) {
	if r.facade == nil {
		return runtimeauthorization.CreditBalance{}, nil
	}

	custID := customer.CustomerID{
		Namespace: namespace,
		ID:        customerID,
	}

	if err := custID.Validate(); err != nil {
		return runtimeauthorization.CreditBalance{}, fmt.Errorf("validate customer ID: %w", err)
	}

	now := time.Now().UTC()
	balances, err := r.facade.GetBalances(ctx, customerbalance.GetBalancesInput{
		CustomerID:    custID,
		FeatureFilter: mo.Option[creditpurchase.FeatureFilters]{},
		AsOf:          &now,
	})
	if err != nil {
		return runtimeauthorization.CreditBalance{}, fmt.Errorf("get balances: %w", err)
	}

	var total int64
	for _, b := range balances {
		f, _ := b.Balance.Settled().Float64()
		total += int64(f)
	}

	if total < 0 {
		total = 0
	}

	return runtimeauthorization.CreditBalance{
		SpendableCredits:           total,
		EnterpriseAvailableCredits: total,
	}, nil
}

// Compile-time interface check
var _ runtimeauthorization.SubjectBalanceReader = ledgerBalanceReader{}

// Unused import guard — keeps slog referenced for future logging additions.
var (
	_ *slog.Logger
	_ time.Duration
)
