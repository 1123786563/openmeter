package refund

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/commerce"
	"github.com/openmeterio/openmeter/openmeter/commerce/payment"
	"github.com/openmeterio/openmeter/pkg/models"
)

// ---------------------------------------------------------------------------
// Mock: Repository
// ---------------------------------------------------------------------------

type mockRepo struct {
	mu            sync.Mutex
	refunds       map[string]*RefundRequest
	facts         map[string]*RefundFactRecord // keyed by RawHash
	appendErr     error
	idCounter     atomic.Int64
	grantStore    *mockWallet // shared grant store for atomic reserve
	lastListInput ListRefundsInput
}

func newMockRepo(wallet *mockWallet) *mockRepo {
	return &mockRepo{
		refunds:    make(map[string]*RefundRequest),
		facts:      make(map[string]*RefundFactRecord),
		grantStore: wallet,
	}
}

func (m *mockRepo) CreateRefund(_ context.Context, req RefundRequest) (*RefundRequest, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, r := range m.refunds {
		if r.Namespace == req.Namespace && r.CustomerID == req.CustomerID && r.IdempotencyKey == req.IdempotencyKey {
			cp := *r
			return &cp, false, nil
		}
	}
	if req.ID == "" {
		req.ID = fmt.Sprintf("rf-%d", m.idCounter.Add(1))
	}
	req.UpdatedAt = time.Now()
	m.refunds[req.ID] = &req
	cp := req
	return &cp, true, nil
}

func (m *mockRepo) GetRefund(_ context.Context, namespace, id string) (*RefundRequest, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.refunds[id]
	if !ok || r.Namespace != namespace {
		return nil, ErrRefundNotFound
	}
	cp := *r
	return &cp, nil
}

func (m *mockRepo) GetRefundByIdempotencyKey(_ context.Context, namespace, customerID, key string) (*RefundRequest, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, r := range m.refunds {
		if r.Namespace == namespace && r.CustomerID == customerID && r.IdempotencyKey == key {
			cp := *r
			return &cp, nil
		}
	}
	return nil, ErrRefundNotFound
}

func (m *mockRepo) GetRefundByProviderRefundID(_ context.Context, namespace, providerRefundID string) (*RefundRequest, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, r := range m.refunds {
		if r.Namespace == namespace && r.ProviderRefundID == providerRefundID {
			cp := *r
			return &cp, nil
		}
	}
	return nil, ErrRefundNotFound
}

func (m *mockRepo) TransitionStatus(_ context.Context, namespace, id string, expectedFrom, to RefundStatus) (*RefundRequest, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.refunds[id]
	if !ok || r.Namespace != namespace {
		return nil, ErrRefundNotFound
	}
	if r.Status != expectedFrom {
		return nil, fmt.Errorf("%w: current=%s, expected=%s", ErrInvalidRefundTransition, r.Status, expectedFrom)
	}
	r.Status = to
	r.UpdatedAt = time.Now()
	cp := *r
	return &cp, nil
}

func (m *mockRepo) SaveQuantum(_ context.Context, namespace, id string, q QuantumReservation) (*RefundRequest, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.refunds[id]
	if !ok || r.Namespace != namespace {
		return nil, ErrRefundNotFound
	}
	r.CreditQuantum = q.CreditQuantum
	r.RefundQuantumFen = q.RefundQuantumFen
	r.ReservedCredits = q.ReservedCredits
	r.RefundFen = q.RefundFen
	r.RemainderCredits = q.RemainderCredits
	r.UpdatedAt = time.Now()
	cp := *r
	return &cp, nil
}

func (m *mockRepo) SetProviderRefundID(_ context.Context, namespace, id, providerName, providerRefundID string) (*RefundRequest, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.refunds[id]
	if !ok || r.Namespace != namespace {
		return nil, ErrRefundNotFound
	}
	r.ProviderName = providerName
	r.ProviderRefundID = providerRefundID
	r.UpdatedAt = time.Now()
	cp := *r
	return &cp, nil
}

func (m *mockRepo) SetFence(_ context.Context, namespace, id, fenceSequence string) (*RefundRequest, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.refunds[id]
	if !ok || r.Namespace != namespace {
		return nil, ErrRefundNotFound
	}
	r.FenceSequence = fenceSequence
	r.UpdatedAt = time.Now()
	cp := *r
	return &cp, nil
}

func (m *mockRepo) MarkFailed(_ context.Context, namespace, id, reason string) (*RefundRequest, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.refunds[id]
	if !ok || r.Namespace != namespace {
		return nil, ErrRefundNotFound
	}
	r.Status = RefundStatusFailed
	r.FailureReason = &reason
	r.UpdatedAt = time.Now()
	cp := *r
	return &cp, nil
}

// ReserveCredits atomically computes available refundable credits and reserves.
func (m *mockRepo) ReserveCredits(_ context.Context, refundID string, in ReservationInput) (ReservationResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Sum already-reserved credits from non-failed refunds for the same order.
	alreadyReserved := int64(0)
	for _, r := range m.refunds {
		if r.ID == refundID {
			continue
		}
		if r.CommerceOrderID == in.OrderID && r.Status != RefundStatusFailed {
			alreadyReserved += r.ReservedCredits
		}
	}

	available := in.RefundableCredits - alreadyReserved
	if available <= 0 {
		return ReservationResult{
			Granted:           false,
			RefundableCredits: in.RefundableCredits,
			AlreadyReserved:   alreadyReserved,
		}, nil
	}

	refundFen, reservedCredits, remainderCredits := ComputeRefundable(available, in.RequestedFen)
	if refundFen <= 0 {
		return ReservationResult{
			Granted:           false,
			RefundableCredits: in.RefundableCredits,
			AlreadyReserved:   alreadyReserved,
		}, nil
	}

	// Reserve on the refund record.
	r, ok := m.refunds[refundID]
	if !ok {
		return ReservationResult{}, ErrRefundNotFound
	}
	r.ReservedCredits = reservedCredits
	r.RefundFen = refundFen
	r.RemainderCredits = remainderCredits

	return ReservationResult{
		Granted:           true,
		RefundFen:         refundFen,
		ReservedCredits:   reservedCredits,
		RemainderCredits:  remainderCredits,
		RefundableCredits: in.RefundableCredits,
		AlreadyReserved:   alreadyReserved,
	}, nil
}

func (m *mockRepo) AppendFact(_ context.Context, fact RefundFactRecord) (*RefundFactRecord, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.appendErr != nil {
		return nil, false, m.appendErr
	}
	if fact.RawHash != "" {
		if existing, ok := m.facts[fact.RawHash]; ok {
			cp := *existing
			return &cp, false, nil
		}
	}
	m.facts[fact.RawHash] = &fact
	cp := fact
	return &cp, true, nil
}

func (m *mockRepo) GetFacts(_ context.Context, namespace, refundID string) ([]RefundFactRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []RefundFactRecord
	for _, f := range m.facts {
		if f.Namespace == namespace && f.RefundRequestID == refundID {
			result = append(result, *f)
		}
	}
	return result, nil
}

func (m *mockRepo) ListRefunds(_ context.Context, in ListRefundsInput) ([]RefundRequest, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastListInput = in
	var result []RefundRequest
	for _, r := range m.refunds {
		if r.Namespace != in.Namespace {
			continue
		}
		if in.CustomerID != "" && r.CustomerID != in.CustomerID {
			continue
		}
		if in.Status != nil && r.Status != *in.Status {
			continue
		}
		result = append(result, *r)
	}
	total := len(result)
	if in.Offset < len(result) {
		result = result[in.Offset:]
	} else {
		result = nil
	}
	if in.Limit > 0 && len(result) > in.Limit {
		result = result[:in.Limit]
	}
	return result, total, nil
}

// ---------------------------------------------------------------------------
// Mock: WalletDataPort
// ---------------------------------------------------------------------------

type mockWallet struct {
	mu     sync.Mutex
	grants map[string][]commerce.AllocationGrant // keyed by customerID
}

func newMockWallet() *mockWallet {
	return &mockWallet{grants: make(map[string][]commerce.AllocationGrant)}
}

func (w *mockWallet) GetGrants(_ context.Context, _ string, customerID string) ([]commerce.AllocationGrant, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	cp := make([]commerce.AllocationGrant, len(w.grants[customerID]))
	copy(cp, w.grants[customerID])
	return cp, nil
}

func (w *mockWallet) setGrants(customerID string, grants []commerce.AllocationGrant) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.grants[customerID] = grants
}

// ---------------------------------------------------------------------------
// Mock: OrderReader
// ---------------------------------------------------------------------------

type mockOrders struct {
	mu     sync.Mutex
	orders map[string]*commerce.Order
}

func newMockOrders() *mockOrders {
	return &mockOrders{orders: make(map[string]*commerce.Order)}
}

func (o *mockOrders) GetOrder(_ context.Context, _ string, id string) (*commerce.Order, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	ord, ok := o.orders[id]
	if !ok {
		return nil, commerce.ErrOrderNotFound
	}
	cp := *ord
	return &cp, nil
}

func (o *mockOrders) addFulfilledOrder(namespace, orderID, customerID string, amount int64) *commerce.Order {
	ord := &commerce.Order{
		NamespacedID: models.NamespacedID{Namespace: namespace, ID: orderID},
		PublicID:     "pub-" + orderID,
		CustomerID:   customerID,
		Kind:         commerce.OrderKindWalletTopUp,
		Status:       commerce.OrderStatusFulfilled,
		AmountMinor:  amount,
		Currency:     "CNY",
		Lines: []commerce.OrderLineSnapshot{
			{Credits: amount * 1000, Currency: "CNY", SubtotalMinor: amount},
		},
	}
	o.orders[orderID] = ord
	return ord
}

// ---------------------------------------------------------------------------
// Mock: FenceClient (per-customer mutex serialization)
// ---------------------------------------------------------------------------

type mockFenceClient struct {
	mu       sync.Mutex
	locks    map[string]*sync.Mutex // per-customer fence lock
	repo     *mockRepo
	released atomic.Int64
	failNext atomic.Bool
}

func newMockFenceClient(repo *mockRepo) *mockFenceClient {
	return &mockFenceClient{locks: make(map[string]*sync.Mutex), repo: repo}
}

func (f *mockFenceClient) getLock(customerID string) *sync.Mutex {
	f.mu.Lock()
	defer f.mu.Unlock()
	l, ok := f.locks[customerID]
	if !ok {
		l = &sync.Mutex{}
		f.locks[customerID] = l
	}
	return l
}

func (f *mockFenceClient) EstablishFence(ctx context.Context, namespace, customerID, refundID string) (FenceResult, error) {
	if f.failNext.Swap(false) {
		return FenceResult{}, ErrFenceTimeout
	}
	lock := f.getLock(customerID)
	lock.Lock()
	sequence := "fence-" + customerID
	if _, err := f.repo.SetFence(ctx, namespace, refundID, sequence); err != nil {
		lock.Unlock()
		return FenceResult{}, err
	}
	return FenceResult{Sequence: sequence, Established: true}, nil
}

func (f *mockFenceClient) ReleaseFence(_ context.Context, _ string, customerID, _, _ string) error {
	f.getLock(customerID).Unlock()
	f.released.Add(1)
	return nil
}

// ---------------------------------------------------------------------------
// Mock: CreditReverser
// ---------------------------------------------------------------------------

type mockReverser struct {
	mu       sync.Mutex
	reversed map[string]int64 // refundID -> credits reversed
	callN    atomic.Int64
}

func newMockReverser() *mockReverser {
	return &mockReverser{reversed: make(map[string]int64)}
}

func (r *mockReverser) ReverseCredits(_ context.Context, in ReverseCreditsInput) (ReverseCreditsResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.callN.Add(1)
	if _, ok := r.reversed[in.RefundID]; ok {
		return ReverseCreditsResult{LedgerEntryID: "ledger-" + in.RefundID, Credits: r.reversed[in.RefundID]}, nil
	}
	r.reversed[in.RefundID] = in.Credits
	return ReverseCreditsResult{LedgerEntryID: "ledger-" + in.RefundID, Credits: in.Credits}, nil
}

func (r *mockReverser) totalReversed() int64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	total := int64(0)
	for _, v := range r.reversed {
		total += v
	}
	return total
}

// ---------------------------------------------------------------------------
// Mock: ProviderRefunder
// ---------------------------------------------------------------------------

type mockProvider struct {
	mu                  sync.Mutex
	name                payment.Provider
	refundResult        func(input payment.RefundInput) (payment.RefundSubmission, error)
	queryResult         func(input payment.RefundQueryInput) (payment.RefundFact, error)
	callbackFact        payment.RefundFact
	callbackErr         error
	callbackHeaders     http.Header
	callbackBody        []byte
	queryCallN          atomic.Int64
	refundCallN         atomic.Int64
	refundCallbackCallN atomic.Int64
}

