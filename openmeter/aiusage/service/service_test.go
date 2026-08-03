package service_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/aiusage"
	"github.com/openmeterio/openmeter/openmeter/aiusage/service"
	"github.com/openmeterio/openmeter/openmeter/currencies"
)

type mockProfileResolver struct {
	profile service.CustomerProfile
	err     error
}

func (m *mockProfileResolver) Resolve(_ context.Context, _, _ string) (service.CustomerProfile, error) {
	return m.profile, m.err
}

type mockScopeResolver struct {
	scope aiusage.SettlementScope
	err   error
}

func (m *mockScopeResolver) ResolveScope(_ context.Context, _, _ string) (aiusage.SettlementScope, error) {
	return m.scope, m.err
}

type mockAllocationFetcher struct {
	allocs []aiusage.Allocation
	err    error
}

func (m *mockAllocationFetcher) GetAllocations(_ context.Context, _, _, _ string) ([]aiusage.Allocation, error) {
	return m.allocs, m.err
}

func TestCustomerProfile_DefaultValues(t *testing.T) {
	p := service.CustomerProfile{
		ChargeID: "charge-1",
		Currency: currencies.NewCurrencyReference("USD"),
	}
	require.Equal(t, "charge-1", p.ChargeID)
}

func TestCorrectionInput_Validate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		in := aiusage.CorrectionInput{
			Namespace:       "ns-1",
			CustomerID:      "cust-1",
			SubjectID:       "subj-1",
			OriginalBatchID: "batch-001",
			TenantSeq:       2,
			PayloadHash:     "hash",
		}
		require.NoError(t, in.Validate())
	})

	t.Run("missing namespace", func(t *testing.T) {
		in := aiusage.CorrectionInput{
			CustomerID:      "cust-1",
			SubjectID:       "subj-1",
			OriginalBatchID: "batch-001",
			TenantSeq:       2,
			PayloadHash:     "hash",
		}
		require.Error(t, in.Validate())
	})

	t.Run("zero tenant_seq", func(t *testing.T) {
		in := aiusage.CorrectionInput{
			Namespace:       "ns-1",
			CustomerID:      "cust-1",
			SubjectID:       "subj-1",
			OriginalBatchID: "batch-001",
			TenantSeq:       0,
			PayloadHash:     "hash",
		}
		require.Error(t, in.Validate())
	})
}

func TestSettle_ValidationFailure(t *testing.T) {
	err := aiusage.IngestBatchInput{}.Validate()
	require.Error(t, err)
}

func TestSettlementScope_Validate(t *testing.T) {
	require.NoError(t, aiusage.SettlementScopeFormal.Validate())
	require.NoError(t, aiusage.SettlementScopeShadow.Validate())
	require.Error(t, aiusage.SettlementScope("invalid").Validate())
}

func TestAllocation_LedgerProvenance(t *testing.T) {
	a := aiusage.Allocation{
		Amount: 42,
		Ledger: aiusage.LedgerProvenance{
			TransactionGroupID: "grp-001",
			RealizationID:      "real-001",
			SortHint:           3,
		},
	}
	require.Equal(t, "grp-001", a.Ledger.TransactionGroupID)
	require.Equal(t, "real-001", a.Ledger.RealizationID)
	require.Equal(t, 3, a.Ledger.SortHint)
	require.Equal(t, int64(42), a.Amount)
}

var _ service.ScopeResolver = (*mockScopeResolver)(nil)
var _ service.CustomerProfileResolver = (*mockProfileResolver)(nil)
var _ service.AllocationFetcher = (*mockAllocationFetcher)(nil)

var (
	_ = errors.New
	_ = sync.Mutex{}
	_ = time.Now
)
