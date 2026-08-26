package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	api "github.com/openmeterio/openmeter/api/v3"
	commercehandler "github.com/openmeterio/openmeter/api/v3/handlers/commerce"
	"github.com/openmeterio/openmeter/openmeter/commerce"
	"github.com/openmeterio/openmeter/openmeter/commerce/payment"
	"github.com/openmeterio/openmeter/openmeter/commerce/refund"
)

// ordersListProbe counts ListOrders calls reaching the commerce handler.
type ordersListProbe struct {
	commerce.OrderService
	called bool
}

func (s *ordersListProbe) ListOrders(context.Context, commerce.ListOrdersInput) ([]commerce.Order, int, error) {
	s.called = true
	return nil, 0, nil
}

// refundsListProbe counts ListRefunds calls reaching the commerce handler.
type refundsListProbe struct {
	refund.Service
	called bool
}

func (s *refundsListProbe) ListRefunds(context.Context, refund.ListRefundsInput) ([]refund.RefundRequest, int, error) {
	s.called = true
	return nil, 0, nil
}

func TestGeneratedRoutesRegisterCommerceListEndpoints(t *testing.T) {
	orders := &ordersListProbe{}
	refunds := &refundsListProbe{}
	catalog := &catalogMutationProbe{}
	commerceHandler := commercehandler.New(
		func(context.Context) (string, error) { return "test-ns", nil },
		commercehandler.Services{Catalog: catalog, Orders: orders, Refund: refunds},
	)
	server := &Server{commerceHandler: commerceHandler}
	router := chi.NewRouter()
	api.HandlerFromMux(server, router)

	tests := []struct {
		name   string
		method string
		path   string
		want   int
	}{
		{name: "list orders", method: http.MethodGet, path: "/orders?page[number]=1&page[size]=10", want: http.StatusOK},
		{name: "list refunds", method: http.MethodGet, path: "/refunds?customer_id=01ARZ3NDEKTSV4RRFFQ69G5FAV", want: http.StatusOK},
		{name: "create recharge product", method: http.MethodPost, path: "/recharge-products", want: http.StatusCreated},
		{name: "update recharge product", method: http.MethodPatch, path: "/recharge-products/01ARZ3NDEKTSV4RRFFQ69G5FAV", want: http.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body strings.Reader
			if tt.method != http.MethodGet {
				body = *strings.NewReader(`{}`)
			}
			req := httptest.NewRequest(tt.method, tt.path, &body)
			req.Header.Set("Content-Type", "application/json")
			recorder := httptest.NewRecorder()

			router.ServeHTTP(recorder, req)

			if recorder.Code != tt.want {
				t.Fatalf("expected %d, got %d: %s", tt.want, recorder.Code, recorder.Body.String())
			}
		})
	}
	if !orders.called {
		t.Fatal("expected generated /orders route to reach the commerce handler")
	}
	if !refunds.called {
		t.Fatal("expected generated /refunds route to reach the commerce handler")
	}
	if !catalog.createCalled || !catalog.updateCalled {
		t.Fatal("expected generated recharge-product mutation routes to reach the commerce handler")
	}
}

type refundCallbackServiceProbe struct {
	refund.Service
	called   bool
	provider payment.Provider
	body     []byte
}

func (s *refundCallbackServiceProbe) HandleCallback(
	_ context.Context,
	_ string,
	provider payment.Provider,
	_ http.Header,
	body []byte,
) (*refund.RefundRequest, error) {
	s.called = true
	s.provider = provider
	s.body = append([]byte(nil), body...)
	return &refund.RefundRequest{}, nil
}

func TestGeneratedRouteRegistersWechatRefundCallback(t *testing.T) {
	refundService := &refundCallbackServiceProbe{}
	commerceHandler := commercehandler.New(
		func(context.Context) (string, error) { return "test-ns", nil },
		commercehandler.Services{Refund: refundService},
	)
	server := &Server{commerceHandler: commerceHandler}
	router := chi.NewRouter()
	api.HandlerFromMux(server, router)

	req := httptest.NewRequest(
		http.MethodPost,
		"/payment-providers/wechat/refund-callback",
		strings.NewReader(`{"event_type":"REFUND.SUCCESS"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected generated refund callback route to return 204, got %d", recorder.Code)
	}
	if !refundService.called {
		t.Fatal("expected generated refund callback route to reach the real commerce handler")
	}
	if refundService.provider != payment.ProviderWeChat {
		t.Fatalf("refund callback provider = %q, want %q", refundService.provider, payment.ProviderWeChat)
	}
	if got := string(refundService.body); got != `{"event_type":"REFUND.SUCCESS"}` {
		t.Fatalf("refund callback body = %q", got)
	}
}

// catalogMutationProbe counts catalog mutations reaching the commerce handler.
type catalogMutationProbe struct {
	commerce.CatalogService
	createCalled bool
	updateCalled bool
}

func (s *catalogMutationProbe) CreateProduct(context.Context, commerce.CreateProductInput) (*commerce.Product, error) {
	s.createCalled = true
	return &commerce.Product{DisplayName: "probe", Active: true}, nil
}

func (s *catalogMutationProbe) UpdateProduct(context.Context, commerce.UpdateProductInput) (*commerce.Product, error) {
	s.updateCalled = true
	return &commerce.Product{DisplayName: "probe", Active: true}, nil
}
