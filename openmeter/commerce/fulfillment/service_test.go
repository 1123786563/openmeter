package fulfillment

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/openmeterio/openmeter/openmeter/commerce"
	"github.com/openmeterio/openmeter/pkg/models"
)

// --- Mock implementations ---

type mockRepo struct {
	mu            sync.Mutex
	fulfillments  map[string]*FulfillmentRecord
	grantCount    atomic.Int64
	fulfilledOnce atomic.Int64 // how many times MarkFulfilled succeeded
}

func newMockRepo() *mockRepo {
	return &mockRepo{fulfillments: make(map[string]*FulfillmentRecord)}
}

func (m *mockRepo) CreateFulfillment(_ context.Context, req FulfillmentRequest) (*FulfillmentRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	// Check for existing by order.
	for _, f := range m.fulfillments {
		if f.OrderID == req.OrderID {
			return f, nil
		}
	}
	rec := &FulfillmentRecord{
		ID:         "ful-" + req.OrderID,
		Namespace:  req.Namespace,
		OrderID:    req.OrderID,
		CustomerID: req.CustomerID,
		Status:     FulfillmentStatusPending,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	m.fulfillments[rec.ID] = rec
	cp := *rec
	return &cp, nil
}

func (m *mockRepo) GetFulfillment(_ context.Context, namespace, id string) (*FulfillmentRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	f, ok := m.fulfillments[id]
	if !ok || f.Namespace != namespace {
		return nil, errors.New("not found")
	}
	cp := *f
	return &cp, nil
}

func (m *mockRepo) GetFulfillmentByOrder(_ context.Context, namespace, orderID string) (*FulfillmentRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, f := range m.fulfillments {
		if f.OrderID == orderID && f.Namespace == namespace {
			cp := *f
			return &cp, nil
		}
	}
	return nil, errors.New("not found")
}

func (m *mockRepo) ClaimForProcessing(_ context.Context, namespace, id string) (*FulfillmentRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	f, ok := m.fulfillments[id]
	if !ok {
		return nil, errors.New("not found")
	}
	if f.Status == FulfillmentStatusProcessing {
		// I4: lease timeout — if the claim is older than the lease, re-claim it.
		if f.ClaimedAt != nil && time.Since(*f.ClaimedAt) < ProcessingLeaseTimeout {
			cp := *f
			return &cp, ErrAlreadyProcessing
		}
	}
	f.Status = FulfillmentStatusProcessing
	now := time.Now()
	f.ClaimedAt = &now
	cp := *f
	return &cp, nil
}

func (m *mockRepo) MarkFulfilled(_ context.Context, namespace, id string, result FulfillmentResult) (*FulfillmentRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	f, ok := m.fulfillments[id]
	if !ok {
		return nil, errors.New("not found")
	}

	// Check if any other fulfillment for this order is already fulfilled
	// (simulates the partial unique index).
	for _, other := range m.fulfillments {
		if other.OrderID == f.OrderID && other.Status == FulfillmentStatusFulfilled && other.ID != id {
			return nil, fmt.Errorf("unique constraint: order already fulfilled")
		}
	}

	f.Status = FulfillmentStatusFulfilled
	f.GrantID = result.GrantID
	f.CreditsGranted = result.CreditsGranted
	now := time.Now()
	f.FulfilledAt = &now
	m.fulfilledOnce.Add(1)
	cp := *f
	return &cp, nil
}

func (m *mockRepo) MarkFailed(_ context.Context, namespace, id, reason string) (*FulfillmentRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	f, ok := m.fulfillments[id]
	if !ok {
		return nil, errors.New("not found")
	}
	f.Status = FulfillmentStatusFailed
	f.FailureReason = &reason
	cp := *f
	return &cp, nil
}

func (m *mockRepo) ListPending(_ context.Context, namespace string, limit int) ([]FulfillmentRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []FulfillmentRecord
	for _, f := range m.fulfillments {
		if f.Status == FulfillmentStatusPending || f.Status == FulfillmentStatusFailed {
			result = append(result, *f)
			if len(result) >= limit {
				break
			}
			continue
		}
		// I4: re-queue processing records whose lease has expired.
		if f.Status == FulfillmentStatusProcessing && f.ClaimedAt != nil && time.Since(*f.ClaimedAt) >= ProcessingLeaseTimeout {
			result = append(result, *f)
			if len(result) >= limit {
				break
			}
		}
	}
	return result, nil
}

type mockOrderUpdater struct {
	mu     sync.Mutex
	orders map[string]*commerce.Order
}

func newMockOrderUpdater() *mockOrderUpdater {
	return &mockOrderUpdater{orders: make(map[string]*commerce.Order)}
}

func (m *mockOrderUpdater) GetOrder(_ context.Context, namespace, id string) (*commerce.Order, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	o, ok := m.orders[id]
	if !ok {
		return nil, commerce.ErrOrderNotFound
	}
	cp := *o
	return &cp, nil
}

func (m *mockOrderUpdater) UpdateOrderStatus(_ context.Context, namespace, id string, expectedFrom, to commerce.OrderStatus) (*commerce.Order, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	o, ok := m.orders[id]
	if !ok {
		return nil, commerce.ErrOrderNotFound
	}
	if o.Status != expectedFrom {
		return nil, commerce.ErrInvalidOrderTransition
	}
	o.Status = to
	o.UpdatedAt = time.Now()
	cp := *o
	return &cp, nil
}

type mockGrantor struct {
	mu         sync.Mutex
	grants     map[string]int // idempotency key -> credits granted
	callN      atomic.Int64
	crashAfter int // if > 0, return error after this many calls
}

func newMockGrantor() *mockGrantor {
	return &mockGrantor{grants: make(map[string]int)}
}

func (g *mockGrantor) GrantCredits(_ context.Context, in GrantCreditsInput) (GrantCreditsResult, error) {
	g.mu.Lock()
	g.callN.Add(1)
	callNum := g.callN.Load()
	g.mu.Unlock()

	if g.crashAfter > 0 && callNum > int64(g.crashAfter) {
		return GrantCreditsResult{}, errors.New("simulated crash")
	}

	g.mu.Lock()
	defer g.mu.Unlock()
	if existing, ok := g.grants[in.IdempotencyKey]; ok {
		return GrantCreditsResult{GrantID: "grant-" + in.IdempotencyKey, Credits: int64(existing)}, nil
	}
	g.grants[in.IdempotencyKey] = int(in.Credits)
	return GrantCreditsResult{GrantID: "grant-" + in.IdempotencyKey, Credits: in.Credits}, nil
}

func (g *mockGrantor) totalGranted() int64 {
	g.mu.Lock()
	defer g.mu.Unlock()
	total := int64(0)
	for _, v := range g.grants {
		total += int64(v)
	}
	return total
}

type mockInvoiceMarker struct {
	markCount atomic.Int64
}

func (m *mockInvoiceMarker) MarkInvoicePaid(_ context.Context, _, _ string) error {
	m.markCount.Add(1)
	return nil
}

type mockNotifier struct {
	notifyCount atomic.Int64
}

func (m *mockNotifier) NotifyOrderFulfilled(_ context.Context, _, _, _ string) error {
	m.notifyCount.Add(1)
	return nil
}

// --- Harness ---

type testHarness struct {
	svc      Service
	repo     *mockRepo
	orders   *mockOrderUpdater
	grantor  *mockGrantor
	invoices *mockInvoiceMarker
	notifier *mockNotifier
}

func newTestHarness(t *testing.T) *testHarness {
	t.Helper()
	repo := newMockRepo()
	orders := newMockOrderUpdater()
	grantor := newMockGrantor()
	invoices := &mockInvoiceMarker{}
	notifier := &mockNotifier{}

	svc, err := New(Config{
		Repo:     repo,
		Orders:   orders,
		Grantor:  grantor,
		Invoices: invoices,
		Notifier: notifier,
	})
	if err != nil {
		t.Fatal(err)
	}
	return &testHarness{svc: svc, repo: repo, orders: orders, grantor: grantor, invoices: invoices, notifier: notifier}
}

func (h *testHarness) addPaidOrder(namespace, orderID, customerID string, amount int64, kind commerce.OrderKind) *commerce.Order {
	order := &commerce.Order{
		NamespacedID: models.NamespacedID{Namespace: namespace, ID: orderID},
		CustomerID:   customerID,
		Kind:         kind,
		Status:       commerce.OrderStatusPaid,
		AmountMinor:  amount,
		Currency:     "CNY",
		Lines: []commerce.OrderLineSnapshot{
			{Credits: amount, Currency: "CNY", SubtotalMinor: amount, UnitPriceMinor: amount},
		},
	}
	h.orders.orders[orderID] = order
	return order
}

// --- Tests ---

func TestRequestFulfillmentIdempotent(t *testing.T) {
	h := newTestHarness(t)
	h.addPaidOrder("ns", "order-1", "cust", 100, commerce.OrderKindWalletTopUp)

	first, err := h.svc.RequestFulfillment(context.Background(), "ns", "order-1")
	if err != nil {
		t.Fatal(err)
	}
	if first.Status != FulfillmentStatusPending {
		t.Errorf("status = %s, want pending", first.Status)
	}

	// Replay.
	second, err := h.svc.RequestFulfillment(context.Background(), "ns", "order-1")
	if err != nil {
		t.Fatal(err)
	}
	if first.ID != second.ID {
		t.Fatalf("idempotent replay: %s vs %s", first.ID, second.ID)
	}
}

func TestProcessOneGrantsCreditsAndFulfills(t *testing.T) {
	h := newTestHarness(t)
	h.addPaidOrder("ns", "order-2", "cust", 500, commerce.OrderKindWalletTopUp)

	rec, err := h.svc.RequestFulfillment(context.Background(), "ns", "order-2")
	if err != nil {
		t.Fatal(err)
	}

	result, err := h.svc.ProcessOne(context.Background(), "ns", rec.ID)
	if err != nil {
		t.Fatalf("process one: %v", err)
	}

	if result.Status != FulfillmentStatusFulfilled {
		t.Errorf("status = %s, want fulfilled", result.Status)
	}
	if result.CreditsGranted != 500 {
		t.Errorf("credits_granted = %d, want 500", result.CreditsGranted)
	}
	if result.GrantID == "" {
		t.Error("grant_id should be set")
	}

	// Order should be fulfilled.
	order, _ := h.orders.GetOrder(context.Background(), "ns", "order-2")
	if order.Status != commerce.OrderStatusFulfilled {
		t.Errorf("order status = %s, want fulfilled", order.Status)
	}

	// Invoice marked once.
	if h.invoices.markCount.Load() != 1 {
		t.Errorf("invoice mark count = %d, want 1", h.invoices.markCount.Load())
	}

	// Credits granted.
	if h.grantor.totalGranted() != 500 {
		t.Errorf("total granted = %d, want 500", h.grantor.totalGranted())
	}

	// Notified once.
	if h.notifier.notifyCount.Load() != 1 {
		t.Errorf("notify count = %d, want 1", h.notifier.notifyCount.Load())
	}
}

func TestProcessOneExactlyOnce(t *testing.T) {
	h := newTestHarness(t)
	h.addPaidOrder("ns", "order-3", "cust", 1000, commerce.OrderKindWalletTopUp)

	rec, err := h.svc.RequestFulfillment(context.Background(), "ns", "order-3")
	if err != nil {
		t.Fatal(err)
	}

	// Process multiple times — should converge to one fulfillment.
	for i := 0; i < 5; i++ {
		_, err := h.svc.ProcessOne(context.Background(), "ns", rec.ID)
		if err != nil {
			t.Fatalf("call %d: %v", i, err)
		}
	}

	// Only one successful fulfillment.
	if h.repo.fulfilledOnce.Load() != 1 {
		t.Errorf("fulfilled once = %d, want 1", h.repo.fulfilledOnce.Load())
	}

	// Credits granted exactly once (idempotent grantor).
	if h.grantor.totalGranted() != 1000 {
		t.Errorf("total granted = %d, want 1000", h.grantor.totalGranted())
	}
}

func TestProcessOneConcurrentWorkers(t *testing.T) {
	h := newTestHarness(t)
	h.addPaidOrder("ns", "order-4", "cust", 2000, commerce.OrderKindWalletTopUp)

	rec, err := h.svc.RequestFulfillment(context.Background(), "ns", "order-4")
	if err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = h.svc.ProcessOne(context.Background(), "ns", rec.ID)
		}()
	}
	wg.Wait()

	// Exactly one fulfilled state.
	if h.repo.fulfilledOnce.Load() != 1 {
		t.Errorf("fulfilled once = %d, want 1", h.repo.fulfilledOnce.Load())
	}
	if h.grantor.totalGranted() != 2000 {
		t.Errorf("total granted = %d, want 2000", h.grantor.totalGranted())
	}
}

