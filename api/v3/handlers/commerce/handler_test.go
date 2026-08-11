package commerce

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/openmeter/commerce"
	"github.com/openmeterio/openmeter/openmeter/commerce/payment"
	"github.com/openmeterio/openmeter/openmeter/commerce/refund"
	"github.com/openmeterio/openmeter/pkg/models"
)

func ptrULID(value string) *api.ULID {
	id := value
	return &id
}

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
	products               []commerce.Product
	product                *commerce.Product
	productBySKU           *commerce.Product
	lastRequestedProductID string
	lastRequestedSKU       string
	err                    error
}

func (m *mockCatalog) CreateProduct(_ context.Context, _ commerce.CreateProductInput) (*commerce.Product, error) {
	return m.product, m.err
}

func (m *mockCatalog) GetProduct(_ context.Context, _, id string) (*commerce.Product, error) {
	m.lastRequestedProductID = id
	return m.product, m.err
}

func (m *mockCatalog) GetProductBySKU(_ context.Context, _, sku string) (*commerce.Product, error) {
	m.lastRequestedSKU = sku
	if m.productBySKU != nil {
		return m.productBySKU, m.err
	}
	return m.product, m.err
}

func (m *mockCatalog) ListProducts(_ context.Context, _ string, _ *commerce.ProductKind, _ bool) ([]commerce.Product, error) {
	return m.products, m.err
}

func (m *mockCatalog) UpdateProduct(_ context.Context, _ commerce.UpdateProductInput) (*commerce.Product, error) {
	return m.product, m.err
}

type mockOrders struct {
	order     *commerce.Order
	created   bool
	err       error
	conflict  bool
	lastInput commerce.CreateOrderInput
}

func (m *mockOrders) CreateOrder(_ context.Context, input commerce.CreateOrderInput) (*commerce.Order, bool, error) {
	m.lastInput = input
	return m.order, m.created, m.err
}

func (m *mockOrders) GetOrder(_ context.Context, _, _ string) (*commerce.Order, error) {
	return m.order, m.err
}

func (m *mockOrders) TransitionStatus(_ context.Context, _, _ string, _ commerce.OrderStatus) (*commerce.Order, error) {
	return m.order, m.err
}

type mockPayment struct {
	attempt        *payment.PaymentAttempt
	err            error
	callbackCalled bool
	callbackBody   []byte
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

func (m *mockPayment) HandleCallback(_ context.Context, _ string, _ payment.Provider, _ map[string][]string, body []byte) (payment.CallbackResult, error) {
	m.callbackCalled = true
	m.callbackBody = append([]byte(nil), body...)
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
		RechargeProductId: ptrULID("product-recharge-100"),
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
		RechargeProductId: ptrULID("product-recharge-100"),
	}
	rr := doRequest(t, h.CreateOrder(), http.MethodPost, "/orders", body, nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 on idempotent replay, got %d", rr.Code)
	}
}

