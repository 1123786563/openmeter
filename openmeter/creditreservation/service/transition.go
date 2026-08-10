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

func (s *service) Execute(ctx context.Context, input creditreservation.ExecuteInput) (reservation creditreservation.Reservation, err error) {
	defer func() { s.metrics.commandOutcome(ctx, "execute", err) }()
	if input.ExecutionDeadline.IsZero() {
		input.ExecutionDeadline = s.now().Add(s.executionDeadline)
	}
	return s.withReservation(ctx, input.ID, func(tx reservationadapter.TxAdapter, current creditreservation.Reservation) (creditreservation.Reservation, error) {
		if _, err := tx.EnsureLifecycleCommand(ctx, input.ID, "execute", creditreservation.CommandIdentity{IdempotencyKey: input.IdempotencyKey, PayloadHash: input.PayloadHash}); err != nil {
			return creditreservation.Reservation{}, err
		}
		if current.State == creditreservation.ReservationStateExecuting {
			return current, nil
		}
		return tx.UpdateReservation(ctx, reservationadapter.UpdateReservationInput{ID: input.ID, ExpectedStates: []creditreservation.ReservationState{creditreservation.ReservationStateActive}, State: creditreservation.ReservationStateExecuting, ExecutionDeadline: &input.ExecutionDeadline})
	})
}

func (s *service) Release(ctx context.Context, input creditreservation.ReleaseInput) (reservation creditreservation.Reservation, err error) {
	defer func() { s.metrics.commandOutcome(ctx, "release", err) }()
	return s.withReservation(ctx, input.ID, func(tx reservationadapter.TxAdapter, current creditreservation.Reservation) (creditreservation.Reservation, error) {
		if _, err := tx.EnsureLifecycleCommand(ctx, input.ID, "release", creditreservation.CommandIdentity{IdempotencyKey: input.IdempotencyKey, PayloadHash: input.PayloadHash}); err != nil {
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

func (s *service) MarkUnknown(ctx context.Context, input creditreservation.UnknownInput) (reservation creditreservation.Reservation, err error) {
	defer func() { s.metrics.commandOutcome(ctx, "unknown", err) }()
	return s.withReservation(ctx, input.ID, func(tx reservationadapter.TxAdapter, current creditreservation.Reservation) (creditreservation.Reservation, error) {
		if _, err := tx.EnsureLifecycleCommand(ctx, input.ID, "unknown", creditreservation.CommandIdentity{IdempotencyKey: input.IdempotencyKey, PayloadHash: input.PayloadHash}); err != nil {
			return creditreservation.Reservation{}, err
		}
		if current.State == creditreservation.ReservationStateUnknown {
			return current, nil
		}
		update := reservationadapter.UpdateReservationInput{ID: input.ID, ExpectedStates: []creditreservation.ReservationState{creditreservation.ReservationStateActive, creditreservation.ReservationStateExecuting}, State: creditreservation.ReservationStateUnknown}
		if current.State == creditreservation.ReservationStateActive {
			deadline := s.now().Add(s.executionDeadline)
			update.ExecutionDeadline = &deadline
		}
		return tx.UpdateReservation(ctx, update)
	})
}

func matchesIdentity(reservation creditreservation.Reservation, idempotencyKey, payloadHash string) error {
	if reservation.CommandIdentity.IdempotencyKey != idempotencyKey || reservation.CommandIdentity.PayloadHash != payloadHash {
		return creditreservation.ErrIdempotencyConflict
	}
	return nil
}

func (s *service) SweepExpired(ctx context.Context, now time.Time, limit int) (creditreservation.SweepResult, error) {
	rows, err := s.adapter.ListExpiredReservations(ctx, now, now.Add(-s.unknownManualReviewAfter), limit)
	if err != nil {
		return creditreservation.SweepResult{}, err
	}
	result := creditreservation.SweepResult{}
	for _, row := range rows {
		deadline := row.ExpiresAt
		if row.State == creditreservation.ReservationStateExecuting || row.State == creditreservation.ReservationStateUnknown {
			deadline = row.ExecutionDeadline
		}
		target := sweepTransition(row.State, valueOrZero(deadline), s.unknownManualReviewAfter, now)
		if target == "" {
			continue
		}
		_, err := s.withReservation(ctx, models.NamespacedID{Namespace: row.Namespace, ID: row.ID}, func(tx reservationadapter.TxAdapter, current creditreservation.Reservation) (creditreservation.Reservation, error) {
			currentDeadline := current.ExpiresAt
			if current.State == creditreservation.ReservationStateExecuting || current.State == creditreservation.ReservationStateUnknown {
				currentDeadline = current.ExecutionDeadline
			}
			if sweepTransition(current.State, valueOrZero(currentDeadline), s.unknownManualReviewAfter, now) != target {
				return current, nil
			}
			return tx.UpdateReservation(ctx, reservationadapter.UpdateReservationInput{ID: models.NamespacedID{Namespace: current.Namespace, ID: current.ID}, ExpectedStates: []creditreservation.ReservationState{current.State}, State: target})
		})
		if err != nil {
			return result, err
		}
		if target == creditreservation.ReservationStateExpired {
			result.Expired++
		} else if target == creditreservation.ReservationStateUnknown {
			result.Unknown++
		} else {
			result.ManualReview++
		}
		s.metrics.transition(ctx, string(target))
	}
	return result, nil
}

func valueOrZero(value *time.Time) time.Time {
	if value == nil {
		return time.Time{}
	}
	return *value
}