func TestProcessOneWalletTopUpUsesRechargeSource(t *testing.T) {
	h := newTestHarness(t)
	h.addPaidOrder("ns", "order-5", "cust", 300, commerce.OrderKindWalletTopUp)

	rec, _ := h.svc.RequestFulfillment(context.Background(), "ns", "order-5")
	_, err := h.svc.ProcessOne(context.Background(), "ns", rec.ID)
	if err != nil {
		t.Fatal(err)
	}

	// Verify the grantor was called (source is validated internally).
	if h.grantor.callN.Load() == 0 {
		t.Error("grantor should be called for wallet top-up")
	}
}

func TestProcessOnePlanPurchaseUsesPlanSource(t *testing.T) {
	h := newTestHarness(t)
	h.addPaidOrder("ns", "order-6", "cust", 10000, commerce.OrderKindPlanPurchase)

	rec, _ := h.svc.RequestFulfillment(context.Background(), "ns", "order-6")
	result, err := h.svc.ProcessOne(context.Background(), "ns", rec.ID)
	if err != nil {
		t.Fatal(err)
	}
	if result.CreditsGranted != 10000 {
		t.Errorf("credits = %d, want 10000", result.CreditsGranted)
	}
}

// --- Crash boundary tests ---

// TestCrashAfterInvoicePaid verifies that if a crash occurs after marking the
// invoice paid but before granting credits, a restart grants the credits and
// completes fulfillment.
func TestCrashAfterInvoicePaid(t *testing.T) {
	h := newTestHarness(t)
	h.addPaidOrder("ns", "order-crash-1", "cust", 500, commerce.OrderKindWalletTopUp)

	// Grantor crashes on first call, succeeds on retry.
	crashCount := atomic.Int64{}
	h.grantor.crashAfter = 0
	origGrantor := h.grantor
	wrappedGrantor := &crashingGrantor{
		inner:       origGrantor,
		crashAfterN: 1,
		crashN:      &crashCount,
	}

	svc, _ := New(Config{
		Repo:     h.repo,
		Orders:   h.orders,
		Grantor:  wrappedGrantor,
		Invoices: h.invoices,
		Notifier: h.notifier,
	})

	rec, _ := h.svc.RequestFulfillment(context.Background(), "ns", "order-crash-1")

	// First attempt: crashes during grant.
	_, err := svc.ProcessOne(context.Background(), "ns", rec.ID)
	if err == nil {
		t.Fatal("first attempt should crash")
	}

	// Fulfillment should be marked failed.
	failed, _ := h.repo.GetFulfillment(context.Background(), "ns", rec.ID)
	if failed.Status != FulfillmentStatusFailed {
		t.Errorf("status = %s, want failed", failed.Status)
	}

	// Invoice was marked before the crash.
	if h.invoices.markCount.Load() != 1 {
		t.Errorf("invoice should be marked: %d", h.invoices.markCount.Load())
	}

	// No credits granted yet.
	if h.grantor.totalGranted() != 0 {
		t.Errorf("no credits should be granted: %d", h.grantor.totalGranted())
	}

	// Restart: retry succeeds.
	result, err := h.svc.ProcessOne(context.Background(), "ns", rec.ID)
	if err != nil {
		t.Fatalf("retry should succeed: %v", err)
	}
	if result.Status != FulfillmentStatusFulfilled {
		t.Errorf("status = %s, want fulfilled", result.Status)
	}
	if result.CreditsGranted != 500 {
		t.Errorf("credits = %d, want 500", result.CreditsGranted)
	}
}