func newMockProvider(name payment.Provider) *mockProvider {
	return &mockProvider{
		name: name,
		refundResult: func(input payment.RefundInput) (payment.RefundSubmission, error) {
			return payment.RefundSubmission{
				Provider:         name,
				ProviderRefundID: input.IdempotencyKey,
				Status:           "success",
			}, nil
		},
		queryResult: func(input payment.RefundQueryInput) (payment.RefundFact, error) {
			return payment.RefundFact{
				Provider:         name,
				ProviderRefundID: input.ProviderRefundID,
				Success:          true,
				RawHash:          "hash-" + input.ProviderRefundID,
				AmountMinor:      0,
				Timestamp:        time.Now(),
			}, nil
		},
	}
}

func (p *mockProvider) Refund(ctx context.Context, input payment.RefundInput) (payment.RefundSubmission, error) {
	p.refundCallN.Add(1)
	return p.refundResult(input)
}

func (p *mockProvider) QueryRefund(_ context.Context, input payment.RefundQueryInput) (payment.RefundFact, error) {
	p.queryCallN.Add(1)
	return p.queryResult(input)
}

func (p *mockProvider) Name() payment.Provider { return p.name }

type mockProviderResolver struct {
	provider payment.Provider
	err      error
}

type mockPaymentAuthorityResolver struct {
	orders     *mockOrders
	provider   payment.Provider
	merchantID string
	authority  *PaymentAuthority
	err        error
}

func (r mockPaymentAuthorityResolver) ResolvePaymentAuthorityForOrder(ctx context.Context, namespace, orderID string) (PaymentAuthority, error) {
	if r.err != nil {
		return PaymentAuthority{}, r.err
	}
	if r.authority != nil {
		return *r.authority, nil
	}
	order, err := r.orders.GetOrder(ctx, namespace, orderID)
	if err != nil {
		return PaymentAuthority{}, err
	}
	return PaymentAuthority{
		Provider:          r.provider,
		ProviderOrderID:   order.PublicID,
		ProviderPaymentID: "payment-" + orderID,
		MerchantID:        r.merchantID,
		AmountMinor:       order.AmountMinor,
		Currency:          order.Currency,
	}, nil
}

func (r mockProviderResolver) ResolveProviderForOrder(context.Context, string, string) (payment.Provider, error) {
	return r.provider, r.err
}

// ---------------------------------------------------------------------------
// Test harness
// ---------------------------------------------------------------------------

type testHarness struct {
	svc      Service
	repo     *mockRepo
	wallet   *mockWallet
	orders   *mockOrders
	fence    *mockFenceClient
	reverser *mockReverser
	provider *mockProvider
}

func newTestHarness(t *testing.T) *testHarness {
	return newTestHarnessForProvider(t, payment.ProviderWeChat)
}

func newTestHarnessForProvider(t *testing.T, providerName payment.Provider) *testHarness {
	t.Helper()
	wallet := newMockWallet()
	orders := newMockOrders()
	reverser := newMockReverser()
	provider := newMockProvider(providerName)
	repo := newMockRepo(wallet)
	fence := newMockFenceClient(repo)
	merchantID := "wx-mch"
	if providerName == payment.ProviderAlipay {
		merchantID = "ali-seller"
	}

	svc, err := New(Config{
		Repo:                     repo,
		Orders:                   orders,
		Wallet:                   wallet,
		Fence:                    fence,
		Reverser:                 reverser,
		Providers:                map[payment.Provider]ProviderRefunder{providerName: provider},
		PaymentAuthorityResolver: mockPaymentAuthorityResolver{orders: orders, provider: providerName, merchantID: merchantID},
	})
	if err != nil {
		t.Fatal(err)
	}
	return &testHarness{
		svc: svc, repo: repo, wallet: wallet, orders: orders,
		fence: fence, reverser: reverser, provider: provider,
	}
}

func verifiedRefundFact(provider payment.Provider, orderID, providerRefundID, status, rawHash string, amountMinor int64) payment.RefundFact {
	merchantID := "wx-mch"
	successStatus := "SUCCESS"
	processingStatus := "PROCESSING"
	if provider == payment.ProviderAlipay {
		merchantID = "ali-seller"
		successStatus = "REFUND_SUCCESS"
		processingStatus = "REFUND_PROCESSING"
	}
	success := status == successStatus
	terminal := status != processingStatus
	return payment.RefundFact{
		Provider: provider, ProviderRefundID: providerRefundID,
		ProviderOrderID: "pub-" + orderID, ProviderPaymentID: "payment-" + orderID,
		MerchantID: merchantID, AmountMinor: amountMinor, TotalAmountMinor: 10000,
		Currency: "CNY", Status: status, Success: success, Terminal: terminal,
		RawHash: rawHash, Timestamp: time.Now(),
	}
}

func verifiedWechatRefundCallbackFact(orderID, providerRefundID, status, rawHash string, amountMinor int64) payment.RefundFact {
	return verifiedRefundFact(payment.ProviderWeChat, orderID, providerRefundID, status, rawHash, amountMinor)
}

// rechargeGrant creates a recharge-source grant with the given granted, consumed,
// and refundable credits.
func rechargeGrant(grantID string, granted, consumed, refundable int64) commerce.AllocationGrant {
	return commerce.AllocationGrant{
		GrantID:    grantID,
		Source:     commerce.BucketSourceRecharge,
		Granted:    granted,
		Consumed:   consumed,
		Refundable: refundable,
		Priority:   commerce.SourcePriority(commerce.BucketSourceRecharge),
		CreatedAt:  time.Now(),
	}
}

// ===========================================================================
// Step 1: Exact refund arithmetic tests
// ===========================================================================

func TestComputeRefundableZeroConsumed(t *testing.T) {
	// CNY 100.00 = 10,000 fen granting 100,000 Credit.
	refundFen, reservedCredits, remainder := ComputeRefundable(100000, 0)
	if refundFen != 10000 {
		t.Errorf("refundFen = %d, want 10000", refundFen)
	}
	if reservedCredits != 100000 {
		t.Errorf("reservedCredits = %d, want 100000", reservedCredits)
	}
	if remainder != 0 {
		t.Errorf("remainder = %d, want 0", remainder)
	}
	// 10,000 fen = CNY 100.00
	if refundFen != 10000 {
		t.Errorf("money = CNY %.2f, want CNY 100.00", float64(refundFen)/100.0)
	}
}

func TestComputeRefundableOneConsumed(t *testing.T) {
	// 1 consumed -> 99,990 refundable, CNY 99.99.
	refundFen, reservedCredits, remainder := ComputeRefundable(99990, 0)
	if refundFen != 9999 {
		t.Errorf("refundFen = %d, want 9999", refundFen)
	}
	if reservedCredits != 99990 {
		t.Errorf("reservedCredits = %d, want 99990", reservedCredits)
	}
	if remainder != 0 {
		t.Errorf("remainder = %d, want 0", remainder)
	}
}

func TestComputeRefundableHalfConsumed(t *testing.T) {
	// 50,010 consumed -> 49,990 refundable, CNY 49.99.
	refundFen, reservedCredits, remainder := ComputeRefundable(49990, 0)
	if refundFen != 4999 {
		t.Errorf("refundFen = %d, want 4999", refundFen)
	}
	if reservedCredits != 49990 {
		t.Errorf("reservedCredits = %d, want 49990", reservedCredits)
	}
	if remainder != 0 {
		t.Errorf("remainder = %d, want 0", remainder)
	}
}

func TestComputeRefundableFullyConsumed(t *testing.T) {
	// 100,000 consumed -> no refundable Credit.
	refundFen, reservedCredits, remainder := ComputeRefundable(0, 0)
	if refundFen != 0 {
		t.Errorf("refundFen = %d, want 0", refundFen)
	}
	if reservedCredits != 0 {
		t.Errorf("reservedCredits = %d, want 0", reservedCredits)
	}
	if remainder != 0 {
		t.Errorf("remainder = %d, want 0", remainder)
	}
}

func TestComputeRefundableSubQuantumRemainder(t *testing.T) {
	// 99,995 refundable -> 9,999 fen (99,990 Credit), remainder = 5 Credit.
	refundFen, reservedCredits, remainder := ComputeRefundable(99995, 0)
	if refundFen != 9999 {
		t.Errorf("refundFen = %d, want 9999", refundFen)
	}
	if reservedCredits != 99990 {
		t.Errorf("reservedCredits = %d, want 99990", reservedCredits)
	}
	if remainder != 5 {
		t.Errorf("remainder = %d, want 5 (sub-quantum stays available)", remainder)
	}
}

func TestComputeRefundablePartialRefund(t *testing.T) {
	// 100,000 refundable, request CNY 50.00 (5,000 fen).
	refundFen, reservedCredits, remainder := ComputeRefundable(100000, 5000)
	if refundFen != 5000 {
		t.Errorf("refundFen = %d, want 5000", refundFen)
	}
	if reservedCredits != 50000 {
		t.Errorf("reservedCredits = %d, want 50000", reservedCredits)
	}
	if remainder != 50000 {
		t.Errorf("remainder = %d, want 50000", remainder)
	}
}

func TestSumRefundableRejectsNonRechargeSources(t *testing.T) {
	// gift/plan/receivable Credit -> always rejected.
	grants := []commerce.AllocationGrant{
		{GrantID: "g1", Source: commerce.BucketSourcePlan, Refundable: 50000},
		{GrantID: "g2", Source: commerce.BucketSourceGift, Refundable: 30000},
		{GrantID: "g3", Source: commerce.BucketSourceEnterpriseReceivable, Refundable: 20000},
		{GrantID: "g4", Source: commerce.BucketSourceRecharge, Refundable: 100000},
	}
	total := sumRefundableCredits(grants)
	if total != 100000 {
		t.Errorf("refundable = %d, want 100000 (only recharge)", total)
	}
}

func TestSumRefundableAllNonRecharge(t *testing.T) {
	// Only non-recharge sources -> 0 refundable.
	grants := []commerce.AllocationGrant{
		{GrantID: "g1", Source: commerce.BucketSourcePlan, Refundable: 50000},
		{GrantID: "g2", Source: commerce.BucketSourceGift, Refundable: 30000},
	}
	total := sumRefundableCredits(grants)
	if total != 0 {
		t.Errorf("refundable = %d, want 0", total)
	}
}

// ===========================================================================
// Step 1 + 3: Full lifecycle arithmetic through the service
// ===========================================================================

func TestProcessOneFullRefundZeroConsumed(t *testing.T) {
	h := newTestHarness(t)
	h.wallet.setGrants("cust", []commerce.AllocationGrant{
		rechargeGrant("grant-1", 100000, 0, 100000),
	})
	h.orders.addFulfilledOrder("ns", "order-1", "cust", 10000)

	rec, _, err := h.svc.CreateRefund(context.Background(), CreateRefundInput{
		Namespace: "ns", OrderID: "order-1", CustomerID: "cust",
		AmountCents: 0, Currency: "CNY", IdempotencyKey: "idem-1",
	})
	if err != nil {
		t.Fatal(err)
	}

	result, err := h.svc.ProcessOne(context.Background(), "ns", rec.ID)
	if err != nil {
		t.Fatalf("process one: %v", err)
	}

	if result.Status != RefundStatusFulfilled {
		t.Errorf("status = %s, want fulfilled", result.Status)
	}
	if result.RefundFen != 10000 {
		t.Errorf("refundFen = %d, want 10000 (CNY 100.00)", result.RefundFen)
	}
	if result.ReservedCredits != 100000 {
		t.Errorf("reservedCredits = %d, want 100000", result.ReservedCredits)
	}
	if result.CreditQuantum != 10 {
		t.Errorf("creditQuantum = %d, want 10", result.CreditQuantum)
	}
	if result.RefundQuantumFen != 1 {
		t.Errorf("refundQuantumFen = %d, want 1", result.RefundQuantumFen)
	}

	// Credits reversed.
	if h.reverser.totalReversed() != 100000 {
		t.Errorf("total reversed = %d, want 100000", h.reverser.totalReversed())
	}
	// Fence released.
	if h.fence.released.Load() != 1 {
		t.Errorf("fence released = %d, want 1", h.fence.released.Load())
	}
}

func TestProcessOneFullRefundFullyConsumed(t *testing.T) {
	h := newTestHarness(t)
	h.wallet.setGrants("cust", []commerce.AllocationGrant{
		rechargeGrant("grant-1", 100000, 100000, 0),
	})
	h.orders.addFulfilledOrder("ns", "order-2", "cust", 10000)

	rec, _, err := h.svc.CreateRefund(context.Background(), CreateRefundInput{
		Namespace: "ns", OrderID: "order-2", CustomerID: "cust",
		AmountCents: 0, Currency: "CNY", IdempotencyKey: "idem-2",
	})
	if err != nil {
		t.Fatal(err)
	}

	result, err := h.svc.ProcessOne(context.Background(), "ns", rec.ID)
	if err == nil {
		t.Fatal("expected error for fully consumed credits")
	}
	if !errors.Is(err, ErrInsufficientRefundable) {
		t.Errorf("expected ErrInsufficientRefundable, got %v", err)
	}
	if result.Status != RefundStatusFailed {
		t.Errorf("status = %s, want failed", result.Status)
	}
	// No credits reversed.
	if h.reverser.totalReversed() != 0 {
		t.Errorf("total reversed = %d, want 0", h.reverser.totalReversed())
	}
}