func TestCreateOrderRouteAcceptsPlanWithoutIDAndResolvesCatalogSKU(t *testing.T) {
	orderService := &mockOrders{order: &commerce.Order{PublicID: "ord-1", CustomerID: "customer-1", AmountMinor: 9900, Currency: "CNY"}, created: true}
	plan := &commerce.Product{NamespacedID: models.NamespacedID{Namespace: "test-ns", ID: "product-plan-pro-monthly"}, SKU: "PLAN-PRO-MONTHLY", Kind: commerce.ProductKindPlanPurchase, AmountMinor: 9900, Currency: "CNY", Active: true}
	catalog := &mockCatalog{productBySKU: plan}
	h := testHandler(Services{Catalog: catalog, Orders: orderService})
	router := http.NewServeMux()
	router.Handle("POST /orders", h.CreateOrder())
	req := httptest.NewRequest(http.MethodPost, "/orders", strings.NewReader(`{
		"billing_customer_id":"customer-1",
		"idempotency_key":"plan-idem-1",
		"kind":"plan_purchase",
		"currency":"CNY",
		"plan":{"plan_key":"pro","plan_version":"monthly"}
	}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
	if catalog.lastRequestedSKU != "PLAN-PRO-MONTHLY" {
		t.Fatalf("catalog SKU = %q, want PLAN-PRO-MONTHLY", catalog.lastRequestedSKU)
	}
	if got := orderService.lastInput.ProductIDs; len(got) != 1 || got[0] != "product-plan-pro-monthly" {
		t.Fatalf("resolved product IDs = %v, want [product-plan-pro-monthly]", got)
	}
	var response api.CommerceOrder
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.AmountFen != 9900 {
		t.Fatalf("response amount_fen = %d, want 9900", response.AmountFen)
	}
}

func TestCreateOrderAcceptsMatchingPlanIDAndSKU(t *testing.T) {
	const planID = "01ARZ3NDEKTSV4RRFFQ69G5FAV"
	plan := &commerce.Product{NamespacedID: models.NamespacedID{Namespace: "test-ns", ID: planID}, SKU: "PLAN-PRO-MONTHLY", Kind: commerce.ProductKindPlanPurchase, AmountMinor: 9900, Currency: "CNY", Active: true}
	catalog := &mockCatalog{product: plan}
	orders := &mockOrders{order: &commerce.Order{PublicID: "ord-1", CustomerID: "customer-1", AmountMinor: 9900}, created: true}
	h := testHandler(Services{Catalog: catalog, Orders: orders})
	body := api.CommerceOrderCreate{
		BillingCustomerId: "customer-1",
		IdempotencyKey:    "plan-idem-matching-id",
		Kind:              api.CommerceOrderKindPlanPurchase,
		Currency:          "CNY",
		Plan:              &api.CommerceOrderCreatePlanRef{PlanId: ptrULID(planID), PlanKey: "pro", PlanVersion: "monthly"},
	}

	rr := doRequest(t, h.CreateOrder(), http.MethodPost, "/orders", body, nil)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
	if catalog.lastRequestedProductID != planID {
		t.Fatalf("catalog product ID = %q, want %q", catalog.lastRequestedProductID, planID)
	}
	if got := orders.lastInput.ProductIDs; len(got) != 1 || got[0] != planID {
		t.Fatalf("resolved product IDs = %v, want [%s]", got, planID)
	}
}

func TestCreateOrderRejectsMismatchingPlanIDAndSKU(t *testing.T) {
	const planID = "01ARZ3NDEKTSV4RRFFQ69G5FAV"
	plan := &commerce.Product{NamespacedID: models.NamespacedID{Namespace: "test-ns", ID: planID}, SKU: "PLAN-TEAM-MONTHLY", Kind: commerce.ProductKindPlanPurchase, AmountMinor: 19900, Currency: "CNY", Active: true}
	catalog := &mockCatalog{product: plan}
	orders := &mockOrders{order: &commerce.Order{PublicID: "ord-1", CustomerID: "customer-1", AmountMinor: 9900}, created: true}
	h := testHandler(Services{Catalog: catalog, Orders: orders})
	body := api.CommerceOrderCreate{
		BillingCustomerId: "customer-1",
		IdempotencyKey:    "plan-idem-mismatching-id",
		Kind:              api.CommerceOrderKindPlanPurchase,
		Currency:          "CNY",
		Plan:              &api.CommerceOrderCreatePlanRef{PlanId: ptrULID(planID), PlanKey: "pro", PlanVersion: "monthly"},
	}

	rr := doRequest(t, h.CreateOrder(), http.MethodPost, "/orders", body, nil)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for mismatching plan_id/SKU, got %d: %s", rr.Code, rr.Body.String())
	}
	requireValidationIssue(t, rr, http.StatusBadRequest, "commerce_invalid_plan_reference", "invalid plan reference")
	if catalog.lastRequestedProductID != planID {
		t.Fatalf("catalog product ID = %q, want %q", catalog.lastRequestedProductID, planID)
	}
	if len(orders.lastInput.ProductIDs) != 0 {
		t.Fatalf("order service received product IDs after plan mismatch: %v", orders.lastInput.ProductIDs)
	}
}

func TestCreateOrderRejectsUnknownPlan(t *testing.T) {
	h := testHandler(Services{Catalog: &mockCatalog{err: commerce.ErrProductNotFound}, Orders: &mockOrders{}})
	body := api.CommerceOrderCreate{BillingCustomerId: "customer-1", IdempotencyKey: "plan-idem-unknown", Kind: api.CommerceOrderKindPlanPurchase, Currency: "CNY", Plan: &api.CommerceOrderCreatePlanRef{PlanKey: "missing", PlanVersion: "monthly"}}
	rr := doRequest(t, h.CreateOrder(), http.MethodPost, "/orders", body, nil)
	requireValidationIssue(t, rr, http.StatusBadRequest, "commerce_invalid_plan_reference", "invalid plan reference")
}

func TestCreateOrderCatalogInfrastructureFailureIsSanitized500(t *testing.T) {
	const internalDetail = "catalog database timeout at postgres://billing-secret"
	h := testHandler(Services{Catalog: &mockCatalog{err: errors.New(internalDetail)}, Orders: &mockOrders{}})
	body := api.CommerceOrderCreate{BillingCustomerId: "customer-1", IdempotencyKey: "plan-idem-infra", Kind: api.CommerceOrderKindPlanPurchase, Currency: "CNY", Plan: &api.CommerceOrderCreatePlanRef{PlanKey: "pro", PlanVersion: "monthly"}}
	rr := doRequest(t, h.CreateOrder(), http.MethodPost, "/orders", body, nil)
	requireProblemResponse(t, rr, http.StatusInternalServerError)
	if strings.Contains(rr.Body.String(), internalDetail) {
		t.Fatalf("500 response exposed catalog failure: %s", rr.Body.String())
	}
}

func TestCreateOrderRejectsMixedProductReference(t *testing.T) {
	h := testHandler(Services{Catalog: &mockCatalog{}, Orders: &mockOrders{}})
	body := api.CommerceOrderCreate{BillingCustomerId: "customer-1", IdempotencyKey: "mixed-idem", Kind: api.CommerceOrderKindPlanPurchase, Currency: "CNY", Plan: &api.CommerceOrderCreatePlanRef{PlanKey: "pro", PlanVersion: "monthly"}, RechargeProductId: ptrULID("recharge-1")}
	rr := doRequest(t, h.CreateOrder(), http.MethodPost, "/orders", body, nil)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for mixed product references, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestCreateOrderRejectsWalletTopUpWithoutProduct(t *testing.T) {
	h := testHandler(Services{Catalog: &mockCatalog{}, Orders: &mockOrders{}})
	body := api.CommerceOrderCreate{BillingCustomerId: "customer-1", IdempotencyKey: "wallet-missing-product", Kind: api.CommerceOrderKindWalletTopUp, Currency: "CNY"}
	rr := doRequest(t, h.CreateOrder(), http.MethodPost, "/orders", body, nil)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for wallet top-up without product, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestCreateOrderRejectsMismatchedPlanProductKind(t *testing.T) {
	product := &commerce.Product{NamespacedID: models.NamespacedID{Namespace: "test-ns", ID: "product-wallet"}, SKU: "PLAN-PRO-MONTHLY", Kind: commerce.ProductKindWalletTopUp, AmountMinor: 1000, Currency: "CNY", Active: true}
	h := testHandler(Services{Catalog: &mockCatalog{productBySKU: product}, Orders: &mockOrders{}})
	body := api.CommerceOrderCreate{BillingCustomerId: "customer-1", IdempotencyKey: "plan-kind-mismatch", Kind: api.CommerceOrderKindPlanPurchase, Currency: "CNY", Plan: &api.CommerceOrderCreatePlanRef{PlanKey: "pro", PlanVersion: "monthly"}}
	rr := doRequest(t, h.CreateOrder(), http.MethodPost, "/orders", body, nil)
	requireValidationIssue(t, rr, http.StatusBadRequest, "commerce_product_not_purchasable", "product cannot be purchased for this order")
}

func TestCreateOrderMapsOrderServiceProductMismatchToSanitized400(t *testing.T) {
	h := testHandler(Services{Orders: &mockOrders{err: fmt.Errorf("%w: hidden wallet/plan mismatch", commerce.ErrProductNotPurchasable)}})
	body := api.CommerceOrderCreate{BillingCustomerId: "customer-1", IdempotencyKey: "wallet-kind-mismatch", Kind: api.CommerceOrderKindWalletTopUp, Currency: "CNY", RechargeProductId: ptrULID("product-plan")}
	rr := doRequest(t, h.CreateOrder(), http.MethodPost, "/orders", body, nil)
	requireValidationIssue(t, rr, http.StatusBadRequest, "commerce_product_not_purchasable", "product cannot be purchased for this order")
	if strings.Contains(rr.Body.String(), "hidden wallet/plan mismatch") {
		t.Fatalf("validation response exposed wrapped detail: %s", rr.Body.String())
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
	requireProblemResponse(t, rr, http.StatusBadRequest)
}

func requireProblemResponse(t *testing.T, rr *httptest.ResponseRecorder, wantStatus int) {
	t.Helper()

	if rr.Code != wantStatus {
		t.Fatalf("expected %d, got %d: %s", wantStatus, rr.Code, rr.Body.String())
	}
	if got := rr.Header().Get("Content-Type"); got != models.ProblemContentType {
		t.Fatalf("expected %q content type, got %q", models.ProblemContentType, got)
	}

	var problem struct {
		Status int    `json:"status"`
		Title  string `json:"title"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&problem); err != nil {
		t.Fatalf("decode problem response: %v", err)
	}
	if problem.Status != wantStatus {
		t.Errorf("expected problem status %d, got %d", wantStatus, problem.Status)
	}
	if want := http.StatusText(wantStatus); problem.Title != want {
		t.Errorf("expected problem title %q, got %q", want, problem.Title)
	}
}

func requireValidationIssue(t *testing.T, rr *httptest.ResponseRecorder, wantStatus int, wantCode, wantMessage string) {
	t.Helper()
	if rr.Code != wantStatus {
		t.Fatalf("expected %d, got %d: %s", wantStatus, rr.Code, rr.Body.String())
	}
	var issue map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&issue); err != nil {
		t.Fatalf("decode validation issue: %v", err)
	}
	if issue["code"] != wantCode || issue["message"] != wantMessage {
		t.Fatalf("validation issue = %#v, want code=%q message=%q", issue, wantCode, wantMessage)
	}
}

