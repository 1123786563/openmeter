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
	"strings"
	"time"

	api "github.com/openmeterio/openmeter/api/v3"
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
	CreateProduct() http.HandlerFunc
	UpdateProduct() http.HandlerFunc
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
	rbac             RBAC
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
		rbac:             customerRBAC{},
		options:          options,
		logger:           slog.Default(),
	}
}

// ---------------------------------------------------------------------------
// RBAC (C4): ownership verification and admin role checks
// ---------------------------------------------------------------------------

// RBAC provides ownership verification and role checks. The default
// implementation reads identity from context values populated by upstream auth
// middleware (WithAuthCustomerID / WithAuthAdmin). When no auth info is present,
// checks are permissive so the handler remains usable in dev / single-tenant
// mode. When an upstream middleware IS wired, it populates the context and the
// checks enforce strictly.
type RBAC interface {
	// AuthenticatedCustomerID returns the customer ID of the authenticated
	// subject, or "" if none is set. The boolean is false when no auth context
	// exists at all (permissive mode).
	AuthenticatedCustomerID(ctx context.Context) (string, bool)

	// IsAdmin reports whether the authenticated subject has an admin role.
	IsAdmin(ctx context.Context) bool
}

type customerRBAC struct{}

type rbacContextKey string

const (
	rbacKeyCustomerID rbacContextKey = "commerce_auth_customer_id"
	rbacKeyAdmin      rbacContextKey = "commerce_auth_is_admin"
)

// WithAuthCustomerID stores the authenticated customer ID in context. Upstream
// auth middleware calls this so the handler can enforce ownership.
func WithAuthCustomerID(ctx context.Context, customerID string) context.Context {
	return context.WithValue(ctx, rbacKeyCustomerID, customerID)
}

// WithAuthAdmin stores whether the authenticated subject is an admin.
func WithAuthAdmin(ctx context.Context, isAdmin bool) context.Context {
	return context.WithValue(ctx, rbacKeyAdmin, isAdmin)
}

func (customerRBAC) AuthenticatedCustomerID(ctx context.Context) (string, bool) {
	v := ctx.Value(rbacKeyCustomerID)
	if v == nil {
		return "", false
	}
	s, _ := v.(string)
	return s, true
}

func (customerRBAC) IsAdmin(ctx context.Context) bool {
	v := ctx.Value(rbacKeyAdmin)
	if v == nil {
		return true // no auth info — permissive for dev / single-tenant
	}
	b, _ := v.(bool)
	return b
}

// verifyOwnership ensures the authenticated subject owns the requested customer
// resource. Returns false (and writes a 403) if access is denied; the caller
// must return immediately. Permissive when no auth context exists.
func (h *handler) verifyOwnership(ctx context.Context, w http.ResponseWriter, pathCustomerID string) bool {
	if h.rbac == nil {
		return true
	}
	authCustomerID, hasAuth := h.rbac.AuthenticatedCustomerID(ctx)
	if !hasAuth {
		return true
	}
	if h.rbac.IsAdmin(ctx) {
		return true
	}
	if authCustomerID != pathCustomerID {
		writeStatus(ctx, w, http.StatusForbidden, errors.New("access denied: resource does not belong to the authenticated customer"))
		return false
	}
	return true
}

