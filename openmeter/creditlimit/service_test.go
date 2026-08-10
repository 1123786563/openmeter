package creditlimit

import (
	"context"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/currencies"
)

func TestRemainingAllowanceSubtractsReceivableAndSharedHolds(t *testing.T) {
	// The same result is used for every feature: feature-specific receivable
	// filtering would incorrectly let separate features spend the limit twice.
	limit := alpacadecimal.NewFromInt(100)
	receivable := alpacadecimal.NewFromInt(-30)
	held := alpacadecimal.NewFromInt(20)

	require.True(t, remainingAllowance(limit, receivable, held).Equal(alpacadecimal.NewFromInt(50)))
}

func TestRemainingFailsClosedWithoutActiveHoldReader(t *testing.T) {
	s := &service{}
	_, err := s.Remaining(context.Background(), RemainingInput{
		Namespace: "ns", CustomerID: "customer", Currency: testManagedCurrency(t), AsOf: time.Now(),
	})
	require.ErrorIs(t, err, ErrActiveHoldReaderUnavailable)
}

func testManagedCurrency(t *testing.T) currencies.CurrencyReference {
	t.Helper()
	currency, err := currencies.ParseCurrencyReference([]byte("custom|v1|CREDIT|01J00000000000000000000000|0"))
	require.NoError(t, err)
	return currency
}