func TestWechatCallbackAckIsNoContent(t *testing.T) {
	h := testHandler(Services{
		Payment: &mockPayment{attempt: &payment.PaymentAttempt{}},
	})
	rr := doRequest(t, h.WechatPaymentCallback(), http.MethodPost, "/payment-providers/wechat/callback", "raw-body", nil)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rr.Code, rr.Body.String())
	}
	if got := rr.Body.String(); got != "" {
		t.Fatalf("expected empty body, got %q", got)
	}
	if got := rr.Header().Get("Content-Type"); got != "" {
		t.Fatalf("expected no content type, got %q", got)
	}
}

func TestAlipayCallbackAckIsPlainSuccess(t *testing.T) {
	h := testHandler(Services{
		Payment: &mockPayment{attempt: &payment.PaymentAttempt{}},
	})
	rr := doRequest(t, h.AlipayPaymentCallback(), http.MethodPost, "/payment-providers/alipay/callback", "raw-body", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if got := rr.Header().Get("Content-Type"); got != "text/plain; charset=utf-8" {
		t.Fatalf("expected text/plain content type, got %q", got)
	}
	if got := rr.Body.String(); got != "success" {
		t.Fatalf("expected success body, got %q", got)
	}
}

func TestPaymentCallbackBodyLimit(t *testing.T) {
	t.Run("accepts exactly one MiB", func(t *testing.T) {
		paymentService := &mockPayment{attempt: &payment.PaymentAttempt{}}
		h := testHandler(Services{Payment: paymentService})
		body := strings.Repeat("x", 1<<20)
		req := httptest.NewRequest(http.MethodPost, "/payment-providers/wechat/callback", strings.NewReader(body))
		rr := httptest.NewRecorder()

		h.WechatPaymentCallback().ServeHTTP(rr, req)

		if rr.Code != http.StatusNoContent {
			t.Fatalf("expected 204 at the body limit, got %d: %s", rr.Code, rr.Body.String())
		}
		if !paymentService.callbackCalled {
			t.Fatal("payment service was not called at the body limit")
		}
		if got := len(paymentService.callbackBody); got != 1<<20 {
			t.Fatalf("callback body length = %d, want %d", got, 1<<20)
		}
	})

	t.Run("rejects more than one MiB before verification", func(t *testing.T) {
		paymentService := &mockPayment{attempt: &payment.PaymentAttempt{}}
		h := testHandler(Services{Payment: paymentService})
		req := httptest.NewRequest(http.MethodPost, "/payment-providers/wechat/callback", strings.NewReader(strings.Repeat("x", 1<<20+1)))
		rr := httptest.NewRecorder()

		h.WechatPaymentCallback().ServeHTTP(rr, req)

		requireProblemResponse(t, rr, http.StatusRequestEntityTooLarge)
		if paymentService.callbackCalled {
			t.Fatal("payment service was called with an oversized callback body")
		}
	})
}

func TestPaymentCallbackSignatureRejectionIsBadRequest(t *testing.T) {
	h := testHandler(Services{
		Payment: &mockPayment{err: payment.ErrInvalidSignature},
	})
	rr := doRequest(t, h.AlipayPaymentCallback(), http.MethodPost, "/payment-providers/alipay/callback", "raw-body", nil)
	requireProblemResponse(t, rr, http.StatusBadRequest)
	if got := rr.Body.String(); got == "success" || containsStr(got, "<xml>") {
		t.Fatalf("invalid signature must not receive provider success ACK, got %q", got)
	}
}

func TestPaymentCallbackFactMismatchIsBadRequest(t *testing.T) {
	h := testHandler(Services{
		Payment: &mockPayment{err: payment.ErrPaymentFactMismatch},
	})
	rr := doRequest(t, h.WechatPaymentCallback(), http.MethodPost, "/payment-providers/wechat/callback", "raw-body", nil)
	requireProblemResponse(t, rr, http.StatusBadRequest)
}

func TestPaymentCallbackContradictoryFactIsBadRequest(t *testing.T) {
	h := testHandler(Services{
		Payment: &mockPayment{err: payment.ErrContradictoryPaymentFact},
	})
	rr := doRequest(t, h.WechatPaymentCallback(), http.MethodPost, "/payment-providers/wechat/callback", "raw-body", nil)
	requireProblemResponse(t, rr, http.StatusBadRequest)
}

func TestPaymentCallbackLegalReplayUsesProviderSuccessAck(t *testing.T) {
	tests := []struct {
		name        string
		provider    payment.Provider
		err         error
		wantStatus  int
		wantBody    string
		contentType string
	}{
		{name: "WeChat duplicate event", provider: payment.ProviderWeChat, err: payment.ErrDuplicateProviderEvent, wantStatus: http.StatusNoContent},
		{name: "WeChat already fulfilled", provider: payment.ProviderWeChat, err: payment.ErrFulfillmentAlreadyDone, wantStatus: http.StatusNoContent},
		{name: "Alipay duplicate event", provider: payment.ProviderAlipay, err: payment.ErrDuplicateProviderEvent, wantStatus: http.StatusOK, wantBody: "success", contentType: "text/plain; charset=utf-8"},
		{name: "Alipay already fulfilled", provider: payment.ProviderAlipay, err: payment.ErrFulfillmentAlreadyDone, wantStatus: http.StatusOK, wantBody: "success", contentType: "text/plain; charset=utf-8"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := testHandler(Services{Payment: &mockPayment{err: tt.err}})
			var callback http.Handler
			if tt.provider == payment.ProviderWeChat {
				callback = h.WechatPaymentCallback()
			} else {
				callback = h.AlipayPaymentCallback()
			}
			rr := doRequest(t, callback, http.MethodPost, "/callback", "raw-body", nil)

			if rr.Code != tt.wantStatus {
				t.Fatalf("expected %d on legal replay, got %d: %s", tt.wantStatus, rr.Code, rr.Body.String())
			}
			if got := rr.Body.String(); got != tt.wantBody {
				t.Fatalf("provider ACK body = %q, want %q", got, tt.wantBody)
			}
			if got := rr.Header().Get("Content-Type"); got != tt.contentType {
				t.Fatalf("provider ACK content type = %q, want %q", got, tt.contentType)
			}
		})
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
	issuedAt := time.Now().UTC()
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

// ---------------------------------------------------------------------------
// Idempotency conflict test (Important #5)
// ---------------------------------------------------------------------------

func TestCreateOrder_IdempotencyConflict_409(t *testing.T) {
	h := testHandler(Services{
		Orders: &mockOrders{
			err: commerce.ErrOrderIdempotencyConflict,
		},
	})
	body := api.CommerceOrderCreate{
		BillingCustomerId: "cust-1",
		Kind:              api.CommerceOrderKindWalletTopUp,
		Currency:          "CNY",
		IdempotencyKey:    "idem-conflict",
		RechargeProductId: ptrULID("product-recharge-100"),
	}
	rr := doRequest(t, h.CreateOrder(), http.MethodPost, "/orders", body, nil)
	if rr.Code != http.StatusConflict {
		t.Fatalf("expected 409 for idempotency conflict, got %d: %s", rr.Code, rr.Body.String())
	}
}

// ---------------------------------------------------------------------------
// Catalog mutation tests (Important #6)
// ---------------------------------------------------------------------------

func TestCreateProduct_Success(t *testing.T) {
	h := testHandler(Services{
		Catalog: &mockCatalog{
			product: &commerce.Product{
				DisplayName: "100 Credits",
				SKU:         "SKU-100",
				Kind:        commerce.ProductKindWalletTopUp,
				Credits:     100,
				AmountMinor: 1000,
				Currency:    "CNY",
				Active:      true,
			},
		},
	})
	body := map[string]any{
		"sku":          "SKU-100",
		"display_name": "100 Credits",
		"kind":         "wallet_top_up",
		"credits":      100,
		"amount_fen":   1000,
		"currency":     "CNY",
	}
	rr := doRequest(t, h.CreateProduct(), http.MethodPost, "/recharge-products", body, nil)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
	var prod api.CommerceRechargeProduct
	if err := json.NewDecoder(rr.Body).Decode(&prod); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if prod.PriceFen != 1000 {
		t.Errorf("expected price 1000, got %d", prod.PriceFen)
	}
}

func TestCreateProduct_SKUConflict(t *testing.T) {
	h := testHandler(Services{
		Catalog: &mockCatalog{err: commerce.ErrSKUNotUnique},
	})
	body := map[string]any{
		"sku":          "SKU-DUP",
		"display_name": "Dup",
		"kind":         "wallet_top_up",
		"credits":      50,
		"amount_fen":   500,
		"currency":     "CNY",
	}
	rr := doRequest(t, h.CreateProduct(), http.MethodPost, "/recharge-products", body, nil)
	if rr.Code != http.StatusConflict {
		t.Errorf("expected 409 for duplicate SKU, got %d", rr.Code)
	}
}

func TestUpdateProduct_Success(t *testing.T) {
	active := true
	h := testHandler(Services{
		Catalog: &mockCatalog{
			product: &commerce.Product{
				DisplayName: "Updated Name",
				AmountMinor: 2000,
				Currency:    "CNY",
				Active:      true,
			},
		},
	})
	body := map[string]any{
		"display_name": "Updated Name",
		"amount_fen":   2000,
		"active":       active,
	}
	rr := doRequest(t, h.UpdateProduct(), http.MethodPut, "/recharge-products/prod-1", body,
		map[string]string{"productId": "prod-1"})
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var prod api.CommerceRechargeProduct
	if err := json.NewDecoder(rr.Body).Decode(&prod); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if prod.PriceFen != 2000 {
		t.Errorf("expected price 2000, got %d", prod.PriceFen)
	}
}

func TestUpdateProduct_NotFound(t *testing.T) {
	h := testHandler(Services{
		Catalog: &mockCatalog{err: commerce.ErrProductNotFound},
	})
	body := map[string]any{"display_name": "x"}
	rr := doRequest(t, h.UpdateProduct(), http.MethodPut, "/recharge-products/none", body,
		map[string]string{"productId": "none"})
	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

// ---------------------------------------------------------------------------
// Nil-service 501 tests (Critical #2 verification)
// ---------------------------------------------------------------------------

func TestCreateCheckoutSession_NilPayment_501(t *testing.T) {
	h := testHandler(Services{})
	rr := doRequest(t, h.CreateCheckoutSession(), http.MethodPost, "/orders/ord-1/checkout", nil,
		map[string]string{"orderId": "ord-1"})
	if rr.Code != http.StatusNotImplemented {
		t.Errorf("expected 501 for nil payment service, got %d", rr.Code)
	}
}

func TestCreateRefund_NilRefund_501(t *testing.T) {
	h := testHandler(Services{})
	body := api.CommerceRefundCreate{
		BillingCustomerId: "cust-1",
		OrderId:           "ord-1",
		AmountFen:         500,
		IdempotencyKey:    "idem-r1",
	}
	rr := doRequest(t, h.CreateRefund(), http.MethodPost, "/refunds", body, nil)
	if rr.Code != http.StatusNotImplemented {
		t.Errorf("expected 501 for nil refund service, got %d", rr.Code)
	}
}

func TestAlipayCallback_NilPayment_501(t *testing.T) {
	h := testHandler(Services{})
	rr := doRequest(t, h.AlipayPaymentCallback(), http.MethodPost, "/payment-providers/alipay/callback", "body", nil)
	if rr.Code != http.StatusNotImplemented {
		t.Errorf("expected 501 for nil payment callback, got %d", rr.Code)
	}
}

func TestWechatCallback_NilPayment_501(t *testing.T) {
	h := testHandler(Services{})
	rr := doRequest(t, h.WechatPaymentCallback(), http.MethodPost, "/payment-providers/wechat/callback", "body", nil)
	if rr.Code != http.StatusNotImplemented {
		t.Errorf("expected 501 for nil payment callback, got %d", rr.Code)
	}
}

// ---------------------------------------------------------------------------
// Payment callback error provenance
// ---------------------------------------------------------------------------

func TestPaymentCallbackErrorClassification(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{name: "unknown provider order from production repository", err: commerce.ErrPaymentAttemptNotFound, wantStatus: http.StatusBadRequest},
		{name: "unknown provider order from payment repository", err: payment.ErrPaymentAttemptNotFound, wantStatus: http.StatusBadRequest},
		{name: "paid transition wraps missing order", err: fmt.Errorf("%w: payment: paid transition: %w", payment.ErrRetryableCallback, commerce.ErrOrderNotFound), wantStatus: http.StatusInternalServerError},
		{name: "retryable provenance wins over deterministic sentinel", err: fmt.Errorf("%w: paid transition: %w", payment.ErrRetryableCallback, payment.ErrPaymentAttemptNotFound), wantStatus: http.StatusInternalServerError},
		{name: "paid transition database failure", err: errors.New("paid TxRunner: database connection lost"), wantStatus: http.StatusInternalServerError},
		{name: "wrapped context cancellation", err: fmt.Errorf("paid TxRunner dependency canceled: %w", context.Canceled), wantStatus: http.StatusInternalServerError},
		{name: "wrapped deadline exceeded", err: fmt.Errorf("payment database timed out: %w", context.DeadlineExceeded), wantStatus: http.StatusInternalServerError},
		{name: "non-whitelisted payment validation issue", err: payment.ErrPaymentNotVerified, wantStatus: http.StatusInternalServerError},
		{name: "non-whitelisted commerce validation issue", err: commerce.ErrOrderIdempotencyConflict, wantStatus: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := testHandler(Services{Payment: &mockPayment{err: tt.err}})
			rr := doRequest(t, h.WechatPaymentCallback(), http.MethodPost, "/payment-providers/wechat/callback", "raw-body", nil)
			requireProblemResponse(t, rr, tt.wantStatus)
		})
	}
}

func TestPaymentCallback_SignatureRejection_400(t *testing.T) {
	// Signature verification failures are deterministic client errors.
	h := testHandler(Services{
		Payment: &mockPayment{err: payment.ErrInvalidSignature},
	})
	rr := doRequest(t, h.AlipayPaymentCallback(), http.MethodPost, "/payment-providers/alipay/callback", "raw-body", nil)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 on definitive rejection, got %d: %s", rr.Code, rr.Body.String())
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

// ---------------------------------------------------------------------------
// RBAC ownership check tests (C4)
// ---------------------------------------------------------------------------

// doRequestWithAuth is like doRequest but injects auth context values.
func doRequestWithAuth(t *testing.T, h http.Handler, method, path string, body any, pathValues map[string]string, authCustomerID string, isAdmin bool) *httptest.ResponseRecorder {
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
	ctx := req.Context()
	if authCustomerID != "" || authCustomerID == "" && isAdmin {
		ctx = WithAuthCustomerID(ctx, authCustomerID)
	}
	ctx = WithAuthAdmin(ctx, isAdmin)
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr
}

// TestGetCustomerWallet_OwnershipForbidden verifies that an authenticated
// non-admin customer gets 403 when trying to read another customer's wallet.
func TestGetCustomerWallet_OwnershipForbidden(t *testing.T) {
	h := testHandler(Services{
		Wallet: &mockWallet{wallet: &commerce.Wallet{
			CustomerID: "cust-1",
		}},
	})
	// Authenticated as cust-2, trying to read cust-1's wallet.
	rr := doRequestWithAuth(t, h.GetCustomerWallet(), http.MethodGet,
		"/customers/cust-1/wallet", nil,
		map[string]string{"customerId": "cust-1"},
		"cust-2", false)
	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403 for cross-customer access, got %d: %s", rr.Code, rr.Body.String())
	}
}

// TestGetCustomerWallet_OwnershipAllowed verifies that the authenticated
// customer can read their own wallet.
func TestGetCustomerWallet_OwnershipAllowed(t *testing.T) {
	h := testHandler(Services{
		Wallet: &mockWallet{wallet: &commerce.Wallet{
			CustomerID: "cust-1",
		}},
	})
	rr := doRequestWithAuth(t, h.GetCustomerWallet(), http.MethodGet,
		"/customers/cust-1/wallet", nil,
		map[string]string{"customerId": "cust-1"},
		"cust-1", false)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 for own wallet, got %d", rr.Code)
	}
}

