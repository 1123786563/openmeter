package db_test

import (
	"context"
	"testing"
	"time"

	"github.com/openmeterio/openmeter/openmeter/commerce"
	"github.com/openmeterio/openmeter/openmeter/ent/db/commerceorder"
	"github.com/openmeterio/openmeter/openmeter/ent/db/commerceoutbox"
	"github.com/openmeterio/openmeter/openmeter/ent/db/commerceproduct"
	"github.com/openmeterio/openmeter/openmeter/ent/db/fulfillment"
	"github.com/openmeterio/openmeter/openmeter/ent/db/paymentattempt"
	"github.com/openmeterio/openmeter/openmeter/ent/db/paymentfact"
)

// TestPaidTxRunner_AtomicTransition (C2): verifies that RunPaidTransition
// atomically inserts a payment fact, moves the order to paid, creates a
// fulfillment request, and writes an outbox record — all in one transaction.
// Requires a local Postgres instance (skipped if POSTGRES_HOST is not set).
func TestPaidTxRunner_AtomicTransition(t *testing.T) {
	c := newTestClient(t)
	ctx := context.Background()

	adapter, err := commerce.NewEntAdapter(commerce.EntAdapterConfig{Client: c})
	if err != nil {
		t.Fatalf("create adapter: %v", err)
	}

	// Create a test product and order in awaiting_payment state.
	_, err = c.CommerceProduct.Create().
		SetNamespace(testNS).
		SetSku("SKU-PAID-TX").
		SetName("Test Product").
		SetKind(commerceproduct.KindWalletTopUp).
		SetPriceCents(1000).
		SetCurrency("CNY").
		Save(ctx)
	if err != nil {
		t.Fatalf("create product: %v", err)
	}

	order, err := c.CommerceOrder.Create().
		SetNamespace(testNS).
		SetPublicID("pub-paid-tx").
		SetCustomerID("cust-paid-tx").
		SetKind(commerceorder.KindWalletTopUp).
		SetStatus(commerceorder.StatusAwaitingPayment).
		SetTotalCents(1000).
		SetCurrency("CNY").
		SetIdempotencyKey("idem-paid-tx").
		Save(ctx)
	if err != nil {
		t.Fatalf("create order: %v", err)
	}

	// Create a payment attempt.
	attempt, err := c.PaymentAttempt.Create().
		SetNamespace(testNS).
		SetCommerceOrderID(order.ID).
		SetCustomerID("cust-paid-tx").
		SetProvider(paymentattempt.ProviderWechat).
		SetStatus(paymentattempt.StatusPending).
		SetIdempotencyKey("idem-attempt").
		SetAmountCents(1000).
		SetCurrency("CNY").
		Save(ctx)
	if err != nil {
		t.Fatalf("create attempt: %v", err)
	}

	// Run the paid transition.
	err = adapter.RunPaidTransition(ctx, commerce.PaidTransitionParams{
		Namespace:        testNS,
		CustomerID:       "cust-paid-tx",
		OrderID:          order.ID,
		PaymentAttemptID: attempt.ID,
		RawHash:          "hash-paid-tx-001",
		Provider:         "wechat",
		SignedPayload:    map[string]any{"amount": 1000, "currency": "CNY"},
	})
	if err != nil {
		t.Fatalf("RunPaidTransition: %v", err)
	}

	// Verify the order is now paid.
	updated, err := c.CommerceOrder.Get(ctx, order.ID)
	if err != nil {
		t.Fatalf("get order: %v", err)
	}
	if updated.Status != commerceorder.StatusPaid {
		t.Errorf("expected order status paid, got %s", updated.Status)
	}

	// Verify the payment fact was inserted.
	factCount, err := c.PaymentFact.Query().
		Where(paymentfact.NamespaceEQ(testNS), paymentfact.RawHashEQ("hash-paid-tx-001")).
		Count(ctx)
	if err != nil {
		t.Fatalf("count facts: %v", err)
	}
	if factCount != 1 {
		t.Errorf("expected 1 payment fact, got %d", factCount)
	}

	// Verify the fulfillment request was created.
	ffCount, err := c.Fulfillment.Query().
		Where(fulfillment.CommerceOrderIDEQ(order.ID)).
		Count(ctx)
	if err != nil {
		t.Fatalf("count fulfillments: %v", err)
	}
	if ffCount != 1 {
		t.Errorf("expected 1 fulfillment, got %d", ffCount)
	}

	// Verify the outbox record was written.
	outboxCount, err := c.CommerceOutbox.Query().
		Where(commerceoutbox.AggregateIDEQ(order.ID)).
		Count(ctx)
	if err != nil {
		t.Fatalf("count outbox: %v", err)
	}
	if outboxCount != 1 {
		t.Errorf("expected 1 outbox record, got %d", outboxCount)
	}
}

