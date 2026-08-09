// Package creditreservations exposes the credit reservation lifecycle over HTTP.
//
// It deliberately uses net/http handlers until the generated v3 server binding
// contains the corresponding operations. This keeps the domain boundary usable
// and testable without coupling it to a generated artifact that is not present.
package creditreservations

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/openmeterio/openmeter/openmeter/creditreservation"
	"github.com/openmeterio/openmeter/pkg/models"
)

// Service is the portion of creditreservation.Service needed by the HTTP API.
// Keeping the dependency narrow makes the handler easy to test and prevents
// lifecycle sweep operations from leaking into the transport surface.
type Service interface {
	Reserve(context.Context, creditreservation.ReserveInput) (creditreservation.Reservation, error)
	Get(context.Context, models.NamespacedID) (creditreservation.Reservation, error)
	Execute(context.Context, creditreservation.ExecuteInput) (creditreservation.Reservation, error)
	Settle(context.Context, creditreservation.SettleInput) (creditreservation.Reservation, error)
	Release(context.Context, creditreservation.ReleaseInput) (creditreservation.Reservation, error)
	MarkUnknown(context.Context, creditreservation.UnknownInput) (creditreservation.Reservation, error)
	Charge(context.Context, creditreservation.ChargeInput) (creditreservation.Charge, error)
	ReverseCharge(context.Context, creditreservation.ReverseChargeInput) (creditreservation.Charge, error)
}

var _ Service = (creditreservation.Service)(nil)

// Handler is the HTTP surface for the eight reservation and direct-charge
// operations. Route registration supplies reservationId and chargeId through
// net/http request path values.
type Handler interface {
	Reserve() http.HandlerFunc
	Get() http.HandlerFunc
	Execute() http.HandlerFunc
	Settle() http.HandlerFunc
	Release() http.HandlerFunc
	Unknown() http.HandlerFunc
	Charge() http.HandlerFunc
	ReverseCharge() http.HandlerFunc
}

type handler struct {
	resolveNamespace func(context.Context) (string, error)
	service          Service
}

// New constructs a credit reservation HTTP handler.
func New(resolveNamespace func(context.Context) (string, error), service Service) Handler {
	return &handler{
		resolveNamespace: resolveNamespace,
		service:          service,
	}
}

// Reserve handles POST /credit-reservations.
func (h *handler) Reserve() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		namespace, ok := h.namespace(w, r)
		if !ok {
			return
		}
		var body reserveRequest
		if err := decodeJSON(r.Body, &body); err != nil {
			writeError(r.Context(), w, r, err)
			return
		}
		input, err := toReserveInput(namespace, body)
		if err != nil {
			writeError(r.Context(), w, r, err)
			return
		}
		reservation, err := h.service.Reserve(r.Context(), input)
		if err != nil {
			writeError(r.Context(), w, r, err)
			return
		}
		writeJSON(w, http.StatusCreated, toReservationResponse(reservation))
	}
}

// Get handles GET /credit-reservations/{reservationId}.
func (h *handler) Get() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		namespace, ok := h.namespace(w, r)
		if !ok {
			return
		}
		id := r.PathValue("reservationId")
		if err := required("reservation_id", id); err != nil {
			writeError(r.Context(), w, r, err)
			return
		}
		reservation, err := h.service.Get(r.Context(), models.NamespacedID{Namespace: namespace, ID: id})
		if err != nil {
			writeError(r.Context(), w, r, err)
			return
		}
		writeJSON(w, http.StatusOK, toReservationResponse(reservation))
	}
}

// Execute handles POST /credit-reservations/{reservationId}/execute.
func (h *handler) Execute() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		namespace, ok := h.namespace(w, r)
		if !ok {
			return
		}
		var body executeRequest
		if err := decodeJSON(r.Body, &body); err != nil {
			writeError(r.Context(), w, r, err)
			return
		}
		input, err := toExecuteInput(namespace, r.PathValue("reservationId"), body)
		if err != nil {
			writeError(r.Context(), w, r, err)
			return
		}
		reservation, err := h.service.Execute(r.Context(), input)
		if err != nil {
			writeError(r.Context(), w, r, err)
			return
		}
		writeJSON(w, http.StatusOK, toReservationResponse(reservation))
	}
}

