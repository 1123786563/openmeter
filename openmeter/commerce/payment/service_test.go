package payment

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/openmeterio/openmeter/openmeter/commerce"
	"github.com/openmeterio/openmeter/pkg/models"
)

// --- Mock implementations ---

type mockAttemptRepo struct {
	mu       sync.Mutex
	attempts map[string]*PaymentAttempt
	byIdem   map[string]string
	statusN  atomic.Int64
}

func newMockAttemptRepo() *mockAttemptRepo {
	return &mockAttemptRepo{
		attempts: make(map[string]*PaymentAttempt),
		byIdem:   make(map[string]string),
	}
}

func (m *mockAttemptRepo) CreateAttempt(_ context.Context, a PaymentAttempt) (*PaymentAttempt, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := a.Namespace + "/" + a.CustomerID + "/" + a.IdempotencyKey
	if id, ok := m.byIdem[key]; ok {
		return m.attempts[id], false, nil
	}
	saved := a
	m.attempts[a.ID] = &saved
	m.byIdem[key] = a.ID
	return &saved, true, nil
}

func (m *mockAttemptRepo) GetAttempt(_ context.Context, namespace, id string) (*PaymentAttempt, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	a, ok := m.attempts[id]
	if !ok || a.Namespace != namespace {
		return nil, ErrPaymentAttemptNotFound
	}
	cp := *a
	return &cp, nil
}

func (m *mockAttemptRepo) GetAttemptByIdempotencyKey(_ context.Context, namespace, customerID, key string) (*PaymentAttempt, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id, ok := m.byIdem[namespace+"/"+customerID+"/"+key]
	if !ok {
		return nil, ErrPaymentAttemptNotFound
	}
	cp := *m.attempts[id]
	return &cp, nil
}

func (m *mockAttemptRepo) GetAttemptByProviderOrder(_ context.Context, namespace string, provider Provider, providerOrderID string) (*PaymentAttempt, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, a := range m.attempts {
		if (namespace == "" || a.Namespace == namespace) && a.Provider == provider && a.ProviderOrderID == providerOrderID {
			cp := *a
			return &cp, nil
		}
	}
	return nil, ErrPaymentAttemptNotFound
}

func (m *mockAttemptRepo) UpdateAttemptStatus(_ context.Context, namespace, id string, expectedFrom, to AttemptStatus) (*PaymentAttempt, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.statusN.Add(1)
	a, ok := m.attempts[id]
	if !ok || a.Namespace != namespace {
		return nil, ErrPaymentAttemptNotFound
	}
	if a.Status != expectedFrom {
		return nil, fmt.Errorf("status mismatch: %s -> %s", a.Status, to)
	}
	a.Status = to
	a.UpdatedAt = time.Now()
	cp := *a
	return &cp, nil
}

func (m *mockAttemptRepo) SetProviderIDs(_ context.Context, namespace, id, orderID, paymentID, sessionID string) (*PaymentAttempt, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	a, ok := m.attempts[id]
	if !ok || a.Namespace != namespace {
		return nil, ErrPaymentAttemptNotFound
	}
	a.ProviderOrderID = orderID
	a.ProviderPaymentID = paymentID
	a.ProviderSessionID = sessionID
	return a, nil
}

type mockFactRepo struct {
	mu        sync.Mutex
	facts     map[string]*PaymentFactRecord
	byRawHash map[string]string
	byEvent   map[string]string
	insertN   atomic.Int64
}

func newMockFactRepo() *mockFactRepo {
	return &mockFactRepo{
		facts:     make(map[string]*PaymentFactRecord),
		byRawHash: make(map[string]string),
		byEvent:   make(map[string]string),
	}
}

func (m *mockFactRepo) InsertFact(_ context.Context, f PaymentFactRecord) (*PaymentFactRecord, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.insertN.Add(1)

	// Dedup by raw hash.
	if id, ok := m.byRawHash[f.Namespace+"/"+f.RawHash]; ok {
		return m.facts[id], false, nil
	}
	saved := f
	m.facts[f.ID] = &saved
	m.byRawHash[f.Namespace+"/"+f.RawHash] = f.ID
	if f.ProviderEventID != "" {
		m.byEvent[f.Namespace+"/"+string(f.Provider)+"/"+f.ProviderEventID] = f.ID
	}
	return &saved, true, nil
}