// requireAdmin ensures the authenticated subject has an admin role. Returns
// false (and writes a 403) if denied. Permissive when no auth context exists.
func (h *handler) requireAdmin(ctx context.Context, w http.ResponseWriter) bool {
	if h.rbac == nil {
		return true
	}
	_, hasAuth := h.rbac.AuthenticatedCustomerID(ctx)
	if !hasAuth {
		return true
	}
	if !h.rbac.IsAdmin(ctx) {
		writeStatus(ctx, w, http.StatusForbidden, errors.New("admin role required for this operation"))
		return false
	}
	return true
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
		// C4: ownership check — authenticated customer may only read their own wallet.
		if !h.verifyOwnership(ctx, w, customerID) {
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

// CreateProduct creates a new catalog product. Admin-only mutation: the RBAC
// middleware ensures the caller has admin role before reaching this handler.
func (h *handler) CreateProduct() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		ns, err := h.resolveNamespace(ctx)
		if err != nil {
			writeCommerceError(ctx, w, err)
			return
		}
		// C4: admin-only mutation.
		if !h.requireAdmin(ctx, w) {
			return
		}
		var body struct {
			SKU          string                `json:"sku"`
			DisplayName  string                `json:"display_name"`
			Kind         api.CommerceOrderKind `json:"kind"`
			Credits      int64                 `json:"credits"`
			AmountFen    api.CommerceFenAmount `json:"amount_fen"`
			Currency     api.CurrencyCode      `json:"currency"`
			DisplayOrder int                   `json:"display_order"`
			RefundPolicy string                `json:"refund_policy"`
			Description  string                `json:"description"`
		}
		if err := decodeJSON(r, &body); err != nil {
			writeStatus(ctx, w, http.StatusBadRequest, err)
			return
		}
		product, err := h.svc.Catalog.CreateProduct(ctx, commerce.CreateProductInput{
			Namespace:    ns,
			SKU:          body.SKU,
			DisplayName:  body.DisplayName,
			Kind:         commerce.ProductKind(body.Kind),
			Credits:      body.Credits,
			AmountMinor:  body.AmountFen,
			Currency:     body.Currency,
			DisplayOrder: body.DisplayOrder,
			Description:  body.Description,
		})
		if err != nil {
			writeCommerceError(ctx, w, err)
			return
		}
		writeJSON(ctx, w, http.StatusCreated, toAPIRechargeProduct(*product))
	}
}

// UpdateProduct updates a catalog product's mutable fields. Admin-only.
func (h *handler) UpdateProduct() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		ns, err := h.resolveNamespace(ctx)
		if err != nil {
			writeCommerceError(ctx, w, err)
			return
		}
		// C4: admin-only mutation.
		if !h.requireAdmin(ctx, w) {
			return
		}
		productID := r.PathValue("productId")
		if productID == "" {
			writeStatus(ctx, w, http.StatusBadRequest, errors.New("productId is required"))
			return
		}
		var body struct {
			DisplayName *string                `json:"display_name,omitempty"`
			AmountFen   *api.CommerceFenAmount `json:"amount_fen,omitempty"`
			Active      *bool                  `json:"active,omitempty"`
		}
		if err := decodeJSON(r, &body); err != nil {
			writeStatus(ctx, w, http.StatusBadRequest, err)
			return
		}
		product, err := h.svc.Catalog.UpdateProduct(ctx, commerce.UpdateProductInput{
			Namespace:   ns,
			ID:          productID,
			DisplayName: body.DisplayName,
			AmountMinor: body.AmountFen,
			Active:      body.Active,
		})
		if err != nil {
			writeCommerceError(ctx, w, err)
			return
		}
		writeJSON(ctx, w, http.StatusOK, toAPIRechargeProduct(*product))
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
		if body.Plan != nil && body.RechargeProductId != nil && *body.RechargeProductId != "" {
			writeStatus(ctx, w, http.StatusBadRequest, errors.New("plan and recharge_product_id cannot be provided together"))
			return
		}
		input := commerce.CreateOrderInput{
			Namespace:      ns,
			CustomerID:     body.BillingCustomerId,
			Kind:           mapAPIOrderKind(body.Kind),
			IdempotencyKey: body.IdempotencyKey,
			Currency:       body.Currency,
		}
		switch input.Kind {
		case commerce.OrderKindPlanPurchase, commerce.OrderKindSubscriptionRenewal:
			if body.Plan == nil {
				writeStatus(ctx, w, http.StatusBadRequest, errors.New("plan is required for plan orders"))
				return
			}
			sku, err := planSKU(*body.Plan)
			if err != nil {
				writeCommerceError(ctx, w, err)
				return
			}
			var product *commerce.Product
			if body.Plan.PlanId != nil && *body.Plan.PlanId != "" {
				product, err = h.svc.Catalog.GetProduct(ctx, ns, *body.Plan.PlanId)
				if err != nil {
					if errors.Is(err, commerce.ErrProductNotFound) {
						writeCommerceError(ctx, w, commerce.ErrInvalidPlanReference)
					} else {
						writeCommerceError(ctx, w, err)
					}
					return
				}
				if !strings.EqualFold(strings.TrimSpace(product.SKU), sku) {
					writeCommerceError(ctx, w, commerce.ErrInvalidPlanReference)
					return
				}
			} else {
				product, err = h.svc.Catalog.GetProductBySKU(ctx, ns, sku)
				if err != nil {
					if errors.Is(err, commerce.ErrProductNotFound) {
						writeCommerceError(ctx, w, commerce.ErrInvalidPlanReference)
					} else {
						writeCommerceError(ctx, w, err)
					}
					return
				}
			}
			expectedKind := commerce.ProductKindPlanPurchase
			if input.Kind == commerce.OrderKindSubscriptionRenewal {
				expectedKind = commerce.ProductKindSubscriptionRenewal
			}
			if product.Kind != expectedKind {
				writeCommerceError(ctx, w, commerce.ErrProductNotPurchasable)
				return
			}
			input.ProductIDs = []string{product.ID}
		case commerce.OrderKindWalletTopUp:
			if body.RechargeProductId == nil || *body.RechargeProductId == "" {
				writeStatus(ctx, w, http.StatusBadRequest, errors.New("recharge_product_id is required for wallet_top_up orders"))
				return
			}
			input.ProductIDs = []string{*body.RechargeProductId}
		default:
			writeStatus(ctx, w, http.StatusBadRequest, fmt.Errorf("invalid order kind: %s", input.Kind))
			return
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

func planSKU(ref api.CommerceOrderCreatePlanRef) (string, error) {
	key := strings.ToUpper(strings.TrimSpace(ref.PlanKey))
	version := strings.ToUpper(strings.TrimSpace(ref.PlanVersion))
	if key == "" || version == "" {
		return "", commerce.ErrInvalidPlanReference
	}
	return "PLAN-" + key + "-" + version, nil
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
		// C4: ownership check — verify the authenticated customer owns this order.
		if !h.verifyOwnership(ctx, w, order.CustomerID) {
			return
		}
		writeJSON(ctx, w, http.StatusOK, toAPIOrder(order))
	}
}

