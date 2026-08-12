package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/commerce/payment"
	openmeterserver "github.com/openmeterio/openmeter/openmeter/server"
)

type recordingPaidTxRunner struct {
	calls int
}

func (r *recordingPaidTxRunner) RunPaidTransition(context.Context, payment.PaidTransitionInput) (payment.PaidTransitionResult, error) {
	r.calls++
	return payment.PaidTransitionResult{}, nil
}

func TestOneShotPaidTransitionFaultInjector(t *testing.T) {
	inner := &recordingPaidTxRunner{}
	injector := newOneShotPaidTransitionFaultInjector(inner)
	injector.Arm()

	_, err := injector.RunPaidTransition(t.Context(), payment.PaidTransitionInput{})
	require.ErrorIs(t, err, errInjectedPaidTransitionFailure)
	require.Equal(t, 0, inner.calls, "injected failure must happen before the authoritative transaction")

	_, err = injector.RunPaidTransition(t.Context(), payment.PaidTransitionInput{})
	require.NoError(t, err)
	require.Equal(t, 1, inner.calls, "fault must be consumed exactly once")
}

type stubCommerceOracle struct {
	result commerceTestOrderOracle
	err    error
}

func (s stubCommerceOracle) ReadOrder(context.Context, string, string) (commerceTestOrderOracle, error) {
	return s.result, s.err
}

func TestCommerceTestControlsAreAuthenticatedAndNotRegisteredWhenDisabled(t *testing.T) {
	t.Run("disabled", func(t *testing.T) {
		hooks := &openmeterserver.RouterHooks{}
		registerCommerceTestControls(hooks, nil)
		router := chi.NewRouter()
		for _, hook := range hooks.Routes {
			hook(router)
		}

		request := httptest.NewRequest(http.MethodPost, commerceTestFaultPath, nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		require.Equal(t, http.StatusNotFound, response.Code)
	})

	t.Run("enabled requires bearer token", func(t *testing.T) {
		injector := newOneShotPaidTransitionFaultInjector(&recordingPaidTxRunner{})
		control := &commerceTestControl{token: "phase2-loopback-control-token", injector: injector, oracle: stubCommerceOracle{}}
		hooks := &openmeterserver.RouterHooks{}
		registerCommerceTestControls(hooks, control)
		router := chi.NewRouter()
		for _, hook := range hooks.Routes {
			hook(router)
		}

		for _, token := range []string{"", "Bearer wrong-token"} {
			request := httptest.NewRequest(http.MethodPost, commerceTestFaultPath, nil)
			request.Host = "127.0.0.1:8889"
			request.Header.Set("Authorization", token)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			require.Equal(t, http.StatusUnauthorized, response.Code)
		}

		request := httptest.NewRequest(http.MethodPost, commerceTestFaultPath, nil)
		request.Host = "127.0.0.1:8889"
		request.Header.Set("Authorization", "Bearer phase2-loopback-control-token")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		require.Equal(t, http.StatusNoContent, response.Code)

		external := httptest.NewRequest(http.MethodPost, commerceTestFaultPath, nil)
		external.Host = "openmeter.example.com"
		external.Header.Set("Authorization", "Bearer phase2-loopback-control-token")
		externalResponse := httptest.NewRecorder()
		router.ServeHTTP(externalResponse, external)
		require.Equal(t, http.StatusNotFound, externalResponse.Code)
	})
}

func TestCommerceTestOracleHandlerReturnsAuthoritativeCardinalities(t *testing.T) {
	want := commerceTestOrderOracle{
		OrderID:          "01ORDER",
		OrderNumber:      "01PUBLIC",
		OrderStatus:      "fulfilled",
		AttemptStatus:    "succeeded",
		PaymentFactCount: 1,
		FulfillmentCount: 1,
		GrantEffectCount: 1,
		CreditsGranted:   100000,
		ProviderOrderID:  "01PUBLIC",
	}
	control := &commerceTestControl{
		token:    "phase2-loopback-control-token",
		injector: newOneShotPaidTransitionFaultInjector(&recordingPaidTxRunner{}),
		oracle:   stubCommerceOracle{result: want},
	}
	hooks := &openmeterserver.RouterHooks{}
	registerCommerceTestControls(hooks, control)
	router := chi.NewRouter()
	for _, hook := range hooks.Routes {
		hook(router)
	}

	request := httptest.NewRequest(http.MethodGet, "/__test/commerce/oracle/orders/01ORDER?provider_order_id=01PUBLIC", nil)
	request.Host = "127.0.0.1:8889"
	request.Header.Set("Authorization", "Bearer phase2-loopback-control-token")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	require.Equal(t, http.StatusOK, response.Code)
	for _, fragment := range []string{
		`"order_id":"01ORDER"`, `"order_number":"01PUBLIC"`, `"attempt_status":"succeeded"`,
		`"payment_fact_count":1`, `"fulfillment_count":1`, `"grant_effect_count":1`, `"credits_granted":100000`,
	} {
		require.True(t, strings.Contains(response.Body.String(), fragment), response.Body.String())
	}
}

func TestCommerceTestOracleHandlerMapsNotFoundAndInternalErrors(t *testing.T) {
	for _, tt := range []struct {
		name string
		err  error
		want int
	}{
		{name: "not found", err: errCommerceTestOrderNotFound, want: http.StatusNotFound},
		{name: "database error", err: errors.New("database unavailable"), want: http.StatusInternalServerError},
	} {
		t.Run(tt.name, func(t *testing.T) {
			control := &commerceTestControl{token: "phase2-loopback-control-token", oracle: stubCommerceOracle{err: tt.err}}
			hooks := &openmeterserver.RouterHooks{}
			registerCommerceTestControls(hooks, control)
			router := chi.NewRouter()
			for _, hook := range hooks.Routes {
				hook(router)
			}
			request := httptest.NewRequest(http.MethodGet, "/__test/commerce/oracle/orders/01ORDER", nil)
			request.Host = "127.0.0.1:8889"
			request.Header.Set("Authorization", "Bearer phase2-loopback-control-token")
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			require.Equal(t, tt.want, response.Code)
			require.NotContains(t, response.Body.String(), "database unavailable")
		})
	}
}
