package main

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync/atomic"

	"entgo.io/ent/dialect/sql"
	"github.com/go-chi/chi/v5"

	"github.com/openmeterio/openmeter/openmeter/commerce/payment"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ent/db/commerceorder"
	"github.com/openmeterio/openmeter/openmeter/ent/db/fulfillment"
	dbgrant "github.com/openmeterio/openmeter/openmeter/ent/db/grant"
	"github.com/openmeterio/openmeter/openmeter/ent/db/paymentattempt"
	"github.com/openmeterio/openmeter/openmeter/ent/db/paymentfact"
	openmeterserver "github.com/openmeterio/openmeter/openmeter/server"
)

const (
	commerceTestFaultPath  = "/__test/commerce/fault/inject-db-error-before-paid-transition"
	commerceTestOraclePath = "/__test/commerce/oracle/orders/{orderID}"
)

var (
	errInjectedPaidTransitionFailure = errors.New("test-only injected paid transition failure")
	errCommerceTestOrderNotFound     = errors.New("commerce test order not found")
)

// oneShotPaidTransitionFaultInjector fails before the authoritative Ent
// transaction. Arm is consumed atomically, so a provider retry can run the
// untouched transaction exactly once after the injected 500 response.
type oneShotPaidTransitionFaultInjector struct {
	inner payment.PaidTxRunner
	armed atomic.Bool
}

func newOneShotPaidTransitionFaultInjector(inner payment.PaidTxRunner) *oneShotPaidTransitionFaultInjector {
	return &oneShotPaidTransitionFaultInjector{inner: inner}
}

func (f *oneShotPaidTransitionFaultInjector) Arm() {
	f.armed.Store(true)
}

func (f *oneShotPaidTransitionFaultInjector) RunPaidTransition(ctx context.Context, in payment.PaidTransitionInput) (payment.PaidTransitionResult, error) {
	if f.armed.CompareAndSwap(true, false) {
		return payment.PaidTransitionResult{}, errInjectedPaidTransitionFailure
	}
	return f.inner.RunPaidTransition(ctx, in)
}

type commerceTestOrderOracle struct {
	OrderID             string `json:"order_id"`
	OrderNumber         string `json:"order_number"`
	OrderStatus         string `json:"order_status"`
	PaymentAttemptCount int    `json:"payment_attempt_count"`
	AttemptStatus       string `json:"attempt_status,omitempty"`
	ProviderOrderID     string `json:"provider_order_id,omitempty"`
	PaymentFactCount    int    `json:"payment_fact_count"`
	FulfillmentCount    int    `json:"fulfillment_count"`
	GrantEffectCount    int    `json:"grant_effect_count"`
	CreditsGranted      int64  `json:"credits_granted"`
}

type commerceTestOracleReader interface {
	ReadOrder(ctx context.Context, orderID, providerOrderID string) (commerceTestOrderOracle, error)
}

type entCommerceTestOracle struct {
	client    *entdb.Client
	namespace string
}

