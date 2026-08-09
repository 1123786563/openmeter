package commerce

import (
	"sync"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"

	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ent/db/commerceorder"
	"github.com/openmeterio/openmeter/openmeter/ent/db/commerceoutbox"
	"github.com/openmeterio/openmeter/openmeter/ent/db/fulfillment"
	"github.com/openmeterio/openmeter/openmeter/ent/db/paymentattempt"
	"github.com/openmeterio/openmeter/openmeter/ent/db/paymentfact"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/stretchr/testify/require"
)

func TestPaidTransitionConcurrent(t *testing.T) {
	testDB := testutils.InitPostgresDB(t, testutils.PostgresDBStateEntMigrated)
	defer testDB.Close(t)
	client := testDB.EntDriver.Client()
	adapter, err := NewEntAdapter(EntAdapterConfig{Client: client, Logger: testutils.NewLogger(t)})
	require.NoError(t, err)

	order, attempt := createPaidTransitionFixture(t, client, "concurrent")
	now := time.Date(2026, 8, 9, 10, 0, 0, 0, time.UTC)
	params := PaidTransitionParams{
		Namespace:         "default",
		CustomerID:        "customer-concurrent",
		OrderID:           order.ID,
		PaymentAttemptID:  attempt.ID,
		Provider:          "wechat",
		ProviderOrderID:   "provider-order-concurrent",
		ProviderPaymentID: "provider-payment-concurrent",
		ProviderEventID:   "provider-event-concurrent",
		MerchantID:        "merchant-concurrent",
		ApplicationID:     "application-concurrent",
		AmountMinor:       100,
		Currency:          "CNY",
		Success:           true,
		RawHash:           "raw-hash-concurrent",
		Timestamp:         now,
		SignedPayload:     map[string]any{"trade_state": "SUCCESS"},
	}

	type transitionResult struct {
		result PaidTransitionResult
		err    error
	}
	results := make(chan transitionResult, 2)
	var ready sync.WaitGroup
	ready.Add(2)
	start := make(chan struct{})
	for range 2 {
		go func() {
			ready.Done()
			<-start
			result, err := adapter.RunPaidTransition(t.Context(), params)
			results <- transitionResult{result: result, err: err}
		}()
	}
	ready.Wait()
	close(start)

	alreadyPaid := 0
	for range 2 {
		result := <-results
		require.NoError(t, result.err)
		if result.result.AlreadyPaid {
			alreadyPaid++
		}
	}
	require.Equal(t, 1, alreadyPaid)

	require.Equal(t, 1, countPaymentFacts(t, client, "provider-event-concurrent"))
	fulfillmentCount, err := client.Fulfillment.Query().
		Where(fulfillment.NamespaceEQ("default"), fulfillment.CommerceOrderIDEQ(order.ID)).
		Count(t.Context())
	require.NoError(t, err)
	require.Equal(t, 1, fulfillmentCount)
	outboxCount, err := client.CommerceOutbox.Query().
		Where(commerceoutbox.NamespaceEQ("default"), commerceoutbox.AggregateIDEQ(order.ID)).
		Count(t.Context())
	require.NoError(t, err)
	require.Equal(t, 1, outboxCount)

	savedAttempt, err := client.PaymentAttempt.Get(t.Context(), attempt.ID)
	require.NoError(t, err)
	require.Equal(t, paymentattempt.StatusSucceeded, savedAttempt.Status)
	savedOrder, err := client.CommerceOrder.Get(t.Context(), order.ID)
	require.NoError(t, err)
	require.Equal(t, commerceorder.StatusPaid, savedOrder.Status)

	savedFact, err := client.PaymentFact.Query().
		Where(paymentfact.NamespaceEQ("default"), paymentfact.ProviderEventIDEQ("provider-event-concurrent")).
		Only(t.Context())
	require.NoError(t, err)
	require.Equal(t, "provider-order-concurrent", savedFact.ProviderOrderID)
	require.Equal(t, "provider-payment-concurrent", *savedFact.ProviderPaymentID)
	require.Equal(t, "merchant-concurrent", *savedFact.MerchantID)
	require.Equal(t, "application-concurrent", *savedFact.ApplicationID)
	require.Equal(t, int64(100), savedFact.AmountMinor)
	require.Equal(t, "CNY", savedFact.Currency)
	require.True(t, savedFact.Success)
}

