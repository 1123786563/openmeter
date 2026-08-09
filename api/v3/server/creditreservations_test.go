package server

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	creditreservationshandler "github.com/openmeterio/openmeter/api/v3/handlers/creditreservations"
	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/creditreservation"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestCreditReservationRoutesReachHandlerWhenEnabled(t *testing.T) {
	service := creditReservationRouteService{
		reserve: func(_ context.Context, input creditreservation.ReserveInput) (creditreservation.Reservation, error) {
			return creditreservation.Reservation{ID: input.ID.ID, CustomerID: input.CustomerID}, nil
		},
	}
	s := Server{
		Config:                    &Config{Credits: config.CreditsConfiguration{Enabled: true, ReservationsEnabled: true}},
		creditReservationsHandler: creditreservationshandler.New(func(context.Context) (string, error) { return "acme", nil }, service),
	}
	r := chi.NewRouter()
	s.registerCreditReservationRoutes(r)

	req := httptest.NewRequest(http.MethodPost, "/credit-reservations", strings.NewReader(`{
		"id":"reserve-1","customer_id":"customer-1","subject_id":"subject-1","client_call_id":"call-1","operation":"completion",
		"idempotency_key":"reserve-key","payload_hash":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"authorization_expires_at":"2026-08-10T12:00:00Z",
		"lines":[{"feature_key":"ai_usage","resource_code":"input_tokens","quantity":1}]
	}`))
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)
}

func TestCreditChargeRouteReachesHandlerWithoutTrailingSlash(t *testing.T) {
	service := creditReservationRouteService{
		charge: func(_ context.Context, input creditreservation.ChargeInput) (creditreservation.Charge, error) {
			return creditreservation.Charge{ID: input.ID.ID, CustomerID: input.CustomerID}, nil
		},
	}
	s := Server{
		Config:                    &Config{Credits: config.CreditsConfiguration{Enabled: true, ReservationsEnabled: true}},
		creditReservationsHandler: creditreservationshandler.New(func(context.Context) (string, error) { return "acme", nil }, service),
	}
	r := chi.NewRouter()
	s.registerCreditReservationRoutes(r)

	req := httptest.NewRequest(http.MethodPost, "/credit-charges", strings.NewReader(`{
		"id":"charge-1","customer_id":"customer-1","subject_id":"subject-1","operation":"completion",
		"idempotency_key":"charge-key","payload_hash":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"booked_at":"2026-08-10T12:00:00Z",
		"lines":[{"feature_key":"ai_usage","resource_code":"input_tokens","quantity":1}]
	}`))
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)
}

func TestCreditReservationRoutesAreAbsentWhenReservationFeatureDisabled(t *testing.T) {
	s := Server{
		Config:                    &Config{Credits: config.CreditsConfiguration{Enabled: true, ReservationsEnabled: false}},
		creditReservationsHandler: creditreservationshandler.New(func(context.Context) (string, error) { return "acme", nil }, creditReservationRouteService{}),
	}
	r := chi.NewRouter()
	s.registerCreditReservationRoutes(r)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/credit-reservations/", nil))

	require.Equal(t, http.StatusNotFound, rec.Code)
}

type creditReservationRouteService struct {
	reserve func(context.Context, creditreservation.ReserveInput) (creditreservation.Reservation, error)
	charge  func(context.Context, creditreservation.ChargeInput) (creditreservation.Charge, error)
}

func (s creditReservationRouteService) Reserve(ctx context.Context, input creditreservation.ReserveInput) (creditreservation.Reservation, error) {
	if s.reserve == nil {
		return creditreservation.Reservation{}, errors.New("unexpected Reserve")
	}
	return s.reserve(ctx, input)
}

func (creditReservationRouteService) Get(context.Context, models.NamespacedID) (creditreservation.Reservation, error) {
	return creditreservation.Reservation{}, errors.New("unexpected Get")
}

func (creditReservationRouteService) Execute(context.Context, creditreservation.ExecuteInput) (creditreservation.Reservation, error) {
	return creditreservation.Reservation{}, errors.New("unexpected Execute")
}

func (creditReservationRouteService) Settle(context.Context, creditreservation.SettleInput) (creditreservation.Reservation, error) {
	return creditreservation.Reservation{}, errors.New("unexpected Settle")
}

func (creditReservationRouteService) Release(context.Context, creditreservation.ReleaseInput) (creditreservation.Reservation, error) {
	return creditreservation.Reservation{}, errors.New("unexpected Release")
}

func (creditReservationRouteService) MarkUnknown(context.Context, creditreservation.UnknownInput) (creditreservation.Reservation, error) {
	return creditreservation.Reservation{}, errors.New("unexpected MarkUnknown")
}

func (s creditReservationRouteService) Charge(ctx context.Context, input creditreservation.ChargeInput) (creditreservation.Charge, error) {
	if s.charge == nil {
		return creditreservation.Charge{}, errors.New("unexpected Charge")
	}
	return s.charge(ctx, input)
}

func (creditReservationRouteService) ReverseCharge(context.Context, creditreservation.ReverseChargeInput) (creditreservation.Charge, error) {
	return creditreservation.Charge{}, errors.New("unexpected ReverseCharge")
}

var _ creditreservationshandler.Service = creditReservationRouteService{}