// TestGetCustomerWallet_AdminCanReadAny verifies that an admin can read any
// customer's wallet.
func TestGetCustomerWallet_AdminCanReadAny(t *testing.T) {
	h := testHandler(Services{
		Wallet: &mockWallet{wallet: &commerce.Wallet{
			CustomerID: "cust-1",
		}},
	})
	rr := doRequestWithAuth(t, h.GetCustomerWallet(), http.MethodGet,
		"/customers/cust-1/wallet", nil,
		map[string]string{"customerId": "cust-1"},
		"admin-user", true)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 for admin reading any wallet, got %d", rr.Code)
	}
}

// TestGetOrder_OwnershipForbidden verifies cross-customer order access is denied.
func TestGetOrder_OwnershipForbidden(t *testing.T) {
	h := testHandler(Services{
		Orders: &mockOrders{order: &commerce.Order{
			CustomerID: "cust-owner",
		}},
	})
	rr := doRequestWithAuth(t, h.GetOrder(), http.MethodGet,
		"/orders/ord-1", nil,
		map[string]string{"orderId": "ord-1"},
		"cust-other", false)
	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403 for cross-customer order access, got %d", rr.Code)
	}
}

// TestCreateProduct_NonAdminForbidden verifies that a non-admin cannot create
// catalog products.
func TestCreateProduct_NonAdminForbidden(t *testing.T) {
	h := testHandler(Services{
		Catalog: &mockCatalog{product: &commerce.Product{}},
	})
	body := map[string]any{
		"sku": "SKU-X", "display_name": "X", "kind": "wallet_top_up",
		"credits": 100, "amount_fen": 1000, "currency": "CNY",
	}
	rr := doRequestWithAuth(t, h.CreateProduct(), http.MethodPost,
		"/recharge-products", body, nil,
		"regular-customer", false)
	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403 for non-admin creating product, got %d", rr.Code)
	}
}