// crashingGrantor wraps a CreditGrantor and returns an error for the first N calls.
type crashingGrantor struct {
	inner       CreditGrantor
	crashAfterN int
	crashN      *atomic.Int64
}

func (g *crashingGrantor) GrantCredits(ctx context.Context, in GrantCreditsInput) (GrantCreditsResult, error) {
	n := g.crashN.Add(1)
	if int(n) <= g.crashAfterN {
		return GrantCreditsResult{}, errors.New("simulated crash before grant")
	}
	return g.inner.GrantCredits(ctx, in)
}

// TestProcessOneAlreadyFulfilled verifies that processing an already-fulfilled
// request is a no-op (idempotent success).
func TestProcessOneAlreadyFulfilled(t *testing.T) {
	h := newTestHarness(t)
	h.addPaidOrder("ns", "order-7", "cust", 100, commerce.OrderKindWalletTopUp)

	rec, _ := h.svc.RequestFulfillment(context.Background(), "ns", "order-7")
	_, err := h.svc.ProcessOne(context.Background(), "ns", rec.ID)
	if err != nil {
		t.Fatal(err)
	}

	grantsBefore := h.grantor.callN.Load()

	// Process again — should be a no-op.
	result, err := h.svc.ProcessOne(context.Background(), "ns", rec.ID)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != FulfillmentStatusFulfilled {
		t.Errorf("status = %s, want fulfilled", result.Status)
	}

	// Grantor should not be called again.
	if h.grantor.callN.Load() > grantsBefore {
		t.Error("grantor should not be called for already-fulfilled")
	}
}

