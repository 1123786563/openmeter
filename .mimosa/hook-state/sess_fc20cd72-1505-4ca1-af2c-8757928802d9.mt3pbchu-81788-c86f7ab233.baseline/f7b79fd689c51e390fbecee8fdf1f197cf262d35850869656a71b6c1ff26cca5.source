package adapter

import (
	"testing"

	decimal "github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/creditreservation"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
)

func TestMapReservationReloadsRateSnapshotAndActualLines(t *testing.T) {
	rated := []creditreservation.RatedLine{{
		ResourceLine: creditreservation.ResourceLine{FeatureKey: "chat", ResourceCode: "tokens", Quantity: 1_000},
		RateCardKey:  "tokens-v1", RateVersion: "snapshot-v1", Credits: 1,
		Snapshot: creditreservation.RateSnapshot{
			UnitAmount:   decimal.NewFromInt(1),
			UnitPriceSet: true,
			UnitConfig: &productcatalog.UnitConfig{
				Operation:        productcatalog.UnitConfigOperationDivide,
				ConversionFactor: decimal.NewFromInt(1_000),
			},
		},
	}}
	actual := []creditreservation.RatedLine{{
		ResourceLine: creditreservation.ResourceLine{FeatureKey: "chat", ResourceCode: "tokens", Quantity: 500},
		RateCardKey:  "tokens-v1", RateVersion: "snapshot-v1", Credits: 1, Snapshot: rated[0].Snapshot,
	}}
	ratedRaw, err := marshalRatedLines(rated)
	require.NoError(t, err)
	actualRaw, err := marshalRatedLines(actual)
	require.NoError(t, err)

	reservation, err := mapReservation(&entdb.CreditReservation{
		ID: "reservation", Namespace: "ns", CustomerID: "customer", State: string(creditreservation.ReservationStateSettled),
		RatedLines: ratedRaw, ActualLines: actualRaw,
	})
	require.NoError(t, err)
	require.Equal(t, rated, reservation.Lines)
	require.Equal(t, actual, reservation.ActualLines)
}
