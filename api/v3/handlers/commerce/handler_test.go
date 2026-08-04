package commerce

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/openmeter/commerce"
	"github.com/openmeterio/openmeter/openmeter/commerce/payment"
	"github.com/openmeterio/openmeter/openmeter/commerce/refund"
)

// ---------------------------------------------------------------------------
// Mock implementations
// ---------------------------------------------------------------------------

type mockWallet struct {
	wallet *commerce.Wallet
	err    error
}

func (m *mockWallet) GetWallet(_ context.Context, _, _ string) (*commerce.Wallet, error) {
	return m.wallet, m.err
}

type mockCatalog struct {
	products []commerce.Product
	err      error
}

func (m *mockCatalog) CreateProduct(_ context.Context, _ commerce.CreateProductInput) (*commerce.Product, error) {
	return nil, m.err
}
func (m *mockCatalog) GetProduct(_ context.Context, _, _ string) (*commerce.Product, error) {
	return nil, m.err
}
func (m *mockCatalog) ListProducts(_ context.Context, _ string, _ *commerce.ProductKind, _ bool) ([]commerce.Product, error) {
	return m.products, m.err
}
func (m *mockCatalog) UpdateProduct(_ context.Context, _ commerce.UpdateProductInput) (*commerce.Product, error) {
	return nil, m.err
}

type mockOrders struct {
	order   *commerce.Order
	created bool
	err     error
}

func (m *mockOrders) CreateOrder(_ context.Context, _ commerce.CreateOrderInput) (*commerce.Order, bool, error) {
	return m.order, m.created, m.err
}
func (m *mockOrders) GetOrder(_ context.Context, _, _ string) (*commerce.Order, error) {
	return m.order, m.err
}
func (m *mockOrders) TransitionStatus(_ context.Context, _, _ string, _ commerce.OrderStatus) (*commerce.Order, error) {
	return m.order, m.err
}

type mockPayment struct {
	attempt *payment.PaymentAttempt
	err     error
}

func (m *mockPayment) CreateAttempt(_ context.Context, _ payment.CreateAttemptInput) (*payment.PaymentAttempt, bool, error) {
	return m.attempt, false, m.err
}
func (m *mockPayment) GetAttempt(_ context.Context, _, _ string) (*payment.PaymentAttempt, error) {
	return m.attempt, m.err
}
func (m *mockPayment) InitiateCheckout(_ context.Context, _, _ string) (payment.CheckoutResult, error) {
	return payment.CheckoutResult{Attempt: m.attempt}, m.err
}
func (m *mockPayment) HandleCallback(_ context.Context, _ string, _ payment.Provider, _ map[string][]string, _ []byte) (payment.CallbackResult, error) {
	return payment.CallbackResult{}, m.err
}
func (m *mockPayment) ConfirmPayment(_ context.Context, _, _ string) (payment.CallbackResult, error) {
	return payment.CallbackResult{}, m.err
}

type mockRefund struct {
	rec     *refund.RefundRequest
	created bool
	err     error
}

func (m *mockRefund) CreateRefund(_ context.Context, _ refund.CreateRefundInput) (*refund.RefundRequest, bool, error) {
	return m.rec, m.created, m.err
}
func (m *mockRefund) GetRefund(_ context.Context, _, _ string) (*refund.RefundRequest, error) {
	return m.rec, m.err
}
func (m *mockRefund) ProcessOne(_ context.Context, _, _ string) (*refund.RefundRequest, error) {
	return m.rec, m.err
}
func (m *mockRefund) ApplyRefundCallback(_ context.Context, _ string, _ payment.RefundFact) (*refund.RefundRequest, error) {
	return m.rec, m.err
}

func testHandler(svc Services) Handler {
	return New(func(_ context.Context) (string, error) {
		return "test-ns", nil
	}, svc)
}

func doRequest(t *testing.T, h http.Handler, method, path string, body any, pathValues map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode body: %v", err)
		}
	}
	req := httptest.NewRequest(method, path, &buf)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range pathValues {
		req.SetPathValue(k, v)
	}
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr
}

// ---------------------------------------------------------------------------
// Wallet tests
// ---------------------------------------------------------------------------

func TestGetCustomerWallet_Success(t *testing.T) {
	now := time.Now().UTC()
	h := testHandler(Services{
		Wallet: &mockWallet{wallet: &commerce.Wallet{
			CustomerID:            "cust-1",
			ContractVersion:       commerce.ContractVersion,
			TotalAvailableCredits: 1000,
			Buckets:               []commerce.WalletBucket{},
			RetrievedAt:           now,
		}},
	})
	rr := doRequest(t, h.GetCustomerWallet(), http.MethodGet, "/customers/cust-1/wallet", nil,
		map[string]string{"customerId": "cust-1"})
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var wallet api.CommerceWallet
	if err := json.NewDecoder(rr.Body).Decode(&wallet); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if wallet.CustomerId != "cust-1" {
		t.Errorf("expected cust-1, got %s", wallet.CustomerId)
	}
	if wallet.ContractVersion != commerce.ContractVersion {
		t.Errorf("contract version mismatch")
	}
}

