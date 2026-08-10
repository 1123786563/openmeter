package service

import (
	"context"
	"testing"
	"time"

	decimal "github.com/alpacahq/alpacadecimal"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
	"github.com/openmeterio/openmeter/openmeter/creditreservation"
	"github.com/openmeterio/openmeter/openmeter/ledger/collector"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/stretchr/testify/require"
)

func TestRateActualUsageUsesPersistedReservationRateSnapshot(t *testing.T) {
	reserved := []creditreservation.RatedLine{{
		ResourceLine: creditreservation.ResourceLine{FeatureKey: "chat", ResourceCode: "llm_input_tokens", Quantity: 100},
		RateCardKey:  "tokens-v1", RateVersion: "snapshot-v1", Credits: 10,
		Snapshot: creditreservation.RateSnapshot{UnitAmount: decimal.NewFromFloat(0.1), UnitPriceSet: true},
	}}
	actual, credits, err := rateActualUsage(reserved, []creditreservation.ResourceLine{{FeatureKey: "chat", ResourceCode: "llm_input_tokens", Quantity: 50}})
	require.NoError(t, err)
	require.Equal(t, int64(5), credits)
	require.Len(t, actual, 1)
	require.Equal(t, "snapshot-v1", actual[0].RateVersion)
	require.Equal(t, int64(50), actual[0].Quantity)
}

func TestRateActualUsageRejectsLineWithoutPersistedSnapshot(t *testing.T) {
	reserved := []creditreservation.RatedLine{{
		ResourceLine: creditreservation.ResourceLine{FeatureKey: "chat", ResourceCode: "llm_input_tokens", Quantity: 100},
		RateCardKey:  "tokens-v1", RateVersion: "snapshot-v1", Credits: 10,
	}}
	_, _, err := rateActualUsage(reserved, []creditreservation.ResourceLine{{FeatureKey: "chat", ResourceCode: "llm_input_tokens", Quantity: 50}})
	require.ErrorIs(t, err, creditreservation.ErrRateNotFound)
}

func TestRateActualUsageAppliesPersistedUnitConfig(t *testing.T) {
	reserved := []creditreservation.RatedLine{{
		ResourceLine: creditreservation.ResourceLine{FeatureKey: "chat", ResourceCode: "llm_input_tokens", Quantity: 2_000},
		RateCardKey:  "tokens-v1", RateVersion: "snapshot-v1", Credits: 2,
		Snapshot: creditreservation.RateSnapshot{
			UnitAmount:   decimal.NewFromInt(1),
			UnitPriceSet: true,
			UnitConfig: &productcatalog.UnitConfig{
				Operation:        productcatalog.UnitConfigOperationDivide,
				ConversionFactor: decimal.NewFromInt(1_000),
				Rounding:         productcatalog.UnitConfigRoundingModeCeiling,
			},
		},
	}}
	_, credits, err := rateActualUsage(reserved, []creditreservation.ResourceLine{{FeatureKey: "chat", ResourceCode: "llm_input_tokens", Quantity: 1_001}})
	require.NoError(t, err)
	require.Equal(t, int64(2), credits)
}

func TestRateActualUsageRejectsMismatchedDimensions(t *testing.T) {
	reserved := []creditreservation.RatedLine{{
		ResourceLine: creditreservation.ResourceLine{FeatureKey: "chat", ResourceCode: "llm_input_tokens", Quantity: 100, Dimensions: map[string]string{"region": "us-east-1"}},
		RateCardKey:  "tokens-v1", RateVersion: "snapshot-v1", Credits: 10,
		Snapshot: creditreservation.RateSnapshot{UnitAmount: decimal.NewFromFloat(0.1), UnitPriceSet: true},
	}}
	_, _, err := rateActualUsage(reserved, []creditreservation.ResourceLine{{FeatureKey: "chat", ResourceCode: "llm_input_tokens", Quantity: 50, Dimensions: map[string]string{"region": "eu-west-1"}}})
	require.ErrorIs(t, err, creditreservation.ErrRateNotFound)
}

func TestSettleCapsOpenMeterRatedActualUsageAtReservationCeiling(t *testing.T) {
	store := &memoryAdapter{rows: map[string]creditreservation.Reservation{}}
	reservation := creditreservation.Reservation{
		ID: "reservation", Namespace: "ns", CustomerID: "customer", State: creditreservation.ReservationStateExecuting,
		TotalCredits: 10, PrepaidHold: 10,
		Lines: []creditreservation.RatedLine{{
			ResourceLine: creditreservation.ResourceLine{FeatureKey: "chat", ResourceCode: "tokens", Quantity: 100},
			RateCardKey:  "tokens-v1", RateVersion: "snapshot-v1", Credits: 10,
			Snapshot: creditreservation.RateSnapshot{UnitAmount: decimal.NewFromFloat(0.1), UnitPriceSet: true},
		}},
	}
	store.rows[reservation.ID] = reservation
	svc, err := New(Config{Adapter: store, Prices: fixedPrice{credits: 10}, Collector: fixedCollector{amount: 10}, SettlementCollector: settlingCollector{}})
	require.NoError(t, err)

	settled, err := svc.Settle(t.Context(), creditreservation.SettleInput{
		ID:              models.NamespacedID{Namespace: reservation.Namespace, ID: reservation.ID},
		CommandIdentity: creditreservation.CommandIdentity{IdempotencyKey: "settle", PayloadHash: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"},
		ActualLines:     []creditreservation.ResourceLine{{FeatureKey: "chat", ResourceCode: "tokens", Quantity: 200}},
		SettledAt:       time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC),
	})
	require.NoError(t, err)
	require.Equal(t, int64(10), settled.SettledCredits)
	require.Len(t, settled.ActualLines, 1)
	require.Equal(t, int64(200), settled.ActualLines[0].Quantity)
	require.Len(t, store.events, 1)
	require.Equal(t, int64(20), store.events[0].Payload["rated_actual_credits"])
	require.Equal(t, true, store.events[0].Payload["actual_over_ceiling"])
}

type settlingCollector struct{}

func (settlingCollector) CollectToAccrued(_ context.Context, input collector.CollectToAccruedInput) (creditrealization.CreateAllocationInputs, error) {
	return creditrealization.CreateAllocationInputs{{
		Amount: input.Amount,
		LedgerTransaction: ledgertransaction.GroupReference{
			TransactionGroupID: "settlement-group",
		},
	}}, nil
}

func (settlingCollector) CorrectCollectedAccrued(context.Context, collector.CorrectCollectedAccruedInput) (creditrealization.CreateCorrectionInputs, error) {
	return nil, nil
}
