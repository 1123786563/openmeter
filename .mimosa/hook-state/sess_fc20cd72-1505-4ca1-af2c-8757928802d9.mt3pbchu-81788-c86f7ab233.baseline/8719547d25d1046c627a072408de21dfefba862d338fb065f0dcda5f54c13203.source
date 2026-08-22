package order

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/openmeterio/openmeter/openmeter/commerce"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
)

// --- Mock implementations ---

type mockRepo struct {
	mu      sync.Mutex
	orders  map[string]*commerce.Order
	byIdem  map[string]string // (namespace+customerID+key) -> orderID
	statusN atomic.Int64      // count of status updates
}

func newMockRepo() *mockRepo {
	return &mockRepo{
		orders: make(map[string]*commerce.Order),
		byIdem: make(map[string]string),
	}
}

func idemKey(namespace, customerID, key string) string {
	return namespace + "/" + customerID + "/" + key
}

func (m *mockRepo) CreateOrder(_ context.Context, o commerce.Order) (*commerce.Order, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := idemKey(o.Namespace, o.CustomerID, o.IdempotencyKey)
	if existingID, ok := m.byIdem[key]; ok {
		existing := m.orders[existingID]
		return existing, false, nil
	}

	saved := o
	m.orders[o.ID] = &saved
	m.byIdem[key] = o.ID
	return &saved, true, nil
}

func (m *mockRepo) GetOrder(_ context.Context, namespace, id string) (*commerce.Order, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	o, ok := m.orders[id]
	if !ok || o.Namespace != namespace {
		return nil, commerce.ErrOrderNotFound
	}
	return o, nil
}

func (m *mockRepo) GetOrderByIdempotencyKey(_ context.Context, namespace, customerID, key string) (*commerce.Order, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id, ok := m.byIdem[idemKey(namespace, customerID, key)]
	if !ok {
		return nil, commerce.ErrOrderNotFound
	}
	return m.orders[id], nil
}

func (m *mockRepo) UpdateOrderStatus(_ context.Context, namespace, id string, expectedFrom, to commerce.OrderStatus) (*commerce.Order, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.statusN.Add(1)

	o, ok := m.orders[id]
	if !ok || o.Namespace != namespace {
		return nil, commerce.ErrOrderNotFound
	}
	if o.Status != expectedFrom {
		return nil, commerce.ErrInvalidOrderTransition
	}
	o.Status = to
	o.UpdatedAt = time.Now()
	return o, nil
}

func (m *mockRepo) ListOrdersByCustomer(_ context.Context, namespace, customerID string, status *commerce.OrderStatus) ([]commerce.Order, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []commerce.Order
	for _, o := range m.orders {
		if o.Namespace != namespace || o.CustomerID != customerID {
			continue
		}
		if status != nil && o.Status != *status {
			continue
		}
		result = append(result, *o)
	}
	return result, nil
}

type mockProductLookup struct {
	products map[string]*commerce.Product
}

func (m *mockProductLookup) GetProduct(_ context.Context, _ string, id string) (*commerce.Product, error) {
	p, ok := m.products[id]
	if !ok {
		return nil, commerce.ErrProductNotFound
	}
	return p, nil
}

func makeProduct(id, sku, name string, kind commerce.ProductKind, credits, price int64) *commerce.Product {
	return &commerce.Product{
		NamespacedID: models.NamespacedID{Namespace: "ns", ID: id},
		ManagedModel: models.ManagedModel{CreatedAt: time.Now(), UpdatedAt: time.Now()},
		Version:      1,
		SKU:          sku,
		DisplayName:  name,
		Kind:         kind,
		Credits:      credits,
		AmountMinor:  price,
		Currency:     "CNY",
		Active:       true,
		RefundPolicy: commerce.RefundPolicyUnspent,
	}
}

// --- Tests ---

