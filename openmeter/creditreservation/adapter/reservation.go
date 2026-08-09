package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"time"

	"github.com/openmeterio/openmeter/openmeter/creditreservation"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	dbcreditreservation "github.com/openmeterio/openmeter/openmeter/ent/db/creditreservation"
	"github.com/openmeterio/openmeter/pkg/models"
)

type CreateReservationInput struct {
	Reservation       creditreservation.Reservation
	SubjectID         string
	ClientCallID      string
	Operation         string
	EstimatedLines    []creditreservation.RatedLine
	RatedLines        []creditreservation.RatedLine
	ActualLines       []creditreservation.RatedLine
	CeilingCredits    int64
	PrepaidHold       int64
	EnterpriseHold    int64
	SettledCredits    int64
	Provider          string
	Model             string
	RequestID         string
	ExecutionDeadline *time.Time
	HoldLedgerGroupID string
	UsageEventID      string
}

type UpdateReservationInput struct {
	ID                      models.NamespacedID
	ExpectedStates          []creditreservation.ReservationState
	State                   creditreservation.ReservationState
	EstimatedLines          []creditreservation.RatedLine
	RatedLines              []creditreservation.RatedLine
	ActualLines             []creditreservation.RatedLine
	CeilingCredits          *int64
	PrepaidHold             *int64
	EnterpriseHold          *int64
	SettledCredits          *int64
	Provider                *string
	Model                   *string
	RequestID               *string
	AuthorizationExpiresAt  *time.Time
	ExecutionDeadline       *time.Time
	HoldLedgerGroupID       *string
	SettlementLedgerGroupID *string
	ReleaseLedgerGroupID    *string
	UsageEventID            *string
}

func (t *txAdapter) CreateReservation(ctx context.Context, input CreateReservationInput) (creditreservation.Reservation, bool, error) {
	if err := input.Reservation.Validate(); err != nil {
		return creditreservation.Reservation{}, false, fmt.Errorf("validate reservation: %w", err)
	}
	if input.Reservation.Namespace != t.customerID.Namespace || input.Reservation.CustomerID != t.customerID.ID {
		return creditreservation.Reservation{}, false, fmt.Errorf("reservation customer must match customer lock")
	}

	existing, err := findReservationCommand(ctx, t.db, input.Reservation.Namespace, input.Reservation.CommandIdentity.IdempotencyKey, input.ClientCallID)
	if err != nil {
		return creditreservation.Reservation{}, false, err
	}
	if existing != nil {
		return matchReservationCommand(existing, input.Reservation.CommandIdentity)
	}

	ratedLines := input.RatedLines
	if ratedLines == nil {
		ratedLines = input.Reservation.Lines
	}
	estimatedLines := input.EstimatedLines
	if estimatedLines == nil {
		estimatedLines = ratedLines
	}
	persistedEstimatedLines, err := marshalRatedLines(estimatedLines)
	if err != nil {
		return creditreservation.Reservation{}, false, err
	}
	persistedRatedLines, err := marshalRatedLines(ratedLines)
	if err != nil {
		return creditreservation.Reservation{}, false, err
	}
	create := t.db.CreditReservation.Create().
		SetNamespace(input.Reservation.Namespace).
		SetCustomerID(input.Reservation.CustomerID).
		SetSubjectID(input.SubjectID).
		SetClientCallID(input.ClientCallID).
		SetOperation(input.Operation).
		SetIdempotencyKey(input.Reservation.CommandIdentity.IdempotencyKey).
		SetPayloadHash(input.Reservation.CommandIdentity.PayloadHash).
		SetCurrency(input.Reservation.Currency).
		SetEstimatedLines(persistedEstimatedLines).
		SetRatedLines(persistedRatedLines).
		SetCeilingCredits(input.CeilingCredits).
		SetPrepaidHold(input.PrepaidHold).
		SetEnterpriseHold(input.EnterpriseHold).
		SetSettledCredits(input.SettledCredits).
		SetRateVersion(input.Reservation.RateVersion).
		SetState(string(input.Reservation.State)).
		SetProvider(input.Provider).
		SetModel(input.Model).
		SetRequestID(input.RequestID).
		SetNillableAuthorizationExpiresAt(input.Reservation.ExpiresAt).
		SetNillableExecutionDeadline(input.ExecutionDeadline).
		SetHoldLedgerGroupID(input.HoldLedgerGroupID).
		SetUsageEventID(input.UsageEventID)
	if input.Reservation.ID != "" {
		create.SetID(input.Reservation.ID)
	}
	if input.Reservation.Currency.CustomCurrencyID != nil {
		create.SetCustomCurrencyID(*input.Reservation.Currency.CustomCurrencyID)
	}
	if input.ActualLines != nil {
		persistedActualLines, err := marshalRatedLines(input.ActualLines)
		if err != nil {
			return creditreservation.Reservation{}, false, err
		}
		create.SetActualLines(persistedActualLines)
	}

	row, err := create.Save(ctx)
	if err != nil {
		if !entdb.IsConstraintError(err) {
			return creditreservation.Reservation{}, false, fmt.Errorf("create reservation: %w", err)
		}
		existing, readErr := findReservationCommand(ctx, t.db, input.Reservation.Namespace, input.Reservation.CommandIdentity.IdempotencyKey, input.ClientCallID)
		if readErr != nil {
			return creditreservation.Reservation{}, false, readErr
		}
		if existing != nil {
			return matchReservationCommand(existing, input.Reservation.CommandIdentity)
		}
		return creditreservation.Reservation{}, false, fmt.Errorf("create reservation: %w", err)
	}
	reservation, err := mapReservation(row)
	return reservation, true, err
}

