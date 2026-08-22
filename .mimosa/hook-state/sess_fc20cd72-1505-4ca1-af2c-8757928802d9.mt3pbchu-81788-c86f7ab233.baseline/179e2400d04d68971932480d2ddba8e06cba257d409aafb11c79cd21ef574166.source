package server

import "net/http"

func (s *Server) creditReservationsEnabled() bool {
	return s.Config != nil &&
		s.Credits.Enabled &&
		s.Credits.ReservationsEnabled &&
		s.CreditReservation.Enabled &&
		s.creditReservationsHandler != nil
}

func (s *Server) CreateCreditReservation(w http.ResponseWriter, r *http.Request) {
	if !s.creditReservationsEnabled() {
		http.NotFound(w, r)
		return
	}

	s.creditReservationsHandler.Reserve().ServeHTTP(w, r)
}

func (s *Server) GetCreditReservation(w http.ResponseWriter, r *http.Request, reservationID string) {
	if !s.creditReservationsEnabled() {
		http.NotFound(w, r)
		return
	}

	r.SetPathValue("reservationId", reservationID)
	s.creditReservationsHandler.Get().ServeHTTP(w, r)
}

func (s *Server) ExecuteCreditReservation(w http.ResponseWriter, r *http.Request, reservationID string) {
	if !s.creditReservationsEnabled() {
		http.NotFound(w, r)
		return
	}

	r.SetPathValue("reservationId", reservationID)
	s.creditReservationsHandler.Execute().ServeHTTP(w, r)
}

func (s *Server) SettleCreditReservation(w http.ResponseWriter, r *http.Request, reservationID string) {
	if !s.creditReservationsEnabled() {
		http.NotFound(w, r)
		return
	}

	r.SetPathValue("reservationId", reservationID)
	s.creditReservationsHandler.Settle().ServeHTTP(w, r)
}

func (s *Server) ReleaseCreditReservation(w http.ResponseWriter, r *http.Request, reservationID string) {
	if !s.creditReservationsEnabled() {
		http.NotFound(w, r)
		return
	}

	r.SetPathValue("reservationId", reservationID)
	s.creditReservationsHandler.Release().ServeHTTP(w, r)
}

func (s *Server) MarkCreditReservationUnknown(w http.ResponseWriter, r *http.Request, reservationID string) {
	if !s.creditReservationsEnabled() {
		http.NotFound(w, r)
		return
	}

	r.SetPathValue("reservationId", reservationID)
	s.creditReservationsHandler.Unknown().ServeHTTP(w, r)
}

func (s *Server) CreateCreditCharge(w http.ResponseWriter, r *http.Request) {
	if !s.creditReservationsEnabled() {
		http.NotFound(w, r)
		return
	}

	s.creditReservationsHandler.Charge().ServeHTTP(w, r)
}

func (s *Server) ReverseCreditCharge(w http.ResponseWriter, r *http.Request, chargeID string) {
	if !s.creditReservationsEnabled() {
		http.NotFound(w, r)
		return
	}

	r.SetPathValue("chargeId", chargeID)
	s.creditReservationsHandler.ReverseCharge().ServeHTTP(w, r)
}
