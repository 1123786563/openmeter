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
	"github.com/openmeterio/openmeter/openmeter/commerce/payment"
	"github.com/openmeterio/openmeter/openmeter/commerce/refund"
)

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
