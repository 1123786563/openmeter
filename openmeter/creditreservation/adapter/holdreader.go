package adapter

import (
	"context"
	"fmt"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/creditreservation"
	"github.com/openmeterio/openmeter/openmeter/creditlimit"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	dbcreditreservation "github.com/openmeterio/openmeter/openmeter/ent/db/creditreservation"
)

// reservationHoldReader implements creditlimit.ActiveHoldReader by summing
// active enterprise holds from the credit_reservation table. It is used
// whenever reservations are enabled so that the credit-limit service accounts
// for outstanding authorizations when computing remaining enterprise allowance.
type reservationHoldReader struct {
	db *entdb.Client
}

// NewReservationHoldReader creates an ActiveHoldReader backed by the
// credit_reservation table.
func NewReservationHoldReader(db *entdb.Client) creditlimit.ActiveHoldReader {
	return &reservationHoldReader{db: db}
}

func (r *reservationHoldReader) ActiveHeldAmount(ctx context.Context, input creditlimit.ActiveHoldInput) (alpacadecimal.Decimal, error) {
	rows, err := r.db.CreditReservation.Query().Where(
		dbcreditreservation.NamespaceEQ(input.Namespace),
		dbcreditreservation.CustomerIDEQ(input.CustomerID),
		dbcreditreservation.StateIn(
			string(creditreservation.ReservationStateActive),
			string(creditreservation.ReservationStateExecuting),
			string(creditreservation.ReservationStateUnknown),
			string(creditreservation.ReservationStateManualReview),
		),
	).All(ctx)
	if err != nil {
		return alpacadecimal.Zero, fmt.Errorf("list active reservation holds: %w", err)
	}

	var held int64
	for _, row := range rows {
		if !row.Currency.Equal(input.Currency) {
			continue
		}
		held += row.EnterpriseHold
	}
	return alpacadecimal.NewFromInt(held), nil
}

var _ creditlimit.ActiveHoldReader = (*reservationHoldReader)(nil)
