package commerce

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/ent/db/paymentattempt"
	"github.com/openmeterio/openmeter/openmeter/testutils"
)

func TestResolvePaymentAuthorityForOrderRequiresSucceededAttempt(t *testing.T) {
	testDB := testutils.InitPostgresDB(t, testutils.PostgresDBStateEntMigrated)
	defer testDB.Close(t)
	client := testDB.EntDriver.Client()
	adapter, err := NewEntAdapter(EntAdapterConfig{Client: client, Logger: testutils.NewLogger(t)})
	require.NoError(t, err)

	order, _ := createPaidTransitionFixture(t, client, "refund-authority-pending")

	_, err = adapter.ResolvePaymentAuthorityForOrder(t.Context(), "default", order.ID)
	require.ErrorIs(t, err, ErrPaymentAttemptNotFound)

	provider, err := adapter.ResolvePaymentProviderForOrder(t.Context(), "default", order.ID)
	require.NoError(t, err)
	require.Equal(t, string(paymentattempt.ProviderWechat), provider)
}