func (m *mockFactRepo) GetFactByRawHash(_ context.Context, namespace, rawHash string) (*PaymentFactRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id, ok := m.byRawHash[namespace+"/"+rawHash]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	return m.facts[id], nil
}

func (m *mockFactRepo) GetFactsByProviderOrder(_ context.Context, namespace string, provider Provider, providerOrderID string) ([]PaymentFactRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []PaymentFactRecord
	for _, f := range m.facts {
		if f.Namespace == namespace && f.Provider == provider && f.ProviderOrderID == providerOrderID {
			result = append(result, *f)
		}
	}
	return result, nil
}

func (m *mockFactRepo) GetFactByProviderEvent(_ context.Context, namespace string, provider Provider, eventID string) (*PaymentFactRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id, ok := m.byEvent[namespace+"/"+string(provider)+"/"+eventID]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	return m.facts[id], nil
}

type mockOrderStatusUpdater struct {
	mu     sync.Mutex
	orders map[string]*commerce.Order
	paidN  atomic.Int64
}

func newMockOrderUpdater() *mockOrderStatusUpdater {
	return &mockOrderStatusUpdater{orders: make(map[string]*commerce.Order)}
}

func (m *mockOrderStatusUpdater) GetOrder(_ context.Context, namespace, id string) (*commerce.Order, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	o, ok := m.orders[id]
	if !ok {
		return nil, commerce.ErrOrderNotFound
	}
	return o, nil
}

func (m *mockOrderStatusUpdater) UpdateOrderStatus(_ context.Context, namespace, id string, expectedFrom, to commerce.OrderStatus) (*commerce.Order, error) {
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
	if to == commerce.OrderStatusPaid {
		m.paidN.Add(1)
	}
	return o, nil
}

// mockProvider is a test ProviderAdapter that returns a configured PaymentFact.
type mockProvider struct {
	name     Provider
	verifyFn func(ctx context.Context, headers http.Header, body []byte) (PaymentFact, error)
	queryFn  func(ctx context.Context, providerOrderID string) (PaymentFact, error)
	qrFn     func(ctx context.Context, in CheckoutInput) (CheckoutFact, error)
}

func (m *mockProvider) CreateQRCode(ctx context.Context, in CheckoutInput) (CheckoutFact, error) {
	if m.qrFn != nil {
		return m.qrFn(ctx, in)
	}
	return CheckoutFact{Provider: m.name, ProviderOrderID: in.OrderPublicID}, nil
}

func (m *mockProvider) VerifyCallback(ctx context.Context, headers http.Header, body []byte) (PaymentFact, error) {
	return m.verifyFn(ctx, headers, body)
}

func (m *mockProvider) QueryPayment(ctx context.Context, providerOrderID string) (PaymentFact, error) {
	if m.queryFn != nil {
		return m.queryFn(ctx, providerOrderID)
	}
	return PaymentFact{}, nil
}

func (m *mockProvider) Refund(_ context.Context, _ RefundInput) (RefundSubmission, error) {
	return RefundSubmission{Provider: m.name, Status: "processing"}, nil
}

func (m *mockProvider) QueryRefund(_ context.Context, _ string) (RefundFact, error) {
	return RefundFact{Provider: m.name}, nil
}
func (m *mockProvider) Name() Provider { return m.name }

// --- Helpers to set up a complete test scenario ---

type testHarness struct {
	svc      Service
	attempts *mockAttemptRepo
	facts    *mockFactRepo
	orders   *mockOrderStatusUpdater
	provider *mockProvider
}

func newTestHarness(t *testing.T) *testHarness {
	t.Helper()
	attempts := newMockAttemptRepo()
	facts := newMockFactRepo()
	orders := newMockOrderUpdater()
	prov := &mockProvider{name: ProviderWeChat}

	svc, err := New(Config{
		Attempts:  attempts,
		Facts:     facts,
		Orders:    orders,
		Providers: map[Provider]ProviderAdapter{ProviderWeChat: prov},
	})
	if err != nil {
		t.Fatal(err)
	}

	return &testHarness{svc: svc, attempts: attempts, facts: facts, orders: orders, provider: prov}
}