// TestCreateProduct_AdminAllowed verifies that an admin can create products.
func TestCreateProduct_AdminAllowed(t *testing.T) {
	h := testHandler(Services{
		Catalog: &mockCatalog{product: &commerce.Product{
			DisplayName: "X", SKU: "SKU-X", Active: true,
		}},
	})
	body := map[string]any{
		"sku": "SKU-X", "display_name": "X", "kind": "wallet_top_up",
		"credits": 100, "amount_fen": 1000, "currency": "CNY",
	}
	rr := doRequestWithAuth(t, h.CreateProduct(), http.MethodPost,
		"/recharge-products", body, nil,
		"admin-user", true)
	if rr.Code != http.StatusCreated {
		t.Errorf("expected 201 for admin creating product, got %d: %s", rr.Code, rr.Body.String())
	}
}

// TestUpdateProduct_NonAdminForbidden verifies that a non-admin cannot update
// catalog products.
func TestUpdateProduct_NonAdminForbidden(t *testing.T) {
	h := testHandler(Services{
		Catalog: &mockCatalog{product: &commerce.Product{}},
	})
	body := map[string]any{"display_name": "updated"}
	rr := doRequestWithAuth(t, h.UpdateProduct(), http.MethodPut,
		"/recharge-products/prod-1", body,
		map[string]string{"productId": "prod-1"},
		"regular-customer", false)
	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403 for non-admin updating product, got %d", rr.Code)
	}
}

