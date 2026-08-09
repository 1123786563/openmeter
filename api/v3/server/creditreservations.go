package server

import "github.com/go-chi/chi/v5"

// registerCreditReservationRoutes mounts the unstable reservation lifecycle
// outside the generated router until the v3 OpenAPI generation can complete.
// The configuration gate keeps the routes absent (rather than merely returning
// an error) when credit reservations are disabled.
func (s *Server) registerCreditReservationRoutes(r chi.Router) {
	if !s.Credits.Enabled || !s.Credits.ReservationsEnabled || s.creditReservationsHandler == nil {
		return
	}

	r.Route("/credit-reservations", func(r chi.Router) {
		r.Post("/", s.creditReservationsHandler.Reserve())
		r.Get("/{reservationId}", s.creditReservationsHandler.Get())
		r.Post("/{reservationId}/execute", s.creditReservationsHandler.Execute())
		r.Post("/{reservationId}/settle", s.creditReservationsHandler.Settle())
		r.Post("/{reservationId}/release", s.creditReservationsHandler.Release())
		r.Post("/{reservationId}/unknown", s.creditReservationsHandler.Unknown())
	})
	r.Route("/credit-charges", func(r chi.Router) {
		r.Post("/", s.creditReservationsHandler.Charge())
		r.Post("/{chargeId}/reverse", s.creditReservationsHandler.ReverseCharge())
	})
}