// setupPaidOrder creates an order + attempt and transitions the order to awaiting_payment.
func (h *testHarness) setupPaidOrder(namespace, customerID, orderID, providerOrderID string, amount int64) (*commerce.Order, *PaymentAttempt) {
	order := &commerce.Order{
		NamespacedID: models.NamespacedID{Namespace: namespace, ID: orderID},
		CustomerID:   customerID,
		Kind:         commerce.OrderKindWalletTopUp,
		Status:       commerce.OrderStatusAwaitingPayment,
		AmountMinor:  amount,
		Currency:     "CNY",
		PublicID:     "PUB-" + orderID,
		Lines: []commerce.OrderLineSnapshot{
			{Credits: amount, Currency: "CNY", SubtotalMinor: amount, UnitPriceMinor: amount},
		},
	}
	h.orders.orders[orderID] = order

	attempt := &PaymentAttempt{
		ID:              "att-" + orderID,
		Namespace:       namespace,
		OrderID:         orderID,
		CustomerID:      customerID,
		Provider:        ProviderWeChat,
		ProviderOrderID: providerOrderID,
		Status:          AttemptStatusPending,
		IdempotencyKey:  "idem-" + orderID,
		AmountMinor:     amount,
		Currency:        "CNY",
	}
	h.attempts.attempts[attempt.ID] = attempt
	h.attempts.byIdem[namespace+"/"+customerID+"/"+attempt.IdempotencyKey] = attempt.ID

	return order, attempt
}

// --- Tests ---

// TestHandleCallbackValidPayment verifies the happy path: valid callback moves
// the order to paid (not fulfilled).
func TestHandleCallbackValidPayment(t *testing.T) {
	h := newTestHarness(t)
	_, _ = h.setupPaidOrder("ns", "cust", "order-1", "PROV-ORDER-1", 100)

	h.provider.verifyFn = func(_ context.Context, _ http.Header, body []byte) (PaymentFact, error) {
		return PaymentFact{
			Provider:        ProviderWeChat,
			ProviderOrderID: "PROV-ORDER-1",
			AmountMinor:     100,
			Currency:        "CNY",
			Success:         true,
			RawHash:         HashRawBody(body),
		}, nil
	}

	result, err := h.svc.HandleCallback(context.Background(), ProviderWeChat, nil, []byte("body-1"))
	if err != nil {
		t.Fatalf("callback should succeed: %v", err)
	}

	if result.Fact == nil {
		t.Fatal("fact should be persisted")
	}
	if !result.Fact.Success {
		t.Error("fact should be success")
	}

	// Order should be paid.
	order, _ := h.orders.GetOrder(context.Background(), "ns", "order-1")
	if order.Status != commerce.OrderStatusPaid {
		t.Errorf("order status = %s, want paid", order.Status)
	}

	// Paid does NOT imply fulfilled.
	if order.Status == commerce.OrderStatusFulfilled {
		t.Error("paid should not imply fulfilled")
	}

	// Attempt should be succeeded.
	if result.Attempt.Status != AttemptStatusSucceeded {
		t.Errorf("attempt status = %s, want succeeded", result.Attempt.Status)
	}
}

// TestHandleCallbackDuplicateEventId verifies that a duplicate provider event
// returns the original result without creating a new fact.
func TestHandleCallbackDuplicateEventId(t *testing.T) {
	h := newTestHarness(t)
	_, _ = h.setupPaidOrder("ns", "cust", "order-2", "PROV-ORDER-2", 200)

	body := []byte("body-2")
	hash := HashRawBody(body)

	callCount := 0
	h.provider.verifyFn = func(_ context.Context, _ http.Header, _ []byte) (PaymentFact, error) {
		callCount++
		return PaymentFact{
			Provider:        ProviderWeChat,
			ProviderOrderID: "PROV-ORDER-2",
			ProviderEventID: "EVENT-001",
			AmountMinor:     200,
			Currency:        "CNY",
			Success:         true,
			RawHash:         hash,
		}, nil
	}

	// First callback.
	_, err := h.svc.HandleCallback(context.Background(), ProviderWeChat, nil, body)
	if err != nil {
		t.Fatal(err)
	}

	firstInserts := h.facts.insertN.Load()

	// Duplicate callback (same raw hash + event ID).
	result, err := h.svc.HandleCallback(context.Background(), ProviderWeChat, nil, body)
	if err != nil {
		t.Fatal(err)
	}

	// No new fact should be inserted.
	if h.facts.insertN.Load() != firstInserts {
		t.Errorf("duplicate callback should not insert new fact: inserts=%d", h.facts.insertN.Load())
	}

	if !result.AlreadyPaid {
		t.Error("duplicate callback should set AlreadyPaid")
	}
}