func TestGetCustomerWallet_NotFound(t *testing.T) {
	h := testHandler(Services{
		Wallet: &mockWallet{err: commerce.ErrOrderNotFound},
	})
	rr := doRequest(t, h.GetCustomerWallet(), http.MethodGet, "/customers/other/wallet", nil,
		map[string]string{"customerId": "other"})
	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestGetCustomerWallet_MissingCustomerId(t *testing.T) {
	h := testHandler(Services{Wallet: &mockWallet{}})
	rr := doRequest(t, h.GetCustomerWallet(), http.MethodGet, "/customers//wallet", nil, nil)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing customerId, got %d", rr.Code)
	}
}

func TestWalletResponse_NoSecretFields(t *testing.T) {
	rc := int64(500)
	w := &commerce.Wallet{
		CustomerID:            "cust-1",
		ContractVersion:       commerce.ContractVersion,
		TotalAvailableCredits: 1000,
		Buckets: []commerce.WalletBucket{
			{Source: commerce.BucketSourceRecharge, AvailableCredits: 500, RefundableCredits: &rc},
		},
	}
	apiWallet := toAPIWallet(w)
	data, _ := json.Marshal(apiWallet)
	body := string(data)
	for _, secret := range []string{"api_key", "token", "secret", "app_secret"} {
		if containsStr(body, secret) {
			t.Errorf("wallet response contains forbidden field: %s", secret)
		}
	}
}

// ---------------------------------------------------------------------------
// Catalog tests
// ---------------------------------------------------------------------------