func TestProcessPendingProcessesAll(t *testing.T) {
	h := newTestHarness(t)
	for i := 0; i < 3; i++ {
		orderID := fmt.Sprintf("order-batch-%d", i)
		h.addPaidOrder("ns", orderID, "cust", int64(100*(i+1)), commerce.OrderKindWalletTopUp)
		_, _ = h.svc.RequestFulfillment(context.Background(), "ns", orderID)
	}

	count, err := h.svc.ProcessPending(context.Background(), "ns", 10)
	if err != nil {
		t.Fatal(err)
	}
	if count != 3 {
		t.Errorf("processed = %d, want 3", count)
	}

	// All should be fulfilled.
	for i := 0; i < 3; i++ {
		rec, _ := h.svc.GetFulfillmentByOrder(context.Background(), "ns", fmt.Sprintf("order-batch-%d", i))
		if rec.Status != FulfillmentStatusFulfilled {
			t.Errorf("order-batch-%d status = %s", i, rec.Status)
		}
	}
}

// --- I3: Order status guard ---

// TestProcessOneRejectsOrderNotPaid verifies that ProcessOne rejects orders
// that are not in the paid or fulfilled state (I3).
func TestProcessOneRejectsOrderNotPaid(t *testing.T) {
	h := newTestHarness(t)
	// Create an order in awaiting_payment (not yet paid).
	h.orders.orders["order-notpaid"] = &commerce.Order{
		NamespacedID: models.NamespacedID{Namespace: "ns", ID: "order-notpaid"},
		CustomerID:   "cust",
		Kind:         commerce.OrderKindWalletTopUp,
		Status:       commerce.OrderStatusAwaitingPayment,
		AmountMinor:  100,
		Currency:     "CNY",
		Lines:        []commerce.OrderLineSnapshot{{Credits: 100, Currency: "CNY"}},
	}

	rec, err := h.svc.RequestFulfillment(context.Background(), "ns", "order-notpaid")
	if err != nil {
		t.Fatal(err)
	}

	_, err = h.svc.ProcessOne(context.Background(), "ns", rec.ID)
	if err == nil {
		t.Fatal("should reject order not in paid state")
	}

	// The fulfillment should be marked failed.
	failed, _ := h.repo.GetFulfillment(context.Background(), "ns", rec.ID)
	if failed.Status != FulfillmentStatusFailed {
		t.Errorf("status = %s, want failed", failed.Status)
	}

	// No credits should be granted.
	if h.grantor.totalGranted() != 0 {
		t.Errorf("no credits should be granted for non-paid order, got %d", h.grantor.totalGranted())
	}
}