// TestHandleCallbackWrongAmount verifies that a fact with the wrong amount is rejected.
func TestHandleCallbackWrongAmount(t *testing.T) {
	h := newTestHarness(t)
	_, _ = h.setupPaidOrder("ns", "cust", "order-3", "PROV-ORDER-3", 100)

	h.provider.verifyFn = func(_ context.Context, _ http.Header, body []byte) (PaymentFact, error) {
		return PaymentFact{
			Provider:        ProviderWeChat,
			ProviderOrderID: "PROV-ORDER-3",
			AmountMinor:     999, // wrong!
			Currency:        "CNY",
			Success:         true,
			RawHash:         HashRawBody(body),
		}, nil
	}

	_, err := h.svc.HandleCallback(context.Background(), ProviderWeChat, nil, []byte("body-3"))
	if err == nil {
		t.Fatal("wrong amount should fail")
	}

	// Order should NOT be paid.
	order, _ := h.orders.GetOrder(context.Background(), "ns", "order-3")
	if order.Status == commerce.OrderStatusPaid {
		t.Error("order should not be paid with wrong amount")
	}
}

// TestHandleCallbackWrongCurrency verifies that a fact with the wrong currency is rejected.
func TestHandleCallbackWrongCurrency(t *testing.T) {
	h := newTestHarness(t)
	_, _ = h.setupPaidOrder("ns", "cust", "order-4", "PROV-ORDER-4", 100)

	h.provider.verifyFn = func(_ context.Context, _ http.Header, body []byte) (PaymentFact, error) {
		return PaymentFact{
			Provider:        ProviderWeChat,
			ProviderOrderID: "PROV-ORDER-4",
			AmountMinor:     100,
			Currency:        "USD", // wrong!
			Success:         true,
			RawHash:         HashRawBody(body),
		}, nil
	}

	_, err := h.svc.HandleCallback(context.Background(), ProviderWeChat, nil, []byte("body-4"))
	if err == nil {
		t.Fatal("wrong currency should fail")
	}
}

// TestHandleCallbackContradictoryFact verifies that a contradictory success fact
// for the same provider order is rejected. The second callback has a different
// event ID and raw body, but reports a different success amount for the same
// provider order — that's a contradiction.
func TestHandleCallbackContradictoryFact(t *testing.T) {
	h := newTestHarness(t)
	_, _ = h.setupPaidOrder("ns", "cust", "order-5", "PROV-ORDER-5", 100)

	// First callback: amount=100, event A.
	h.provider.verifyFn = func(_ context.Context, _ http.Header, body []byte) (PaymentFact, error) {
		return PaymentFact{
			Provider:        ProviderWeChat,
			ProviderOrderID: "PROV-ORDER-5",
			ProviderEventID: "EVENT-A",
			AmountMinor:     100,
			Currency:        "CNY",
			Success:         true,
			RawHash:         HashRawBody(body),
		}, nil
	}

	_, err := h.svc.HandleCallback(context.Background(), ProviderWeChat, nil, []byte("body-5a"))
	if err != nil {
		t.Fatal(err)
	}

	// Second callback: contradictory amount (999 vs 100) for same order, different event.
	h.provider.verifyFn = func(_ context.Context, _ http.Header, body []byte) (PaymentFact, error) {
		return PaymentFact{
			Provider:        ProviderWeChat,
			ProviderOrderID: "PROV-ORDER-5",
			ProviderEventID: "EVENT-B",
			AmountMinor:     999, // contradicts the first fact (100)
			Currency:        "CNY",
			Success:         true,
			RawHash:         HashRawBody(body),
		}, nil
	}

	_, err = h.svc.HandleCallback(context.Background(), ProviderWeChat, nil, []byte("body-5b"))
	if err == nil {
		t.Fatal("contradictory fact should fail")
	}
}

// TestHandleCallbackInvalidSignature verifies that an invalid signature is rejected.
func TestHandleCallbackInvalidSignature(t *testing.T) {
	h := newTestHarness(t)
	_, _ = h.setupPaidOrder("ns", "cust", "order-6", "PROV-ORDER-6", 100)

	h.provider.verifyFn = func(_ context.Context, _ http.Header, _ []byte) (PaymentFact, error) {
		return PaymentFact{}, ErrInvalidSignature
	}

	_, err := h.svc.HandleCallback(context.Background(), ProviderWeChat, nil, []byte("body-6"))
	if err == nil {
		t.Fatal("invalid signature should fail")
	}
}