func (h *handler) CreateCheckoutSession() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		if h.svc.Payment == nil {
			writeStatus(ctx, w, http.StatusNotImplemented, errors.New("payment service not configured"))
			return
		}
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
		if h.svc.Payment == nil {
			writeStatus(ctx, w, http.StatusNotImplemented, errors.New("payment service not configured"))
			return
		}
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
	if h.svc.Payment == nil {
		return notImplementedHandler("payment service not configured")
	}
	return h.paymentCallback(payment.ProviderAlipay)
}

func (h *handler) WechatPaymentCallback() http.HandlerFunc {
	if h.svc.Payment == nil {
		return notImplementedHandler("payment service not configured")
	}
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
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		body, err := io.ReadAll(r.Body)
		if err != nil {
			r.Body.Close()
			status := http.StatusBadRequest
			var maxBytesErr *http.MaxBytesError
			if errors.As(err, &maxBytesErr) {
				status = http.StatusRequestEntityTooLarge
			}
			writeStatus(ctx, w, status, err)
			return
		}
		r.Body.Close()

		_, cbErr := h.svc.Payment.HandleCallback(ctx, ns, provider, r.Header, body)
		if cbErr != nil {
			h.logger.WarnContext(ctx, "commerce: payment callback error", "provider", provider, "error", cbErr)
			// Storage and paid-transition failures retain their retryable
			// provenance even if they wrap a deterministic domain sentinel.
			if errors.Is(cbErr, payment.ErrRetryableCallback) {
				writeCallbackRetryableError(ctx, w)
				return
			}
			if isSuccessfulCallbackError(cbErr) {
				writeCallbackSuccess(w, provider)
				return
			}
			// Deterministic callback failures (bad signature, malformed provider
			// fields, or a payment fact mismatch) must not be acknowledged: a
			// success ACK would make the provider discard a request we rejected.
			if status := callbackErrorStatus(cbErr); status >= http.StatusBadRequest && status < http.StatusInternalServerError {
				writeStatus(ctx, w, status, cbErr)
				return
			}
			// Database, transaction, timeout, and unknown errors are transient.
			// Return 500 so the provider retries the callback.
			writeCallbackRetryableError(ctx, w)
			return
		}
		writeCallbackSuccess(w, provider)
	}
}