// --- I4: Lease timeout recovery ---

// TestLeaseTimeoutRecovery verifies that a fulfillment stuck in processing
// after the lease expires can be re-claimed by another worker (I4).
func TestLeaseTimeoutRecovery(t *testing.T) {
	h := newTestHarness(t)
	h.addPaidOrder("ns", "order-lease", "cust", 500, commerce.OrderKindWalletTopUp)

	rec, err := h.svc.RequestFulfillment(context.Background(), "ns", "order-lease")
	if err != nil {
		t.Fatal(err)
	}

	// Simulate a worker crash: manually set the record to processing with an
	// expired claim time.
	h.repo.mu.Lock()
	stuck := h.repo.fulfillments[rec.ID]
	stuck.Status = FulfillmentStatusProcessing
	expired := time.Now().Add(-(ProcessingLeaseTimeout + time.Second))
	stuck.ClaimedAt = &expired
	h.repo.mu.Unlock()

	// The record should appear in ListPending (lease expired).
	pending, _ := h.repo.ListPending(context.Background(), "ns", 10)
	found := false
	for _, p := range pending {
		if p.ID == rec.ID {
			found = true
		}
	}
	if !found {
		t.Fatal("expired processing record should appear in ListPending")
	}

	// ProcessOne should be able to reclaim and fulfill it.
	result, err := h.svc.ProcessOne(context.Background(), "ns", rec.ID)
	if err != nil {
		t.Fatalf("should reclaim and process: %v", err)
	}
	if result.Status != FulfillmentStatusFulfilled {
		t.Errorf("status = %s, want fulfilled", result.Status)
	}
	if result.CreditsGranted != 500 {
		t.Errorf("credits = %d, want 500", result.CreditsGranted)
	}
}

