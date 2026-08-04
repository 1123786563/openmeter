package db_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"entgo.io/ent/dialect"
	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ent/db/commerceorder"
	"github.com/openmeterio/openmeter/openmeter/ent/db/commerceproduct"
	"github.com/openmeterio/openmeter/openmeter/ent/db/enttest"
	"github.com/openmeterio/openmeter/openmeter/ent/db/fulfillment"
	"github.com/openmeterio/openmeter/openmeter/ent/db/paymentattempt"
)

const testNS = "test-namespace"

// newTestClient connects to a local PostgreSQL instance and runs Ent
// auto-migration to create all tables. Each test truncates commerce tables
// before running. Set POSTGRES_HOST to enable; skipped otherwise.
func newTestClient(t *testing.T) *db.Client {
	t.Helper()

	host := os.Getenv("POSTGRES_HOST")
	if host == "" {
		t.Skip("POSTGRES_HOST not set; skipping database invariant tests")
	}
	port := os.Getenv("POSTGRES_PORT")
	if port == "" {
		port = "5432"
	}
	pass := os.Getenv("POSTGRES_PASSWORD")
	if pass == "" {
		pass = "postgres"
	}

	dsn := fmt.Sprintf("host=%s port=%s user=postgres password=%s dbname=commerce_test sslmode=disable", host, port, pass)

	c := enttest.Open(t, dialect.Postgres, dsn)

	// Truncate all commerce tables for test isolation using the Ent driver.
	commerceTables := []string{
		"external_invoice_refs", "offline_payments", "receivable_periods",
		"receivable_accounts", "refund_facts", "refund_requests",
		"fulfillments", "payment_facts", "payment_attempts",
		"commerce_order_lines", "commerce_orders", "commerce_products",
	}
	for _, tbl := range commerceTables {
		_, err := c.ExecContext(context.Background(),
			fmt.Sprintf("TRUNCATE TABLE %s RESTART IDENTITY CASCADE", tbl))
		if err != nil {
			// Table may not exist on first migration; ignore.
			t.Logf("truncate %s: %v", tbl, err)
		}
	}

	return c
}

// --- Idempotency key uniqueness ---

func TestOrderIdempotencyKeyUnique(t *testing.T) {
	ctx := context.Background()
	c := newTestClient(t)
	defer c.Close()

	base := func() *db.CommerceOrderCreate {
		return c.CommerceOrder.Create().
			SetNamespace(testNS).
			SetCustomerID("cust-1").
			SetKind(commerceorder.KindPlanPurchase).
			SetTotalCents(1000).
			SetCurrency("CNY").
			SetIdempotencyKey("idem-order-1")
	}

	if _, err := base().Save(ctx); err != nil {
		t.Fatalf("first order: %v", err)
	}
	if _, err := base().Save(ctx); err == nil {
		t.Fatal("duplicate idempotency_key should fail")
	}

	// Different customer, same idempotency key: allowed.
	if _, err := base().SetCustomerID("cust-2").Save(ctx); err != nil {
		t.Fatalf("different customer same key: %v", err)
	}
}

func TestPaymentAttemptIdempotencyKeyUnique(t *testing.T) {
	ctx := context.Background()
	c := newTestClient(t)
	defer c.Close()

	order := createOrderX(ctx, t, c, "idem-pay-attempt")

	base := func() *db.PaymentAttemptCreate {
		return c.PaymentAttempt.Create().
			SetNamespace(testNS).
			SetCommerceOrderID(order.ID).
			SetCustomerID("cust-1").
			SetProvider(paymentattempt.ProviderWechat).
			SetIdempotencyKey("pay-idem-1").
			SetAmountCents(1000)
	}

	if _, err := base().Save(ctx); err != nil {
		t.Fatalf("first attempt: %v", err)
	}
	if _, err := base().Save(ctx); err == nil {
		t.Fatal("duplicate payment idempotency_key should fail")
	}
}

func TestRefundRequestIdempotencyKeyUnique(t *testing.T) {
	ctx := context.Background()
	c := newTestClient(t)
	defer c.Close()

	order := createOrderX(ctx, t, c, "idem-refund-req")

	base := func() *db.RefundRequestCreate {
		return c.RefundRequest.Create().
			SetNamespace(testNS).
			SetCommerceOrderID(order.ID).
			SetCustomerID("cust-1").
			SetAmountCents(500).
			SetIdempotencyKey("refund-idem-1")
	}

	if _, err := base().Save(ctx); err != nil {
		t.Fatalf("first refund: %v", err)
	}
	if _, err := base().Save(ctx); err == nil {
		t.Fatal("duplicate refund idempotency_key should fail")
	}
}

// --- Provider uniqueness (partial unique indexes) ---