func TestListRechargeProducts_Success(t *testing.T) {
	h := testHandler(Services{
		Catalog: &mockCatalog{products: []commerce.Product{
			{DisplayName: "100 Credits", AmountMinor: 1000, Currency: "CNY", Credits: 100, Active: true},
		}},
	})
	rr := doRequest(t, h.ListRechargeProducts(), http.MethodGet, "/recharge-products", nil, nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var list api.CommerceRechargeProductList
	if err := json.NewDecoder(rr.Body).Decode(&list); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(list.Products) != 1 {
		t.Fatalf("expected 1 product, got %d", len(list.Products))
	}
	if list.Products[0].PriceFen != 1000 {
		t.Errorf("expected 1000, got %d", list.Products[0].PriceFen)
	}
}

func TestListRechargeProducts_CurrencyFilter(t *testing.T) {
	h := testHandler(Services{
		Catalog: &mockCatalog{products: []commerce.Product{
			{DisplayName: "CNY Pack", AmountMinor: 1000, Currency: "CNY", Credits: 100, Active: true},
			{DisplayName: "USD Pack", AmountMinor: 100, Currency: "USD", Credits: 100, Active: true},
		}},
	})
	rr := doRequest(t, h.ListRechargeProducts(), http.MethodGet, "/recharge-products?currency=CNY", nil, nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var list api.CommerceRechargeProductList
	_ = json.NewDecoder(rr.Body).Decode(&list)
	if len(list.Products) != 1 {
		t.Errorf("expected 1 product after filter, got %d", len(list.Products))
	}
}

// ---------------------------------------------------------------------------
// Order tests
// ---------------------------------------------------------------------------

func TestCreateOrder_Success(t *testing.T) {
	h := testHandler(Services{
		Orders: &mockOrders{
			order: &commerce.Order{
				PublicID:    "ord-1",
				CustomerID:  "cust-1",
				Kind:        commerce.OrderKindWalletTopUp,
				Status:      commerce.OrderStatusCreated,
				AmountMinor: 1000,
				Currency:    "CNY",
			},
			created: true,
		},
	})
	body := api.CommerceOrderCreate{
		BillingCustomerId: "cust-1",
		Kind:              api.CommerceOrderKindWalletTopUp,
		Currency:          "CNY",
		IdempotencyKey:    "idem-1",
	}
	rr := doRequest(t, h.CreateOrder(), http.MethodPost, "/orders", body, nil)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestCreateOrder_IdempotentReplay(t *testing.T) {
	h := testHandler(Services{
		Orders: &mockOrders{
			order: &commerce.Order{
				PublicID:   "ord-1",
				CustomerID: "cust-1",
				Kind:       commerce.OrderKindWalletTopUp,
				Status:     commerce.OrderStatusCreated,
			},
			created: false,
		},
	})
	body := api.CommerceOrderCreate{
		BillingCustomerId: "cust-1",
		Kind:              api.CommerceOrderKindWalletTopUp,
		Currency:          "CNY",
		IdempotencyKey:    "idem-1",
	}
	rr := doRequest(t, h.CreateOrder(), http.MethodPost, "/orders", body, nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 on idempotent replay, got %d", rr.Code)
	}
}

func TestGetOrder_NotFound(t *testing.T) {
	h := testHandler(Services{
		Orders: &mockOrders{err: commerce.ErrOrderNotFound},
	})
	rr := doRequest(t, h.GetOrder(), http.MethodGet, "/orders/nonexistent", nil,
		map[string]string{"orderId": "nonexistent"})
	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", rr.Code, rr.Body.String())
	}
}

// ---------------------------------------------------------------------------
// Payment callback tests
// ---------------------------------------------------------------------------

func TestPaymentCallback_NoNamespace(t *testing.T) {
	h := New(func(_ context.Context) (string, error) {
		return "", errors.New("no namespace")
	}, Services{Payment: &mockPayment{}})
	rr := doRequest(t, h.AlipayPaymentCallback(), http.MethodPost, "/payment-providers/alipay/callback", "raw-body", nil)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 when namespace unresolvable, got %d", rr.Code)
	}
}

func TestPaymentCallback_Success(t *testing.T) {
	h := testHandler(Services{
		Payment: &mockPayment{attempt: &payment.PaymentAttempt{}},
	})
	rr := doRequest(t, h.WechatPaymentCallback(), http.MethodPost, "/payment-providers/wechat/callback", "raw-body", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var ack api.CommerceProviderCallbackAck
	if err := json.NewDecoder(rr.Body).Decode(&ack); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if ack.Ack == "" {
		t.Error("expected non-empty ack")
	}
}

// ---------------------------------------------------------------------------
// Refund tests
// ---------------------------------------------------------------------------

func TestCreateRefund_Success(t *testing.T) {
	now := time.Now().UTC()
	h := testHandler(Services{
		Refund: &mockRefund{
			rec: &refund.RefundRequest{
				ID:              "ref-1",
				CommerceOrderID: "ord-1",
				CustomerID:      "cust-1",
				Status:          refund.RefundStatusPendingFence,
				CreatedAt:       now,
				UpdatedAt:       now,
			},
			created: true,
		},
	})
	body := api.CommerceRefundCreate{
		BillingCustomerId: "cust-1",
		OrderId:           "ord-1",
		AmountFen:         500,
		Reason:            "customer request",
		IdempotencyKey:    "idem-ref-1",
	}
	rr := doRequest(t, h.CreateRefund(), http.MethodPost, "/refunds", body, nil)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestGetRefund_Success(t *testing.T) {
	now := time.Now().UTC()
	h := testHandler(Services{
		Refund: &mockRefund{
			rec: &refund.RefundRequest{
				ID:              "ref-1",
				CommerceOrderID: "ord-1",
				CustomerID:      "cust-1",
				Status:          refund.RefundStatusFulfilled,
				CreatedAt:       now,
				UpdatedAt:       now,
			},
		},
	})
	rr := doRequest(t, h.GetRefund(), http.MethodGet, "/refunds/ref-1", nil,
		map[string]string{"refundId": "ref-1"})
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

// ---------------------------------------------------------------------------
// Enterprise tests
// ---------------------------------------------------------------------------

func TestListReceivablePeriods_Success(t *testing.T) {
	h := testHandler(Services{})
	rr := doRequest(t, h.ListReceivablePeriods(), http.MethodGet, "/customers/cust-1/receivable-periods", nil,
		map[string]string{"customerId": "cust-1"})
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var resp api.ReceivablePeriodPaginatedResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Data) != 0 {
		t.Errorf("expected 0 periods, got %d", len(resp.Data))
	}
}

func TestUpdateExternalInvoice_Success(t *testing.T) {
	h := testHandler(Services{})
	issuedAt := api.DateTime(time.Now().UTC())
	body := api.CommerceExternalInvoiceUpdate{
		InvoiceNumber:  "INV-001",
		IdempotencyKey: "idem-inv-1",
		IssuedAt:       &issuedAt,
	}
	rr := doRequest(t, h.UpdateExternalInvoice(), http.MethodPut, "/customers/cust-1/receivable-periods/per-1/external-invoice", body,
		map[string]string{"customerId": "cust-1", "periodId": "per-1"})
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var resp api.CommerceExternalInvoice
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.InvoiceNumber != "INV-001" {
		t.Errorf("expected INV-001, got %s", resp.InvoiceNumber)
	}
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