func (o entCommerceTestOracle) ReadOrder(ctx context.Context, orderID, providerOrderID string) (commerceTestOrderOracle, error) {
	order, err := o.client.CommerceOrder.Query().
		Where(commerceorder.IDEQ(orderID), commerceorder.NamespaceEQ(o.namespace)).
		Only(ctx)
	if entdb.IsNotFound(err) {
		return commerceTestOrderOracle{}, errCommerceTestOrderNotFound
	}
	if err != nil {
		return commerceTestOrderOracle{}, fmt.Errorf("query commerce order: %w", err)
	}

	attemptQuery := o.client.PaymentAttempt.Query().Where(
		paymentattempt.NamespaceEQ(o.namespace),
		paymentattempt.CommerceOrderIDEQ(orderID),
	)
	if providerOrderID != "" {
		attemptQuery = attemptQuery.Where(paymentattempt.ProviderOrderIDEQ(providerOrderID))
	}
	attempts, err := attemptQuery.Order(paymentattempt.ByCreatedAt(sql.OrderDesc())).All(ctx)
	if err != nil {
		return commerceTestOrderOracle{}, fmt.Errorf("query payment attempts: %w", err)
	}

	factQuery := o.client.PaymentFact.Query().Where(
		paymentfact.NamespaceEQ(o.namespace),
		paymentfact.HasAttemptWith(paymentattempt.CommerceOrderIDEQ(orderID)),
	)
	if providerOrderID != "" {
		factQuery = factQuery.Where(paymentfact.ProviderOrderIDEQ(providerOrderID))
	}
	factCount, err := factQuery.Count(ctx)
	if err != nil {
		return commerceTestOrderOracle{}, fmt.Errorf("count payment facts: %w", err)
	}

	fulfillments, err := o.client.Fulfillment.Query().Where(
		fulfillment.NamespaceEQ(o.namespace),
		fulfillment.CommerceOrderIDEQ(orderID),
	).All(ctx)
	if err != nil {
		return commerceTestOrderOracle{}, fmt.Errorf("query fulfillments: %w", err)
	}
	var creditsGranted int64
	for _, record := range fulfillments {
		creditsGranted += record.CreditsGranted
	}

	grants, err := o.client.Grant.Query().Where(
		dbgrant.NamespaceEQ(o.namespace),
		dbgrant.OwnerIDEQ(order.CustomerID),
	).All(ctx)
	if err != nil {
		return commerceTestOrderOracle{}, fmt.Errorf("query grants: %w", err)
	}
	grantCount := 0
	for _, grant := range grants {
		if grant.Metadata["order_id"] == orderID && grant.Metadata["idempotency_key"] == "fulfillment:"+orderID {
			grantCount++
		}
	}

	result := commerceTestOrderOracle{
		OrderID:             order.ID,
		OrderNumber:         order.PublicID,
		OrderStatus:         string(order.Status),
		PaymentAttemptCount: len(attempts),
		PaymentFactCount:    factCount,
		FulfillmentCount:    len(fulfillments),
		GrantEffectCount:    grantCount,
		CreditsGranted:      creditsGranted,
		ProviderOrderID:     providerOrderID,
	}
	if len(attempts) > 0 {
		result.AttemptStatus = string(attempts[0].Status)
		if attempts[0].ProviderOrderID != nil {
			result.ProviderOrderID = *attempts[0].ProviderOrderID
		}
	}
	return result, nil
}

type commerceTestControl struct {
	token    string
	injector *oneShotPaidTransitionFaultInjector
	oracle   commerceTestOracleReader
}

// registerCommerceTestControls is a no-op unless explicit validated test
// configuration constructed a control. These routes also carry their own
// bearer authentication because route hooks run outside generated API auth.
func registerCommerceTestControls(hooks *openmeterserver.RouterHooks, control *commerceTestControl) {
	if hooks == nil || control == nil {
		return
	}
	hooks.Routes = append(hooks.Routes, func(router openmeterserver.RouteManager) {
		router.Post(commerceTestFaultPath, control.authenticate(control.injectFault))
		router.Get(commerceTestOraclePath, control.authenticate(control.readOracle))
	})
}

func (c *commerceTestControl) authenticate(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		host := r.Host
		if parsedHost, _, err := net.SplitHostPort(r.Host); err == nil {
			host = parsedHost
		}
		if !isCommerceTestRequestHost(host) {
			http.NotFound(w, r)
			return
		}
		// Fail closed when no token is configured: an empty Authorization header
		// compares equal to an empty token, which would authorize any caller.
		if c.token == "" {
			writeCommerceTestJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		provided := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if len(provided) != len(c.token) || subtle.ConstantTimeCompare([]byte(provided), []byte(c.token)) != 1 {
			writeCommerceTestJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		next(w, r)
	}
}

func isCommerceTestRequestHost(host string) bool {
	switch strings.ToLower(strings.Trim(host, "[]")) {
	case "127.0.0.1", "::1", "localhost", "openmeter", "payment-provider":
		return true
	default:
		return false
	}
}

func (c *commerceTestControl) injectFault(w http.ResponseWriter, _ *http.Request) {
	if c.injector == nil {
		writeCommerceTestJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "fault injector unavailable"})
		return
	}
	c.injector.Arm()
	w.WriteHeader(http.StatusNoContent)
}

func (c *commerceTestControl) readOracle(w http.ResponseWriter, r *http.Request) {
	if c.oracle == nil {
		writeCommerceTestJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "oracle unavailable"})
		return
	}
	result, err := c.oracle.ReadOrder(r.Context(), chi.URLParam(r, "orderID"), r.URL.Query().Get("provider_order_id"))
	if errors.Is(err, errCommerceTestOrderNotFound) {
		writeCommerceTestJSON(w, http.StatusNotFound, map[string]string{"error": "order not found"})
		return
	}
	if err != nil {
		writeCommerceTestJSON(w, http.StatusInternalServerError, map[string]string{"error": "oracle query failed"})
		return
	}
	writeCommerceTestJSON(w, http.StatusOK, result)
}

func writeCommerceTestJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