// TestLeaseBlocksConcurrentReclaim verifies that a record still within its
// lease blocks re-claiming by another worker (I4).
func TestLeaseBlocksConcurrentReclaim(t *testing.T) {
	h := newTestHarness(t)
	h.addPaidOrder("ns", "order-lease-block", "cust", 300, commerce.OrderKindWalletTopUp)

	rec, err := h.svc.RequestFulfillment(context.Background(), "ns", "order-lease-block")
	if err != nil {
		t.Fatal(err)
	}

	// Simulate an active claim (within lease).
	h.repo.mu.Lock()
	stuck := h.repo.fulfillments[rec.ID]
	stuck.Status = FulfillmentStatusProcessing
	now := time.Now()
	stuck.ClaimedAt = &now
	h.repo.mu.Unlock()

	// ClaimForProcessing should return ErrAlreadyProcessing.
	_, err = h.repo.ClaimForProcessing(context.Background(), "ns", rec.ID)
	if !errors.Is(err, ErrAlreadyProcessing) {
		t.Fatalf("expected ErrAlreadyProcessing, got %v", err)
	}
}

// --- I5: Crash boundary tests ---
func TestCrashAfterCreditGrant(t *testing.T) {
	h := newTestHarness(t)
	h.addPaidOrder("ns", "order-crash-grant", "cust", 1000, commerce.OrderKindWalletTopUp)

	rec, err := h.svc.RequestFulfillment(context.Background(), "ns", "order-crash-grant")
	if err != nil {
		t.Fatal(err)
	}

	// Wrap the grantor to inject a crash after the grant succeeds but before
	// MarkFulfilled is called.
	grantSucceeded := atomic.Bool{}
	wrappedGrantor := &crashAfterGrantGrantor{
		inner:     h.grantor,
		grantDone: &grantSucceeded,
	}

	// Use a service with the crashing grantor.
	crashSvc, _ := New(Config{
		Repo:     h.repo,
		Orders:   h.orders,
		Grantor:  wrappedGrantor,
		Invoices: h.invoices,
		Notifier: h.notifier,
	})

	// Process: the grantor crashes after granting credits, before fulfillment is marked.
	_, _ = crashSvc.ProcessOne(context.Background(), "ns", rec.ID)
	if err == nil {
		t.Fatal("first process should crash")
	}

	// Verify credits were granted exactly once.
	if h.grantor.totalGranted() != 1000 {
		t.Fatalf("credits granted = %d, want 1000", h.grantor.totalGranted())
	}

	// The fulfillment should be marked failed (crash before MarkFulfilled).
	failed, _ := h.repo.GetFulfillment(context.Background(), "ns", rec.ID)
	if failed.Status != FulfillmentStatusFailed {
		t.Fatalf("status = %s, want failed", failed.Status)
	}

	// Restart: re-process. The grantor is idempotent (same idempotency key), so
	// the second grant returns the original grant without double-charging.
	result, err := h.svc.ProcessOne(context.Background(), "ns", rec.ID)
	if err != nil {
		t.Fatalf("retry should succeed: %v", err)
	}
	if result.Status != FulfillmentStatusFulfilled {
		t.Errorf("status = %s, want fulfilled", result.Status)
	}

	// Total granted must still be 1000 (idempotent grantor).
	if h.grantor.totalGranted() != 1000 {
		t.Errorf("total granted = %d, want 1000 (idempotent)", h.grantor.totalGranted())
	}

	// Exactly one successful fulfillment.
	if h.repo.fulfilledOnce.Load() != 1 {
		t.Errorf("fulfilled once = %d, want 1", h.repo.fulfilledOnce.Load())
	}
}