// TestGetRefund_OwnershipForbidden verifies cross-customer refund access is denied.
func TestGetRefund_OwnershipForbidden(t *testing.T) {
	now := time.Now().UTC()
	h := testHandler(Services{
		Refund: &mockRefund{rec: &refund.RefundRequest{
			ID: "ref-1", CustomerID: "cust-owner", Status: refund.RefundStatusFulfilled,
			CreatedAt: now, UpdatedAt: now,
		}},
	})
	rr := doRequestWithAuth(t, h.GetRefund(), http.MethodGet,
		"/refunds/ref-1", nil,
		map[string]string{"refundId": "ref-1"},
		"cust-other", false)
	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403 for cross-customer refund access, got %d", rr.Code)
	}
}

// TestRBAC_NoAuthContext_Permissive verifies that when no auth context is set
// (no middleware wired), the handler is permissive (dev / single-tenant mode).
func TestRBAC_NoAuthContext_Permissive(t *testing.T) {
	h := testHandler(Services{
		Wallet: &mockWallet{wallet: &commerce.Wallet{CustomerID: "cust-1"}},
	})
	// No auth context injected — should pass.
	rr := doRequest(t, h.GetCustomerWallet(), http.MethodGet,
		"/customers/cust-1/wallet", nil,
		map[string]string{"customerId": "cust-1"})
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 with no auth context (permissive), got %d", rr.Code)
	}
}