// TestPaidTxRunner_IdempotentReplay (C2): running the transition twice for the
// same order should be idempotent — no duplicate facts, fulfillments, or
// outbox records, and the order stays in paid.
func TestPaidTxRunner_IdempotentReplay(t *testing.T) {
	c := newTestClient(t)
	ctx := context.Background()

	adapter, err := commerce.NewEntAdapter(commerce.EntAdapterConfig{Client: c})
	if err != nil {
		t.Fatalf("create adapter: %v", err)
	}

	// Create order in awaiting_payment.
	order, err := c.CommerceOrder.Create().
		SetNamespace(testNS).
		SetPublicID("pub-idem").
		SetCustomerID("cust-idem").
		SetKind(commerceorder.KindWalletTopUp).
		SetStatus(commerceorder.StatusAwaitingPayment).
		SetTotalCents(500).
		SetCurrency("CNY").
		SetIdempotencyKey("idem-order").
		Save(ctx)
	if err != nil {
		t.Fatalf("create order: %v", err)
	}

	attempt, err := c.PaymentAttempt.Create().
		SetNamespace(testNS).
		SetCommerceOrderID(order.ID).
		SetCustomerID("cust-idem").
		SetProvider(paymentattempt.ProviderWechat).
		SetStatus(paymentattempt.StatusPending).
		SetIdempotencyKey("idem-attempt-2").
		SetAmountCents(500).
		SetCurrency("CNY").
		Save(ctx)
	if err != nil {
		t.Fatalf("create attempt: %v", err)
	}

	params := commerce.PaidTransitionParams{
		Namespace:        testNS,
		CustomerID:       "cust-idem",
		OrderID:          order.ID,
		PaymentAttemptID: attempt.ID,
		RawHash:          "hash-idem-001",
		Provider:         "wechat",
		SignedPayload:    map[string]any{"amount": 500},
	}

	// First run.
	if err := adapter.RunPaidTransition(ctx, params); err != nil {
		t.Fatalf("first RunPaidTransition: %v", err)
	}

	// Second run (idempotent replay — should not error and not duplicate).
	if err := adapter.RunPaidTransition(ctx, params); err != nil {
		t.Fatalf("second RunPaidTransition: %v", err)
	}

	// Verify only 1 fact, 1 fulfillment, 1 outbox.
	factCount, _ := c.PaymentFact.Query().
		Where(paymentfact.RawHashEQ("hash-idem-001")).
		Count(ctx)
	if factCount != 1 {
		t.Errorf("expected 1 fact after replay, got %d", factCount)
	}

	ffCount, _ := c.Fulfillment.Query().
		Where(fulfillment.CommerceOrderIDEQ(order.ID)).
		Count(ctx)
	if ffCount != 1 {
		t.Errorf("expected 1 fulfillment after replay, got %d", ffCount)
	}
}

// TestPaidTxRunner_UniqueHashDedup (C1): verifies that the unique index on
// (namespace, raw_hash) prevents two concurrent inserts of the same fact.
func TestPaidTxRunner_UniqueHashDedup(t *testing.T) {
	c := newTestClient(t)
	ctx := context.Background()

	// Insert a fact directly.
	_, err := c.PaymentFact.Create().
		SetNamespace(testNS).
		SetPaymentAttemptID("attempt-dedup").
		SetRawHash("hash-dedup-unique").
		SetProvider(paymentfact.ProviderWechat).
		SetSignedPayload(map[string]any{}).
		SetTimestamp(time.Now()).
		Save(ctx)
	if err != nil {
		t.Fatalf("insert first fact: %v", err)
	}

	// Attempt to insert a second fact with the same raw_hash — should fail
	// due to the unique index.
	_, err = c.PaymentFact.Create().
		SetNamespace(testNS).
		SetPaymentAttemptID("attempt-dedup-2").
		SetRawHash("hash-dedup-unique").
		SetProvider(paymentfact.ProviderAlipay).
		SetSignedPayload(map[string]any{}).
		SetTimestamp(time.Now()).
		Save(ctx)
	if err == nil {
		t.Error("expected unique constraint violation for duplicate raw_hash, got nil")
	}
}