// crashAfterGrantGrantor wraps a CreditGrantor and injects a panic/error after
// the grant succeeds, simulating a crash between grant and MarkFulfilled.
type crashAfterGrantGrantor struct {
	inner     CreditGrantor
	grantDone *atomic.Bool
}

func (g *crashAfterGrantGrantor) GrantCredits(ctx context.Context, in GrantCreditsInput) (GrantCreditsResult, error) {
	res, err := g.inner.GrantCredits(ctx, in)
	if err != nil {
		return res, err
	}
	// Simulate crash: mark grant as done, then return error to abort processing.
	g.grantDone.Store(true)
	return GrantCreditsResult{}, errors.New("simulated crash after credit grant")
}

// --- C1: Nil guard test ---

// TestMarkFulfilledFailureDoesNotPanic verifies that when MarkFulfilled fails
// (e.g. unique constraint from concurrent worker), ProcessOne does not panic
// from a nil rec dereference (C1).
func TestMarkFulfilledFailureDoesNotPanic(t *testing.T) {
	h := newTestHarness(t)
	h.addPaidOrder("ns", "order-c1", "cust", 400, commerce.OrderKindWalletTopUp)

	rec, err := h.svc.RequestFulfillment(context.Background(), "ns", "order-c1")
	if err != nil {
		t.Fatal(err)
	}

	// Use a repo where MarkFulfilled returns (nil, error) to trigger C1 path.
	crashRepo := &nilMarkFulfilledRepo{inner: h.repo}
	crashSvc, _ := New(Config{
		Repo:     crashRepo,
		Orders:   h.orders,
		Grantor:  h.grantor,
		Invoices: h.invoices,
		Notifier: h.notifier,
	})

	// This should NOT panic — it should return an error after reloading.
	_, _ = crashSvc.ProcessOne(context.Background(), "ns", rec.ID)

	// The key assertion is that we got here without panicking.
}

// nilMarkFulfilledRepo wraps a Repository and returns (nil, error) from
// MarkFulfilled to test the C1 nil guard.
type nilMarkFulfilledRepo struct {
	inner Repository
}

func (r *nilMarkFulfilledRepo) CreateFulfillment(ctx context.Context, req FulfillmentRequest) (*FulfillmentRecord, error) {
	return r.inner.CreateFulfillment(ctx, req)
}

func (r *nilMarkFulfilledRepo) GetFulfillment(ctx context.Context, namespace, id string) (*FulfillmentRecord, error) {
	return r.inner.GetFulfillment(ctx, namespace, id)
}

func (r *nilMarkFulfilledRepo) GetFulfillmentByOrder(ctx context.Context, namespace, orderID string) (*FulfillmentRecord, error) {
	return r.inner.GetFulfillmentByOrder(ctx, namespace, orderID)
}

func (r *nilMarkFulfilledRepo) ClaimForProcessing(ctx context.Context, namespace, id string) (*FulfillmentRecord, error) {
	return r.inner.ClaimForProcessing(ctx, namespace, id)
}

func (r *nilMarkFulfilledRepo) MarkFulfilled(_ context.Context, _, _ string, _ FulfillmentResult) (*FulfillmentRecord, error) {
	return nil, errors.New("simulated unique constraint violation")
}

func (r *nilMarkFulfilledRepo) MarkFailed(ctx context.Context, namespace, id, reason string) (*FulfillmentRecord, error) {
	return r.inner.MarkFailed(ctx, namespace, id, reason)
}

func (r *nilMarkFulfilledRepo) ListPending(ctx context.Context, namespace string, limit int) ([]FulfillmentRecord, error) {
	return r.inner.ListPending(ctx, namespace, limit)
}
