package service

import (
	"context"
	"fmt"
	"time"

	decimal "github.com/alpacahq/alpacadecimal"
	"github.com/oklog/ulid/v2"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
	"github.com/openmeterio/openmeter/openmeter/creditlimit"
	"github.com/openmeterio/openmeter/openmeter/creditreservation"
	reservationadapter "github.com/openmeterio/openmeter/openmeter/creditreservation/adapter"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger/collector"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

// Settle, Charge, and ReverseCharge are public lifecycle contracts. Their
// concrete ledger implementation is intentionally unavailable until the
// reservation adapter persists the collector allocation provenance required by
// CorrectCollectedAccrued. Returning a typed error is safer than inventing a
// reversal from a current balance or committing a ledger group that cannot be
// deterministically corrected.
func (s *service) Settle(ctx context.Context, input creditreservation.SettleInput) (reservation creditreservation.Reservation, err error) {
	defer func() { s.metrics.commandOutcome(ctx, "settle", err) }()
	if err := validateSettleInput(input); err != nil {
		return creditreservation.Reservation{}, err
	}
	if s.settlement == nil {
		return creditreservation.Reservation{}, creditreservation.ErrSettlementNotConfigured
	}
	return s.withReservation(ctx, input.ID, func(tx reservationadapter.TxAdapter, current creditreservation.Reservation) (creditreservation.Reservation, error) {
		if current.State == creditreservation.ReservationStateSettled {
			if current.SettlementIdentity != input.CommandIdentity {
				return creditreservation.Reservation{}, creditreservation.ErrIdempotencyConflict
			}
			return current, nil
		}
		actual := min(input.ActualCredits, current.TotalCredits)
		if current.State != creditreservation.ReservationStateExecuting && current.State != creditreservation.ReservationStateUnknown {
			return creditreservation.Reservation{}, creditreservation.ErrStateConflict
		}
		groupID, _, err := s.collect(ctx, current.Namespace, current.ID, current.CustomerID, current.Currency, current.Lines, actual, current.EnterpriseHold, input.SettledAt)
		if err != nil {
			return creditreservation.Reservation{}, err
		}
		zero := int64(0)
		updated, err := tx.UpdateReservation(ctx, reservationadapter.UpdateReservationInput{
			ID: input.ID, ExpectedStates: []creditreservation.ReservationState{current.State}, State: creditreservation.ReservationStateSettled,
			SettledCredits: &actual, PrepaidHold: &zero, EnterpriseHold: &zero, SettlementIdentity: &input.CommandIdentity,
			SettlementLedgerGroupID: &groupID, Evidence: creditreservation.TransitionEvidence{Kind: creditreservation.TransitionEvidenceSettlement, Reference: groupID},
		})
		if err != nil {
			return creditreservation.Reservation{}, err
		}
		if err := tx.AppendUsageEvent(ctx, creditreservation.UsageEvent{EventID: "usage:" + current.ID, AggregateType: "credit_reservation", AggregateID: current.ID, EventType: "openmeter.credit.usage", Payload: map[string]any{"customer_id": current.CustomerID, "reservation_id": current.ID, "credits": actual}}); err != nil {
			return creditreservation.Reservation{}, err
		}
		return updated, nil
	})
}

func (s *service) Charge(ctx context.Context, input creditreservation.ChargeInput) (charge creditreservation.Charge, err error) {
	defer func() { s.metrics.commandOutcome(ctx, "charge", err) }()
	if err := validateChargeInput(input); err != nil {
		return creditreservation.Charge{}, err
	}
	if s.settlement == nil {
		return creditreservation.Charge{}, creditreservation.ErrSettlementNotConfigured
	}
	var result creditreservation.Charge
	err = s.adapter.WithCustomerLock(ctx, customer.CustomerID{Namespace: input.ID.Namespace, ID: input.CustomerID}, func(tx reservationadapter.TxAdapter) error {
		existing, found, err := tx.GetChargeByCommand(ctx, input.ID.Namespace, input.CommandIdentity.IdempotencyKey)
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
		priced, err := s.prices.Resolve(ctx, creditreservation.ResolvePriceInput{Namespace: input.ID.Namespace, CustomerID: input.CustomerID, At: input.BookedAt, Lines: input.Lines})
		if err != nil {
			return err
		}
		prepaid, err := s.collector.GetCollectableAmount(ctx, collector.GetCollectableAmountInput{CustomerID: customer.CustomerID{Namespace: input.ID.Namespace, ID: input.CustomerID}, Currency: priced.Currency, FeatureKey: priced.Lines[0].FeatureKey, AsOf: input.BookedAt})
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
		remaining, err := s.creditLimit.Remaining(ctx, creditlimit.RemainingInput{Namespace: input.ID.Namespace, CustomerID: input.CustomerID, Currency: priced.Currency, FeatureKey: priced.Lines[0].FeatureKey, AsOf: input.BookedAt})
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
		groupID, allocations, err := s.collect(ctx, input.ID.Namespace, input.ID.ID, input.CustomerID, priced.Currency, priced.Lines, priced.TotalCredits, split.enterprise, input.BookedAt)
		if err != nil {
			return err
		}
		result, _, err = tx.CreateCharge(ctx, reservationadapter.CreateChargeInput{Charge: creditreservation.Charge{ID: input.ID.ID, Currency: priced.Currency, RateVersion: priced.RateVersion, Lines: priced.Lines, TotalCredits: priced.TotalCredits, CommandIdentity: input.CommandIdentity}, Namespace: input.ID.Namespace, CustomerID: input.CustomerID, SubjectID: input.SubjectID, Operation: input.Operation, State: reservationadapter.ChargeStateSettled, SettlementLedgerGroupID: groupID, UsageEventID: "usage:" + input.ID.ID, SettlementAllocations: allocations})
		if err != nil {
			return err
		}
		return tx.AppendUsageEvent(ctx, creditreservation.UsageEvent{EventID: "usage:" + input.ID.ID, AggregateType: "credit_charge", AggregateID: input.ID.ID, EventType: "openmeter.credit.usage", Payload: map[string]any{"customer_id": input.CustomerID, "charge_id": input.ID.ID, "lines": priced.Lines, "credits": priced.TotalCredits}})
	})
	return result, err
}

func (s *service) ReverseCharge(ctx context.Context, input creditreservation.ReverseChargeInput) (charge creditreservation.Charge, err error) {
	defer func() { s.metrics.commandOutcome(ctx, "reverse", err) }()
	if err := input.ID.Validate(); err != nil {
		return creditreservation.Charge{}, err
	}
	if err := input.CommandIdentity.Validate(); err != nil {
		return creditreservation.Charge{}, err
	}
	if s.settlement == nil {
		return creditreservation.Charge{}, creditreservation.ErrSettlementNotConfigured
	}
	original, err := s.adapter.GetCharge(ctx, input.ID)
	if err != nil {
		return creditreservation.Charge{}, err
	}
	if len(original.SettlementAllocations) == 0 {
		return creditreservation.Charge{}, creditreservation.ErrSettlementProvenanceAbsent
	}
	var result creditreservation.Charge
	err = s.adapter.WithCustomerLock(ctx, customer.CustomerID{Namespace: original.Namespace, ID: original.CustomerID}, func(tx reservationadapter.TxAdapter) error {
		current, err := tx.GetCharge(ctx, input.ID)
		if err != nil {
			return err
		}
		if current.State == string(reservationadapter.ChargeStateReversed) {
			if err := matchesReverseIdentity(current.ReversalIdentity, input.CommandIdentity); err != nil {
				return err
			}
			result = current
			return nil
		}
		requests := make(creditrealization.CorrectionRequest, 0, len(current.SettlementAllocations))
		for _, allocation := range current.SettlementAllocations {
			requests = append(requests, creditrealization.CorrectionRequestItem{Allocation: creditrealization.Realization{NamespacedModel: models.NamespacedModel{Namespace: current.Namespace}, CreateInput: creditrealization.CreateInput{ID: allocation.ID, LedgerTransaction: ledgertransaction.GroupReference{TransactionGroupID: allocation.GroupID}, Amount: decimal.NewFromInt(allocation.Amount), Type: creditrealization.TypeAllocation}, SortHint: allocation.SortHint}, Amount: decimal.NewFromInt(-allocation.Amount)})
		}
		corrections, err := s.settlement.CorrectCollectedAccrued(ctx, collector.CorrectCollectedAccruedInput{Namespace: current.Namespace, ChargeID: current.ID, CustomerID: current.CustomerID, AllocateAt: input.ReversedAt, Corrections: requests})
		if err != nil {
			return err
		}
		if len(corrections) == 0 {
			return creditreservation.ErrSettlementProvenanceAbsent
		}
		result, err = tx.ReverseCharge(ctx, input.ID, input.CommandIdentity, corrections[0].LedgerTransaction.TransactionGroupID)
		return err
	})
	return result, err
}

func matchesReverseIdentity(stored, requested creditreservation.CommandIdentity) error {
	if stored != requested {
		return creditreservation.ErrIdempotencyConflict
	}
	return nil
}

func (s *service) collect(ctx context.Context, namespace, chargeID, customerID string, currency currencies.CurrencyReference, lines []creditreservation.RatedLine, credits, enterpriseHold int64, bookedAt time.Time) (string, []creditreservation.SettlementAllocation, error) {
	if len(lines) == 0 {
		return "", nil, creditreservation.ErrResourceLinesRequired
	}
	var receivableLimit *decimal.Decimal
	if enterpriseHold > 0 {
		value := decimal.NewFromInt(enterpriseHold)
		receivableLimit = &value
	}
	allocations, err := s.settlement.CollectToAccrued(ctx, collector.CollectToAccruedInput{Namespace: namespace, ChargeID: chargeID, CustomerID: customerID, BookedAt: bookedAt, SourceBalanceAsOf: bookedAt, Currency: currency, FeatureKey: lines[0].FeatureKey, SettlementMode: productcatalog.CreditOnlySettlementMode, ServicePeriod: timeutil.ClosedPeriod{From: bookedAt, To: bookedAt}, Amount: decimal.NewFromInt(credits), ReceivableLimit: receivableLimit})
	if err != nil {
		return "", nil, err
	}
	if len(allocations) == 0 {
		return "", nil, fmt.Errorf("collector returned no ledger provenance")
	}
	provenance := make([]creditreservation.SettlementAllocation, 0, len(allocations))
	for index, allocation := range allocations {
		provenance = append(provenance, creditreservation.SettlementAllocation{ID: ulid.Make().String(), GroupID: allocation.LedgerTransaction.TransactionGroupID, Amount: allocation.Amount.IntPart(), SortHint: index})
	}
	return allocations[0].LedgerTransaction.TransactionGroupID, provenance, nil
}

func validateSettleInput(input creditreservation.SettleInput) error {
	if err := input.ID.Validate(); err != nil {
		return err
	}
	if err := input.CommandIdentity.Validate(); err != nil {
		return err
	}
	if input.ActualCredits < 0 || input.SettledAt.IsZero() {
		return fmt.Errorf("invalid settle input")
	}
	return nil
}

func validateChargeInput(input creditreservation.ChargeInput) error {
	if err := input.ID.Validate(); err != nil {
		return err
	}
	if err := input.CommandIdentity.Validate(); err != nil {
		return err
	}
	if input.CustomerID == "" || input.SubjectID == "" || input.Operation == "" || len(input.Lines) == 0 || input.BookedAt.IsZero() {
		return fmt.Errorf("invalid charge input")
	}
	return nil
}