func TestProcessOneGiftPlanReceivableRejected(t *testing.T) {
	h := newTestHarness(t)
	h.wallet.setGrants("cust", []commerce.AllocationGrant{
		{GrantID: "g1", Source: commerce.BucketSourcePlan, Refundable: 50000},
		{GrantID: "g2", Source: commerce.BucketSourceGift, Refundable: 30000},
		{GrantID: "g3", Source: commerce.BucketSourceEnterpriseReceivable, Refundable: 20000},
	})
	h.orders.addFulfilledOrder("ns", "order-3", "cust", 10000)

	rec, _, err := h.svc.CreateRefund(context.Background(), CreateRefundInput{
		Namespace: "ns", OrderID: "order-3", CustomerID: "cust",
		AmountCents: 0, Currency: "CNY", IdempotencyKey: "idem-3",
	})
	if err != nil {
		t.Fatal(err)
	}

	result, err := h.svc.ProcessOne(context.Background(), "ns", rec.ID)
	if err == nil {
		t.Fatal("expected error for non-recharge credits")
	}
	if result.Status != RefundStatusFailed {
		t.Errorf("status = %s, want failed", result.Status)
	}
	if h.reverser.totalReversed() != 0 {
		t.Errorf("total reversed = %d, want 0", h.reverser.totalReversed())
	}
}

func TestProcessOnePartialRefund(t *testing.T) {
	h := newTestHarness(t)
	h.wallet.setGrants("cust", []commerce.AllocationGrant{
		rechargeGrant("grant-1", 100000, 0, 100000),
	})
	h.orders.addFulfilledOrder("ns", "order-4", "cust", 10000)

	// Request CNY 50.00 = 5,000 fen.
	rec, _, err := h.svc.CreateRefund(context.Background(), CreateRefundInput{
		Namespace: "ns", OrderID: "order-4", CustomerID: "cust",
		AmountCents: 5000, Currency: "CNY", IdempotencyKey: "idem-4",
	})
	if err != nil {
		t.Fatal(err)
	}

	result, err := h.svc.ProcessOne(context.Background(), "ns", rec.ID)
	if err != nil {
		t.Fatalf("process one: %v", err)
	}
	if result.Status != RefundStatusFulfilled {
		t.Errorf("status = %s, want fulfilled", result.Status)
	}
	if result.RefundFen != 5000 {
		t.Errorf("refundFen = %d, want 5000", result.RefundFen)
	}
	if result.ReservedCredits != 50000 {
		t.Errorf("reservedCredits = %d, want 50000", result.ReservedCredits)
	}
	if result.RemainderCredits != 50000 {
		t.Errorf("remainderCredits = %d, want 50000", result.RemainderCredits)
	}
}

func TestProcessOneSubQuantumRemainder(t *testing.T) {
	h := newTestHarness(t)
	// 99,995 refundable -> 9,999 fen (99,990 Credit), remainder = 5 Credit.
	h.wallet.setGrants("cust", []commerce.AllocationGrant{
		rechargeGrant("grant-1", 99995, 5, 99995),
	})
	h.orders.addFulfilledOrder("ns", "order-5", "cust", 10000)

	rec, _, err := h.svc.CreateRefund(context.Background(), CreateRefundInput{
		Namespace: "ns", OrderID: "order-5", CustomerID: "cust",
		AmountCents: 0, Currency: "CNY", IdempotencyKey: "idem-5",
	})
	if err != nil {
		t.Fatal(err)
	}

	result, err := h.svc.ProcessOne(context.Background(), "ns", rec.ID)
	if err != nil {
		t.Fatalf("process one: %v", err)
	}
	if result.RefundFen != 9999 {
		t.Errorf("refundFen = %d, want 9999", result.RefundFen)
	}
	if result.ReservedCredits != 99990 {
		t.Errorf("reservedCredits = %d, want 99990", result.ReservedCredits)
	}
	if result.RemainderCredits != 5 {
		t.Errorf("remainderCredits = %d, want 5 (sub-quantum stays available)", result.RemainderCredits)
	}
	// Only the quantum-rounded credits are reversed.
	if h.reverser.totalReversed() != 99990 {
		t.Errorf("total reversed = %d, want 99990", h.reverser.totalReversed())
	}
}

// ===========================================================================
// Step 2: Fence behavior
// ===========================================================================

func TestFenceTimeoutLeavesPendingFence(t *testing.T) {
	h := newTestHarness(t)
	h.wallet.setGrants("cust", []commerce.AllocationGrant{
		rechargeGrant("grant-1", 100000, 0, 100000),
	})
	h.orders.addFulfilledOrder("ns", "order-fence", "cust", 10000)
	h.fence.failNext.Store(true)

	rec, _, err := h.svc.CreateRefund(context.Background(), CreateRefundInput{
		Namespace: "ns", OrderID: "order-fence", CustomerID: "cust",
		AmountCents: 0, Currency: "CNY", IdempotencyKey: "idem-fence",
	})
	if err != nil {
		t.Fatal(err)
	}

	result, err := h.svc.ProcessOne(context.Background(), "ns", rec.ID)
	if err == nil {
		t.Fatal("expected fence timeout error")
	}
	if result.Status != RefundStatusPendingFence {
		t.Errorf("status = %s, want pending_fence (no bypass mutation)", result.Status)
	}
}

func TestCreateRefundIdempotent(t *testing.T) {
	h := newTestHarness(t)
	h.orders.addFulfilledOrder("ns", "order-idem", "cust", 10000)

	first, _, err := h.svc.CreateRefund(context.Background(), CreateRefundInput{
		Namespace: "ns", OrderID: "order-idem", CustomerID: "cust",
		AmountCents: 0, Currency: "CNY", IdempotencyKey: "idem-key",
	})
	if err != nil {
		t.Fatal(err)
	}
	second, inserted, err := h.svc.CreateRefund(context.Background(), CreateRefundInput{
		Namespace: "ns", OrderID: "order-idem", CustomerID: "cust",
		AmountCents: 0, Currency: "CNY", IdempotencyKey: "idem-key",
	})
	if err != nil {
		t.Fatal(err)
	}
	if first.ID != second.ID {
		t.Errorf("idempotent replay: %s vs %s", first.ID, second.ID)
	}
	if inserted {
		t.Error("second create should be a replay (inserted=false)")
	}
}

func TestCreateRefundRejectsNonWalletTopUp(t *testing.T) {
	h := newTestHarness(t)
	h.orders.orders["order-plan"] = &commerce.Order{
		NamespacedID: models.NamespacedID{Namespace: "ns", ID: "order-plan"},
		CustomerID:   "cust",
		Kind:         commerce.OrderKindPlanPurchase,
		Status:       commerce.OrderStatusFulfilled,
		Currency:     "CNY",
	}
	_, _, err := h.svc.CreateRefund(context.Background(), CreateRefundInput{
		Namespace: "ns", OrderID: "order-plan", CustomerID: "cust",
		AmountCents: 0, Currency: "CNY", IdempotencyKey: "idem-plan",
	})
	if !errors.Is(err, ErrOrderNotRefundable) {
		t.Errorf("expected ErrOrderNotRefundable, got %v", err)
	}
}

// TestCreateRefundRejectsPaidOrder (I6): a paid-but-not-fulfilled order has not
// granted credits yet, so it must not be refundable.
func TestCreateRefundRejectsPaidOrder(t *testing.T) {
	h := newTestHarness(t)
	h.orders.orders["order-paid"] = &commerce.Order{
		NamespacedID: models.NamespacedID{Namespace: "ns", ID: "order-paid"},
		CustomerID:   "cust",
		Kind:         commerce.OrderKindWalletTopUp,
		Status:       commerce.OrderStatusPaid, // paid but NOT fulfilled
		Currency:     "CNY",
	}
	_, _, err := h.svc.CreateRefund(context.Background(), CreateRefundInput{
		Namespace: "ns", OrderID: "order-paid", CustomerID: "cust",
		AmountCents: 0, Currency: "CNY", IdempotencyKey: "idem-paid",
	})
	if !errors.Is(err, ErrOrderNotRefundable) {
		t.Errorf("expected ErrOrderNotRefundable for paid (not fulfilled) order, got %v", err)
	}
}

// ===========================================================================
// Step 5: Concurrency / race tests
// ===========================================================================

func TestConcurrentLast100CreditOneFenceSucceeds(t *testing.T) {
	h := newTestHarness(t)
	// Provider returns immediate success so the fence is released in one pass.
	h.provider.refundResult = func(input payment.RefundInput) (payment.RefundSubmission, error) {
		return payment.RefundSubmission{Provider: payment.ProviderWeChat, ProviderRefundID: input.IdempotencyKey, Status: "success"}, nil
	}
	// Only 100 refundable Credit remaining = 10 fen = CNY 0.10.
	h.wallet.setGrants("cust", []commerce.AllocationGrant{
		rechargeGrant("grant-1", 100, 0, 100),
	})
	h.orders.addFulfilledOrder("ns", "order-race", "cust", 10)

	// Two concurrent refund requests.
	rec1, _, err := h.svc.CreateRefund(context.Background(), CreateRefundInput{
		Namespace: "ns", OrderID: "order-race", CustomerID: "cust",
		AmountCents: 0, Currency: "CNY", IdempotencyKey: "idem-race-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	rec2, _, err := h.svc.CreateRefund(context.Background(), CreateRefundInput{
		Namespace: "ns", OrderID: "order-race", CustomerID: "cust",
		AmountCents: 0, Currency: "CNY", IdempotencyKey: "idem-race-2",
	})
	if err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	results := make([]*RefundRequest, 2)
	wg.Add(2)
	go func() {
		defer wg.Done()
		results[0], _ = h.svc.ProcessOne(context.Background(), "ns", rec1.ID)
	}()
	go func() {
		defer wg.Done()
		results[1], _ = h.svc.ProcessOne(context.Background(), "ns", rec2.ID)
	}()
	wg.Wait()

	// Exactly one should be fulfilled, the other failed.
	fulfilled := 0
	failed := 0
	for _, r := range results {
		if r != nil {
			switch r.Status {
			case RefundStatusFulfilled:
				fulfilled++
			case RefundStatusFailed:
				failed++
			}
		}
	}
	if fulfilled != 1 {
		t.Errorf("fulfilled = %d, want exactly 1", fulfilled)
	}
	if failed != 1 {
		t.Errorf("failed = %d, want exactly 1", failed)
	}
	// Total credits reversed must be exactly 100 (conservation).
	if h.reverser.totalReversed() != 100 {
		t.Errorf("total reversed = %d, want 100 (exact conservation)", h.reverser.totalReversed())
	}
}

func TestConcurrentMultipleFullRefunds(t *testing.T) {
	h := newTestHarness(t)
	h.provider.refundResult = func(input payment.RefundInput) (payment.RefundSubmission, error) {
		return payment.RefundSubmission{Provider: payment.ProviderWeChat, ProviderRefundID: input.IdempotencyKey, Status: "success"}, nil
	}
	// 1,000,000 refundable Credit = 100,000 fen = CNY 1000.00.
	h.wallet.setGrants("cust", []commerce.AllocationGrant{
		rechargeGrant("grant-1", 1000000, 0, 1000000),
	})
	h.orders.addFulfilledOrder("ns", "order-multi", "cust", 100000)

	const N = 8
	recs := make([]*RefundRequest, N)
	for i := 0; i < N; i++ {
		// Each requests CNY 125.00 = 12,500 fen = 125,000 Credit.
		// 8 * 125,000 = 1,000,000 exactly.
		rec, _, err := h.svc.CreateRefund(context.Background(), CreateRefundInput{
			Namespace: "ns", OrderID: "order-multi", CustomerID: "cust",
			AmountCents: 12500, Currency: "CNY",
			IdempotencyKey: fmt.Sprintf("idem-multi-%d", i),
		})
		if err != nil {
			t.Fatal(err)
		}
		recs[i] = rec
	}

	var wg sync.WaitGroup
	results := make([]*RefundRequest, N)
	errs := make([]error, N)
	for i := 0; i < N; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			results[idx], errs[idx] = h.svc.ProcessOne(context.Background(), "ns", recs[idx].ID)
		}(i)
	}
	wg.Wait()

	// All should be fulfilled (exact division, no contention).
	for i, r := range results {
		if r == nil || r.Status != RefundStatusFulfilled {
			t.Errorf("refund %d: status = %v, want fulfilled (err=%v)", i, r, errs[i])
		}
	}
	// Total reversed must be exactly 1,000,000.
	if h.reverser.totalReversed() != 1000000 {
		t.Errorf("total reversed = %d, want 1000000 (exact conservation)", h.reverser.totalReversed())
	}
}

