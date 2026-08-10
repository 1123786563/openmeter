package service

import (
	"time"

	"github.com/openmeterio/openmeter/openmeter/creditreservation"
)

// sweepTransition returns the only transition a time-based sweep may make.
// UNKNOWN is intentionally omitted: elapsed time is never evidence that an
// external provider did not execute a request.
func sweepTransition(state creditreservation.ReservationState, deadline time.Time, unknownManualReviewAfter time.Duration, now time.Time) creditreservation.ReservationState {
	if deadline.IsZero() || deadline.After(now) {
		return ""
	}
	switch state {
	case creditreservation.ReservationStateActive:
		return creditreservation.ReservationStateExpired
	case creditreservation.ReservationStateExecuting:
		return creditreservation.ReservationStateUnknown
	case creditreservation.ReservationStateUnknown:
		if deadline.Add(unknownManualReviewAfter).After(now) {
			return ""
		}
		return creditreservation.ReservationStateManualReview
	default:
		return ""
	}
}
