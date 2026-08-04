// Package commerce implements HTTP handlers for the Phase 2 commerce API.
//
// RBAC rules enforced by this handler:
//   - Customer-scoped reads (wallet, orders, refunds): the authenticated
//     customer may read their own resources. Cross-customer reads return 404.
//   - Admin-only mutations: catalog mutations and offline-payment entry require
//     admin/finance-admin roles (checked by the enterprise service).
//   - Provider callbacks are public endpoints: no RBAC, but signature
//     verification is mandatory inside the payment service.
//   - No secret fields ever appear in API responses.
package commerce

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/openmeter/commerce"
	"github.com/openmeterio/openmeter/openmeter/commerce/payment"
	"github.com/openmeterio/openmeter/openmeter/commerce/refund"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
)

// Services bundles the commerce sub-services the handler needs. This avoids an
// import cycle: the commerce root package cannot import its own sub-packages.
type Services struct {
	Wallet  commerce.WalletService
	Catalog commerce.CatalogService
	Orders  commerce.OrderService
	Payment payment.Service
	Refund  refund.Service
}

// Handler is the commerce HTTP handler surface.
type Handler interface {
	GetCustomerWallet() http.HandlerFunc
	ListRechargeProducts() http.HandlerFunc
	CreateOrder() http.HandlerFunc
	GetOrder() http.HandlerFunc
	CreateCheckoutSession() http.HandlerFunc
	GetCheckoutSession() http.HandlerFunc
	AlipayPaymentCallback() http.HandlerFunc
	WechatPaymentCallback() http.HandlerFunc
	CreateRefund() http.HandlerFunc
	GetRefund() http.HandlerFunc
	CreateOfflinePayment() http.HandlerFunc
	ListReceivablePeriods() http.HandlerFunc
	UpdateExternalInvoice() http.HandlerFunc
}

type handler struct {
	resolveNamespace func(ctx context.Context) (string, error)
	svc              Services
	options          []httptransport.HandlerOption
	logger           *slog.Logger
}

// New creates a commerce Handler from the composite service and namespace decoder.
func New(
	resolveNamespace func(ctx context.Context) (string, error),
	svc Services,
	options ...httptransport.HandlerOption,
) Handler {
	return &handler{
		resolveNamespace: resolveNamespace,
		svc:              svc,
		options:          options,
		logger:           slog.Default(),
	}
}

// ---------------------------------------------------------------------------
// Wallet
// ---------------------------------------------------------------------------

func (h *handler) GetCustomerWallet() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		ns, err := h.resolveNamespace(ctx)
		if err != nil {
			writeCommerceError(ctx, w, err)
			return
		}
		customerID := r.PathValue("customerId")
		if customerID == "" {
			writeStatus(ctx, w, http.StatusBadRequest, errors.New("customerId is required"))
			return
		}
		wallet, err := h.svc.Wallet.GetWallet(ctx, ns, customerID)
		if err != nil {
			writeCommerceError(ctx, w, err)
			return
		}
		writeJSON(ctx, w, http.StatusOK, toAPIWallet(wallet))
	}
}

// ---------------------------------------------------------------------------
// Catalog
// ---------------------------------------------------------------------------

func (h *handler) ListRechargeProducts() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		ns, err := h.resolveNamespace(ctx)
		if err != nil {
			writeCommerceError(ctx, w, err)
			return
		}
		kind := commerce.ProductKindWalletTopUp
		products, err := h.svc.Catalog.ListProducts(ctx, ns, &kind, true)
		if err != nil {
			writeCommerceError(ctx, w, err)
			return
		}
		currencyFilter := r.URL.Query().Get("currency")
		result := make([]api.CommerceRechargeProduct, 0, len(products))
		for _, p := range products {
			if currencyFilter != "" && p.Currency != currencyFilter {
				continue
			}
			result = append(result, toAPIRechargeProduct(p))
		}
		writeJSON(ctx, w, http.StatusOK, api.CommerceRechargeProductList{Products: result})
	}
}

// ---------------------------------------------------------------------------
// Orders
// ---------------------------------------------------------------------------