func TestPaymentAttemptProviderOrderIDUnique(t *testing.T) {
	ctx := context.Background()
	c := newTestClient(t)
	defer c.Close()

	order1 := createOrderX(ctx, t, c, "idem-prov-order-a")
	order2 := createOrderX(ctx, t, c, "idem-prov-order-b")

	// Two attempts with the same provider + provider_order_id should fail.
	c.PaymentAttempt.Create().
		SetNamespace(testNS).
		SetCommerceOrderID(order1.ID).
		SetCustomerID("cust-1").
		SetProvider(paymentattempt.ProviderWechat).
		SetIdempotencyKey("prov-pay-1").
		SetAmountCents(1000).
		SetProviderOrderID("wx-order-123").
		SaveX(ctx)

	_, err := c.PaymentAttempt.Create().
		SetNamespace(testNS).
		SetCommerceOrderID(order2.ID).
		SetCustomerID("cust-1").
		SetProvider(paymentattempt.ProviderWechat).
		SetIdempotencyKey("prov-pay-2").
		SetAmountCents(2000).
		SetProviderOrderID("wx-order-123").
		Save(ctx)
	if err == nil {
		t.Fatal("duplicate (provider, provider_order_id) should fail")
	}

	// NULL provider_order_id should not conflict.
	c.PaymentAttempt.Create().
		SetNamespace(testNS).
		SetCommerceOrderID(order1.ID).
		SetCustomerID("cust-1").
		SetProvider(paymentattempt.ProviderWechat).
		SetIdempotencyKey("prov-pay-3").
		SetAmountCents(3000).
		SaveX(ctx)
	c.PaymentAttempt.Create().
		SetNamespace(testNS).
		SetCommerceOrderID(order2.ID).
		SetCustomerID("cust-1").
		SetProvider(paymentattempt.ProviderWechat).
		SetIdempotencyKey("prov-pay-4").
		SetAmountCents(4000).
		SaveX(ctx)
}

func TestPaymentAttemptProviderPaymentIDUnique(t *testing.T) {
	ctx := context.Background()
	c := newTestClient(t)
	defer c.Close()

	order1 := createOrderX(ctx, t, c, "idem-prov-pay-a")
	order2 := createOrderX(ctx, t, c, "idem-prov-pay-b")

	c.PaymentAttempt.Create().
		SetNamespace(testNS).
		SetCommerceOrderID(order1.ID).
		SetCustomerID("cust-1").
		SetProvider(paymentattempt.ProviderAlipay).
		SetIdempotencyKey("ppay-1").
		SetAmountCents(1000).
		SetProviderPaymentID("ali-pay-456").
		SaveX(ctx)

	_, err := c.PaymentAttempt.Create().
		SetNamespace(testNS).
		SetCommerceOrderID(order2.ID).
		SetCustomerID("cust-1").
		SetProvider(paymentattempt.ProviderAlipay).
		SetIdempotencyKey("ppay-2").
		SetAmountCents(2000).
		SetProviderPaymentID("ali-pay-456").
		Save(ctx)
	if err == nil {
		t.Fatal("duplicate (provider, provider_payment_id) should fail")
	}
}

// --- One successful fulfillment per order ---

func TestOneFulfilledPerOrder(t *testing.T) {
	ctx := context.Background()
	c := newTestClient(t)
	defer c.Close()

	order := createOrderX(ctx, t, c, "idem-fulfill-1")

	// First fulfilled: OK.
	c.Fulfillment.Create().
		SetNamespace(testNS).
		SetCommerceOrderID(order.ID).
		SetCustomerID("cust-1").
		SetStatus(fulfillment.StatusFulfilled).
		SaveX(ctx)

	// Second fulfilled on same order: should fail.
	_, err := c.Fulfillment.Create().
		SetNamespace(testNS).
		SetCommerceOrderID(order.ID).
		SetCustomerID("cust-1").
		SetStatus(fulfillment.StatusFulfilled).
		Save(ctx)
	if err == nil {
		t.Fatal("second fulfilled fulfillment on same order should fail")
	}

	// Non-fulfilled on same order: OK.
	c.Fulfillment.Create().
		SetNamespace(testNS).
		SetCommerceOrderID(order.ID).
		SetCustomerID("cust-1").
		SetStatus(fulfillment.StatusFailed).
		SaveX(ctx)
}

// --- Product SKU uniqueness ---

func TestProductSKUUnique(t *testing.T) {
	ctx := context.Background()
	c := newTestClient(t)
	defer c.Close()

	c.CommerceProduct.Create().
		SetNamespace(testNS).
		SetSku("SKU-001").
		SetName("Test Plan").
		SetKind(commerceproduct.KindPlanPurchase).
		SetPriceCents(9900).
		SaveX(ctx)

	_, err := c.CommerceProduct.Create().
		SetNamespace(testNS).
		SetSku("SKU-001").
		SetName("Another Plan").
		SetKind(commerceproduct.KindWalletTopUp).
		SetPriceCents(5000).
		Save(ctx)
	if err == nil {
		t.Fatal("duplicate (namespace, sku) should fail")
	}

	// Different namespace: OK.
	c.CommerceProduct.Create().
		SetNamespace("other-namespace").
		SetSku("SKU-001").
		SetName("Different NS").
		SetKind(commerceproduct.KindWalletTopUp).
		SetPriceCents(5000).
		SaveX(ctx)
}

// --- Helper ---

func createOrderX(ctx context.Context, t *testing.T, c *db.Client, idemKey string) *db.CommerceOrder {
	t.Helper()
	return c.CommerceOrder.Create().
		SetNamespace(testNS).
		SetCustomerID("cust-1").
		SetKind(commerceorder.KindPlanPurchase).
		SetTotalCents(1000).
		SetCurrency("CNY").
		SetIdempotencyKey(idemKey).
		SaveX(ctx)
}
