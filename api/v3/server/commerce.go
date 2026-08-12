package server

import (
	"net/http"

	api "github.com/openmeterio/openmeter/api/v3"
)

// ---------------------------------------------------------------------------
// Commerce routes — implement the v3 ServerInterface methods for the Commerce
// tag. Each method delegates to the commerce handler. If no commerce handler is
// configured (CommerceHandler is nil), the route returns 501 Not Implemented.
// ---------------------------------------------------------------------------

func (s *Server) GetCustomerWallet(w http.ResponseWriter, r *http.Request, customerId api.ULID) {
	if s.commerceHandler == nil {
		w.WriteHeader(http.StatusNotImplemented)
		return
	}
	r.SetPathValue("customerId", customerId)
	s.commerceHandler.GetCustomerWallet().ServeHTTP(w, r)
}

func (s *Server) ListRechargeProducts(w http.ResponseWriter, r *http.Request, params api.ListRechargeProductsParams) {
	if s.commerceHandler == nil {
		w.WriteHeader(http.StatusNotImplemented)
		return
	}
	s.commerceHandler.ListRechargeProducts().ServeHTTP(w, r)
}

func (s *Server) CreateOrder(w http.ResponseWriter, r *http.Request) {
	if s.commerceHandler == nil {
		w.WriteHeader(http.StatusNotImplemented)
		return
	}
	s.commerceHandler.CreateOrder().ServeHTTP(w, r)
}

func (s *Server) GetOrder(w http.ResponseWriter, r *http.Request, orderId api.ULID) {
	if s.commerceHandler == nil {
		w.WriteHeader(http.StatusNotImplemented)
		return
	}
	r.SetPathValue("orderId", orderId)
	s.commerceHandler.GetOrder().ServeHTTP(w, r)
}

func (s *Server) CreateCheckoutSession(w http.ResponseWriter, r *http.Request, orderId api.ULID) {
	if s.commerceHandler == nil {
		w.WriteHeader(http.StatusNotImplemented)
		return
	}
	r.SetPathValue("orderId", orderId)
	s.commerceHandler.CreateCheckoutSession().ServeHTTP(w, r)
}

func (s *Server) GetCheckoutSession(w http.ResponseWriter, r *http.Request, sessionId api.ULID) {
	if s.commerceHandler == nil {
		w.WriteHeader(http.StatusNotImplemented)
		return
	}
	r.SetPathValue("sessionId", sessionId)
	s.commerceHandler.GetCheckoutSession().ServeHTTP(w, r)
}

func (s *Server) AlipayPaymentCallback(w http.ResponseWriter, r *http.Request) {
	if s.commerceHandler == nil {
		w.WriteHeader(http.StatusNotImplemented)
		return
	}
	s.commerceHandler.AlipayPaymentCallback().ServeHTTP(w, r)
}

func (s *Server) WechatPaymentCallback(w http.ResponseWriter, r *http.Request) {
	if s.commerceHandler == nil {
		w.WriteHeader(http.StatusNotImplemented)
		return
	}
	s.commerceHandler.WechatPaymentCallback().ServeHTTP(w, r)
}

func (s *Server) WechatRefundCallback(w http.ResponseWriter, r *http.Request) {
	if s.commerceHandler == nil {
		w.WriteHeader(http.StatusNotImplemented)
		return
	}
	s.commerceHandler.WechatRefundCallback().ServeHTTP(w, r)
}

func (s *Server) CreateRefund(w http.ResponseWriter, r *http.Request) {
	if s.commerceHandler == nil {
		w.WriteHeader(http.StatusNotImplemented)
		return
	}
	s.commerceHandler.CreateRefund().ServeHTTP(w, r)
}

func (s *Server) GetRefund(w http.ResponseWriter, r *http.Request, refundId api.ULID) {
	if s.commerceHandler == nil {
		w.WriteHeader(http.StatusNotImplemented)
		return
	}
	r.SetPathValue("refundId", refundId)
	s.commerceHandler.GetRefund().ServeHTTP(w, r)
}

func (s *Server) CreateOfflinePayment(w http.ResponseWriter, r *http.Request, customerId api.ULID) {
	if s.commerceHandler == nil {
		w.WriteHeader(http.StatusNotImplemented)
		return
	}
	r.SetPathValue("customerId", customerId)
	s.commerceHandler.CreateOfflinePayment().ServeHTTP(w, r)
}

func (s *Server) ListReceivablePeriods(w http.ResponseWriter, r *http.Request, customerId api.ULID, params api.ListReceivablePeriodsParams) {
	if s.commerceHandler == nil {
		w.WriteHeader(http.StatusNotImplemented)
		return
	}
	r.SetPathValue("customerId", customerId)
	s.commerceHandler.ListReceivablePeriods().ServeHTTP(w, r)
}

func (s *Server) UpdateExternalInvoice(w http.ResponseWriter, r *http.Request, customerId api.ULID, periodId api.ULID) {
	if s.commerceHandler == nil {
		w.WriteHeader(http.StatusNotImplemented)
		return
	}
	r.SetPathValue("customerId", customerId)
	r.SetPathValue("periodId", periodId)
	s.commerceHandler.UpdateExternalInvoice().ServeHTTP(w, r)
}