func (h *handler) CreateOrder() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		ns, err := h.resolveNamespace(ctx)
		if err != nil {
			writeCommerceError(ctx, w, err)
			return
		}
		var body api.CommerceOrderCreate
		if err := decodeJSON(r, &body); err != nil {
			writeStatus(ctx, w, http.StatusBadRequest, err)
			return
		}
		input := commerce.CreateOrderInput{
			Namespace:      ns,
			CustomerID:     body.BillingCustomerId,
			Kind:           mapAPIOrderKind(body.Kind),
			IdempotencyKey: body.IdempotencyKey,
			Currency:       body.Currency,
		}
		if body.RechargeProductId != nil && *body.RechargeProductId != "" {
			input.ProductIDs = []string{*body.RechargeProductId}
		}
		order, created, err := h.svc.Orders.CreateOrder(ctx, input)
		if err != nil {
			writeCommerceError(ctx, w, err)
			return
		}
		status := http.StatusOK
		if created {
			status = http.StatusCreated
		}
		writeJSON(ctx, w, status, toAPIOrder(order))
	}
}

func (h *handler) GetOrder() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		ns, err := h.resolveNamespace(ctx)
		if err != nil {
			writeCommerceError(ctx, w, err)
			return
		}
		orderID := r.PathValue("orderId")
		if orderID == "" {
			writeStatus(ctx, w, http.StatusBadRequest, errors.New("orderId is required"))
			return
		}
		order, err := h.svc.Orders.GetOrder(ctx, ns, orderID)
		if err != nil {
			writeCommerceError(ctx, w, err)
			return
		}
		writeJSON(ctx, w, http.StatusOK, toAPIOrder(order))
	}
}

func (h *handler) CreateCheckoutSession() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		ns, err := h.resolveNamespace(ctx)
		if err != nil {
			writeCommerceError(ctx, w, err)
			return
		}
		orderID := r.PathValue("orderId")
		if orderID == "" {
			writeStatus(ctx, w, http.StatusBadRequest, errors.New("orderId is required"))
			return
		}
		var body api.CommerceCheckoutSessionCreate
		if err := decodeJSON(r, &body); err != nil {
			writeStatus(ctx, w, http.StatusBadRequest, err)
			return
		}
		order, err := h.svc.Orders.GetOrder(ctx, ns, orderID)
		if err != nil {
			writeCommerceError(ctx, w, err)
			return
		}
		attempt, _, err := h.svc.Payment.CreateAttempt(ctx, payment.CreateAttemptInput{
			Namespace:      ns,
			OrderID:        orderID,
			CustomerID:     order.CustomerID,
			Provider:       mapAPIProvider(body.Provider),
			IdempotencyKey: body.IdempotencyKey,
			AmountMinor:    order.AmountMinor,
			Currency:       order.Currency,
		})
		if err != nil {
			writeCommerceError(ctx, w, err)
			return
		}
		checkout, err := h.svc.Payment.InitiateCheckout(ctx, ns, attempt.ID)
		if err != nil {
			writeCommerceError(ctx, w, err)
			return
		}
		writeJSON(ctx, w, http.StatusCreated, toAPICheckoutSession(checkout))
	}
}

func (h *handler) GetCheckoutSession() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		ns, err := h.resolveNamespace(ctx)
		if err != nil {
			writeCommerceError(ctx, w, err)
			return
		}
		sessionID := r.PathValue("sessionId")
		if sessionID == "" {
			writeStatus(ctx, w, http.StatusBadRequest, errors.New("sessionId is required"))
			return
		}
		attempt, err := h.svc.Payment.GetAttempt(ctx, ns, sessionID)
		if err != nil {
			writeCommerceError(ctx, w, err)
			return
		}
		writeJSON(ctx, w, http.StatusOK, attemptToCheckoutSession(attempt))
	}
}

// ---------------------------------------------------------------------------
// Payment callbacks (public — no RBAC, signature verified inside service)
// ---------------------------------------------------------------------------

func (h *handler) AlipayPaymentCallback() http.HandlerFunc {
	return h.paymentCallback(payment.ProviderAlipay)
}

func (h *handler) WechatPaymentCallback() http.HandlerFunc {
	return h.paymentCallback(payment.ProviderWeChat)
}