func (t *txAdapter) UpdateReservation(ctx context.Context, input UpdateReservationInput) (creditreservation.Reservation, error) {
	if err := input.ID.Validate(); err != nil {
		return creditreservation.Reservation{}, fmt.Errorf("validate reservation id: %w", err)
	}
	if input.ID.Namespace != t.customerID.Namespace {
		return creditreservation.Reservation{}, fmt.Errorf("reservation namespace must match customer lock")
	}
	if len(input.ExpectedStates) == 0 {
		return creditreservation.Reservation{}, creditreservation.ErrStateConflict
	}
	for _, from := range input.ExpectedStates {
		if err := creditreservation.ValidateTransition(from, input.State); err != nil {
			return creditreservation.Reservation{}, err
		}
	}

	states := make([]string, len(input.ExpectedStates))
	for i, state := range input.ExpectedStates {
		states[i] = string(state)
	}
	update := t.db.CreditReservation.Update().
		Where(
			dbcreditreservation.IDEQ(input.ID.ID),
			dbcreditreservation.NamespaceEQ(input.ID.Namespace),
			dbcreditreservation.StateIn(states...),
		).
		SetState(string(input.State))
	if input.EstimatedLines != nil {
		lines, err := marshalRatedLines(input.EstimatedLines)
		if err != nil {
			return creditreservation.Reservation{}, err
		}
		update.SetEstimatedLines(lines)
	}
	if input.RatedLines != nil {
		lines, err := marshalRatedLines(input.RatedLines)
		if err != nil {
			return creditreservation.Reservation{}, err
		}
		update.SetRatedLines(lines)
	}
	if input.ActualLines != nil {
		lines, err := marshalRatedLines(input.ActualLines)
		if err != nil {
			return creditreservation.Reservation{}, err
		}
		update.SetActualLines(lines)
	}
	if input.CeilingCredits != nil {
		update.SetCeilingCredits(*input.CeilingCredits)
	}
	if input.PrepaidHold != nil {
		update.SetPrepaidHold(*input.PrepaidHold)
	}
	if input.EnterpriseHold != nil {
		update.SetEnterpriseHold(*input.EnterpriseHold)
	}
	if input.SettledCredits != nil {
		update.SetSettledCredits(*input.SettledCredits)
	}
	if input.Provider != nil {
		update.SetProvider(*input.Provider)
	}
	if input.Model != nil {
		update.SetModel(*input.Model)
	}
	if input.RequestID != nil {
		update.SetRequestID(*input.RequestID)
	}
	if input.AuthorizationExpiresAt != nil {
		update.SetAuthorizationExpiresAt(*input.AuthorizationExpiresAt)
	}
	if input.ExecutionDeadline != nil {
		update.SetExecutionDeadline(*input.ExecutionDeadline)
	}
	if input.HoldLedgerGroupID != nil {
		update.SetHoldLedgerGroupID(*input.HoldLedgerGroupID)
	}
	if input.SettlementLedgerGroupID != nil {
		update.SetSettlementLedgerGroupID(*input.SettlementLedgerGroupID)
	}
	if input.ReleaseLedgerGroupID != nil {
		update.SetReleaseLedgerGroupID(*input.ReleaseLedgerGroupID)
	}
	if input.UsageEventID != nil {
		update.SetUsageEventID(*input.UsageEventID)
	}

	affected, err := update.Save(ctx)
	if err != nil {
		return creditreservation.Reservation{}, fmt.Errorf("update reservation: %w", err)
	}
	if affected != 1 {
		row, err := getReservation(ctx, t.db, input.ID)
		if err != nil {
			return creditreservation.Reservation{}, err
		}
		if row.State == input.State {
			return row, nil
		}
		return creditreservation.Reservation{}, creditreservation.ErrStateConflict
	}
	return getReservation(ctx, t.db, input.ID)
}