func TestConcurrentRefundOverSubscription(t *testing.T) {
	h := newTestHarness(t)
	h.provider.refundResult = func(input payment.RefundInput) (payment.RefundSubmission, error) {
		return payment.RefundSubmission{Provider: payment.ProviderWeChat, ProviderRefundID: input.IdempotencyKey, Status: "success"}, nil
	}
	// 50,000 refundable Credit = 5,000 fen = CNY 50.00.
	h.wallet.setGrants("cust", []commerce.AllocationGrant{
		rechargeGrant("grant-1", 50000, 0, 50000),
	})
	h.orders.addFulfilledOrder("ns", "order-over", "cust", 5000)

	const N = 5
	recs := make([]*RefundRequest, N)
	for i := 0; i < N; i++ {
		// Each requests CNY 20.00 = 2,000 fen = 20,000 Credit.
		// 5 * 20,000 = 100,000 > 50,000 available.
		// Only 2 can succeed (40,000), 3 must fail.
		rec, _, err := h.svc.CreateRefund(context.Background(), CreateRefundInput{
			Namespace: "ns", OrderID: "order-over", CustomerID: "cust",
			AmountCents: 2000, Currency: "CNY",
			IdempotencyKey: fmt.Sprintf("idem-over-%d", i),
		})
		if err != nil {
			t.Fatal(err)
		}
		recs[i] = rec
	}

	var wg sync.WaitGroup
	results := make([]*RefundRequest, N)
	for i := 0; i < N; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			results[idx], _ = h.svc.ProcessOne(context.Background(), "ns", recs[idx].ID)
		}(i)
	}
	wg.Wait()

	fulfilled := 0
	failed := 0
	for _, r := range results {
		if r == nil {
			continue
		}
		switch r.Status {
		case RefundStatusFulfilled:
			fulfilled++
		case RefundStatusFailed:
			failed++
		}
	}
	// 50,000 / 20,000 = 2 full + 1 partial (10,000) = 3 fulfilled, 2 failed.
	if fulfilled != 3 {
		t.Errorf("fulfilled = %d, want 3", fulfilled)
	}
	if failed != 2 {
		t.Errorf("failed = %d, want 2", failed)
	}
	// Total reversed must be exactly 50,000 (20k + 20k + 10k).
	if h.reverser.totalReversed() != 50000 {
		t.Errorf("total reversed = %d, want 50000 (exact conservation)", h.reverser.totalReversed())
	}
}

// ===========================================================================
// State machine + idempotency tests
// ===========================================================================

func TestProcessOneIdempotentFulfilled(t *testing.T) {
	h := newTestHarness(t)
	h.wallet.setGrants("cust", []commerce.AllocationGrant{
		rechargeGrant("grant-1", 100000, 0, 100000),
	})
	h.orders.addFulfilledOrder("ns", "order-idem-proc", "cust", 10000)

	rec, _, err := h.svc.CreateRefund(context.Background(), CreateRefundInput{
		Namespace: "ns", OrderID: "order-idem-proc", CustomerID: "cust",
		AmountCents: 0, Currency: "CNY", IdempotencyKey: "idem-proc",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Process multiple times.
	for i := 0; i < 5; i++ {
		if _, err := h.svc.ProcessOne(context.Background(), "ns", rec.ID); err != nil {
			t.Fatalf("call %d: %v", i, err)
		}
	}

	// Reverser should be called exactly once (idempotent reversal).
	if h.reverser.callN.Load() != 1 {
		t.Errorf("reverser calls = %d, want 1 (idempotent)", h.reverser.callN.Load())
	}
	if h.reverser.totalReversed() != 100000 {
		t.Errorf("total reversed = %d, want 100000", h.reverser.totalReversed())
	}
}

func TestProcessOneProviderFailure(t *testing.T) {
	h := newTestHarness(t)
	h.wallet.setGrants("cust", []commerce.AllocationGrant{
		rechargeGrant("grant-1", 100000, 0, 100000),
	})
	h.orders.addFulfilledOrder("ns", "order-prov-fail", "cust", 10000)

	// Provider returns failure on submission.
	h.provider.refundResult = func(input payment.RefundInput) (payment.RefundSubmission, error) {
		return payment.RefundSubmission{
			Provider:         payment.ProviderWeChat,
			ProviderRefundID: input.IdempotencyKey,
			Status:           "failed",
		}, nil
	}

	rec, _, err := h.svc.CreateRefund(context.Background(), CreateRefundInput{
		Namespace: "ns", OrderID: "order-prov-fail", CustomerID: "cust",
		AmountCents: 0, Currency: "CNY", IdempotencyKey: "idem-prov-fail",
	})
	if err != nil {
		t.Fatal(err)
	}

	result, err := h.svc.ProcessOne(context.Background(), "ns", rec.ID)
	if err == nil {
		t.Fatal("expected provider failure error")
	}
	if result.Status != RefundStatusFailed {
		t.Errorf("status = %s, want failed", result.Status)
	}
	// No credits reversed on failure.
	if h.reverser.totalReversed() != 0 {
		t.Errorf("total reversed = %d, want 0", h.reverser.totalReversed())
	}
	// Fence released on failure.
	if h.fence.released.Load() != 1 {
		t.Errorf("fence released = %d, want 1 (released on failure)", h.fence.released.Load())
	}
}

func TestProcessOneProviderTransientErrorRetainsFence(t *testing.T) {
	h := newTestHarness(t)
	h.wallet.setGrants("cust", []commerce.AllocationGrant{
		rechargeGrant("grant-1", 100000, 0, 100000),
	})
	h.orders.addFulfilledOrder("ns", "order-provider-transient", "cust", 10000)
	wantErr := &payment.ProviderError{
		Provider: payment.ProviderWeChat, Operation: "refund", Kind: payment.ProviderErrorRetryable,
	}
	h.provider.refundResult = func(payment.RefundInput) (payment.RefundSubmission, error) {
		return payment.RefundSubmission{}, wantErr
	}

	rec, _, err := h.svc.CreateRefund(t.Context(), CreateRefundInput{
		Namespace: "ns", OrderID: "order-provider-transient", CustomerID: "cust",
		Currency: "CNY", IdempotencyKey: "idem-provider-transient",
	})
	require.NoError(t, err)

	result, err := h.svc.ProcessOne(t.Context(), "ns", rec.ID)
	require.ErrorIs(t, err, payment.ErrRetryableProvider)
	require.Equal(t, RefundStatusProviderProcessing, result.Status)
	require.Zero(t, h.fence.released.Load())
	require.Zero(t, h.reverser.totalReversed())
}

func TestProcessOneProviderPermanentProtocolErrorRetainsFence(t *testing.T) {
	h := newTestHarness(t)
	h.wallet.setGrants("cust", []commerce.AllocationGrant{
		rechargeGrant("grant-1", 100000, 0, 100000),
	})
	h.orders.addFulfilledOrder("ns", "order-provider-protocol", "cust", 10000)
	wantErr := &payment.ProviderError{
		Provider: payment.ProviderWeChat, Operation: "refund", Kind: payment.ProviderErrorPermanent,
		Cause: payment.ErrPermanentProviderProtocol,
	}
	h.provider.refundResult = func(payment.RefundInput) (payment.RefundSubmission, error) {
		return payment.RefundSubmission{}, wantErr
	}

	rec, _, err := h.svc.CreateRefund(t.Context(), CreateRefundInput{
		Namespace: "ns", OrderID: "order-provider-protocol", CustomerID: "cust",
		Currency: "CNY", IdempotencyKey: "idem-provider-protocol",
	})
	require.NoError(t, err)

	result, err := h.svc.ProcessOne(t.Context(), "ns", rec.ID)
	require.ErrorIs(t, err, payment.ErrPermanentProviderProtocol)
	require.Equal(t, RefundStatusProviderProcessing, result.Status)
	require.Zero(t, h.fence.released.Load())
	require.Zero(t, h.reverser.totalReversed())
}

func TestProcessOneProviderAuthorityPassesOriginalPaymentAndTotal(t *testing.T) {
	wallet := newMockWallet()
	orders := newMockOrders()
	repo := newMockRepo(wallet)
	fence := newMockFenceClient(repo)
	reverser := newMockReverser()
	provider := newMockProvider(payment.ProviderWeChat)
	wallet.setGrants("cust", []commerce.AllocationGrant{
		rechargeGrant("grant-1", 100000, 0, 100000),
	})
	orders.addFulfilledOrder("ns", "order-provider-authority", "cust", 10000)

	var refundInput payment.RefundInput
	provider.refundResult = func(input payment.RefundInput) (payment.RefundSubmission, error) {
		refundInput = input
		return payment.RefundSubmission{Provider: payment.ProviderWeChat, ProviderRefundID: input.IdempotencyKey, Status: "processing"}, nil
	}
	svc, err := New(Config{
		Repo:      repo,
		Orders:    orders,
		Wallet:    wallet,
		Fence:     fence,
		Reverser:  reverser,
		Providers: map[payment.Provider]ProviderRefunder{payment.ProviderWeChat: provider},
		PaymentAuthorityResolver: mockPaymentAuthorityResolver{
			orders: orders, provider: payment.ProviderWeChat, merchantID: "wx-mch",
		},
	})
	require.NoError(t, err)

	rec, _, err := svc.CreateRefund(t.Context(), CreateRefundInput{
		Namespace: "ns", OrderID: "order-provider-authority", CustomerID: "cust",
		Currency: "CNY", IdempotencyKey: "idem-provider-authority",
	})
	require.NoError(t, err)

	result, err := svc.ProcessOne(t.Context(), "ns", rec.ID)
	require.NoError(t, err)
	require.Equal(t, RefundStatusProviderProcessing, result.Status)
	require.Equal(t, "pub-order-provider-authority", refundInput.ProviderOrderID)
	require.Equal(t, "payment-order-provider-authority", refundInput.ProviderPaymentID)
	require.Equal(t, int64(10000), refundInput.TotalAmountMinor)

	for _, authority := range []PaymentAuthority{
		{Provider: payment.ProviderAlipay, ProviderOrderID: "provider-order", ProviderPaymentID: "provider-payment", AmountMinor: 10000, Currency: "CNY"},
		{Provider: payment.ProviderWeChat, ProviderOrderID: "provider-order", ProviderPaymentID: "provider-payment", AmountMinor: 10000, Currency: "USD"},
		{Provider: payment.ProviderWeChat, ProviderOrderID: "provider-order", ProviderPaymentID: "provider-payment", AmountMinor: 9999, Currency: "CNY"},
	} {
		t.Run("rejects mismatched authority", func(t *testing.T) {
			wallet := newMockWallet()
			orders := newMockOrders()
			repo := newMockRepo(wallet)
			fence := newMockFenceClient(repo)
			provider := newMockProvider(payment.ProviderWeChat)
			wallet.setGrants("cust", []commerce.AllocationGrant{rechargeGrant("grant-1", 100000, 0, 100000)})
			orders.addFulfilledOrder("ns", "order-provider-authority-reject", "cust", 10000)
			svc, err := New(Config{
				Repo: repo, Orders: orders, Wallet: wallet, Fence: fence, Reverser: newMockReverser(),
				Providers:                map[payment.Provider]ProviderRefunder{payment.ProviderWeChat: provider},
				PaymentAuthorityResolver: mockPaymentAuthorityResolver{authority: &authority},
			})
			require.NoError(t, err)
			rec, _, err := svc.CreateRefund(t.Context(), CreateRefundInput{
				Namespace: "ns", OrderID: "order-provider-authority-reject", CustomerID: "cust",
				Currency: "CNY", IdempotencyKey: "idem-provider-authority-reject",
			})
			require.NoError(t, err)

			result, err := svc.ProcessOne(t.Context(), "ns", rec.ID)
			require.Error(t, err)
			require.Equal(t, RefundStatusProviderProcessing, result.Status)
			require.Zero(t, provider.refundCallN.Load())
			require.Zero(t, fence.released.Load())
		})
	}
}

func TestProcessOneProviderProcessingThenSuccess(t *testing.T) {
	h := newTestHarness(t)
	h.wallet.setGrants("cust", []commerce.AllocationGrant{
		rechargeGrant("grant-1", 100000, 0, 100000),
	})
	h.orders.addFulfilledOrder("ns", "order-proc", "cust", 10000)

	// Override: submission returns "processing", then QueryRefund returns success.
	h.provider.refundResult = func(input payment.RefundInput) (payment.RefundSubmission, error) {
		return payment.RefundSubmission{Provider: payment.ProviderWeChat, ProviderRefundID: input.IdempotencyKey, Status: "processing"}, nil
	}
	queryCalled := atomic.Bool{}
	var queryInput payment.RefundQueryInput
	h.provider.queryResult = func(input payment.RefundQueryInput) (payment.RefundFact, error) {
		queryCalled.Store(true)
		queryInput = input
		return verifiedRefundFact(payment.ProviderWeChat, "order-proc", input.ProviderRefundID, "SUCCESS", "hash-"+input.ProviderRefundID, 10000), nil
	}

	rec, _, err := h.svc.CreateRefund(context.Background(), CreateRefundInput{
		Namespace: "ns", OrderID: "order-proc", CustomerID: "cust",
		AmountCents: 0, Currency: "CNY", IdempotencyKey: "idem-proc",
	})
	if err != nil {
		t.Fatal(err)
	}

	// First ProcessOne: fence + reserve + submit -> stays in provider_processing.
	result, err := h.svc.ProcessOne(context.Background(), "ns", rec.ID)
	if err != nil {
		t.Fatalf("first process: %v", err)
	}
	if result.Status != RefundStatusProviderProcessing {
		t.Errorf("first process status = %s, want provider_processing", result.Status)
	}

	// Second ProcessOne: query -> success -> ledger_reversing -> fulfilled.
	result, err = h.svc.ProcessOne(context.Background(), "ns", rec.ID)
	if err != nil {
		t.Fatalf("second process: %v", err)
	}
	if result.Status != RefundStatusFulfilled {
		t.Errorf("second process status = %s, want fulfilled", result.Status)
	}
	if !queryCalled.Load() {
		t.Error("QueryRefund should have been called")
	}
	if queryInput.ProviderRefundID != "idem-proc" || queryInput.ProviderOrderID != "pub-order-proc" || queryInput.AmountMinor != 10000 || queryInput.Currency != "CNY" {
		t.Errorf("QueryRefund input = %+v, want persisted refund and order context", queryInput)
	}
	if h.reverser.totalReversed() != 100000 {
		t.Errorf("total reversed = %d, want 100000", h.reverser.totalReversed())
	}
}

func TestProcessOneInvalidResolverResultDoesNotFallBackToAnotherProvider(t *testing.T) {
	wantResolverErr := errors.New("payment attempt lookup failed")
	for _, tt := range []struct {
		name     string
		resolver mockProviderResolver
		wantErr  error
	}{
		{name: "resolver error", resolver: mockProviderResolver{err: wantResolverErr}, wantErr: wantResolverErr},
		{name: "empty provider", resolver: mockProviderResolver{}},
		{name: "unconfigured provider", resolver: mockProviderResolver{provider: payment.ProviderOffline}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			h := newTestHarness(t)
			h.wallet.setGrants("cust", []commerce.AllocationGrant{
				rechargeGrant("grant-1", 100000, 0, 100000),
			})
			h.orders.addFulfilledOrder("ns", "order-resolver-error", "cust", 10000)
			otherProvider := newMockProvider(payment.ProviderAlipay)

			service, err := New(Config{
				Repo: h.repo, Orders: h.orders, Wallet: h.wallet, Fence: h.fence, Reverser: h.reverser,
				Providers: map[payment.Provider]ProviderRefunder{
					payment.ProviderAlipay: otherProvider,
					payment.ProviderWeChat: h.provider,
				},
				ProviderResolver: tt.resolver,
			})
			require.NoError(t, err)

			rec, _, err := service.CreateRefund(t.Context(), CreateRefundInput{
				Namespace: "ns", OrderID: "order-resolver-error", CustomerID: "cust",
				Currency: "CNY", IdempotencyKey: "idem-resolver-error",
			})
			require.NoError(t, err)

			rec, err = service.ProcessOne(t.Context(), "ns", rec.ID)
			require.Error(t, err)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			}
			require.Equal(t, RefundStatusProviderProcessing, rec.Status)
			require.Zero(t, otherProvider.refundCallN.Load())
			require.Zero(t, h.provider.refundCallN.Load())
			require.Zero(t, h.reverser.totalReversed())
			require.Zero(t, h.fence.released.Load())
		})
	}
}

