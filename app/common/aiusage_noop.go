package common

import (
	"context"
	"log/slog"
	"time"

	"github.com/openmeterio/openmeter/openmeter/aiusage/runtimeauthorization"
	"github.com/openmeterio/openmeter/openmeter/aiusage/signing"
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

// Unused import guard — keeps slog referenced for future logging additions.
var (
	_ *slog.Logger
	_ time.Duration
)