func getReservation(ctx context.Context, db *entdb.Client, id models.NamespacedID) (creditreservation.Reservation, error) {
	row, err := db.CreditReservation.Query().Where(
		dbcreditreservation.IDEQ(id.ID), dbcreditreservation.NamespaceEQ(id.Namespace),
	).Only(ctx)
	if err != nil {
		return creditreservation.Reservation{}, fmt.Errorf("get reservation: %w", err)
	}
	return mapReservation(row)
}

func findReservationCommand(ctx context.Context, db *entdb.Client, namespace, idempotencyKey, clientCallID string) (*entdb.CreditReservation, error) {
	byIdempotencyKey, err := findReservationByIdempotencyKey(ctx, db, namespace, idempotencyKey)
	if err != nil {
		return nil, err
	}
	byClientCall, err := findReservationByClientCall(ctx, db, namespace, clientCallID)
	if err != nil {
		return nil, err
	}
	return selectReservationIdentity(byIdempotencyKey, byClientCall)
}

func selectReservationIdentity(byIdempotencyKey, byClientCall *entdb.CreditReservation) (*entdb.CreditReservation, error) {
	if byIdempotencyKey != nil && byClientCall != nil && byIdempotencyKey.ID != byClientCall.ID {
		return nil, creditreservation.ErrIdempotencyConflict
	}
	if byIdempotencyKey != nil {
		return byIdempotencyKey, nil
	}
	return byClientCall, nil
}

func findReservationByIdempotencyKey(ctx context.Context, db *entdb.Client, namespace, idempotencyKey string) (*entdb.CreditReservation, error) {
	row, err := db.CreditReservation.Query().Where(
		dbcreditreservation.NamespaceEQ(namespace),
		dbcreditreservation.IdempotencyKeyEQ(idempotencyKey),
	).Only(ctx)
	if entdb.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find reservation by idempotency key: %w", err)
	}
	return row, nil
}

func findReservationByClientCall(ctx context.Context, db *entdb.Client, namespace, clientCallID string) (*entdb.CreditReservation, error) {
	row, err := db.CreditReservation.Query().Where(
		dbcreditreservation.NamespaceEQ(namespace),
		dbcreditreservation.ClientCallIDEQ(clientCallID),
	).Only(ctx)
	if entdb.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find reservation by client call: %w", err)
	}
	return row, nil
}

func matchReservationCommand(row *entdb.CreditReservation, identity creditreservation.CommandIdentity) (creditreservation.Reservation, bool, error) {
	if row.IdempotencyKey != identity.IdempotencyKey || row.PayloadHash != identity.PayloadHash {
		return creditreservation.Reservation{}, false, creditreservation.ErrIdempotencyConflict
	}
	reservation, err := mapReservation(row)
	return reservation, false, err
}

func mapReservation(row *entdb.CreditReservation) (creditreservation.Reservation, error) {
	lines, err := unmarshalRatedLines(row.RatedLines)
	if err != nil {
		return creditreservation.Reservation{}, err
	}
	return creditreservation.Reservation{
		ID: row.ID, Namespace: row.Namespace, CustomerID: row.CustomerID, Currency: row.Currency,
		State: creditreservation.ReservationState(row.State), RateVersion: row.RateVersion,
		Lines: lines, TotalCredits: row.CeilingCredits, ExpiresAt: row.AuthorizationExpiresAt,
		CommandIdentity: creditreservation.CommandIdentity{IdempotencyKey: row.IdempotencyKey, PayloadHash: row.PayloadHash},
	}, nil
}

func marshalRatedLines(lines []creditreservation.RatedLine) ([]json.RawMessage, error) {
	persisted := make([]json.RawMessage, 0, len(lines))
	for _, line := range lines {
		encoded, err := json.Marshal(line)
		if err != nil {
			return nil, fmt.Errorf("marshal rated line: %w", err)
		}
		persisted = append(persisted, encoded)
	}
	return persisted, nil
}

func unmarshalRatedLines(persisted []json.RawMessage) ([]creditreservation.RatedLine, error) {
	lines := make([]creditreservation.RatedLine, 0, len(persisted))
	for _, line := range persisted {
		var decoded creditreservation.RatedLine
		if err := json.Unmarshal(line, &decoded); err != nil {
			return nil, fmt.Errorf("unmarshal rated line: %w", err)
		}
		lines = append(lines, decoded)
	}
	return slices.Clone(lines), nil
}