func TestProcessOneProviderKeyNameMismatchDoesNotCallRefundChannel(t *testing.T) {
	h := newTestHarness(t)
	h.wallet.setGrants("cust", []commerce.AllocationGrant{
		rechargeGrant("grant-1", 100000, 0, 100000),
	})
	h.orders.addFulfilledOrder("ns", "order-provider-mismatch", "cust", 10000)
	wrongChannel := newMockProvider(payment.ProviderAlipay)

	service, err := New(Config{
		Repo: h.repo, Orders: h.orders, Wallet: h.wallet, Fence: h.fence, Reverser: h.reverser,
		Providers: map[payment.Provider]ProviderRefunder{
			payment.ProviderWeChat: wrongChannel,
		},
		ProviderResolver: mockProviderResolver{provider: payment.ProviderWeChat},
	})
	require.NoError(t, err)

	rec, _, err := service.CreateRefund(t.Context(), CreateRefundInput{
		Namespace: "ns", OrderID: "order-provider-mismatch", CustomerID: "cust",
		Currency: "CNY", IdempotencyKey: "idem-provider-mismatch",
	})
	require.NoError(t, err)

	rec, err = service.ProcessOne(t.Context(), "ns", rec.ID)
	require.ErrorIs(t, err, payment.ErrPermanentProviderProtocol)
	require.Equal(t, RefundStatusProviderProcessing, rec.Status)
	require.Zero(t, wrongChannel.refundCallN.Load())
	require.Zero(t, wrongChannel.queryCallN.Load())
}

func TestProcessOnePersistedUnavailableProviderDoesNotFallBackWithoutResolver(t *testing.T) {
	h := newTestHarness(t)
	h.orders.addFulfilledOrder("ns", "order-persisted-provider", "cust", 10000)
	alipayProvider := newMockProvider(payment.ProviderAlipay)
	wechatProvider := newMockProvider(payment.ProviderWeChat)

	service, err := New(Config{
		Repo: h.repo, Orders: h.orders, Wallet: h.wallet, Reverser: h.reverser,
		Providers: map[payment.Provider]ProviderRefunder{
			payment.ProviderAlipay: alipayProvider,
			payment.ProviderWeChat: wechatProvider,
		},
	})
	require.NoError(t, err)

	rec, _, err := h.repo.CreateRefund(t.Context(), RefundRequest{
		ID: "refund-persisted-provider", Namespace: "ns", CommerceOrderID: "order-persisted-provider",
		CustomerID: "cust", Currency: "CNY", Status: RefundStatusProviderProcessing,
		ProviderName: string(payment.ProviderOffline), ProviderRefundID: "provider-refund-1",
		RefundFen: 10000, ReservedCredits: 100000, FenceSequence: "fence-1",
	})
	require.NoError(t, err)

	rec, err = service.ProcessOne(t.Context(), "ns", rec.ID)
	require.Error(t, err)
	require.Equal(t, RefundStatusProviderProcessing, rec.Status)
	require.Zero(t, alipayProvider.queryCallN.Load())
	require.Zero(t, wechatProvider.queryCallN.Load())
	require.Zero(t, alipayProvider.refundCallN.Load())
	require.Zero(t, wechatProvider.refundCallN.Load())
	require.Zero(t, h.reverser.totalReversed())
	require.Zero(t, h.fence.released.Load())
}

func TestProcessOneProviderSuccessStopsWhenFactPersistenceFails(t *testing.T) {
	h := newTestHarness(t)
	h.wallet.setGrants("cust", []commerce.AllocationGrant{
		rechargeGrant("grant-1", 100000, 0, 100000),
	})
	h.orders.addFulfilledOrder("ns", "order-fact-error", "cust", 10000)
	h.provider.refundResult = func(input payment.RefundInput) (payment.RefundSubmission, error) {
		return payment.RefundSubmission{
			Provider: payment.ProviderWeChat, ProviderRefundID: input.IdempotencyKey, Status: "processing",
		}, nil
	}
	h.provider.queryResult = func(input payment.RefundQueryInput) (payment.RefundFact, error) {
		return verifiedRefundFact(payment.ProviderWeChat, "order-fact-error", input.ProviderRefundID, "SUCCESS", "fact-error-hash", 10000), nil
	}

	rec, _, err := h.svc.CreateRefund(t.Context(), CreateRefundInput{
		Namespace: "ns", OrderID: "order-fact-error", CustomerID: "cust",
		Currency: "CNY", IdempotencyKey: "idem-fact-error",
	})
	require.NoError(t, err)

	rec, err = h.svc.ProcessOne(t.Context(), "ns", rec.ID)
	require.NoError(t, err)
	require.Equal(t, RefundStatusProviderProcessing, rec.Status)

	wantErr := errors.New("append refund fact")
	h.repo.appendErr = wantErr
	rec, err = h.svc.ProcessOne(t.Context(), "ns", rec.ID)
	require.ErrorIs(t, err, wantErr)
	require.Equal(t, RefundStatusProviderProcessing, rec.Status)
	require.Zero(t, h.reverser.totalReversed())
	require.Zero(t, h.fence.released.Load())
}

func TestProcessOneProviderProcessingResultRetainsFence(t *testing.T) {
	h := newTestHarness(t)
	h.wallet.setGrants("cust", []commerce.AllocationGrant{
		rechargeGrant("grant-1", 100000, 0, 100000),
	})
	h.orders.addFulfilledOrder("ns", "order-unknown", "cust", 10000)

	// Override: submission returns "processing" so refund stays in provider_processing.
	h.provider.refundResult = func(input payment.RefundInput) (payment.RefundSubmission, error) {
		return payment.RefundSubmission{Provider: payment.ProviderWeChat, ProviderRefundID: input.IdempotencyKey, Status: "processing"}, nil
	}
	// QueryRefund returns a verified provider-processing result (no success and
	// no definitive RawHash), so the service must retain the fence.
	h.provider.queryResult = func(input payment.RefundQueryInput) (payment.RefundFact, error) {
		fact := verifiedRefundFact(payment.ProviderWeChat, "order-unknown", input.ProviderRefundID, "PROCESSING", "", input.AmountMinor)
		fact.SignedPayload = map[string]any{"status": "PROCESSING"}
		return fact, nil
	}

	rec, _, err := h.svc.CreateRefund(context.Background(), CreateRefundInput{
		Namespace: "ns", OrderID: "order-unknown", CustomerID: "cust",
		AmountCents: 0, Currency: "CNY", IdempotencyKey: "idem-unknown",
	})
	if err != nil {
		t.Fatal(err)
	}

	// First: submit -> provider_processing.
	result, err := h.svc.ProcessOne(context.Background(), "ns", rec.ID)
	if err != nil {
		t.Fatalf("first: %v", err)
	}
	if result.Status != RefundStatusProviderProcessing {
		t.Fatalf("status = %s, want provider_processing", result.Status)
	}

	// Second: query returns unknown -> stays in provider_processing, fence retained.
	result, err = h.svc.ProcessOne(context.Background(), "ns", rec.ID)
	if err != nil {
		t.Fatalf("second: %v", err)
	}
	if result.Status != RefundStatusProviderProcessing {
		t.Errorf("status = %s, want provider_processing (fence retained)", result.Status)
	}
	// Fence NOT released.
	if h.fence.released.Load() != 0 {
		t.Errorf("fence released = %d, want 0 (fence retained for unknown result)", h.fence.released.Load())
	}
}

func TestProcessOneProviderDefinitiveFailureReleasesFence(t *testing.T) {
	h := newTestHarness(t)
	h.wallet.setGrants("cust", []commerce.AllocationGrant{
		rechargeGrant("grant-1", 100000, 0, 100000),
	})
	h.orders.addFulfilledOrder("ns", "order-def-fail", "cust", 10000)

	// Override: submission returns "processing" so refund stays in provider_processing.
	h.provider.refundResult = func(input payment.RefundInput) (payment.RefundSubmission, error) {
		return payment.RefundSubmission{Provider: payment.ProviderWeChat, ProviderRefundID: input.IdempotencyKey, Status: "processing"}, nil
	}
	// QueryRefund returns definitive failure (non-success with raw hash).
	h.provider.queryResult = func(input payment.RefundQueryInput) (payment.RefundFact, error) {
		return verifiedRefundFact(payment.ProviderWeChat, "order-def-fail", input.ProviderRefundID, "CLOSED", "def-hash-"+input.ProviderRefundID, input.AmountMinor), nil
	}

	rec, _, err := h.svc.CreateRefund(context.Background(), CreateRefundInput{
		Namespace: "ns", OrderID: "order-def-fail", CustomerID: "cust",
		AmountCents: 0, Currency: "CNY", IdempotencyKey: "idem-def-fail",
	})
	if err != nil {
		t.Fatal(err)
	}

	// First: submit -> provider_processing.
	_, _ = h.svc.ProcessOne(context.Background(), "ns", rec.ID)

	// Second: query -> definitive failure -> failed + release fence.
	result, err := h.svc.ProcessOne(context.Background(), "ns", rec.ID)
	if err == nil {
		t.Fatal("expected definitive failure error")
	}
	if result.Status != RefundStatusFailed {
		t.Errorf("status = %s, want failed", result.Status)
	}
	if h.fence.released.Load() != 1 {
		t.Errorf("fence released = %d, want 1", h.fence.released.Load())
	}
	if h.reverser.totalReversed() != 0 {
		t.Errorf("total reversed = %d, want 0", h.reverser.totalReversed())
	}
}

func TestRefundFactDeduplication(t *testing.T) {
	repo := newMockRepo(newMockWallet())
	ctx := context.Background()

	fact := RefundFactRecord{
		ID: "fact-1", Namespace: "ns", RefundRequestID: "rf-1",
		Provider: payment.ProviderWeChat, ProviderRefundID: "pr-1",
		RawHash: "hash-1", Success: true,
	}

	first, inserted, err := repo.AppendFact(ctx, fact)
	if err != nil {
		t.Fatal(err)
	}
	if !inserted {
		t.Error("first insert should be inserted=true")
	}

	// Same raw hash -> dedup.
	_, inserted, err = repo.AppendFact(ctx, fact)
	if err != nil {
		t.Fatal(err)
	}
	if inserted {
		t.Error("second insert with same hash should be inserted=false (dedup)")
	}

	facts, _ := repo.GetFacts(ctx, "ns", "rf-1")
	if len(facts) != 1 {
		t.Errorf("facts count = %d, want 1 (deduped)", len(facts))
	}
	_ = first
}

