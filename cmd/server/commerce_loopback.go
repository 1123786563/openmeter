package main

import (
	"context"

	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/commerce/refund"
)

// Loopback-only automatic-refund collaborators.
//
// Production wiring passes nil runtime dependencies and commerce wiring
// fails closed until real refund-fence collaborators (the WeKnora fence API)
// are assembled. When commerce.test.enabled is true, configuration
// validation already guarantees that every enabled provider endpoint is
// loopback HTTP and a >=24 char control token is set, so this stand-in only
// ever exists on the loopback protocol test stand. It must not be used for
// production.
type loopbackFenceClient struct{}

func (loopbackFenceClient) EstablishFence(context.Context, string, string, string) (refund.FenceResult, error) {
	return refund.FenceResult{Sequence: "loopback-fence-1", Established: true}, nil
}

func (loopbackFenceClient) ReleaseFence(context.Context, string, string, string, string) error {
	return nil
}

type loopbackCreditReverser struct{}

func (loopbackCreditReverser) ReverseCredits(_ context.Context, in refund.ReverseCreditsInput) (refund.ReverseCreditsResult, error) {
	return refund.ReverseCreditsResult{LedgerEntryID: "loopback-reversal-1", Credits: in.Credits}, nil
}

func loopbackTestRuntimeDependencies(cfg config.CommerceConfiguration) *commerceRuntimeDependencies {
	if !cfg.Test.Enabled {
		return nil
	}
	return &commerceRuntimeDependencies{
		RefundFence:          loopbackFenceClient{},
		RefundCreditReverser: loopbackCreditReverser{},
	}
}