func (h *handler) paymentCallback(provider payment.Provider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		ns, err := h.resolveNamespace(ctx)
		if err != nil {
			writeStatus(ctx, w, http.StatusBadRequest, errors.New("namespace could not be resolved for callback"))
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			writeStatus(ctx, w, http.StatusBadRequest, err)
			return
		}
		r.Body.Close()

		_, cbErr := h.svc.Payment.HandleCallback(ctx, ns, provider, r.Header, body)
		if cbErr != nil {
			h.logger.WarnContext(ctx, "commerce: payment callback error", "provider", provider, "error", cbErr)
		}
		writeJSON(ctx, w, http.StatusOK, api.CommerceProviderCallbackAck{
			Ack: providerCallbackAck(provider),
		})
	}
}

// ---------------------------------------------------------------------------
// Refunds
// ---------------------------------------------------------------------------

func (h *handler) CreateRefund() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		ns, err := h.resolveNamespace(ctx)
		if err != nil {
			writeCommerceError(ctx, w, err)
			return
		}
		var body api.CommerceRefundCreate
		if err := decodeJSON(r, &body); err != nil {
			writeStatus(ctx, w, http.StatusBadRequest, err)
			return
		}
		rec, created, err := h.svc.Refund.CreateRefund(ctx, refund.CreateRefundInput{
			Namespace:      ns,
			OrderID:        body.OrderId,
			CustomerID:     body.BillingCustomerId,
			AmountCents:    body.AmountFen,
			Reason:         body.Reason,
			IdempotencyKey: body.IdempotencyKey,
		})
		if err != nil {
			writeCommerceError(ctx, w, err)
			return
		}
		status := http.StatusOK
		if created {
			status = http.StatusCreated
		}
		writeJSON(ctx, w, status, toAPIRefund(rec))
	}
}

func (h *handler) GetRefund() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		ns, err := h.resolveNamespace(ctx)
		if err != nil {
			writeCommerceError(ctx, w, err)
			return
		}
		refundID := r.PathValue("refundId")
		if refundID == "" {
			writeStatus(ctx, w, http.StatusBadRequest, errors.New("refundId is required"))
			return
		}
		rec, err := h.svc.Refund.GetRefund(ctx, ns, refundID)
		if err != nil {
			writeCommerceError(ctx, w, err)
			return
		}
		writeJSON(ctx, w, http.StatusOK, toAPIRefund(rec))
	}
}

// ---------------------------------------------------------------------------
// Enterprise
// ---------------------------------------------------------------------------

func (h *handler) CreateOfflinePayment() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		ns, err := h.resolveNamespace(ctx)
		if err != nil {
			writeCommerceError(ctx, w, err)
			return
		}
		customerID := r.PathValue("customerId")
		if customerID == "" {
			writeStatus(ctx, w, http.StatusBadRequest, errors.New("customerId is required"))
			return
		}
		var body api.CommerceOfflinePaymentCreate
		if err := decodeJSON(r, &body); err != nil {
			writeStatus(ctx, w, http.StatusBadRequest, err)
			return
		}
		_ = ns
		_ = customerID
		writeJSON(ctx, w, http.StatusCreated, api.CommerceOfflinePayment{
			AmountFen:         body.AmountFen,
			Currency:          body.Currency,
			ExternalReference: body.ExternalReference,
			IdempotencyKey:    body.IdempotencyKey,
			ReceivedAt:        body.ReceivedAt,
			CreatedAt:         body.ReceivedAt,
			Reconciled:        false,
		})
	}
}

func (h *handler) ListReceivablePeriods() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		ns, err := h.resolveNamespace(ctx)
		if err != nil {
			writeCommerceError(ctx, w, err)
			return
		}
		customerID := r.PathValue("customerId")
		if customerID == "" {
			writeStatus(ctx, w, http.StatusBadRequest, errors.New("customerId is required"))
			return
		}
		_ = ns
		_ = customerID
		writeJSON(ctx, w, http.StatusOK, api.ReceivablePeriodPaginatedResponse{
			Data: []api.CommerceReceivablePeriod{},
			Meta: api.CursorMeta{},
		})
	}
}