// Settle handles POST /credit-reservations/{reservationId}/settle.
func (h *handler) Settle() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		namespace, ok := h.namespace(w, r)
		if !ok {
			return
		}
		var body settleRequest
		if err := decodeJSON(r.Body, &body); err != nil {
			writeError(r.Context(), w, r, err)
			return
		}
		input, err := toSettleInput(namespace, r.PathValue("reservationId"), body)
		if err != nil {
			writeError(r.Context(), w, r, err)
			return
		}
		reservation, err := h.service.Settle(r.Context(), input)
		if err != nil {
			writeError(r.Context(), w, r, err)
			return
		}
		writeJSON(w, http.StatusOK, toReservationResponse(reservation))
	}
}

// Release handles POST /credit-reservations/{reservationId}/release.
func (h *handler) Release() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		namespace, ok := h.namespace(w, r)
		if !ok {
			return
		}
		var body releaseRequest
		if err := decodeJSON(r.Body, &body); err != nil {
			writeError(r.Context(), w, r, err)
			return
		}
		input, err := toReleaseInput(namespace, r.PathValue("reservationId"), body)
		if err != nil {
			writeError(r.Context(), w, r, err)
			return
		}
		reservation, err := h.service.Release(r.Context(), input)
		if err != nil {
			writeError(r.Context(), w, r, err)
			return
		}
		writeJSON(w, http.StatusOK, toReservationResponse(reservation))
	}
}

// Unknown handles POST /credit-reservations/{reservationId}/unknown.
func (h *handler) Unknown() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		namespace, ok := h.namespace(w, r)
		if !ok {
			return
		}
		var body unknownRequest
		if err := decodeJSON(r.Body, &body); err != nil {
			writeError(r.Context(), w, r, err)
			return
		}
		input, err := toUnknownInput(namespace, r.PathValue("reservationId"), body)
		if err != nil {
			writeError(r.Context(), w, r, err)
			return
		}
		reservation, err := h.service.MarkUnknown(r.Context(), input)
		if err != nil {
			writeError(r.Context(), w, r, err)
			return
		}
		writeJSON(w, http.StatusOK, toReservationResponse(reservation))
	}
}

// Charge handles POST /credit-charges.
func (h *handler) Charge() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		namespace, ok := h.namespace(w, r)
		if !ok {
			return
		}
		var body chargeRequest
		if err := decodeJSON(r.Body, &body); err != nil {
			writeError(r.Context(), w, r, err)
			return
		}
		input, err := toChargeInput(namespace, body)
		if err != nil {
			writeError(r.Context(), w, r, err)
			return
		}
		charge, err := h.service.Charge(r.Context(), input)
		if err != nil {
			writeError(r.Context(), w, r, err)
			return
		}
		writeJSON(w, http.StatusCreated, toChargeResponse(charge))
	}
}

// ReverseCharge handles POST /credit-charges/{chargeId}/reverse.
func (h *handler) ReverseCharge() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		namespace, ok := h.namespace(w, r)
		if !ok {
			return
		}
		var body reverseChargeRequest
		if err := decodeJSON(r.Body, &body); err != nil {
			writeError(r.Context(), w, r, err)
			return
		}
		input, err := toReverseChargeInput(namespace, r.PathValue("chargeId"), body)
		if err != nil {
			writeError(r.Context(), w, r, err)
			return
		}
		charge, err := h.service.ReverseCharge(r.Context(), input)
		if err != nil {
			writeError(r.Context(), w, r, err)
			return
		}
		writeJSON(w, http.StatusOK, toChargeResponse(charge))
	}
}

func (h *handler) namespace(w http.ResponseWriter, r *http.Request) (string, bool) {
	namespace, err := h.resolveNamespace(r.Context())
	if err != nil {
		writeError(r.Context(), w, r, err)
		return "", false
	}
	return namespace, true
}

func writeJSON(w http.ResponseWriter, status int, response any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response)
}
