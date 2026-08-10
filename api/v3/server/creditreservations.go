package server

import "github.com/go-chi/chi/v5"

// registerCreditReservationRoutes mounts the unstable reservation lifecycle
// outside the generated router until the v3 OpenAPI generation can complete.
// The configuration gate keeps the routes absent (rather than merely returning
// an error) when credit reservations are disabled.
func (s *Server) registerCreditReservationRoutes(r chi.Router) {
	if !s.Credits.Enabled || !s.Credits.ReservationsEnabled || !s.CreditReservation.Enabled || s.creditReservationsHandler == nil {
		return
	}

	// Register routes without trailing slashes to match SDK client paths.
	r.Post("/credit-reservations", s.creditReservationsHandler.Reserve())
	r.Get("/credit-reservations/{reservationId}", s.creditReservationsHandler.Get())
	r.Post("/credit-reservations/{reservationId}/execute", s.creditReservationsHandler.Execute())
	r.Post("/credit-reservations/{reservationId}/settle", s.creditReservationsHandler.Settle())
	r.Post("/credit-reservations/{reservationId}/release", s.creditReservationsHandler.Release())
	r.Post("/credit-reservations/{reservationId}/unknown", s.creditReservationsHandler.Unknown())
	r.Post("/credit-charges", s.creditReservationsHandler.Charge())
	r.Post("/credit-charges/{chargeId}/reverse", s.creditReservationsHandler.ReverseCharge())
}