// TestConfirmPaymentCallbackLost verifies that ConfirmPayment queries the
// provider when the callback is lost.
func TestConfirmPaymentCallbackLost(t *testing.T) {
	h := newTestHarness(t)
	_, attempt := h.setupPaidOrder("ns", "cust", "order-7", "PROV-ORDER-7", 100)

	queryCalled := false
	h.provider.queryFn = func(_ context.Context, providerOrderID string) (PaymentFact, error) {
		queryCalled = true
		return PaymentFact{
			Provider:        ProviderWeChat,
			ProviderOrderID: providerOrderID,
			AmountMinor:     100,
			Currency:        "CNY",
			Success:         true,
			RawHash:         "query-hash-7",
		}, nil
	}

	result, err := h.svc.ConfirmPayment(context.Background(), "ns", attempt.ID)
	if err != nil {
		t.Fatalf("confirm payment should succeed: %v", err)
	}

	if !queryCalled {
		t.Fatal("provider query should be called")
	}

	if !result.Fact.Success {
		t.Error("fact should be success")
	}

	order, _ := h.orders.GetOrder(context.Background(), "ns", "order-7")
	if order.Status != commerce.OrderStatusPaid {
		t.Errorf("order status = %s, want paid", order.Status)
	}
}

// TestCreateAttemptIdempotent verifies that creating an attempt with the same
// idempotency key returns the existing attempt.
func TestCreateAttemptIdempotent(t *testing.T) {
	h := newTestHarness(t)
	h.orders.orders["order-8"] = &commerce.Order{
		NamespacedID: models.NamespacedID{Namespace: "ns", ID: "order-8"},
		Status:       commerce.OrderStatusCreated,
		AmountMinor:  100,
		Currency:     "CNY",
	}

	in := CreateAttemptInput{
		Namespace:      "ns",
		OrderID:        "order-8",
		CustomerID:     "cust",
		Provider:       ProviderWeChat,
		IdempotencyKey: "idem-8",
		AmountMinor:    100,
		Currency:       "CNY",
	}

	first, isNew, err := h.svc.CreateAttempt(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	if !isNew {
		t.Fatal("first create should be fresh")
	}

	second, isNew2, err := h.svc.CreateAttempt(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	if isNew2 {
		t.Fatal("replay should not be fresh")
	}
	if first.ID != second.ID {
		t.Fatalf("idempotent replay returned different ID: %s vs %s", first.ID, second.ID)
	}
}

// TestConcurrentCallbacks verifies that concurrent callbacks for the same order
// result in exactly one paid transition.
func TestConcurrentCallbacks(t *testing.T) {
	h := newTestHarness(t)
	_, _ = h.setupPaidOrder("ns", "cust", "order-9", "PROV-ORDER-9", 100)

	// All callbacks use the same body (same raw hash + event ID) so the dedup
	// path is exercised under concurrency.
	body := []byte("body-9")
	h.provider.verifyFn = func(_ context.Context, _ http.Header, b []byte) (PaymentFact, error) {
		return PaymentFact{
			Provider:        ProviderWeChat,
			ProviderOrderID: "PROV-ORDER-9",
			ProviderEventID: "EVENT-9",
			AmountMinor:     100,
			Currency:        "CNY",
			Success:         true,
			RawHash:         HashRawBody(b),
		}, nil
	}

	var wg sync.WaitGroup
	var successCount atomic.Int64
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := h.svc.HandleCallback(context.Background(), ProviderWeChat, nil, body)
			if err == nil {
				successCount.Add(1)
			}
		}()
	}
	wg.Wait()

	// The order should be paid.
	order, _ := h.orders.GetOrder(context.Background(), "ns", "order-9")
	if order.Status != commerce.OrderStatusPaid {
		t.Fatalf("order status = %s, want paid", order.Status)
	}

	// The order must be in the paid state (exactly one successful transition).
	// Note: the mock has a TOCTOU window between the dedup check and InsertFact
	// (both under separate locks), so concurrent first-time callbacks for the
	// same fact may insert multiple records. In production, the database unique
	// index on raw_hash handles this atomically. The invariant we verify here
	// is that the order converges to paid.
	if successCount.Load() == 0 {
		t.Error("at least one callback should succeed")
	}

	// The order must be paid exactly once.
	if h.orders.paidN.Load() != 1 {
		t.Errorf("expected exactly 1 paid transition, got %d", h.orders.paidN.Load())
	}
}