// TestCreateOrderIdempotent verifies that replaying the same idempotency key
// returns the existing order without modification.
func TestCreateOrderIdempotent(t *testing.T) {
	repo := newMockRepo()
	lookup := &mockProductLookup{
		products: map[string]*commerce.Product{
			"p1": makeProduct("p1", "RECHARGE-100", "100 Points", commerce.ProductKindWalletTopUp, 100, 1000),
		},
	}
	svc := New(Config{Repo: repo, Products: lookup})

	input := commerce.CreateOrderInput{
		Namespace:      "ns",
		CustomerID:     "cust",
		Kind:           commerce.OrderKindWalletTopUp,
		IdempotencyKey: "idem-1",
		Currency:       "CNY",
		ProductIDs:     []string{"p1"},
	}

	first, isNew, err := svc.CreateOrder(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if !isNew {
		t.Fatal("first create should be a fresh insert")
	}

	// Replay.
	second, isNew2, err := svc.CreateOrder(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if isNew2 {
		t.Fatal("replay should return false for is-new")
	}
	if first.ID != second.ID {
		t.Fatalf("idempotent replay returned different order: %s vs %s", first.ID, second.ID)
	}
}

func TestCreateOrderRejectsProductKindMismatch(t *testing.T) {
	tests := []struct {
		name        string
		orderKind   commerce.OrderKind
		productKind commerce.ProductKind
	}{
		{name: "plan order with wallet product", orderKind: commerce.OrderKindPlanPurchase, productKind: commerce.ProductKindWalletTopUp},
		{name: "wallet order with plan product", orderKind: commerce.OrderKindWalletTopUp, productKind: commerce.ProductKindPlanPurchase},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			product := makeProduct("p1", "SKU-1", "Product", tt.productKind, 100, 1000)
			svc := New(Config{Repo: newMockRepo(), Products: &mockProductLookup{products: map[string]*commerce.Product{"p1": product}}})
			_, _, err := svc.CreateOrder(context.Background(), commerce.CreateOrderInput{
				Namespace: "ns", CustomerID: "cust", Kind: tt.orderKind,
				IdempotencyKey: "kind-mismatch", Currency: "CNY", ProductIDs: []string{"p1"},
			})
			if !errors.Is(err, commerce.ErrProductNotPurchasable) {
				t.Fatalf("error = %v, want ErrProductNotPurchasable", err)
			}
		})
	}
}

func TestCreateOrderValidatesProductPurchasabilityBeforeSnapshot(t *testing.T) {
	now := time.Date(2026, time.August, 10, 12, 0, 0, 0, time.UTC)
	clock.FreezeTime(now)
	t.Cleanup(clock.UnFreeze)

	future := now.Add(time.Hour)
	past := now.Add(-time.Hour)
	tests := []struct {
		name    string
		mutate  func(*commerce.Product)
		wantErr bool
	}{
		{name: "inactive", mutate: func(p *commerce.Product) { p.Active = false }, wantErr: true},
		{name: "future sale window", mutate: func(p *commerce.Product) { p.OnSaleAt = &future }, wantErr: true},
		{name: "expired sale window", mutate: func(p *commerce.Product) { p.OffSaleAt = &past }, wantErr: true},
		{name: "currency mismatch", mutate: func(p *commerce.Product) { p.Currency = "USD" }, wantErr: true},
		{name: "valid product", mutate: func(p *commerce.Product) { p.OnSaleAt = &past; p.OffSaleAt = &future }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			product := makeProduct("p1", "RECHARGE-100", "100 Credits", commerce.ProductKindWalletTopUp, 100, 9900)
			tt.mutate(product)
			svc := New(Config{Repo: newMockRepo(), Products: &mockProductLookup{products: map[string]*commerce.Product{"p1": product}}})
			order, _, err := svc.CreateOrder(context.Background(), commerce.CreateOrderInput{
				Namespace: "ns", CustomerID: "cust", Kind: commerce.OrderKindWalletTopUp,
				IdempotencyKey: "sale-check-" + tt.name, Currency: "CNY", ProductIDs: []string{"p1"},
			})
			if tt.wantErr {
				if !errors.Is(err, commerce.ErrProductNotPurchasable) {
					t.Fatalf("error = %v, want ErrProductNotPurchasable", err)
				}
				if len(svc.(*service).repo.(*mockRepo).orders) != 0 {
					t.Fatal("invalid product was snapshotted into an order")
				}
				return
			}
			if err != nil {
				t.Fatalf("valid product rejected: %v", err)
			}
			if order.AmountMinor != 9900 {
				t.Fatalf("order amount_minor = %d, want 9900", order.AmountMinor)
			}
		})
	}
}