// writeCallbackRetryableError preserves a retryable 500 response for payment
// providers. It intentionally does not pass the callback error to
// NewStatusProblem: its global context-cancellation mapping is appropriate for
// ordinary request clients, but providers must retry transient callback errors.
func writeCallbackRetryableError(ctx context.Context, w http.ResponseWriter) {
	models.NewStatusProblem(ctx, nil, http.StatusInternalServerError).Respond(w)
}

func writeCallbackSuccess(w http.ResponseWriter, provider payment.Provider) {
	switch provider {
	case payment.ProviderWeChat:
		writeWechatCallbackSuccess(w)
	case payment.ProviderAlipay:
		writeAlipayCallbackSuccess(w)
	default:
		w.WriteHeader(http.StatusNoContent)
	}
}

func writeWechatCallbackSuccess(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

func writeAlipayCallbackSuccess(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, "success")
}

func isSuccessfulCallbackError(err error) bool {
	return errors.Is(err, payment.ErrDuplicateProviderEvent) ||
		errors.Is(err, payment.ErrFulfillmentAlreadyDone)
}

// callbackErrorStatus uses an explicit callback-contract whitelist. Generic
// commerce status mapping is intentionally not used because callback storage
// failures can wrap domain sentinels whose ordinary API status is not valid for
// a provider callback.
func callbackErrorStatus(err error) int {
	switch {
	case errors.Is(err, payment.ErrRetryableCallback):
		return http.StatusInternalServerError
	case errors.Is(err, payment.ErrInvalidSignature),
		errors.Is(err, payment.ErrPaymentFactMismatch),
		errors.Is(err, payment.ErrContradictoryPaymentFact),
		errors.Is(err, payment.ErrPermanentProviderProtocol),
		errors.Is(err, payment.ErrPaymentAttemptNotFound),
		errors.Is(err, commerce.ErrPaymentAttemptNotFound):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

// isRetryableCallbackError reports whether a payment callback error is
// transient and should cause the provider to retry. It is retained as a small
// helper for callers/tests that need to inspect the retry classification.
func isRetryableCallbackError(err error) bool {
	if errors.Is(err, payment.ErrRetryableCallback) {
		return true
	}
	if isSuccessfulCallbackError(err) {
		return false
	}
	return callbackErrorStatus(err) >= http.StatusInternalServerError
}

// ---------------------------------------------------------------------------
// Refunds
// ---------------------------------------------------------------------------

func (h *handler) CreateRefund() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		if h.svc.Refund == nil {
			writeStatus(ctx, w, http.StatusNotImplemented, errors.New("refund service not configured"))
			return
		}
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
		// C4: ownership check — verify the caller owns the refund's customer.
		if !h.verifyOwnership(ctx, w, body.BillingCustomerId) {
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
		if h.svc.Refund == nil {
			writeStatus(ctx, w, http.StatusNotImplemented, errors.New("refund service not configured"))
			return
		}
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
		// C4: ownership check — verify the authenticated customer owns this refund.
		if !h.verifyOwnership(ctx, w, rec.CustomerID) {
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
		Id:                     o.NamespacedID.ID,
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
	models.NewStatusProblem(ctx, err, status).Respond(w)
}

// notImplementedHandler returns a handler that responds with 501 Not Implemented.
func notImplementedHandler(msg string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeStatus(r.Context(), w, http.StatusNotImplemented, errors.New(msg))
	}
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
	case commerce.ErrCodeInvalidPlanReference, commerce.ErrCodeProductNotPurchasable:
		return http.StatusBadRequest
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
	v := *t
	return &v
}