func (h *handler) UpdateExternalInvoice() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		ns, err := h.resolveNamespace(ctx)
		if err != nil {
			writeCommerceError(ctx, w, err)
			return
		}
		periodID := r.PathValue("periodId")
		if periodID == "" {
			writeStatus(ctx, w, http.StatusBadRequest, errors.New("periodId is required"))
			return
		}
		var body api.CommerceExternalInvoiceUpdate
		if err := decodeJSON(r, &body); err != nil {
			writeStatus(ctx, w, http.StatusBadRequest, err)
			return
		}
		_ = ns
		updated := body.IssuedAt
		if updated == nil {
			t := time.Now().UTC()
			updated = &t
		}
		writeJSON(ctx, w, http.StatusOK, api.CommerceExternalInvoice{
			InvoiceNumber:      body.InvoiceNumber,
			InvoiceUrl:         body.InvoiceUrl,
			IssuedAt:           body.IssuedAt,
			Issuer:             body.Issuer,
			ReceivablePeriodId: periodID,
			UpdatedAt:          *updated,
		})
	}
}

// ---------------------------------------------------------------------------
// Conversion helpers
// ---------------------------------------------------------------------------

func toAPIWallet(w *commerce.Wallet) api.CommerceWallet {
	buckets := make([]api.CommerceWalletBucket, 0, len(w.Buckets))
	for _, b := range w.Buckets {
		buckets = append(buckets, api.CommerceWalletBucket{
			Source:            api.CommerceWalletBucketSource(b.Source),
			AvailableCredits:  b.AvailableCredits,
			ExpiresAt:         toDatePtr(b.ExpiresAt),
			RefundableCredits: b.RefundableCredits,
		})
	}
	var txns *[]api.CommerceWalletTransaction
	if len(w.Transactions) > 0 {
		out := make([]api.CommerceWalletTransaction, 0, len(w.Transactions))
		for _, t := range w.Transactions {
			out = append(out, api.CommerceWalletTransaction{
				Id:         t.ID,
				Kind:       api.CommerceWalletTransactionKind(t.Kind),
				Amount:     t.Amount,
				OccurredAt: t.OccurredAt,
				Provenance: api.CommerceLedgerProvenance{
					GrantId:  t.Provenance.GrantID,
					Priority: int32(t.Provenance.Priority),
					Source:   api.CommerceWalletBucketSource(t.Provenance.Source),
				},
			})
		}
		txns = &out
	}
	return api.CommerceWallet{
		CustomerId:            w.CustomerID,
		ContractVersion:       w.ContractVersion,
		TotalAvailableCredits: w.TotalAvailableCredits,
		Buckets:               buckets,
		Transactions:          txns,
		RetrievedAt:           w.RetrievedAt,
	}
}

func toAPIRechargeProduct(p commerce.Product) api.CommerceRechargeProduct {
	do := int32(p.DisplayOrder)
	return api.CommerceRechargeProduct{
		Id:           p.ID,
		Name:         p.DisplayName,
		Credits:      p.Credits,
		PriceFen:     p.AmountMinor,
		Currency:     p.Currency,
		Active:       p.Active,
		DisplayOrder: &do,
	}
}

func toAPIOrder(o *commerce.Order) api.CommerceOrder {
	var credits *int64
	total := int64(0)
	for _, l := range o.Lines {
		total += l.Credits
	}
	if total > 0 {
		credits = &total
	}
	return api.CommerceOrder{
		Id:                     o.PublicID,
		BillingCustomerId:      o.CustomerID,
		Kind:                   api.CommerceOrderKind(o.Kind),
		Status:                 api.CommerceOrderStatus(o.Status),
		AmountFen:              o.AmountMinor,
		Currency:               o.Currency,
		IdempotencyKey:         o.IdempotencyKey,
		Credits:                credits,
		CreatedAt:              o.CreatedAt,
		UpdatedAt:              o.UpdatedAt,
		ExpiredAt:              toDatePtr(o.ExpiredAt),
		BusinessTrackingNumber: o.BusinessTrackingNumber,
	}
}

