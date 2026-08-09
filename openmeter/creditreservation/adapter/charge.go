package adapter

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/creditreservation"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	dbcreditcharge "github.com/openmeterio/openmeter/openmeter/ent/db/creditcharge"
)

type ChargeState string

const (
	ChargeStateSettled  ChargeState = "SETTLED"
	ChargeStateReversed ChargeState = "REVERSED"
)

type CreateChargeInput struct {
	Charge                  creditreservation.Charge
	Namespace               string
	CustomerID              string
	SubjectID               string
	Operation               string
	State                   ChargeState
	SettlementLedgerGroupID string
	ReversalLedgerGroupID   string
	UsageEventID            string
}

func (t *txAdapter) GetChargeByCommand(ctx context.Context, namespace, idempotencyKey string) (creditreservation.Charge, bool, error) {
	if namespace != t.customerID.Namespace {
		return creditreservation.Charge{}, false, fmt.Errorf("charge namespace must match customer lock")
	}
	row, err := findCharge(ctx, t.db, namespace, idempotencyKey)
	if err != nil || row == nil {
		return creditreservation.Charge{}, false, err
	}
	charge, err := mapCharge(row)
	return charge, err == nil, err
}

func (t *txAdapter) CreateCharge(ctx context.Context, input CreateChargeInput) (creditreservation.Charge, bool, error) {
	if err := input.Charge.Validate(); err != nil {
		return creditreservation.Charge{}, false, fmt.Errorf("validate charge: %w", err)
	}
	if input.Namespace != t.customerID.Namespace || input.CustomerID != t.customerID.ID {
		return creditreservation.Charge{}, false, fmt.Errorf("charge customer must match customer lock")
	}
	if input.State != ChargeStateSettled && input.State != ChargeStateReversed {
		return creditreservation.Charge{}, false, fmt.Errorf("invalid charge state: %s", input.State)
	}

	existing, err := findCharge(ctx, t.db, input.Namespace, input.Charge.CommandIdentity.IdempotencyKey)
	if err != nil {
		return creditreservation.Charge{}, false, err
	}
	if existing != nil {
		return matchChargeCommand(existing, input.Charge.CommandIdentity)
	}

	persistedLines, err := marshalRatedLines(input.Charge.Lines)
	if err != nil {
		return creditreservation.Charge{}, false, err
	}
	create := t.db.CreditCharge.Create().
		SetNamespace(input.Namespace).
		SetCustomerID(input.CustomerID).
		SetSubjectID(input.SubjectID).
		SetOperation(input.Operation).
		SetIdempotencyKey(input.Charge.CommandIdentity.IdempotencyKey).
		SetPayloadHash(input.Charge.CommandIdentity.PayloadHash).
		SetCurrency(input.Charge.Currency).
		SetRatedLines(persistedLines).
		SetAmount(input.Charge.TotalCredits).
		SetState(dbcreditcharge.State(input.State)).
		SetSettlementLedgerGroupID(input.SettlementLedgerGroupID).
		SetReversalLedgerGroupID(input.ReversalLedgerGroupID).
		SetUsageEventID(input.UsageEventID)
	if input.Charge.ID != "" {
		create.SetID(input.Charge.ID)
	}
	if input.Charge.ReservationID != "" {
		create.SetReservationID(input.Charge.ReservationID)
	}
	if input.Charge.Currency.CustomCurrencyID != nil {
		create.SetCustomCurrencyID(*input.Charge.Currency.CustomCurrencyID)
	}

	row, err := create.Save(ctx)
	if err != nil {
		if !entdb.IsConstraintError(err) {
			return creditreservation.Charge{}, false, fmt.Errorf("create charge: %w", err)
		}
		existing, readErr := findCharge(ctx, t.db, input.Namespace, input.Charge.CommandIdentity.IdempotencyKey)
		if readErr != nil {
			return creditreservation.Charge{}, false, readErr
		}
		if existing != nil {
			return matchChargeCommand(existing, input.Charge.CommandIdentity)
		}
		return creditreservation.Charge{}, false, fmt.Errorf("create charge: %w", err)
	}
	charge, err := mapCharge(row)
	return charge, true, err
}

func findCharge(ctx context.Context, db *entdb.Client, namespace, idempotencyKey string) (*entdb.CreditCharge, error) {
	row, err := db.CreditCharge.Query().Where(
		dbcreditcharge.NamespaceEQ(namespace), dbcreditcharge.IdempotencyKeyEQ(idempotencyKey),
	).Only(ctx)
	if entdb.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find charge command: %w", err)
	}
	return row, nil
}

func matchChargeCommand(row *entdb.CreditCharge, identity creditreservation.CommandIdentity) (creditreservation.Charge, bool, error) {
	if row.PayloadHash != identity.PayloadHash {
		return creditreservation.Charge{}, false, creditreservation.ErrIdempotencyConflict
	}
	charge, err := mapCharge(row)
	return charge, false, err
}

func mapCharge(row *entdb.CreditCharge) (creditreservation.Charge, error) {
	lines, err := unmarshalRatedLines(row.RatedLines)
	if err != nil {
		return creditreservation.Charge{}, err
	}
	charge := creditreservation.Charge{
		ID:           row.ID,
		Currency:     row.Currency,
		RateVersion:  "",
		Lines:        lines,
		TotalCredits: row.Amount,
		CommandIdentity: creditreservation.CommandIdentity{
			IdempotencyKey: row.IdempotencyKey,
			PayloadHash:    row.PayloadHash,
		},
		State: string(row.State), SettlementLedgerGroupID: row.SettlementLedgerGroupID,
		ReversalLedgerGroupID: row.ReversalLedgerGroupID,
	}
	if row.ReservationID != nil {
		charge.ReservationID = *row.ReservationID
	}
	return charge, nil
}
