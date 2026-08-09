package service

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/creditreservation"
)

// This catches a regression that creates a durable authorization before
// determining that neither prepaid funds nor an explicit limit can cover it.
func TestReserveRejectsBeforeCreatingRowWhenFundsAreInsufficient(t *testing.T) {
	_, err := splitFunding(9, nil, 10)

	require.ErrorIs(t, err, creditreservation.ErrInsufficientFunds)
}

// This catches accidentally treating a missing explicit credit limit as an
// unlimited receivable allowance.
func TestReserveDoesNotCreateReceivableWithoutExplicitLimit(t *testing.T) {
	_, err := splitFunding(9, nil, 10)

	require.ErrorIs(t, err, creditreservation.ErrInsufficientFunds)
}

func TestReserveFundingUsesPrepaidBeforeExplicitLimit(t *testing.T) {
	limit := int64(4)
	split, err := splitFunding(9, &limit, 10)

	require.NoError(t, err)
	require.Equal(t, int64(9), split.prepaid)
	require.Equal(t, int64(1), split.enterprise)
}
