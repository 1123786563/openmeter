package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/openmeterio/openmeter/openmeter/creditreservation"
	reservationadapter "github.com/openmeterio/openmeter/openmeter/creditreservation/adapter"
	"github.com/openmeterio/openmeter/pkg/models"
)

func validateReleaseEvidence(state creditreservation.ReservationState, evidence creditreservation.Evidence) error {
	switch state {
	case creditreservation.ReservationStateActive:
		if evidence.Kind == "" || evidence.Kind == creditreservation.EvidenceNotSent {
			return nil
		}
	case creditreservation.ReservationStateExecuting, creditreservation.ReservationStateUnknown:
		if evidence.Kind == creditreservation.EvidenceProviderConfirmedNotExecuted && strings.TrimSpace(evidence.Reference) != "" {
			return nil
		}
	}
	return fmt.Errorf("%w: release from %s", creditreservation.ErrTransitionEvidenceRequired, state)
}

func (s *service) Execute(ctx context.Context, input creditreservation.ExecuteInput) (creditreservation.Reservation, error) {
	if input.ExecutionDeadline.IsZero() {
		return creditreservation.Reservation{}, fmt.Errorf("execution deadline is required")
	}
	return s.withReservation(ctx, input.ID, func(tx reservationadapter.TxAdapter, current creditreservation.Reservation) (creditreservation.Reservation, error) {
		if err := matchesIdentity(current, input.IdempotencyKey, input.PayloadHash); err != nil {
			return creditreservation.Reservation{}, err
		}
		if current.State == creditreservation.ReservationStateExecuting {
			return current, nil
		}
		return tx.UpdateReservation(ctx, reservationadapter.UpdateReservationInput{ID: input.ID, ExpectedStates: []creditreservation.ReservationState{creditreservation.ReservationStateActive}, State: creditreservation.ReservationStateExecuting, ExecutionDeadline: &input.ExecutionDeadline})
	})
}

func (s *service) Release(ctx context.Context, input creditreservation.ReleaseInput) (creditreservation.Reservation, error) {
	return s.withReservation(ctx, input.ID, func(tx reservationadapter.TxAdapter, current creditreservation.Reservation) (creditreservation.Reservation, error) {
		if err := matchesIdentity(current, input.IdempotencyKey, input.PayloadHash); err != nil {
			return creditreservation.Reservation{}, err
		}
		if current.State == creditreservation.ReservationStateReleased {
			return current, nil
		}
		if err := validateReleaseEvidence(current.State, input.Evidence); err != nil {
			return creditreservation.Reservation{}, err
		}
		return tx.UpdateReservation(ctx, reservationadapter.UpdateReservationInput{ID: input.ID, ExpectedStates: []creditreservation.ReservationState{current.State}, State: creditreservation.ReservationStateReleased, Evidence: creditreservation.TransitionEvidence{Kind: creditreservation.TransitionEvidenceRelease, Reference: input.Evidence.Reference}})
	})
}

func (s *service) MarkUnknown(ctx context.Context, input creditreservation.UnknownInput) (creditreservation.Reservation, error) {
	return s.withReservation(ctx, input.ID, func(tx reservationadapter.TxAdapter, current creditreservation.Reservation) (creditreservation.Reservation, error) {
		if err := matchesIdentity(current, input.IdempotencyKey, input.PayloadHash); err != nil {
			return creditreservation.Reservation{}, err
		}
		if current.State == creditreservation.ReservationStateUnknown {
			return current, nil
		}
		return tx.UpdateReservation(ctx, reservationadapter.UpdateReservationInput{ID: input.ID, ExpectedStates: []creditreservation.ReservationState{creditreservation.ReservationStateActive, creditreservation.ReservationStateExecuting}, State: creditreservation.ReservationStateUnknown})
	})
}

func matchesIdentity(reservation creditreservation.Reservation, idempotencyKey, payloadHash string) error {
	if reservation.CommandIdentity.IdempotencyKey != idempotencyKey || reservation.CommandIdentity.PayloadHash != payloadHash {
		return creditreservation.ErrIdempotencyConflict
	}
	return nil
}

func (s *service) SweepExpired(ctx context.Context, now time.Time, limit int) (creditreservation.SweepResult, error) {
	rows, err := s.adapter.ListExpiredReservations(ctx, now, limit)
	if err != nil {
		return creditreservation.SweepResult{}, err
	}
	result := creditreservation.SweepResult{}
	for _, row := range rows {
		deadline := row.ExpiresAt
		if row.State == creditreservation.ReservationStateExecuting {
			deadline = row.ExecutionDeadline
		}
		target := sweepTransition(row.State, valueOrZero(deadline), now)
		if target == "" {
			continue
		}
		_, err := s.withReservation(ctx, models.NamespacedID{Namespace: row.Namespace, ID: row.ID}, func(tx reservationadapter.TxAdapter, current creditreservation.Reservation) (creditreservation.Reservation, error) {
			currentDeadline := current.ExpiresAt
			if current.State == creditreservation.ReservationStateExecuting {
				currentDeadline = current.ExecutionDeadline
			}
			if sweepTransition(current.State, valueOrZero(currentDeadline), now) != target {
				return current, nil
			}
			return tx.UpdateReservation(ctx, reservationadapter.UpdateReservationInput{ID: models.NamespacedID{Namespace: current.Namespace, ID: current.ID}, ExpectedStates: []creditreservation.ReservationState{current.State}, State: target})
		})
		if err != nil {
			return result, err
		}
		if target == creditreservation.ReservationStateExpired {
			result.Expired++
		} else {
			result.Unknown++
		}
	}
	return result, nil
}

func valueOrZero(value *time.Time) time.Time {
	if value == nil {
		return time.Time{}
	}
	return *value
}