func TestSetProviderIDsEmpty(t *testing.T) {
	testDB := testutils.InitPostgresDB(t, testutils.PostgresDBStateEntMigrated)
	defer testDB.Close(t)
	client := testDB.EntDriver.Client()
	adapter, err := NewEntAdapter(EntAdapterConfig{Client: client, Logger: testutils.NewLogger(t)})
	require.NoError(t, err)

	_, first := createPaidTransitionFixture(t, client, "empty-payment-id-1")
	_, second := createPaidTransitionFixture(t, client, "empty-payment-id-2")

	_, err = adapter.SetPaymentAttemptProviderIDs(t.Context(), "default", first.ID, "provider-order-empty-1", "", "qr-code-1")
	require.NoError(t, err)
	_, err = adapter.SetPaymentAttemptProviderIDs(t.Context(), "default", second.ID, "provider-order-empty-2", "", "qr-code-2")
	require.NoError(t, err)

	first, err = client.PaymentAttempt.Get(t.Context(), first.ID)
	require.NoError(t, err)
	second, err = client.PaymentAttempt.Get(t.Context(), second.ID)
	require.NoError(t, err)
	require.Nil(t, first.ProviderPaymentID)
	require.Nil(t, second.ProviderPaymentID)
}

func TestInsertPaymentFactProviderEventDedupDifferentRawHashNonSuccess(t *testing.T) {
	testDB := testutils.InitPostgresDB(t, testutils.PostgresDBStateEntMigrated)
	defer testDB.Close(t)
	client := testDB.EntDriver.Client()
	adapter, err := NewEntAdapter(EntAdapterConfig{Client: client, Logger: testutils.NewLogger(t)})
	require.NoError(t, err)

	_, attempt := createPaidTransitionFixture(t, client, "provider-event-dedup")
	now := time.Date(2026, 8, 9, 11, 0, 0, 0, time.UTC)
	first := PaymentFactWire{
		ID:              ulid.Make().String(),
		Namespace:       "default",
		AttemptID:       attempt.ID,
		Provider:        "wechat",
		ProviderOrderID: "provider-order-event-dedup",
		ProviderEventID: "provider-event-dedup",
		AmountMinor:     100,
		Currency:        "CNY",
		Success:         false,
		RawHash:         "raw-hash-event-dedup-first",
		SignedPayload:   map[string]any{"trade_state": "NOTPAY"},
		Timestamp:       now,
		CreatedAt:       now,
	}
	saved, fresh, err := adapter.InsertPaymentFact(t.Context(), first)
	require.NoError(t, err)
	require.True(t, fresh)

	second := first
	second.ID = ulid.Make().String()
	second.RawHash = "raw-hash-event-dedup-second"
	replayed, fresh, err := adapter.InsertPaymentFact(t.Context(), second)
	require.NoError(t, err)
	require.False(t, fresh)
	require.Equal(t, saved.ID, replayed.ID)
	require.Equal(t, first.RawHash, replayed.RawHash)
	require.False(t, replayed.Success)
	require.Equal(t, 1, countPaymentFacts(t, client, first.ProviderEventID))
}

func createPaidTransitionFixture(t *testing.T, client *db.Client, suffix string) (*db.CommerceOrder, *db.PaymentAttempt) {
	t.Helper()
	order, err := client.CommerceOrder.Create().
		SetNamespace("default").
		SetCustomerID("customer-" + suffix).
		SetKind(commerceorder.KindWalletTopUp).
		SetStatus(commerceorder.StatusAwaitingPayment).
		SetTotalCents(100).
		SetCurrency("CNY").
		SetIdempotencyKey("order-idempotency-" + suffix).
		Save(t.Context())
	require.NoError(t, err)

	attempt, err := client.PaymentAttempt.Create().
		SetNamespace("default").
		SetCommerceOrderID(order.ID).
		SetCustomerID("customer-" + suffix).
		SetProvider(paymentattempt.ProviderWechat).
		SetStatus(paymentattempt.StatusPending).
		SetIdempotencyKey("attempt-idempotency-" + suffix).
		SetAmountCents(100).
		SetCurrency("CNY").
		Save(t.Context())
	require.NoError(t, err)

	return order, attempt
}

func countPaymentFacts(t *testing.T, client *db.Client, providerEventID string) int {
	t.Helper()
	count, err := client.PaymentFact.Query().
		Where(paymentfact.NamespaceEQ("default"), paymentfact.ProviderEventIDEQ(providerEventID)).
		Count(t.Context())
	require.NoError(t, err)
	return count
}
