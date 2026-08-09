package service

import (
	"time"

	"github.com/openmeterio/openmeter/openmeter/creditreservation"
)

// sweepTransition returns the only transition a time-based sweep may make.
// UNKNOWN is intentionally omitted: elapsed time is never evidence that an
// external provider did not execute a request.
func sweepTransition(state creditreservation.ReservationState, deadline, now time.Time) creditreservation.ReservationState {
	if deadline.IsZero() || deadline.After(now) {
		return ""
	}
	switch state {
	case creditreservation.ReservationStateActive:
		return creditreservation.ReservationStateExpired
	case creditreservation.ReservationStateExecuting:
		return creditreservation.ReservationStateUnknown
	default:
		return ""
	}
}