// ===========================================================================
// Quantum persistence tests
// ===========================================================================

func testQuantumPersisted(t *testing.T) {
	t.Helper()
	h := newTestHarness(t)
	h.wallet.setGrants("cust", []commerce.AllocationGrant{
		rechargeGrant("grant-1", 100000, 0, 100000),
	})
	h.orders.addFulfilledOrder("ns", "order-q", "cust", 10000)

	rec, _, err := h.svc.CreateRefund(context.Background(), CreateRefundInput{
		Namespace: "ns", OrderID: "order-q", CustomerID: "cust",
		AmountCents: 0, Currency: "CNY", IdempotencyKey: "idem-q",
	})
	if err != nil {
		t.Fatal(err)
	}

	result, err := h.svc.ProcessOne(context.Background(), "ns", rec.ID)
	if err != nil {
		t.Fatal(err)
	}

	// Validate the persisted quantum ratio: credit_quantum:refund_quantum_fen = 10:1.
	if result.CreditQuantum != 10 {
		t.Errorf("creditQuantum = %d, want 10", result.CreditQuantum)
	}
	if result.RefundQuantumFen != 1 {
		t.Errorf("refundQuantumFen = %d, want 1", result.RefundQuantumFen)
	}
	// Validate reserved credits = refundFen * creditQuantum.
	if result.ReservedCredits != result.RefundFen*result.CreditQuantum {
		t.Errorf("quantum mismatch: reserved=%d, refundFen=%d * quantum=%d",
			result.ReservedCredits, result.RefundFen, result.CreditQuantum)
	}
	// Validate remainder = refundable - reserved (sub-quantum stays available).
	expectedRemainder := int64(100000) - result.ReservedCredits
	if result.RemainderCredits != expectedRemainder {
		t.Errorf("remainder = %d, want %d", result.RemainderCredits, expectedRemainder)
	}
}

func TestQuantumPersistedOnFullRefund(t *testing.T) {
	testQuantumPersisted(t)
}

func TestQuantumConservation(t *testing.T) {
	// For various consumption levels, verify credit + money conservation.
	cases := []struct {
		name              string
		refundableCredits int64
		wantRefundFen     int64
		wantReserved      int64
		wantRemainder     int64
	}{
		{"0 consumed", 100000, 10000, 100000, 0},
		{"1 consumed", 99990, 9999, 99990, 0},
		{"50,010 consumed", 49990, 4999, 49990, 0},
		{"100,000 consumed", 0, 0, 0, 0},
		{"sub-quantum", 99995, 9999, 99990, 5},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			refundFen, reserved, remainder := ComputeRefundable(tc.refundableCredits, 0)
			if refundFen != tc.wantRefundFen {
				t.Errorf("refundFen = %d, want %d", refundFen, tc.wantRefundFen)
			}
			if reserved != tc.wantReserved {
				t.Errorf("reserved = %d, want %d", reserved, tc.wantReserved)
			}
			if remainder != tc.wantRemainder {
				t.Errorf("remainder = %d, want %d", remainder, tc.wantRemainder)
			}
			// Conservation: reserved + remainder == refundableCredits.
			if reserved+remainder != tc.refundableCredits {
				t.Errorf("conservation: %d + %d != %d", reserved, remainder, tc.refundableCredits)
			}
		})
	}
}

// ===========================================================================
// Fix #1 tests: Provider money validation
// ===========================================================================

func TestProviderMoneyUnderRefundIsRejectedBeforeFailureTransition(t *testing.T) {
	h := newTestHarness(t)
	h.wallet.setGrants("cust", []commerce.AllocationGrant{
		rechargeGrant("grant-1", 100000, 0, 100000),
	})
	h.orders.addFulfilledOrder("ns", "order-money-under", "cust", 10000)

	// Provider query returns success but with less money than reserved.
	h.provider.refundResult = func(input payment.RefundInput) (payment.RefundSubmission, error) {
		return payment.RefundSubmission{Provider: payment.ProviderWeChat, ProviderRefundID: input.IdempotencyKey, Status: "processing"}, nil
	}
	h.provider.queryResult = func(input payment.RefundQueryInput) (payment.RefundFact, error) {
		return verifiedRefundFact(payment.ProviderWeChat, "order-money-under", input.ProviderRefundID, "SUCCESS", "hash-money-under", 5000), nil
	}

	rec, _, err := h.svc.CreateRefund(context.Background(), CreateRefundInput{
		Namespace: "ns", OrderID: "order-money-under", CustomerID: "cust",
		AmountCents: 0, Currency: "CNY", IdempotencyKey: "idem-money-under",
	})
	if err != nil {
		t.Fatal(err)
	}

	// First: submit -> provider_processing.
	_, _ = h.svc.ProcessOne(context.Background(), "ns", rec.ID)

	// Second: query returns success with insufficient money -> should fail.
	result, err := h.svc.ProcessOne(context.Background(), "ns", rec.ID)
	if err == nil {
		t.Fatal("expected money validation failure")
	}
	if !errors.Is(err, ErrRefundFactMismatch) {
		t.Fatalf("error = %v, want ErrRefundFactMismatch", err)
	}
	if result.Status != RefundStatusProviderProcessing {
		t.Errorf("status = %s, want provider_processing", result.Status)
	}
	// No credits reversed.
	if h.reverser.totalReversed() != 0 {
		t.Errorf("total reversed = %d, want 0 (money validation rejected)", h.reverser.totalReversed())
	}
}

func TestProviderMoneyMatchesSucceeds(t *testing.T) {
	h := newTestHarness(t)
	h.wallet.setGrants("cust", []commerce.AllocationGrant{
		rechargeGrant("grant-1", 100000, 0, 100000),
	})
	h.orders.addFulfilledOrder("ns", "order-money-match", "cust", 10000)

	// Provider query returns success with exact money match.
	h.provider.refundResult = func(input payment.RefundInput) (payment.RefundSubmission, error) {
		return payment.RefundSubmission{Provider: payment.ProviderWeChat, ProviderRefundID: input.IdempotencyKey, Status: "processing"}, nil
	}
	h.provider.queryResult = func(input payment.RefundQueryInput) (payment.RefundFact, error) {
		return verifiedRefundFact(payment.ProviderWeChat, "order-money-match", input.ProviderRefundID, "SUCCESS", "hash-money-match", 10000), nil
	}

	rec, _, err := h.svc.CreateRefund(context.Background(), CreateRefundInput{
		Namespace: "ns", OrderID: "order-money-match", CustomerID: "cust",
		AmountCents: 0, Currency: "CNY", IdempotencyKey: "idem-money-match",
	})
	if err != nil {
		t.Fatal(err)
	}

	_, _ = h.svc.ProcessOne(context.Background(), "ns", rec.ID)
	result, err := h.svc.ProcessOne(context.Background(), "ns", rec.ID)
	if err != nil {
		t.Fatalf("expected success: %v", err)
	}
	if result.Status != RefundStatusFulfilled {
		t.Errorf("status = %s, want fulfilled", result.Status)
	}
	if h.reverser.totalReversed() != 100000 {
		t.Errorf("total reversed = %d, want 100000", h.reverser.totalReversed())
	}
}

// ===========================================================================
// Fix #2 test: Fence retained on post-transition transient error
// ===========================================================================

func TestFenceRetainedOnProviderProcessingTransientError(t *testing.T) {
	h := newTestHarness(t)
	h.wallet.setGrants("cust", []commerce.AllocationGrant{
		rechargeGrant("grant-1", 100000, 0, 100000),
	})
	h.orders.addFulfilledOrder("ns", "order-transient", "cust", 10000)

	// Provider returns processing, so the refund stays in provider_processing.
	h.provider.refundResult = func(input payment.RefundInput) (payment.RefundSubmission, error) {
		return payment.RefundSubmission{Provider: payment.ProviderWeChat, ProviderRefundID: input.IdempotencyKey, Status: "processing"}, nil
	}

	rec, _, err := h.svc.CreateRefund(context.Background(), CreateRefundInput{
		Namespace: "ns", OrderID: "order-transient", CustomerID: "cust",
		AmountCents: 0, Currency: "CNY", IdempotencyKey: "idem-transient",
	})
	if err != nil {
		t.Fatal(err)
	}

	result, err := h.svc.ProcessOne(context.Background(), "ns", rec.ID)
	if err != nil {
		t.Fatalf("process: %v", err)
	}
	if result.Status != RefundStatusProviderProcessing {
		t.Fatalf("status = %s, want provider_processing", result.Status)
	}

	// Fence should NOT be released — reserved credits are still guarded.
	if h.fence.released.Load() != 0 {
		t.Errorf("fence released = %d, want 0 (fence must be retained in provider_processing)", h.fence.released.Load())
	}
}

// ===========================================================================
// Fix #3 tests: ApplyRefundCallback
// ===========================================================================