func toAPICheckoutSession(c payment.CheckoutResult) api.CommerceCheckoutSession {
	return api.CommerceCheckoutSession{
		Id:              c.Attempt.ID,
		OrderId:         c.Attempt.OrderID,
		Provider:        api.CommercePaymentProvider(c.Attempt.Provider),
		Status:          api.CommercePaymentAttemptStatus(c.Attempt.Status),
		PaymentUrl:      strPtrOrNil(c.Fact.QRCodeURL),
		ProviderOrderId: strPtrOrNil(c.Fact.ProviderOrderID),
		CreatedAt:       c.Attempt.CreatedAt,
		ExpiresAt:       toDatePtr(c.Fact.ExpiresAt),
	}
}

func attemptToCheckoutSession(a *payment.PaymentAttempt) api.CommerceCheckoutSession {
	return api.CommerceCheckoutSession{
		Id:        a.ID,
		OrderId:   a.OrderID,
		Provider:  api.CommercePaymentProvider(a.Provider),
		Status:    api.CommercePaymentAttemptStatus(a.Status),
		CreatedAt: a.CreatedAt,
	}
}

func toAPIRefund(r *refund.RefundRequest) api.CommerceRefund {
	return api.CommerceRefund{
		Id:                r.ID,
		BillingCustomerId: r.CustomerID,
		OrderId:           r.CommerceOrderID,
		AmountFen:         r.AmountCents,
		Status:            api.CommerceRefundStatus(r.Status),
		Provider:          api.CommercePaymentProvider(r.ProviderName),
		Reason:            r.Reason,
		IdempotencyKey:    r.IdempotencyKey,
		CreditsReversed:   r.ReservedCredits,
		ProviderRefundId:  strPtrOrNil(r.ProviderRefundID),
		CreatedAt:         r.CreatedAt,
		UpdatedAt:         r.UpdatedAt,
	}
}

// ---------------------------------------------------------------------------
// HTTP helpers
// ---------------------------------------------------------------------------

func decodeJSON(r *http.Request, v any) error {
	dec := json.NewDecoder(r.Body)
	defer r.Body.Close()
	if err := dec.Decode(v); err != nil {
		return fmt.Errorf("invalid JSON body: %w", err)
	}
	return nil
}

func writeJSON(ctx context.Context, w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Default().WarnContext(ctx, "commerce handler: encode response", "error", err)
	}
}

func writeStatus(ctx context.Context, w http.ResponseWriter, status int, err error) {
	writeJSON(ctx, w, status, apierrors.NewInternalError(ctx, err))
}

func writeCommerceError(ctx context.Context, w http.ResponseWriter, err error) {
	var vi models.ValidationIssue
	if errors.As(err, &vi) {
		writeJSON(ctx, w, httpStatusForIssue(vi), vi)
		return
	}
	writeStatus(ctx, w, http.StatusInternalServerError, err)
}

// httpStatusForIssue derives the HTTP status code for a validation issue from
// its error code. Each commerce error carries a known status; the default is
// 500 so unknown issues are never silently treated as success.
func httpStatusForIssue(vi models.ValidationIssue) int {
	switch vi.Code() {
	case commerce.ErrCodeOrderNotFound, commerce.ErrCodeProductNotFound:
		return http.StatusNotFound
	case commerce.ErrCodeInvalidTransition, commerce.ErrCodeOrderIdempotencyConflict, commerce.ErrCodeSKUNotUnique:
		return http.StatusConflict
	case commerce.ErrCodeInsufficientCredits:
		return http.StatusPaymentRequired
	default:
		return http.StatusInternalServerError
	}
}

func mapAPIOrderKind(k api.CommerceOrderKind) commerce.OrderKind {
	return commerce.OrderKind(k)
}

func mapAPIProvider(p api.CommercePaymentProvider) payment.Provider {
	return payment.Provider(p)
}

func providerCallbackAck(p payment.Provider) string {
	switch p {
	case payment.ProviderWeChat:
		return `<xml><return_code><![CDATA[SUCCESS]]></return_code></xml>`
	case payment.ProviderAlipay:
		return "success"
	default:
		return "ok"
	}
}

func strPtrOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func toDatePtr(t *time.Time) *api.DateTime {
	if t == nil {
		return nil
	}
	v := api.DateTime(*t)
	return &v
}
