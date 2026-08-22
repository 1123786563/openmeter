package adapter

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/creditreservation"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
)

func TestSelectReservationIdentityRejectsIntersectingRows(t *testing.T) {
	_, err := selectReservationIdentity(
		&entdb.CreditReservation{ID: "reservation-idempotency"},
		&entdb.CreditReservation{ID: "reservation-client-call"},
	)
	require.ErrorIs(t, err, creditreservation.ErrIdempotencyConflict)
}