func TestApplyRefundCallbackSuccess(t *testing.T) {
	h := newTestHarness(t)
	h.wallet.setGrants("cust", []commerce.AllocationGrant{
		rechargeGrant("grant-1", 100000, 0, 100000),
	})
	h.orders.addFulfilledOrder("ns", "order-cb", "cust", 10000)

	// Submit returns processing so we can test the callback path.
	h.provider.refundResult = func(input payment.RefundInput) (payment.RefundSubmission, error) {
		return payment.RefundSubmission{Provider: payment.ProviderWeChat, ProviderRefundID: "pr-cb-1", Status: "processing"}, nil
	}

	rec, _, err := h.svc.CreateRefund(context.Background(), CreateRefundInput{
		Namespace: "ns", OrderID: "order-cb", CustomerID: "cust",
		AmountCents: 0, Currency: "CNY", IdempotencyKey: "idem-cb",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Drive to provider_processing.
	result, err := h.svc.ProcessOne(context.Background(), "ns", rec.ID)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != RefundStatusProviderProcessing {
		t.Fatalf("status = %s, want provider_processing", result.Status)
	}
	if result.ProviderRefundID != "pr-cb-1" {
		t.Fatalf("providerRefundID = %s, want pr-cb-1", result.ProviderRefundID)
	}

	// Apply a verified refund callback fact.
	callbackFact := verifiedWechatRefundCallbackFact("order-cb", "pr-cb-1", "SUCCESS", "hash-cb-1", 10000)

	result, err = h.svc.ApplyRefundCallback(context.Background(), "ns", callbackFact)
	if err != nil {
		t.Fatalf("apply callback: %v", err)
	}
	if result.Status != RefundStatusFulfilled {
		t.Errorf("status = %s, want fulfilled", result.Status)
	}
	if h.reverser.totalReversed() != 100000 {
		t.Errorf("total reversed = %d, want 100000", h.reverser.totalReversed())
	}
	if h.fence.released.Load() != 1 {
		t.Errorf("fence released = %d, want 1", h.fence.released.Load())
	}
}

func TestHandleCallbackVerifiesThenAppliesRefundFact(t *testing.T) {
	h := newTestHarness(t)
	h.wallet.setGrants("cust", []commerce.AllocationGrant{
		rechargeGrant("grant-1", 100000, 0, 100000),
	})
	h.orders.addFulfilledOrder("ns", "order-http-cb", "cust", 10000)
	h.provider.refundResult = func(payment.RefundInput) (payment.RefundSubmission, error) {
		return payment.RefundSubmission{Provider: payment.ProviderWeChat, ProviderRefundID: "pr-http-cb", Status: "processing"}, nil
	}

	rec, _, err := h.svc.CreateRefund(t.Context(), CreateRefundInput{
		Namespace: "ns", OrderID: "order-http-cb", CustomerID: "cust",
		Currency: "CNY", IdempotencyKey: "idem-http-cb",
	})
	require.NoError(t, err)
	rec, err = h.svc.ProcessOne(t.Context(), "ns", rec.ID)
	require.NoError(t, err)
	require.Equal(t, RefundStatusProviderProcessing, rec.Status)

	h.provider.callbackFact = verifiedWechatRefundCallbackFact("order-http-cb", "pr-http-cb", "SUCCESS", "hash-http-cb", 10000)
	headers := http.Header{"Wechatpay-Serial": []string{"platform-serial"}}
	rec, err = h.svc.HandleCallback(t.Context(), "ns", payment.ProviderWeChat, headers, []byte(`{"id":"refund-event"}`))
	require.NoError(t, err)
	require.Equal(t, RefundStatusFulfilled, rec.Status)
	require.Equal(t, int64(1), h.provider.refundCallbackCallN.Load())
	require.Equal(t, "platform-serial", h.provider.callbackHeaders.Get("Wechatpay-Serial"))
	require.Equal(t, `{"id":"refund-event"}`, string(h.provider.callbackBody))
	require.Equal(t, int64(100000), h.reverser.totalReversed())
}

func TestHandleCallbackVerificationFailureHasNoRefundEffects(t *testing.T) {
	h := newTestHarness(t)
	h.wallet.setGrants("cust", []commerce.AllocationGrant{
		rechargeGrant("grant-1", 100000, 0, 100000),
	})
	h.orders.addFulfilledOrder("ns", "order-http-reject", "cust", 10000)
	h.provider.refundResult = func(payment.RefundInput) (payment.RefundSubmission, error) {
		return payment.RefundSubmission{Provider: payment.ProviderWeChat, ProviderRefundID: "pr-http-reject", Status: "processing"}, nil
	}
	rec, _, err := h.svc.CreateRefund(t.Context(), CreateRefundInput{
		Namespace: "ns", OrderID: "order-http-reject", CustomerID: "cust",
		Currency: "CNY", IdempotencyKey: "idem-http-reject",
	})
	require.NoError(t, err)
	rec, err = h.svc.ProcessOne(t.Context(), "ns", rec.ID)
	require.NoError(t, err)
	require.Equal(t, RefundStatusProviderProcessing, rec.Status)

	h.provider.callbackErr = payment.ErrInvalidSignature
	_, err = h.svc.HandleCallback(t.Context(), "ns", payment.ProviderWeChat, http.Header{}, []byte("invalid"))
	require.ErrorIs(t, err, payment.ErrInvalidSignature)
	require.Equal(t, int64(1), h.provider.refundCallbackCallN.Load())
	require.Equal(t, int64(0), h.reverser.callN.Load())
	rec, err = h.svc.GetRefund(t.Context(), "ns", rec.ID)
	require.NoError(t, err)
	require.Equal(t, RefundStatusProviderProcessing, rec.Status)
}

func TestHandleCallbackRejectsNonWeChatWithoutVerifying(t *testing.T) {
	h := newTestHarness(t)

	_, err := h.svc.HandleCallback(t.Context(), "ns", payment.ProviderAlipay, http.Header{}, []byte("callback"))
	require.ErrorIs(t, err, payment.ErrPermanentProviderProtocol)
	require.Zero(t, h.provider.refundCallbackCallN.Load())
	require.Zero(t, h.reverser.callN.Load())
}

func TestHandleCallbackUnknownProviderRefundIDHasNoEffects(t *testing.T) {
	h := newTestHarness(t)
	h.provider.callbackFact = verifiedWechatRefundCallbackFact("unknown-order", "unknown-provider-refund", "SUCCESS", "hash-unknown-provider-refund", 10000)

	_, err := h.svc.HandleCallback(t.Context(), "ns", payment.ProviderWeChat, http.Header{}, []byte(`{"id":"unknown-refund-event"}`))
	require.ErrorIs(t, err, ErrRefundNotFound)
	require.Equal(t, int64(1), h.provider.refundCallbackCallN.Load())
	require.Zero(t, h.reverser.callN.Load())
	require.Empty(t, h.repo.refunds)
	require.Empty(t, h.repo.facts)
}

func TestApplyRefundCallbackIdempotent(t *testing.T) {
	h := newTestHarness(t)
	h.wallet.setGrants("cust", []commerce.AllocationGrant{
		rechargeGrant("grant-1", 100000, 0, 100000),
	})
	h.orders.addFulfilledOrder("ns", "order-cb-idem", "cust", 10000)

	h.provider.refundResult = func(input payment.RefundInput) (payment.RefundSubmission, error) {
		return payment.RefundSubmission{Provider: payment.ProviderWeChat, ProviderRefundID: "pr-cb-idem", Status: "processing"}, nil
	}

	rec, _, _ := h.svc.CreateRefund(context.Background(), CreateRefundInput{
		Namespace: "ns", OrderID: "order-cb-idem", CustomerID: "cust",
		AmountCents: 0, Currency: "CNY", IdempotencyKey: "idem-cb-idem",
	})
	_, _ = h.svc.ProcessOne(context.Background(), "ns", rec.ID)

	callbackFact := verifiedWechatRefundCallbackFact("order-cb-idem", "pr-cb-idem", "SUCCESS", "hash-cb-idem", 10000)

	// First callback.
	r1, err := h.svc.ApplyRefundCallback(context.Background(), "ns", callbackFact)
	if err != nil {
		t.Fatalf("first callback: %v", err)
	}
	if r1.Status != RefundStatusFulfilled {
		t.Fatalf("first: status = %s, want fulfilled", r1.Status)
	}

	// Replay the same callback — should be idempotent.
	r2, err := h.svc.ApplyRefundCallback(context.Background(), "ns", callbackFact)
	if err != nil {
		t.Fatalf("second callback: %v", err)
	}
	if r2.Status != RefundStatusFulfilled {
		t.Errorf("second: status = %s, want fulfilled", r2.Status)
	}

	// Reverser called exactly once.
	if h.reverser.callN.Load() != 1 {
		t.Errorf("reverser calls = %d, want 1 (idempotent)", h.reverser.callN.Load())
	}
}

func TestApplyRefundCallbackDefinitiveFailure(t *testing.T) {
	h := newTestHarness(t)
	h.wallet.setGrants("cust", []commerce.AllocationGrant{
		rechargeGrant("grant-1", 100000, 0, 100000),
	})
	h.orders.addFulfilledOrder("ns", "order-cb-fail", "cust", 10000)

	h.provider.refundResult = func(input payment.RefundInput) (payment.RefundSubmission, error) {
		return payment.RefundSubmission{Provider: payment.ProviderWeChat, ProviderRefundID: "pr-cb-fail", Status: "processing"}, nil
	}

	rec, _, _ := h.svc.CreateRefund(context.Background(), CreateRefundInput{
		Namespace: "ns", OrderID: "order-cb-fail", CustomerID: "cust",
		AmountCents: 0, Currency: "CNY", IdempotencyKey: "idem-cb-fail",
	})
	_, _ = h.svc.ProcessOne(context.Background(), "ns", rec.ID)

	// Callback with definitive failure.
	callbackFact := verifiedWechatRefundCallbackFact("order-cb-fail", "pr-cb-fail", "CLOSED", "hash-cb-fail", 10000)

	result, err := h.svc.ApplyRefundCallback(context.Background(), "ns", callbackFact)
	if err == nil {
		t.Fatal("expected definitive failure error")
	}
	if result.Status != RefundStatusFailed {
		t.Errorf("status = %s, want failed", result.Status)
	}
	if h.reverser.totalReversed() != 0 {
		t.Errorf("total reversed = %d, want 0", h.reverser.totalReversed())
	}
	if h.fence.released.Load() != 1 {
		t.Errorf("fence released = %d, want 1 (released on failure)", h.fence.released.Load())
	}
}

func TestApplyRefundCallbackMoneyMismatch(t *testing.T) {
	h := newTestHarness(t)
	h.wallet.setGrants("cust", []commerce.AllocationGrant{
		rechargeGrant("grant-1", 100000, 0, 100000),
	})
	h.orders.addFulfilledOrder("ns", "order-cb-money", "cust", 10000)

	h.provider.refundResult = func(input payment.RefundInput) (payment.RefundSubmission, error) {
		return payment.RefundSubmission{Provider: payment.ProviderWeChat, ProviderRefundID: "pr-cb-money", Status: "processing"}, nil
	}

	rec, _, _ := h.svc.CreateRefund(context.Background(), CreateRefundInput{
		Namespace: "ns", OrderID: "order-cb-money", CustomerID: "cust",
		AmountCents: 0, Currency: "CNY", IdempotencyKey: "idem-cb-money",
	})
	_, _ = h.svc.ProcessOne(context.Background(), "ns", rec.ID)

	// Callback with insufficient money.
	callbackFact := verifiedWechatRefundCallbackFact("order-cb-money", "pr-cb-money", "SUCCESS", "hash-cb-money", 5000)

	result, err := h.svc.ApplyRefundCallback(context.Background(), "ns", callbackFact)
	if err == nil {
		t.Fatal("expected money validation failure")
	}
	if result.Status != RefundStatusProviderProcessing {
		t.Errorf("status = %s, want provider_processing", result.Status)
	}
	if h.reverser.totalReversed() != 0 {
		t.Errorf("total reversed = %d, want 0", h.reverser.totalReversed())
	}
}

func TestApplyRefundCallbackRejectsPersistedAuthorityMismatchesBeforeEffects(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*payment.RefundFact)
	}{
		{name: "provider", mutate: func(f *payment.RefundFact) { f.Provider = payment.ProviderAlipay }},
		{name: "merchant order", mutate: func(f *payment.RefundFact) { f.ProviderOrderID = "other-order" }},
		{name: "original payment ID", mutate: func(f *payment.RefundFact) { f.ProviderPaymentID = "other-payment" }},
		{name: "merchant", mutate: func(f *payment.RefundFact) { f.MerchantID = "other-merchant" }},
		{name: "refund amount", mutate: func(f *payment.RefundFact) { f.AmountMinor++ }},
		{name: "original total", mutate: func(f *payment.RefundFact) { f.TotalAmountMinor++ }},
		{name: "currency", mutate: func(f *payment.RefundFact) { f.Currency = "USD" }},
		{name: "non-terminal", mutate: func(f *payment.RefundFact) { f.Terminal = false }},
		{name: "status and success", mutate: func(f *payment.RefundFact) { f.Status = "CLOSED" }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newTestHarness(t)
			h.wallet.setGrants("cust", []commerce.AllocationGrant{rechargeGrant("grant-1", 100000, 0, 100000)})
			const orderID = "order-cb-authority"
			h.orders.addFulfilledOrder("ns", orderID, "cust", 10000)
			h.provider.refundResult = func(payment.RefundInput) (payment.RefundSubmission, error) {
				return payment.RefundSubmission{Provider: payment.ProviderWeChat, ProviderRefundID: "pr-cb-authority", Status: "processing"}, nil
			}
			rec, _, err := h.svc.CreateRefund(t.Context(), CreateRefundInput{
				Namespace: "ns", OrderID: orderID, CustomerID: "cust", Currency: "CNY", IdempotencyKey: "idem-cb-authority",
			})
			require.NoError(t, err)
			rec, err = h.svc.ProcessOne(t.Context(), "ns", rec.ID)
			require.NoError(t, err)
			require.Equal(t, RefundStatusProviderProcessing, rec.Status)

			fact := verifiedWechatRefundCallbackFact(orderID, rec.ProviderRefundID, "SUCCESS", "hash-"+orderID, 10000)
			tt.mutate(&fact)
			result, err := h.svc.ApplyRefundCallback(t.Context(), "ns", fact)
			require.ErrorIs(t, err, ErrRefundFactMismatch)
			require.Equal(t, RefundStatusProviderProcessing, result.Status)
			require.Zero(t, h.reverser.totalReversed())
			require.Zero(t, h.fence.released.Load())
			facts, factsErr := h.repo.GetFacts(t.Context(), "ns", rec.ID)
			require.NoError(t, factsErr)
			require.Empty(t, facts)
		})
	}
}

func TestAlipayQueryRejectsPersistedAuthorityMismatchesBeforeEffects(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*payment.RefundFact)
	}{
		{name: "provider", mutate: func(f *payment.RefundFact) { f.Provider = payment.ProviderWeChat }},
		{name: "refund ID", mutate: func(f *payment.RefundFact) { f.ProviderRefundID = "other-refund" }},
		{name: "merchant order", mutate: func(f *payment.RefundFact) { f.ProviderOrderID = "other-order" }},
		{name: "original payment ID", mutate: func(f *payment.RefundFact) { f.ProviderPaymentID = "other-payment" }},
		{name: "merchant", mutate: func(f *payment.RefundFact) { f.MerchantID = "other-merchant" }},
		{name: "refund amount", mutate: func(f *payment.RefundFact) { f.AmountMinor++ }},
		{name: "original total", mutate: func(f *payment.RefundFact) { f.TotalAmountMinor++ }},
		{name: "currency", mutate: func(f *payment.RefundFact) { f.Currency = "USD" }},
		{name: "non-terminal success", mutate: func(f *payment.RefundFact) { f.Terminal = false }},
		{name: "success without hash", mutate: func(f *payment.RefundFact) { f.RawHash = "" }},
		{name: "status success mismatch", mutate: func(f *payment.RefundFact) { f.Success = false }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newTestHarnessForProvider(t, payment.ProviderAlipay)
			h.wallet.setGrants("cust", []commerce.AllocationGrant{rechargeGrant("grant-1", 100000, 0, 100000)})
			const orderID = "order-alipay-query-authority"
			h.orders.addFulfilledOrder("ns", orderID, "cust", 10000)
			h.provider.refundResult = func(input payment.RefundInput) (payment.RefundSubmission, error) {
				return payment.RefundSubmission{Provider: payment.ProviderAlipay, ProviderRefundID: input.IdempotencyKey, Status: "processing"}, nil
			}
			h.provider.queryResult = func(input payment.RefundQueryInput) (payment.RefundFact, error) {
				fact := verifiedRefundFact(payment.ProviderAlipay, orderID, input.ProviderRefundID, "REFUND_SUCCESS", "hash-"+orderID, input.AmountMinor)
				tt.mutate(&fact)
				return fact, nil
			}

			rec, _, err := h.svc.CreateRefund(t.Context(), CreateRefundInput{
				Namespace: "ns", OrderID: orderID, CustomerID: "cust", Currency: "CNY", IdempotencyKey: "idem-alipay-query-authority",
			})
			require.NoError(t, err)
			rec, err = h.svc.ProcessOne(t.Context(), "ns", rec.ID)
			require.NoError(t, err)
			require.Equal(t, RefundStatusProviderProcessing, rec.Status)

			result, err := h.svc.ProcessOne(t.Context(), "ns", rec.ID)
			require.ErrorIs(t, err, ErrRefundFactMismatch)
			require.Equal(t, RefundStatusProviderProcessing, result.Status)
			require.Zero(t, h.reverser.totalReversed())
			require.Zero(t, h.fence.released.Load())
			facts, factsErr := h.repo.GetFacts(t.Context(), "ns", rec.ID)
			require.NoError(t, factsErr)
			require.Empty(t, facts)
		})
	}
}

