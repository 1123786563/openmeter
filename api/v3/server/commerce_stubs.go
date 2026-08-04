package server

import (
	"fmt"
	"net/http"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
)

// Commerce Phase 2 stub handlers.
//
// These methods satisfy the generated ServerInterface so the server compiles.
// Each returns 501 Not Implemented until the real commerce domain services are
// built in subsequent tasks.

func notImplemented(w http.ResponseWriter, r *http.Request, what string) {
	apierrors.NewNotImplementedError(
		r.Context(),
		fmt.Errorf("%s not implemented", what),
	).HandleAPIError(w, r)
}

func (s *Server) GetCustomerWallet(w http.ResponseWriter, r *http.Request, customerId api.ULID) {
	notImplemented(w, r, "get customer wallet")
}

func (s *Server) ListRechargeProducts(w http.ResponseWriter, r *http.Request, params api.ListRechargeProductsParams) {
	notImplemented(w, r, "list recharge products")
}

func (s *Server) CreateOrder(w http.ResponseWriter, r *http.Request) {
	notImplemented(w, r, "create order")
}

func (s *Server) GetOrder(w http.ResponseWriter, r *http.Request, orderId api.ULID) {
	notImplemented(w, r, "get order")
}

func (s *Server) CreateCheckoutSession(w http.ResponseWriter, r *http.Request, orderId api.ULID) {
	notImplemented(w, r, "create checkout session")
}

func (s *Server) GetCheckoutSession(w http.ResponseWriter, r *http.Request, sessionId api.ULID) {
	notImplemented(w, r, "get checkout session")
}

func (s *Server) CreateRefund(w http.ResponseWriter, r *http.Request) {
	notImplemented(w, r, "create refund")
}

func (s *Server) GetRefund(w http.ResponseWriter, r *http.Request, refundId api.ULID) {
	notImplemented(w, r, "get refund")
}

func (s *Server) WechatPaymentCallback(w http.ResponseWriter, r *http.Request) {
	notImplemented(w, r, "wechat payment callback")
}

func (s *Server) AlipayPaymentCallback(w http.ResponseWriter, r *http.Request) {
	notImplemented(w, r, "alipay payment callback")
}

func (s *Server) ListReceivablePeriods(w http.ResponseWriter, r *http.Request, customerId api.ULID, params api.ListReceivablePeriodsParams) {
	notImplemented(w, r, "list receivable periods")
}

func (s *Server) CreateOfflinePayment(w http.ResponseWriter, r *http.Request, customerId api.ULID) {
	notImplemented(w, r, "create offline payment")
}

func (s *Server) UpdateExternalInvoice(w http.ResponseWriter, r *http.Request, customerId api.ULID, periodId api.ULID) {
	notImplemented(w, r, "update external invoice")
}
