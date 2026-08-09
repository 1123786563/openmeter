package creditreservations

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/creditreservation"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestReserveDecodesSnakeCaseRequestAndReturnsCreated(t *testing.T) {
	var received creditreservation.ReserveInput
	h := New(func(context.Context) (string, error) { return "acme", nil }, stubService{
		reserve: func(_ context.Context, input creditreservation.ReserveInput) (creditreservation.Reservation, error) {
			received = input
			return testReservation(), nil
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/credit-reservations", strings.NewReader(`{
		"id":"reserve-1","customer_id":"customer-1","subject_id":"subject-1","client_call_id":"call-1","operation":"completion",
		"idempotency_key":"reserve-key","payload_hash":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"authorization_expires_at":"2026-08-10T12:00:00Z",
		"lines":[{"feature_key":"ai_usage","resource_code":"input_tokens","quantity":42,"provider":"openai","model":"gpt-5"}]
	}`))
	rec := httptest.NewRecorder()

	h.Reserve().ServeHTTP(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)
	require.Equal(t, "acme", received.ID.Namespace)
	require.Equal(t, "reserve-1", received.ID.ID)
	require.Equal(t, "customer-1", received.CustomerID)
	require.Equal(t, "reserve-key", received.CommandIdentity.IdempotencyKey)
	require.Equal(t, int64(42), received.Lines[0].Quantity)
	require.Contains(t, rec.Body.String(), `"customer_id":"customer-1"`)
	require.Contains(t, rec.Body.String(), `"ceiling_credits":42`)
}

func TestExecuteMapsInsufficientFundsToPaymentRequired(t *testing.T) {
	h := New(func(context.Context) (string, error) { return "acme", nil }, stubService{
		execute: func(context.Context, creditreservation.ExecuteInput) (creditreservation.Reservation, error) {
			return creditreservation.Reservation{}, creditreservation.ErrInsufficientFunds
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/credit-reservations/reserve-1/execute", strings.NewReader(`{
		"idempotency_key":"execute-key","payload_hash":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","execution_deadline":"2026-08-10T12:00:00Z"
	}`))
	req.SetPathValue("reservationId", "reserve-1")
	rec := httptest.NewRecorder()

	h.Execute().ServeHTTP(rec, req)

	require.Equal(t, http.StatusPaymentRequired, rec.Code)
}

func TestGetMapsNotFoundToNotFound(t *testing.T) {
	h := New(func(context.Context) (string, error) { return "acme", nil }, stubService{
		get: func(context.Context, models.NamespacedID) (creditreservation.Reservation, error) {
			return creditreservation.Reservation{}, models.NewGenericNotFoundError(errors.New("reservation absent"))
		},
	})
	req := httptest.NewRequest(http.MethodGet, "/credit-reservations/missing", nil)
	req.SetPathValue("reservationId", "missing")
	rec := httptest.NewRecorder()

	h.Get().ServeHTTP(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestReserveMalformedRequestMapsToUnprocessableEntity(t *testing.T) {
	h := New(func(context.Context) (string, error) { return "acme", nil }, stubService{})
	req := httptest.NewRequest(http.MethodPost, "/credit-reservations", strings.NewReader(`{"id":`))
	rec := httptest.NewRecorder()

	h.Reserve().ServeHTTP(rec, req)

	require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

type stubService struct {
	reserve       func(context.Context, creditreservation.ReserveInput) (creditreservation.Reservation, error)
	get           func(context.Context, models.NamespacedID) (creditreservation.Reservation, error)
	execute       func(context.Context, creditreservation.ExecuteInput) (creditreservation.Reservation, error)
	settle        func(context.Context, creditreservation.SettleInput) (creditreservation.Reservation, error)
	release       func(context.Context, creditreservation.ReleaseInput) (creditreservation.Reservation, error)
	unknown       func(context.Context, creditreservation.UnknownInput) (creditreservation.Reservation, error)
	charge        func(context.Context, creditreservation.ChargeInput) (creditreservation.Charge, error)
	reverseCharge func(context.Context, creditreservation.ReverseChargeInput) (creditreservation.Charge, error)
}

func (s stubService) Reserve(ctx context.Context, input creditreservation.ReserveInput) (creditreservation.Reservation, error) {
	if s.reserve == nil {
		return creditreservation.Reservation{}, errors.New("unexpected Reserve")
	}
	return s.reserve(ctx, input)
}

func (s stubService) Get(ctx context.Context, input models.NamespacedID) (creditreservation.Reservation, error) {
	if s.get == nil {
		return creditreservation.Reservation{}, errors.New("unexpected Get")
	}
	return s.get(ctx, input)
}

func (s stubService) Execute(ctx context.Context, input creditreservation.ExecuteInput) (creditreservation.Reservation, error) {
	if s.execute == nil {
		return creditreservation.Reservation{}, errors.New("unexpected Execute")
	}
	return s.execute(ctx, input)
}

func (s stubService) Settle(ctx context.Context, input creditreservation.SettleInput) (creditreservation.Reservation, error) {
	if s.settle == nil {
		return creditreservation.Reservation{}, errors.New("unexpected Settle")
	}
	return s.settle(ctx, input)
}

func (s stubService) Release(ctx context.Context, input creditreservation.ReleaseInput) (creditreservation.Reservation, error) {
	if s.release == nil {
		return creditreservation.Reservation{}, errors.New("unexpected Release")
	}
	return s.release(ctx, input)
}

func (s stubService) MarkUnknown(ctx context.Context, input creditreservation.UnknownInput) (creditreservation.Reservation, error) {
	if s.unknown == nil {
		return creditreservation.Reservation{}, errors.New("unexpected MarkUnknown")
	}
	return s.unknown(ctx, input)
}

func (s stubService) Charge(ctx context.Context, input creditreservation.ChargeInput) (creditreservation.Charge, error) {
	if s.charge == nil {
		return creditreservation.Charge{}, errors.New("unexpected Charge")
	}
	return s.charge(ctx, input)
}

func (s stubService) ReverseCharge(ctx context.Context, input creditreservation.ReverseChargeInput) (creditreservation.Charge, error) {
	if s.reverseCharge == nil {
		return creditreservation.Charge{}, errors.New("unexpected ReverseCharge")
	}
	return s.reverseCharge(ctx, input)
}

func testReservation() creditreservation.Reservation {
	expiresAt := time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC)
	customID := "credit-usd"
	return creditreservation.Reservation{
		ID:              "reserve-1",
		CustomerID:      "customer-1",
		Currency:        currencies.CurrencyReference{Code: currencyx.Code("CREDIT"), CustomCurrencyID: &customID},
		State:           creditreservation.ReservationStateActive,
		RateVersion:     "rate-v1",
		Lines:           []creditreservation.RatedLine{{ResourceLine: creditreservation.ResourceLine{FeatureKey: "ai_usage", ResourceCode: "input_tokens", Quantity: 42}, RateCardKey: "rate-card-1", RateVersion: "rate-v1", Credits: 42}},
		TotalCredits:    42,
		ExpiresAt:       &expiresAt,
		PrepaidHold:     40,
		EnterpriseHold:  2,
		SettledCredits:  0,
		CommandIdentity: creditreservation.CommandIdentity{IdempotencyKey: "reserve-key", PayloadHash: strings.Repeat("a", 64)},
	}
}