func TestAlipayQueryWithMatchingPersistedAuthorityFulfills(t *testing.T) {
	h := newTestHarnessForProvider(t, payment.ProviderAlipay)
	h.wallet.setGrants("cust", []commerce.AllocationGrant{rechargeGrant("grant-1", 100000, 0, 100000)})
	const orderID = "order-alipay-query-success"
	h.orders.addFulfilledOrder("ns", orderID, "cust", 10000)
	h.provider.refundResult = func(input payment.RefundInput) (payment.RefundSubmission, error) {
		return payment.RefundSubmission{Provider: payment.ProviderAlipay, ProviderRefundID: input.IdempotencyKey, Status: "processing"}, nil
	}
	h.provider.queryResult = func(input payment.RefundQueryInput) (payment.RefundFact, error) {
		return verifiedRefundFact(payment.ProviderAlipay, orderID, input.ProviderRefundID, "REFUND_SUCCESS", "hash-alipay-query-success", input.AmountMinor), nil
	}
	rec, _, err := h.svc.CreateRefund(t.Context(), CreateRefundInput{
		Namespace: "ns", OrderID: orderID, CustomerID: "cust", Currency: "CNY", IdempotencyKey: "idem-alipay-query-success",
	})
	require.NoError(t, err)
	rec, err = h.svc.ProcessOne(t.Context(), "ns", rec.ID)
	require.NoError(t, err)
	require.Equal(t, RefundStatusProviderProcessing, rec.Status)
	rec, err = h.svc.ProcessOne(t.Context(), "ns", rec.ID)
	require.NoError(t, err)
	require.Equal(t, RefundStatusFulfilled, rec.Status)
	require.Equal(t, int64(100000), h.reverser.totalReversed())
	require.Equal(t, int64(1), h.fence.released.Load())
}

func TestAlipayQueryRejectsMismatchedDefinitiveFailureBeforeTransition(t *testing.T) {
	h := newTestHarnessForProvider(t, payment.ProviderAlipay)
	h.wallet.setGrants("cust", []commerce.AllocationGrant{rechargeGrant("grant-1", 100000, 0, 100000)})
	const orderID = "order-alipay-query-failure-authority"
	h.orders.addFulfilledOrder("ns", orderID, "cust", 10000)
	h.provider.refundResult = func(input payment.RefundInput) (payment.RefundSubmission, error) {
		return payment.RefundSubmission{Provider: payment.ProviderAlipay, ProviderRefundID: input.IdempotencyKey, Status: "processing"}, nil
	}
	h.provider.queryResult = func(input payment.RefundQueryInput) (payment.RefundFact, error) {
		fact := verifiedRefundFact(payment.ProviderAlipay, orderID, input.ProviderRefundID, "REFUND_FAIL", "hash-failure", input.AmountMinor)
		fact.MerchantID = "wrong-merchant"
		return fact, nil
	}
	rec, _, err := h.svc.CreateRefund(t.Context(), CreateRefundInput{
		Namespace: "ns", OrderID: orderID, CustomerID: "cust", Currency: "CNY", IdempotencyKey: "idem-alipay-query-failure-authority",
	})
	require.NoError(t, err)
	rec, err = h.svc.ProcessOne(t.Context(), "ns", rec.ID)
	require.NoError(t, err)
	result, err := h.svc.ProcessOne(t.Context(), "ns", rec.ID)
	require.ErrorIs(t, err, ErrRefundFactMismatch)
	require.Equal(t, RefundStatusProviderProcessing, result.Status)
	require.Zero(t, h.reverser.totalReversed())
	require.Zero(t, h.fence.released.Load())
}

func TestAlipayCallbackRejectsPersistedAuthorityMismatchesBeforeEffects(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*payment.RefundFact)
	}{
		{name: "provider", mutate: func(f *payment.RefundFact) { f.Provider = payment.ProviderWeChat }},
		{name: "merchant order", mutate: func(f *payment.RefundFact) { f.ProviderOrderID = "other-order" }},
		{name: "original payment ID", mutate: func(f *payment.RefundFact) { f.ProviderPaymentID = "other-payment" }},
		{name: "merchant", mutate: func(f *payment.RefundFact) { f.MerchantID = "other-merchant" }},
		{name: "refund amount", mutate: func(f *payment.RefundFact) { f.AmountMinor++ }},
		{name: "original total", mutate: func(f *payment.RefundFact) { f.TotalAmountMinor++ }},
		{name: "currency", mutate: func(f *payment.RefundFact) { f.Currency = "USD" }},
		{name: "non-terminal success", mutate: func(f *payment.RefundFact) { f.Terminal = false }},
		{name: "success without hash", mutate: func(f *payment.RefundFact) { f.RawHash = "" }},
		{name: "status success mismatch", mutate: func(f *payment.RefundFact) { f.Status = "REFUND_FAIL" }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newTestHarnessForProvider(t, payment.ProviderAlipay)
			h.wallet.setGrants("cust", []commerce.AllocationGrant{rechargeGrant("grant-1", 100000, 0, 100000)})
			const orderID = "order-alipay-callback-authority"
			h.orders.addFulfilledOrder("ns", orderID, "cust", 10000)
			h.provider.refundResult = func(payment.RefundInput) (payment.RefundSubmission, error) {
				return payment.RefundSubmission{Provider: payment.ProviderAlipay, ProviderRefundID: "pr-alipay-callback", Status: "processing"}, nil
			}
			rec, _, err := h.svc.CreateRefund(t.Context(), CreateRefundInput{
				Namespace: "ns", OrderID: orderID, CustomerID: "cust", Currency: "CNY", IdempotencyKey: "idem-alipay-callback-authority",
			})
			require.NoError(t, err)
			rec, err = h.svc.ProcessOne(t.Context(), "ns", rec.ID)
			require.NoError(t, err)

			fact := verifiedRefundFact(payment.ProviderAlipay, orderID, rec.ProviderRefundID, "REFUND_SUCCESS", "hash-"+orderID, 10000)
			tt.mutate(&fact)
			result, err := h.svc.ApplyRefundCallback(t.Context(), "ns", fact)
			require.ErrorIs(t, err, ErrRefundFactMismatch)
			require.Equal(t, RefundStatusProviderProcessing, result.Status)
			require.Zero(t, h.reverser.totalReversed())
			require.Zero(t, h.fence.released.Load())
			facts, factsErr := h.repo.GetFacts(t.Context(), "ns", rec.ID)
			require.NoError(t, factsErr)
			require.Empty(t, facts)
		})
	}
}

func TestAlipayCallbackWithMatchingPersistedAuthorityFulfills(t *testing.T) {
	h := newTestHarnessForProvider(t, payment.ProviderAlipay)
	h.wallet.setGrants("cust", []commerce.AllocationGrant{rechargeGrant("grant-1", 100000, 0, 100000)})
	const orderID = "order-alipay-callback-success"
	h.orders.addFulfilledOrder("ns", orderID, "cust", 10000)
	h.provider.refundResult = func(payment.RefundInput) (payment.RefundSubmission, error) {
		return payment.RefundSubmission{Provider: payment.ProviderAlipay, ProviderRefundID: "pr-alipay-callback-success", Status: "processing"}, nil
	}
	rec, _, err := h.svc.CreateRefund(t.Context(), CreateRefundInput{
		Namespace: "ns", OrderID: orderID, CustomerID: "cust", Currency: "CNY", IdempotencyKey: "idem-alipay-callback-success",
	})
	require.NoError(t, err)
	rec, err = h.svc.ProcessOne(t.Context(), "ns", rec.ID)
	require.NoError(t, err)
	fact := verifiedRefundFact(payment.ProviderAlipay, orderID, rec.ProviderRefundID, "REFUND_SUCCESS", "hash-alipay-callback-success", 10000)
	rec, err = h.svc.ApplyRefundCallback(t.Context(), "ns", fact)
	require.NoError(t, err)
	require.Equal(t, RefundStatusFulfilled, rec.Status)
	require.Equal(t, int64(100000), h.reverser.totalReversed())
	require.Equal(t, int64(1), h.fence.released.Load())
}

// Compile-time check that mockProvider satisfies interfaces.
var (
	_ ProviderRefunder       = (*mockProvider)(nil)
	_ RefundCallbackVerifier = (*mockProvider)(nil)
)

// Stub to satisfy RefundCallbackVerifier on mockProvider for compile-time check.
func (p *mockProvider) VerifyRefundCallback(_ context.Context, headers http.Header, body []byte) (payment.RefundFact, error) {
	p.refundCallbackCallN.Add(1)
	p.callbackHeaders = headers.Clone()
	p.callbackBody = append([]byte(nil), body...)
	return p.callbackFact, p.callbackErr
}

// TestListRefundsFiltersAndPageWindow verifies the namespace-level refund
// list: customer and status filters, the applied page-window bounds, and the
// reported total count.
func TestListRefundsFiltersAndPageWindow(t *testing.T) {
	h := newTestHarness(t)
	h.orders.addFulfilledOrder("ns", "order-list-1", "cust-1", 10000)
	h.orders.addFulfilledOrder("ns", "order-list-2", "cust-2", 10000)
	for _, in := range []CreateRefundInput{
		{Namespace: "ns", OrderID: "order-list-1", CustomerID: "cust-1", Currency: "CNY", IdempotencyKey: "list-idem-1"},
		{Namespace: "ns", OrderID: "order-list-2", CustomerID: "cust-2", Currency: "CNY", IdempotencyKey: "list-idem-2"},
	} {
		if _, _, err := h.svc.CreateRefund(t.Context(), in); err != nil {
			t.Fatal(err)
		}
	}

	// given: two pending_fence refunds across two customers
	refunds, total, err := h.svc.ListRefunds(t.Context(), ListRefundsInput{Namespace: "ns"})
	if err != nil {
		t.Fatal(err)
	}
	if total != 2 || len(refunds) != 2 {
		t.Fatalf("unfiltered list = %d items, total %d; want 2/2", len(refunds), total)
	}

	refunds, total, err = h.svc.ListRefunds(t.Context(), ListRefundsInput{Namespace: "ns", CustomerID: "cust-1"})
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(refunds) != 1 {
		t.Fatalf("cust-1 list = %d items, total %d; want 1/1", len(refunds), total)
	}

	pending := RefundStatusPendingFence
	refunds, total, err = h.svc.ListRefunds(t.Context(), ListRefundsInput{Namespace: "ns", Status: &pending})
	if err != nil {
		t.Fatal(err)
	}
	if total != 2 || len(refunds) != 2 {
		t.Fatalf("pending-fence list = %d items, total %d; want 2/2", len(refunds), total)
	}

	fulfilled := RefundStatusFulfilled
	refunds, total, err = h.svc.ListRefunds(t.Context(), ListRefundsInput{Namespace: "ns", CustomerID: "cust-1", Status: &fulfilled})
	if err != nil {
		t.Fatal(err)
	}
	if total != 0 || len(refunds) != 0 {
		t.Fatalf("cust-1 fulfilled list = %d items, total %d; want 0/0", len(refunds), total)
	}

	// then: page window slices the result while total stays whole
	refunds, total, err = h.svc.ListRefunds(t.Context(), ListRefundsInput{Namespace: "ns", Limit: 1, Offset: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(refunds) != 1 || total != 2 {
		t.Fatalf("paged list = %d items, total %d; want 1/2", len(refunds), total)
	}

	// then: input bounds are applied (defaults and clamps)
	if _, _, err := h.svc.ListRefunds(t.Context(), ListRefundsInput{Namespace: "ns", Limit: -5, Offset: -3}); err != nil {
		t.Fatal(err)
	}
	if got := h.repo.lastListInput; got.Limit != 100 || got.Offset != 0 {
		t.Errorf("bounded input = (%d, %d); want (100, 0)", got.Limit, got.Offset)
	}
	if _, _, err := h.svc.ListRefunds(t.Context(), ListRefundsInput{Namespace: "ns", Limit: 5000}); err != nil {
		t.Fatal(err)
	}
	if got := h.repo.lastListInput.Limit; got != 1000 {
		t.Errorf("clamped limit = %d, want 1000", got)
	}

	// then: missing namespace is rejected before the query
	if _, _, err := h.svc.ListRefunds(t.Context(), ListRefundsInput{}); err == nil {
		t.Fatal("expected an error for missing namespace")
	}
}
