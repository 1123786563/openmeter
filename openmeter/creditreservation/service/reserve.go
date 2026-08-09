package service

import (
	"context"
	"fmt"

	decimal "github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/creditlimit"
	"github.com/openmeterio/openmeter/openmeter/creditreservation"
	reservationadapter "github.com/openmeterio/openmeter/openmeter/creditreservation/adapter"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger/collector"
)

type fundingSplit struct {
	prepaid    int64
	enterprise int64
}

func (s *service) Reserve(ctx context.Context, input creditreservation.ReserveInput) (creditreservation.Reservation, error) {
	if input.AuthorizationExpiresAt.IsZero() {
		input.AuthorizationExpiresAt = s.now().Add(s.authorizationTTL)
	}
	if err := validateReserve(input); err != nil {
		return creditreservation.Reservation{}, err
	}
	var result creditreservation.Reservation
	err := s.adapter.WithCustomerLock(ctx, customer.CustomerID{Namespace: input.ID.Namespace, ID: input.CustomerID}, func(tx reservationadapter.TxAdapter) error {
		existing, found, err := tx.GetReservationByCommand(ctx, input.ID.Namespace, input.CommandIdentity.IdempotencyKey)
		if err != nil {
			return err
		}
		if found {
			if existing.CommandIdentity.PayloadHash != input.CommandIdentity.PayloadHash {
				return creditreservation.ErrIdempotencyConflict
			}
			result = existing
			return nil
		}
		fenced, err := tx.HasActiveRefundFence(ctx)
		if err != nil {
			return fmt.Errorf("check customer refund fence: %w", err)
		}
		if fenced {
			return creditreservation.ErrCustomerFenced
		}
		priced, err := s.prices.Resolve(ctx, creditreservation.ResolvePriceInput{Namespace: input.ID.Namespace, CustomerID: input.CustomerID, At: s.now(), Lines: input.Lines})
		if err != nil {
			return err
		}
		prepaid, err := s.collector.GetCollectableAmount(ctx, collector.GetCollectableAmountInput{CustomerID: customer.CustomerID{Namespace: input.ID.Namespace, ID: input.CustomerID}, Currency: priced.Currency, FeatureKey: priced.Lines[0].FeatureKey, AsOf: s.now()})
		if err != nil {
			return err
		}
		held, err := tx.ActivePrepaidHold(ctx, priced.Currency, priced.Lines[0].FeatureKey)
		if err != nil {
			return err
		}
		available := decimalToCredits(prepaid) - held
		if available < 0 {
			available = 0
		}
		remaining, err := s.creditLimit.Remaining(ctx, creditlimit.RemainingInput{Namespace: input.ID.Namespace, CustomerID: input.CustomerID, Currency: priced.Currency, FeatureKey: priced.Lines[0].FeatureKey, AsOf: s.now()})
		if err != nil {
			return err
		}
		var limit *int64
		if remaining != nil {
			value := decimalToCredits(*remaining)
			if value < 0 {
				value = 0
			}
			limit = &value
		}
		split, err := splitFunding(available, limit, priced.TotalCredits)
		if err != nil {
			return err
		}
		expires := input.AuthorizationExpiresAt
		reservation := creditreservation.Reservation{ID: input.ID.ID, Namespace: input.ID.Namespace, CustomerID: input.CustomerID, Currency: priced.Currency, State: creditreservation.ReservationStateActive, RateVersion: priced.RateVersion, Lines: priced.Lines, TotalCredits: priced.TotalCredits, ExpiresAt: &expires, CommandIdentity: input.CommandIdentity}
		var created bool
		result, created, err = tx.CreateReservation(ctx, reservationadapter.CreateReservationInput{Reservation: reservation, SubjectID: input.SubjectID, ClientCallID: input.ClientCallID, Operation: input.Operation, EstimatedLines: priced.Lines, RatedLines: priced.Lines, CeilingCredits: priced.TotalCredits, PrepaidHold: split.prepaid, EnterpriseHold: split.enterprise, Provider: input.Provider, Model: input.Model, RequestID: input.RequestID, UsageEventID: reservation.ID})
		if err != nil || !created {
			return err
		}
		return tx.AppendUsageEvent(ctx, creditreservation.UsageEvent{EventID: result.ID, AggregateType: "credit_reservation", AggregateID: result.ID, EventType: "credit.reservation.created", Payload: map[string]any{"reservation_id": result.ID}})
	})
	if err != nil {
		s.metrics.command(ctx, "reserve", "error")
	} else {
		s.metrics.command(ctx, "reserve", "success")
		s.metrics.ceiling.Record(ctx, result.TotalCredits)
		s.metrics.receivable.Record(ctx, result.EnterpriseHold)
	}
	return result, err
}

func validateReserve(input creditreservation.ReserveInput) error {
	if err := input.ID.Validate(); err != nil {
		return err
	}
	if input.CustomerID == "" || input.SubjectID == "" || input.ClientCallID == "" || input.Operation == "" || input.AuthorizationExpiresAt.IsZero() || len(input.Lines) == 0 {
		return fmt.Errorf("invalid reserve input")
	}
	if err := input.CommandIdentity.Validate(); err != nil {
		return err
	}
	featureKey := input.Lines[0].FeatureKey
	for _, line := range input.Lines[1:] {
		if line.FeatureKey != featureKey {
			return fmt.Errorf("a reservation must contain one feature")
		}
	}
	return nil
}

func decimalToCredits(value decimal.Decimal) int64 { return value.IntPart() }

// splitFunding is the strict prepaid boundary used by Reserve. A nil limit is
// not a zero-value convenience: it explicitly prohibits receivables.
func splitFunding(prepaid int64, enterpriseLimit *int64, required int64) (fundingSplit, error) {
	if prepaid < 0 || required < 0 {
		return fundingSplit{}, fmt.Errorf("%w: negative balance or hold", creditreservation.ErrInsufficientFunds)
	}

	split := fundingSplit{prepaid: min(prepaid, required)}
	remaining := required - split.prepaid
	if remaining == 0 {
		return split, nil
	}
	if enterpriseLimit == nil || *enterpriseLimit < remaining {
		return fundingSplit{}, creditreservation.ErrInsufficientFunds
	}

	split.enterprise = remaining
	return split, nil
}
