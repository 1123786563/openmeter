package adapter_test

import (
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/creditreservation"
	creditreservationadapter "github.com/openmeterio/openmeter/openmeter/creditreservation/adapter"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/customer"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ent/db/creditreservationoutbox"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

func newAdapter(t *testing.T) (creditreservationadapter.Adapter, *entdb.Client) {
	t.Helper()

	testDB := testutils.InitPostgresDB(t, testutils.PostgresDBStateAtlasMigrated)
	t.Cleanup(func() { testDB.Close(t) })

	dbClient := entdb.NewClient(entdb.Driver(testDB.EntDriver.Driver()))
	adp, err := creditreservationadapter.New(creditreservationadapter.Config{
		Client: dbClient,
		Logger: slog.Default(),
	})
	require.NoError(t, err)

	return adp, dbClient
}

func customerID() customer.CustomerID {
	return customer.CustomerID{Namespace: "test", ID: "customer-1"}
}

func reservationInput(key, hash string) creditreservationadapter.CreateReservationInput {
	expiresAt := time.Now().UTC().Add(time.Minute)
	return creditreservationadapter.CreateReservationInput{
		Reservation: creditreservation.Reservation{
			ID:          "reservation-1",
			Namespace:   "test",
			CustomerID:  "customer-1",
			Currency:    currencies.NewCurrencyReference(currencyx.Code("CREDIT")),
			State:       creditreservation.ReservationStateActive,
			RateVersion: "rate-v1",
			Lines: []creditreservation.RatedLine{{
				ResourceLine: creditreservation.ResourceLine{FeatureKey: "chat", ResourceCode: "llm.input", Quantity: 1},
				RateCardKey:  "chat-input",
				RateVersion:  "rate-v1",
				Credits:      10,
			}},
			TotalCredits: 10,
			ExpiresAt:    &expiresAt,
			CommandIdentity: creditreservation.CommandIdentity{
				IdempotencyKey: key,
				PayloadHash:    hash,
			},
		},
		SubjectID:    "subject-1",
		ClientCallID: "call-" + key,
		Operation:    "chat.completions.create",
		PrepaidHold:  10,
	}
}

func TestCreateReservationIsIdempotentByKeyAndHash(t *testing.T) {
	adp, _ := newAdapter(t)
	input := reservationInput("wk:reserve:c1", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")

	var first creditreservation.Reservation
	var created bool
	err := adp.WithCustomerLock(t.Context(), customerID(), func(tx creditreservationadapter.TxAdapter) (err error) {
		first, created, err = tx.CreateReservation(t.Context(), input)
		return err
	})
	require.NoError(t, err)
	require.True(t, created)

	var replay creditreservation.Reservation
	err = adp.WithCustomerLock(t.Context(), customerID(), func(tx creditreservationadapter.TxAdapter) error {
		var replayCreated bool
		var replayErr error
		replay, replayCreated, replayErr = tx.CreateReservation(t.Context(), input)
		require.False(t, replayCreated)
		return replayErr
	})
	require.NoError(t, err)
	require.Equal(t, first.ID, replay.ID)

	conflicting := input
	conflicting.Reservation.CommandIdentity.PayloadHash = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	err = adp.WithCustomerLock(t.Context(), customerID(), func(tx creditreservationadapter.TxAdapter) error {
		_, _, err := tx.CreateReservation(t.Context(), conflicting)
		return err
	})
	require.ErrorIs(t, err, creditreservation.ErrIdempotencyConflict)
}

func TestLifecycleCommandIdentityIsDurablePerKind(t *testing.T) {
	adp, _ := newAdapter(t)
	input := reservationInput("wk:reserve:lifecycle", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	err := adp.WithCustomerLock(t.Context(), customerID(), func(tx creditreservationadapter.TxAdapter) error {
		_, _, err := tx.CreateReservation(t.Context(), input)
		return err
	})
	require.NoError(t, err)
	id := input.Reservation.ID
	execute := creditreservation.CommandIdentity{IdempotencyKey: "wk:execute:1", PayloadHash: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}
	err = adp.WithCustomerLock(t.Context(), customerID(), func(tx creditreservationadapter.TxAdapter) error {
		replay, err := tx.EnsureLifecycleCommand(t.Context(), models.NamespacedID{Namespace: "test", ID: id}, "execute", execute)
		require.False(t, replay)
		return err
	})
	require.NoError(t, err)
	err = adp.WithCustomerLock(t.Context(), customerID(), func(tx creditreservationadapter.TxAdapter) error {
		replay, err := tx.EnsureLifecycleCommand(t.Context(), models.NamespacedID{Namespace: "test", ID: id}, "execute", execute)
		require.True(t, replay)
		return err
	})
	require.NoError(t, err)
	err = adp.WithCustomerLock(t.Context(), customerID(), func(tx creditreservationadapter.TxAdapter) error {
		_, err := tx.EnsureLifecycleCommand(t.Context(), models.NamespacedID{Namespace: "test", ID: id}, "execute", creditreservation.CommandIdentity{IdempotencyKey: execute.IdempotencyKey, PayloadHash: "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"})
		return err
	})
	require.ErrorIs(t, err, creditreservation.ErrIdempotencyConflict)
	err = adp.WithCustomerLock(t.Context(), customerID(), func(tx creditreservationadapter.TxAdapter) error {
		replay, err := tx.EnsureLifecycleCommand(t.Context(), models.NamespacedID{Namespace: "test", ID: id}, "unknown", execute)
		require.False(t, replay)
		return err
	})
	require.NoError(t, err)
}

func TestCreateReservationRejectsIntersectingIdempotencyAndCallIdentities(t *testing.T) {
	adp, _ := newAdapter(t)
	firstInput := reservationInput("wk:reserve:first", "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd")
	firstInput.Reservation.ID = "reservation-first"
	secondInput := reservationInput("wk:reserve:second", "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee")
	secondInput.Reservation.ID = "reservation-second"

	for _, input := range []creditreservationadapter.CreateReservationInput{firstInput, secondInput} {
		err := adp.WithCustomerLock(t.Context(), customerID(), func(tx creditreservationadapter.TxAdapter) error {
			_, created, err := tx.CreateReservation(t.Context(), input)
			require.True(t, created)
			return err
		})
		require.NoError(t, err)
	}

	intersection := firstInput
	intersection.Reservation.ID = "reservation-intersection"
	intersection.ClientCallID = secondInput.ClientCallID
	err := adp.WithCustomerLock(t.Context(), customerID(), func(tx creditreservationadapter.TxAdapter) error {
		_, _, err := tx.CreateReservation(t.Context(), intersection)
		return err
	})
	require.ErrorIs(t, err, creditreservation.ErrIdempotencyConflict)
}

func TestReservationAndUsageOutboxRollbackTogether(t *testing.T) {
	adp, client := newAdapter(t)
	input := reservationInput("wk:reserve:rollback", "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc")

	err := adp.WithCustomerLock(t.Context(), customerID(), func(tx creditreservationadapter.TxAdapter) error {
		reservation, _, err := tx.CreateReservation(t.Context(), input)
		if err != nil {
			return err
		}
		if err := tx.AppendUsageEvent(t.Context(), creditreservation.UsageEvent{
			EventID:       "usage-event-1",
			AggregateType: "credit_reservation",
			AggregateID:   reservation.ID,
			EventType:     "credit.reservation.created",
			Payload:       map[string]any{"reservation_id": reservation.ID},
		}); err != nil {
			return err
		}
		return errors.New("simulate outbox delivery failure after write")
	})
	require.Error(t, err)

	reservationCount, err := client.CreditReservation.Query().Count(t.Context())
	require.NoError(t, err)
	require.Zero(t, reservationCount)
	outboxCount, err := client.CreditReservationOutbox.Query().Where(creditreservationoutbox.EventID("usage-event-1")).Count(t.Context())
	require.NoError(t, err)
	require.Zero(t, outboxCount)
}