// TestInitiateCheckout verifies the checkout flow stores provider IDs and
// transitions the attempt to pending.
func TestInitiateCheckout(t *testing.T) {
	h := newTestHarness(t)
	h.orders.orders["order-10"] = &commerce.Order{
		NamespacedID: models.NamespacedID{Namespace: "ns", ID: "order-10"},
		CustomerID:   "cust",
		Status:       commerce.OrderStatusAwaitingPayment,
		AmountMinor:  100,
		Currency:     "CNY",
		PublicID:     "PUB-10",
	}
	h.attempts.attempts["att-10"] = &PaymentAttempt{
		ID:          "att-10",
		Namespace:   "ns",
		OrderID:     "order-10",
		CustomerID:  "cust",
		Provider:    ProviderWeChat,
		Status:      AttemptStatusCreated,
		AmountMinor: 100,
		Currency:    "CNY",
	}
	h.attempts.byIdem["ns/cust/idem-10"] = "att-10"

	h.provider.qrFn = func(_ context.Context, in CheckoutInput) (CheckoutFact, error) {
		return CheckoutFact{
			Provider:        ProviderWeChat,
			ProviderOrderID: "WX-ORDER-10",
			QRCodeURL:       "weixin://wxpay/bizpayurl?pr=10",
		}, nil
	}

	result, err := h.svc.InitiateCheckout(context.Background(), "ns", "att-10")
	if err != nil {
		t.Fatalf("initiate checkout: %v", err)
	}
	if result.Fact.QRCodeURL == "" {
		t.Error("qr code url should be set")
	}
	if result.Attempt.ProviderOrderID != "WX-ORDER-10" {
		t.Errorf("provider_order_id = %s", result.Attempt.ProviderOrderID)
	}
	if result.Attempt.Status != AttemptStatusPending {
		t.Errorf("status = %s, want pending", result.Attempt.Status)
	}
}

// TestGetAttempt verifies retrieval by ID.
func TestGetAttempt(t *testing.T) {
	h := newTestHarness(t)
	h.attempts.attempts["att-11"] = &PaymentAttempt{
		ID:        "att-11",
		Namespace: "ns",
		OrderID:   "order-11",
		Status:    AttemptStatusPending,
	}

	a, err := h.svc.GetAttempt(context.Background(), "ns", "att-11")
	if err != nil {
		t.Fatal(err)
	}
	if a.ID != "att-11" {
		t.Errorf("id = %s", a.ID)
	}

	// Not found.
	_, err = h.svc.GetAttempt(context.Background(), "ns", "nonexistent")
	if err == nil {
		t.Fatal("should return error for nonexistent attempt")
	}
}

// TestNewValidationErrors verifies that New rejects nil dependencies.
func TestNewValidationErrors(t *testing.T) {
	_, err := New(Config{})
	if err == nil {
		t.Fatal("should reject nil config")
	}
}

// TestCreateAttemptValidation verifies input validation.
func TestCreateAttemptValidation(t *testing.T) {
	h := newTestHarness(t)

	_, _, err := h.svc.CreateAttempt(context.Background(), CreateAttemptInput{
		Namespace: "", // missing
	})
	if err == nil {
		t.Fatal("should reject empty namespace")
	}

	_, _, err = h.svc.CreateAttempt(context.Background(), CreateAttemptInput{
		Namespace:      "ns",
		OrderID:        "o",
		CustomerID:     "c",
		IdempotencyKey: "k",
		Provider:       Provider("invalid"),
		AmountMinor:    100,
		Currency:       "CNY",
	})
	if err == nil {
		t.Fatal("should reject invalid provider")
	}
}

// TestHashRawBody verifies deterministic hashing.
func TestHashRawBody(t *testing.T) {
	h1 := HashRawBody([]byte("test"))
	h2 := HashRawBody([]byte("test"))
	if h1 != h2 {
		t.Fatal("same input should produce same hash")
	}
	if len(h1) != 64 {
		t.Fatalf("sha256 hex should be 64 chars, got %d", len(h1))
	}
}