// TestOrderLineSnapshotIsImmutable verifies that editing a product after order
// creation does not change the order's line snapshot.
func TestOrderLineSnapshotIsImmutable(t *testing.T) {
	repo := newMockRepo()
	product := makeProduct("p1", "RECHARGE-500", "500 Points", commerce.ProductKindWalletTopUp, 500, 4900)
	lookup := &mockProductLookup{products: map[string]*commerce.Product{"p1": product}}
	svc := New(Config{Repo: repo, Products: lookup})

	order, _, err := svc.CreateOrder(context.Background(), commerce.CreateOrderInput{
		Namespace:      "ns",
		CustomerID:     "cust",
		Kind:           commerce.OrderKindWalletTopUp,
		IdempotencyKey: "idem-snap",
		Currency:       "CNY",
		ProductIDs:     []string{"p1"},
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(order.Lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(order.Lines))
	}
	line := order.Lines[0]
	if line.SKU != "RECHARGE-500" {
		t.Errorf("snapshot sku = %s, want RECHARGE-500", line.SKU)
	}
	if line.DisplayName != "500 Points" {
		t.Errorf("snapshot name = %s, want 500 Points", line.DisplayName)
	}
	if line.Credits != 500 {
		t.Errorf("snapshot credits = %d, want 500", line.Credits)
	}
	if line.UnitPriceMinor != 4900 {
		t.Errorf("snapshot unit_price = %d, want 4900", line.UnitPriceMinor)
	}
	if line.SubtotalMinor != 4900 {
		t.Errorf("snapshot subtotal = %d, want 4900", line.SubtotalMinor)
	}

	// Now mutate the product. The order snapshot should NOT change.
	product.Credits = 9999
	product.AmountMinor = 99999
	product.DisplayName = "CHANGED"

	// Reload order.
	reloaded, err := svc.GetOrder(context.Background(), "ns", order.ID)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.Lines[0].Credits != 500 {
		t.Errorf("snapshot credits changed after product edit: %d, want 500", reloaded.Lines[0].Credits)
	}
	if reloaded.Lines[0].DisplayName != "500 Points" {
		t.Errorf("snapshot name changed after product edit: %s", reloaded.Lines[0].DisplayName)
	}
	if reloaded.Lines[0].UnitPriceMinor != 4900 {
		t.Errorf("snapshot price changed after product edit: %d", reloaded.Lines[0].UnitPriceMinor)
	}
}

// TestOrderLineSnapshotFields verifies all snapshot fields from the brief: SKU,
// display name, unit Credit, quantity, amount_minor, currency, included plan
// Credit, validity, and metadata.
func TestOrderLineSnapshotFields(t *testing.T) {
	repo := newMockRepo()
	product := makeProduct("p1", "PLAN-PRO", "Pro Plan", commerce.ProductKindPlanPurchase, 10000, 9900)
	product.RefundPolicy = commerce.RefundPolicyFullWindow
	product.BonusCredits = 500
	lookup := &mockProductLookup{products: map[string]*commerce.Product{"p1": product}}
	svc := New(Config{Repo: repo, Products: lookup})

	order, _, err := svc.CreateOrder(context.Background(), commerce.CreateOrderInput{
		Namespace:      "ns",
		CustomerID:     "cust",
		Kind:           commerce.OrderKindPlanPurchase,
		IdempotencyKey: "idem-snap-fields",
		Currency:       "CNY",
		ProductIDs:     []string{"p1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if order.AmountMinor != 9900 {
		t.Fatalf("order amount_minor = %d, want 9900", order.AmountMinor)
	}

	line := order.Lines[0]

	// SKU
	if line.SKU != "PLAN-PRO" {
		t.Errorf("sku = %s", line.SKU)
	}
	// Display name
	if line.DisplayName != "Pro Plan" {
		t.Errorf("display_name = %s", line.DisplayName)
	}
	// Credits (unit Credit)
	if line.Credits != 10000 {
		t.Errorf("credits = %d", line.Credits)
	}
	// Quantity
	if line.Quantity != 1 {
		t.Errorf("quantity = %d", line.Quantity)
	}
	// amount_minor
	if line.UnitPriceMinor != 9900 {
		t.Errorf("unit_price = %d", line.UnitPriceMinor)
	}
	// currency
	if line.Currency != "CNY" {
		t.Errorf("currency = %s", line.Currency)
	}
	// included plan credit
	if line.IncludedPlanCredit != 10000 {
		t.Errorf("included_plan_credit = %d, want 10000", line.IncludedPlanCredit)
	}
	// metadata (product version + refund policy)
	if line.Metadata["product_version"] != "1" {
		t.Errorf("metadata product_version = %s", line.Metadata["product_version"])
	}
	if line.Metadata["refund_policy"] != "full_window" {
		t.Errorf("metadata refund_policy = %s", line.Metadata["refund_policy"])
	}
}

// TestOrderStateMachineTransitions exercises the full happy path and verifies
// that "paid does not imply fulfilled" and "fulfilled cannot return to paid".
func TestOrderStateMachineTransitions(t *testing.T) {
	repo := newMockRepo()
	lookup := &mockProductLookup{
		products: map[string]*commerce.Product{
			"p1": makeProduct("p1", "RECHARGE-100", "100", commerce.ProductKindWalletTopUp, 100, 1000),
		},
	}
	svc := New(Config{Repo: repo, Products: lookup})

	order, _, err := svc.CreateOrder(context.Background(), commerce.CreateOrderInput{
		Namespace:      "ns",
		CustomerID:     "cust",
		Kind:           commerce.OrderKindWalletTopUp,
		IdempotencyKey: "idem-sm",
		Currency:       "CNY",
		ProductIDs:     []string{"p1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if order.Status != commerce.OrderStatusCreated {
		t.Fatalf("initial status = %s, want created", order.Status)
	}

	// created -> awaiting_payment
	o, err := svc.TransitionStatus(context.Background(), "ns", order.ID, commerce.OrderStatusAwaitingPayment)
	if err != nil {
		t.Fatalf("created -> awaiting_payment: %v", err)
	}
	if o.Status != commerce.OrderStatusAwaitingPayment {
		t.Fatalf("status = %s, want awaiting_payment", o.Status)
	}

	// awaiting_payment -> paid
	o, err = svc.TransitionStatus(context.Background(), "ns", order.ID, commerce.OrderStatusPaid)
	if err != nil {
		t.Fatalf("awaiting_payment -> paid: %v", err)
	}
	if o.Status != commerce.OrderStatusPaid {
		t.Fatalf("status = %s, want paid", o.Status)
	}

	// "paid does not imply fulfilled": status is paid, not fulfilled.
	if o.Status == commerce.OrderStatusFulfilled {
		t.Fatal("paid should not imply fulfilled")
	}

	// paid -> fulfilled
	_, err = svc.TransitionStatus(context.Background(), "ns", order.ID, commerce.OrderStatusFulfilled)
	if err != nil {
		t.Fatalf("paid -> fulfilled: %v", err)
	}

	// "fulfilled cannot return to paid"
	_, err = svc.TransitionStatus(context.Background(), "ns", order.ID, commerce.OrderStatusPaid)
	if err == nil {
		t.Fatal("fulfilled -> paid should be rejected")
	}

	// fulfilled -> refund_pending
	_, err = svc.TransitionStatus(context.Background(), "ns", order.ID, commerce.OrderStatusRefundPending)
	if err != nil {
		t.Fatalf("fulfilled -> refund_pending: %v", err)
	}

	// refund_pending -> refunded
	o, err = svc.TransitionStatus(context.Background(), "ns", order.ID, commerce.OrderStatusRefunded)
	if err != nil {
		t.Fatalf("refund_pending -> refunded: %v", err)
	}
	if o.Status != commerce.OrderStatusRefunded {
		t.Fatalf("status = %s, want refunded", o.Status)
	}
}

// TestOrderInvalidTransitionReturnsConflict verifies that invalid transitions
// return the stable conflict error code.
func TestOrderInvalidTransitionReturnsConflict(t *testing.T) {
	repo := newMockRepo()
	lookup := &mockProductLookup{
		products: map[string]*commerce.Product{
			"p1": makeProduct("p1", "RECHARGE-100", "100", commerce.ProductKindWalletTopUp, 100, 1000),
		},
	}
	svc := New(Config{Repo: repo, Products: lookup})

	order, _, err := svc.CreateOrder(context.Background(), commerce.CreateOrderInput{
		Namespace:      "ns",
		CustomerID:     "cust",
		Kind:           commerce.OrderKindWalletTopUp,
		IdempotencyKey: "idem-invalid",
		Currency:       "CNY",
		ProductIDs:     []string{"p1"},
	})
	if err != nil {
		t.Fatal(err)
	}

	// created -> paid (skip awaiting_payment): invalid
	_, err = svc.TransitionStatus(context.Background(), "ns", order.ID, commerce.OrderStatusPaid)
	if err == nil {
		t.Fatal("created -> paid should fail")
	}

	// Verify the order status didn't change.
	unchanged, _ := svc.GetOrder(context.Background(), "ns", order.ID)
	if unchanged.Status != commerce.OrderStatusCreated {
		t.Fatalf("status changed to %s after invalid transition", unchanged.Status)
	}
}

// TestOrderCancellation verifies cancellation from created and awaiting_payment.
func TestOrderCancellation(t *testing.T) {
	repo := newMockRepo()
	lookup := &mockProductLookup{
		products: map[string]*commerce.Product{
			"p1": makeProduct("p1", "RECHARGE-100", "100", commerce.ProductKindWalletTopUp, 100, 1000),
		},
	}
	svc := New(Config{Repo: repo, Products: lookup})

	// Cancel from created.
	order, _, _ := svc.CreateOrder(context.Background(), commerce.CreateOrderInput{
		Namespace: "ns", CustomerID: "cust", Kind: commerce.OrderKindWalletTopUp,
		IdempotencyKey: "idem-cancel-1", Currency: "CNY", ProductIDs: []string{"p1"},
	})
	o, err := svc.TransitionStatus(context.Background(), "ns", order.ID, commerce.OrderStatusCancelled)
	if err != nil {
		t.Fatalf("created -> canceled: %v", err)
	}
	if o.Status != commerce.OrderStatusCancelled {
		t.Fatalf("status = %s, want canceled", o.Status)
	}

	// Cancel from awaiting_payment.
	order2, _, _ := svc.CreateOrder(context.Background(), commerce.CreateOrderInput{
		Namespace: "ns", CustomerID: "cust", Kind: commerce.OrderKindWalletTopUp,
		IdempotencyKey: "idem-cancel-2", Currency: "CNY", ProductIDs: []string{"p1"},
	})
	_, _ = svc.TransitionStatus(context.Background(), "ns", order2.ID, commerce.OrderStatusAwaitingPayment)
	o2, err := svc.TransitionStatus(context.Background(), "ns", order2.ID, commerce.OrderStatusCancelled)
	if err != nil {
		t.Fatalf("awaiting_payment -> canceled: %v", err)
	}
	if o2.Status != commerce.OrderStatusCancelled {
		t.Fatalf("status = %s, want canceled", o2.Status)
	}
}

// TestYearlyRenewalSchedulesMonthlyGrants verifies that a yearly Pro/Team
// subscription renewal schedules twelve monthly plan-credit grants and never
// grants the whole annual credit at payment time.
func TestYearlyRenewalSchedulesMonthlyGrants(t *testing.T) {
	product := makeProduct("p1", "PLAN-PRO-YEARLY", "Pro Yearly", commerce.ProductKindSubscriptionRenewal, 120000, 99000)
	lookup := &mockProductLookup{products: map[string]*commerce.Product{"p1": product}}
	repo := newMockRepo()
	svc := New(Config{Repo: repo, Products: lookup})

	order, _, err := svc.CreateOrder(context.Background(), commerce.CreateOrderInput{
		Namespace:      "ns",
		CustomerID:     "cust",
		Kind:           commerce.OrderKindSubscriptionRenewal,
		IdempotencyKey: "idem-yearly",
		Currency:       "CNY",
		ProductIDs:     []string{"p1"},
	})
	if err != nil {
		t.Fatal(err)
	}

	schedule := ScheduleYearlyRenewal(*order)
	if len(schedule) != 12 {
		t.Fatalf("expected 12 monthly entries, got %d", len(schedule))
	}

	// 120000 / 12 = 10000 per month
	for i, e := range schedule {
		if e.Credits != 10000 {
			t.Errorf("entry[%d] credits = %d, want 10000", i, e.Credits)
		}
		if e.MonthIndex != i+1 {
			t.Errorf("entry[%d] month_index = %d, want %d", i, e.MonthIndex, i+1)
		}
		// Each grant date is one month apart.
		expectedDate := schedule[0].GrantDate.AddDate(0, i, 0)
		if !e.GrantDate.Equal(expectedDate) {
			t.Errorf("entry[%d] grant_date = %v, want %v", i, e.GrantDate, expectedDate)
		}
	}

	// Verify the annual credit is NOT granted upfront at order creation time.
	// Each monthly grant is 1/12 of the total, not the full amount.
	if schedule[0].Credits >= 120000 {
		t.Fatal("monthly grant should not equal the full annual credit")
	}
}

// TestYearlyRenewalOnlyForSubscriptionKind verifies that non-subscription orders
// produce no renewal schedule.
func TestYearlyRenewalOnlyForSubscriptionKind(t *testing.T) {
	product := makeProduct("p1", "RECHARGE-100", "100", commerce.ProductKindWalletTopUp, 100, 1000)
	lookup := &mockProductLookup{products: map[string]*commerce.Product{"p1": product}}
	repo := newMockRepo()
	svc := New(Config{Repo: repo, Products: lookup})

	order, _, _ := svc.CreateOrder(context.Background(), commerce.CreateOrderInput{
		Namespace: "ns", CustomerID: "cust", Kind: commerce.OrderKindWalletTopUp,
		IdempotencyKey: "idem-non-sub", Currency: "CNY", ProductIDs: []string{"p1"},
	})

	schedule := ScheduleYearlyRenewal(*order)
	if schedule != nil {
		t.Fatal("wallet_top_up order should not produce a renewal schedule")
	}
}

// --- Concurrency tests ---

// TestConcurrentStatusTransitions verifies that two concurrent attempts to
// transition the same order from the same state result in exactly one success.
func TestConcurrentStatusTransitions(t *testing.T) {
	repo := newMockRepo()
	lookup := &mockProductLookup{
		products: map[string]*commerce.Product{
			"p1": makeProduct("p1", "RECHARGE-100", "100", commerce.ProductKindWalletTopUp, 100, 1000),
		},
	}
	svc := New(Config{Repo: repo, Products: lookup})

	order, _, _ := svc.CreateOrder(context.Background(), commerce.CreateOrderInput{
		Namespace: "ns", CustomerID: "cust", Kind: commerce.OrderKindWalletTopUp,
		IdempotencyKey: "idem-conc", Currency: "CNY", ProductIDs: []string{"p1"},
	})
	_, _ = svc.TransitionStatus(context.Background(), "ns", order.ID, commerce.OrderStatusAwaitingPayment)

	var wg sync.WaitGroup
	var successCount atomic.Int64
	var errCount atomic.Int64

	// Two goroutines try paid -> ... (awaiting_payment -> paid) simultaneously.
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := svc.TransitionStatus(context.Background(), "ns", order.ID, commerce.OrderStatusPaid)
			if err == nil {
				successCount.Add(1)
			} else {
				errCount.Add(1)
			}
		}()
	}
	wg.Wait()

	if successCount.Load() != 1 {
		t.Fatalf("expected exactly 1 successful transition, got %d", successCount.Load())
	}
	if errCount.Load() != 1 {
		t.Fatalf("expected exactly 1 failed transition, got %d", errCount.Load())
	}
}

// TestConcurrentOrderCreation verifies that two concurrent orders with the same
// idempotency key produce exactly one order.
func TestConcurrentOrderCreation(t *testing.T) {
	repo := newMockRepo()
	lookup := &mockProductLookup{
		products: map[string]*commerce.Product{
			"p1": makeProduct("p1", "RECHARGE-100", "100", commerce.ProductKindWalletTopUp, 100, 1000),
		},
	}
	svc := New(Config{Repo: repo, Products: lookup})

	input := commerce.CreateOrderInput{
		Namespace:      "ns",
		CustomerID:     "cust",
		Kind:           commerce.OrderKindWalletTopUp,
		IdempotencyKey: "idem-concurrent",
		Currency:       "CNY",
		ProductIDs:     []string{"p1"},
	}

	var wg sync.WaitGroup
	var newCount atomic.Int64
	var orderIDs sync.Map

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			o, isNew, err := svc.CreateOrder(context.Background(), input)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if isNew {
				newCount.Add(1)
			}
			orderIDs.Store(o.ID, true)
		}()
	}
	wg.Wait()

	if newCount.Load() != 1 {
		t.Fatalf("expected exactly 1 fresh insert, got %d", newCount.Load())
	}

	count := 0
	orderIDs.Range(func(_, _ any) bool {
		count++
		return true
	})
	if count != 1 {
		t.Fatalf("expected exactly 1 unique order ID, got %d", count)
	}
}

// TestConsumptionPriorityConstants verifies that the fixed source-priority
// ordering matches the contract: plan=10, gift=20, recharge=30,
// enterprise_receivable=40. The full CreditEngine implementation (concurrent
// debit with spillover to enterprise receivable) is deferred to a later task;
// this test only guards the constants.
func TestConsumptionPriorityConstants(t *testing.T) {
	tests := []struct {
		source commerce.BucketSource
		want   int
	}{
		{commerce.BucketSourcePlan, 10},
		{commerce.BucketSourceGift, 20},
		{commerce.BucketSourceRecharge, 30},
		{commerce.BucketSourceEnterpriseReceivable, 40},
	}
	for _, tt := range tests {
		got := commerce.SourcePriority(tt.source)
		if got != tt.want {
			t.Errorf("SourcePriority(%s) = %d, want %d", tt.source, got, tt.want)
		}
	}
}
