package service

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	decimal "github.com/alpacahq/alpacadecimal"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/creditlimit"
	"github.com/openmeterio/openmeter/openmeter/creditreservation"
	reservationadapter "github.com/openmeterio/openmeter/openmeter/creditreservation/adapter"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger/collector"
	"github.com/openmeterio/openmeter/pkg/models"
)

// This catches a regression that creates a durable authorization before
// determining that neither prepaid funds nor an explicit limit can cover it.
func TestReserveRejectsBeforeCreatingRowWhenFundsAreInsufficient(t *testing.T) {
	_, err := splitFunding(9, nil, 10)

	require.ErrorIs(t, err, creditreservation.ErrInsufficientFunds)
}

// This catches accidentally treating a missing explicit credit limit as an
// unlimited receivable allowance.
func TestReserveDoesNotCreateReceivableWithoutExplicitLimit(t *testing.T) {
	_, err := splitFunding(9, nil, 10)

	require.ErrorIs(t, err, creditreservation.ErrInsufficientFunds)
}

func TestReserveFundingUsesPrepaidBeforeExplicitLimit(t *testing.T) {
	limit := int64(4)
	split, err := splitFunding(9, &limit, 10)

	require.NoError(t, err)
	require.Equal(t, int64(9), split.prepaid)
	require.Equal(t, int64(1), split.enterprise)
}

func TestReservePersistsHoldAndReplaysSameCommand(t *testing.T) {
	svc, store := newReserveService(t, 10, nil, 10)
	input := reserveInput("call-accepted")
	first, err := svc.Reserve(t.Context(), input)
	require.NoError(t, err)
	require.Equal(t, creditreservation.ReservationStateActive, first.State)
	require.Equal(t, 1, store.reservationCount())
	require.Equal(t, 1, store.outboxCount())

	replay, err := svc.Reserve(t.Context(), input)
	require.NoError(t, err)
	require.Equal(t, first.ID, replay.ID)
	require.Equal(t, 1, store.reservationCount())
	require.Equal(t, 1, store.outboxCount())
}

func TestReserveFencePreventsRowCreation(t *testing.T) {
	svc, store := newReserveService(t, 10, nil, 10)
	store.fenced = true
	_, err := svc.Reserve(t.Context(), reserveInput("call-fenced"))
	require.ErrorIs(t, err, creditreservation.ErrCustomerFenced)
	require.Zero(t, store.reservationCount())
}

func TestEstablishedFenceBlocksConcurrentReserveWithStableSequence(t *testing.T) {
	svc, store := newReserveService(t, 10, nil, 10)
	ready := make(chan creditreservation.FenceResult, 1)
	go func() {
		_ = store.WithCustomerLock(context.Background(), customer.CustomerID{Namespace: "ns", ID: "customer"}, func(tx reservationadapter.TxAdapter) error {
			fence, err := tx.EstablishRefundFence(context.Background(), "refund-1")
			ready <- fence
			return err
		})
	}()
	first := <-ready
	second, err := establishMemoryFence(store)
	require.NoError(t, err)
	require.Equal(t, first.Sequence, second.Sequence)
	_, err = svc.Reserve(t.Context(), reserveInput("call-after-established-fence"))
	require.ErrorIs(t, err, creditreservation.ErrCustomerFenced)
	require.Zero(t, store.reservationCount())
}

func establishMemoryFence(store *memoryAdapter) (creditreservation.FenceResult, error) {
	var result creditreservation.FenceResult
	err := store.WithCustomerLock(context.Background(), customer.CustomerID{Namespace: "ns", ID: "customer"}, func(tx reservationadapter.TxAdapter) error {
		var err error
		result, err = tx.EstablishRefundFence(context.Background(), "refund-1")
		return err
	})
	return result, err
}

func TestLifecycleExecuteUnknownReleaseAndSweep(t *testing.T) {
	svc, store := newReserveService(t, 30, nil, 10)
	reserved, err := svc.Reserve(t.Context(), reserveInput("call-life"))
	require.NoError(t, err)
	id := models.NamespacedID{Namespace: reserved.Namespace, ID: reserved.ID}
	executed, err := svc.Execute(t.Context(), creditreservation.ExecuteInput{ID: id, IdempotencyKey: reserved.CommandIdentity.IdempotencyKey, PayloadHash: reserved.CommandIdentity.PayloadHash, ExecutionDeadline: time.Date(2026, 8, 10, 12, 1, 0, 0, time.UTC)})
	require.NoError(t, err)
	require.Equal(t, creditreservation.ReservationStateExecuting, executed.State)
	unknown, err := svc.MarkUnknown(t.Context(), creditreservation.UnknownInput{ID: id, IdempotencyKey: reserved.CommandIdentity.IdempotencyKey, PayloadHash: reserved.CommandIdentity.PayloadHash})
	require.NoError(t, err)
	require.Equal(t, creditreservation.ReservationStateUnknown, unknown.State)
	_, err = svc.Release(t.Context(), creditreservation.ReleaseInput{ID: id, IdempotencyKey: reserved.CommandIdentity.IdempotencyKey, PayloadHash: reserved.CommandIdentity.PayloadHash, Evidence: creditreservation.Evidence{Kind: creditreservation.EvidenceNotSent}})
	require.ErrorIs(t, err, creditreservation.ErrTransitionEvidenceRequired)
	released, err := svc.Release(t.Context(), creditreservation.ReleaseInput{ID: id, IdempotencyKey: reserved.CommandIdentity.IdempotencyKey, PayloadHash: reserved.CommandIdentity.PayloadHash, Evidence: creditreservation.Evidence{Kind: creditreservation.EvidenceProviderConfirmedNotExecuted, Reference: "provider-1"}})
	require.NoError(t, err)
	require.Equal(t, creditreservation.ReservationStateReleased, released.State)

	active := reserveInput("call-sweep")
	active.AuthorizationExpiresAt = time.Date(2026, 8, 10, 11, 59, 0, 0, time.UTC)
	_, err = svc.Reserve(t.Context(), active)
	require.NoError(t, err)
	swept, err := svc.SweepExpired(t.Context(), time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC), 10)
	require.NoError(t, err)
	require.Equal(t, 1, swept.Expired)
	require.Equal(t, creditreservation.ReservationStateExpired, store.rows[active.ID.ID].State)
}

func newReserveService(t *testing.T, prepaid int64, limit *int64, cost int64) (creditreservation.Service, *memoryAdapter) {
	t.Helper()
	store := &memoryAdapter{rows: map[string]creditreservation.Reservation{}}
	svc, err := New(Config{Adapter: store, Prices: fixedPrice{credits: cost}, Collector: fixedCollector{amount: prepaid}, CreditLimit: fixedLimit{amount: limit}, Now: func() time.Time { return time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC) }})
	require.NoError(t, err)
	return svc, store
}

func reserveInput(call string) creditreservation.ReserveInput {
	return creditreservation.ReserveInput{ID: models.NamespacedID{Namespace: "ns", ID: ulid.Make().String()}, CustomerID: "customer", SubjectID: "subject", ClientCallID: call, Operation: "chat", CommandIdentity: creditreservation.CommandIdentity{IdempotencyKey: "reserve:" + call, PayloadHash: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}, Lines: []creditreservation.ResourceLine{{FeatureKey: "chat", ResourceCode: "tokens", Quantity: 1}}, AuthorizationExpiresAt: time.Date(2026, 8, 10, 12, 2, 0, 0, time.UTC)}
}

type fixedPrice struct{ credits int64 }

func (p fixedPrice) Resolve(context.Context, creditreservation.ResolvePriceInput) (creditreservation.ResolvedPrice, error) {
	return creditreservation.ResolvedPrice{Currency: currencies.CurrencyReference{}, RateVersion: "rate-v1", Lines: []creditreservation.RatedLine{{ResourceLine: creditreservation.ResourceLine{FeatureKey: "chat", ResourceCode: "tokens", Quantity: 1}, Credits: p.credits}}, TotalCredits: p.credits}, nil
}

type fixedCollector struct{ amount int64 }

func (c fixedCollector) GetCollectableAmount(context.Context, collector.GetCollectableAmountInput) (decimal.Decimal, error) {
	return decimal.NewFromInt(c.amount), nil
}

type fixedLimit struct{ amount *int64 }

func (l fixedLimit) Remaining(context.Context, creditlimit.RemainingInput) (*decimal.Decimal, error) {
	if l.amount == nil {
		return nil, nil
	}
	value := decimal.NewFromInt(*l.amount)
	return &value, nil
}

type memoryAdapter struct {
	mu       sync.Mutex
	rows     map[string]creditreservation.Reservation
	fenced   bool
	sequence string
	outbox   int
}

func (m *memoryAdapter) WithCustomerLock(ctx context.Context, _ customer.CustomerID, fn func(reservationadapter.TxAdapter) error) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return fn(memoryTx{m})
}
func (m *memoryAdapter) GetReservation(_ context.Context, id models.NamespacedID) (creditreservation.Reservation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	row, ok := m.rows[id.ID]
	if !ok {
		return creditreservation.Reservation{}, fmt.Errorf("not found")
	}
	return row, nil
}
func (m *memoryAdapter) GetCharge(context.Context, models.NamespacedID) (creditreservation.Charge, error) {
	return creditreservation.Charge{}, fmt.Errorf("not found")
}
func (m *memoryAdapter) ListExpiredReservations(_ context.Context, now time.Time, limit int) ([]creditreservation.Reservation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	rows := make([]creditreservation.Reservation, 0)
	for _, row := range m.rows {
		if len(rows) == limit {
			break
		}
		if (row.State == creditreservation.ReservationStateActive && row.ExpiresAt != nil && !row.ExpiresAt.After(now)) || (row.State == creditreservation.ReservationStateExecuting && row.ExecutionDeadline != nil && !row.ExecutionDeadline.After(now)) {
			rows = append(rows, row)
		}
	}
	return rows, nil
}
func (m *memoryAdapter) reservationCount() int { m.mu.Lock(); defer m.mu.Unlock(); return len(m.rows) }
func (m *memoryAdapter) outboxCount() int      { m.mu.Lock(); defer m.mu.Unlock(); return m.outbox }

type memoryTx struct{ m *memoryAdapter }

func (t memoryTx) GetReservation(_ context.Context, id models.NamespacedID) (creditreservation.Reservation, error) {
	row, ok := t.m.rows[id.ID]
	if !ok {
		return creditreservation.Reservation{}, fmt.Errorf("not found")
	}
	return row, nil
}
func (t memoryTx) GetReservationByCommand(_ context.Context, _ string, key string) (creditreservation.Reservation, bool, error) {
	for _, row := range t.m.rows {
		if row.CommandIdentity.IdempotencyKey == key {
			return row, true, nil
		}
	}
	return creditreservation.Reservation{}, false, nil
}
func (t memoryTx) GetChargeByCommand(context.Context, string, string) (creditreservation.Charge, bool, error) {
	return creditreservation.Charge{}, false, nil
}
func (t memoryTx) GetCharge(context.Context, models.NamespacedID) (creditreservation.Charge, error) {
	return creditreservation.Charge{}, fmt.Errorf("not found")
}
func (t memoryTx) ReverseCharge(context.Context, models.NamespacedID, string) (creditreservation.Charge, error) {
	return creditreservation.Charge{}, fmt.Errorf("unused")
}
func (t memoryTx) ActivePrepaidHold(_ context.Context, _ currencies.CurrencyReference, _ string) (int64, error) {
	var held int64
	for _, row := range t.m.rows {
		if row.State == creditreservation.ReservationStateActive || row.State == creditreservation.ReservationStateExecuting || row.State == creditreservation.ReservationStateUnknown || row.State == creditreservation.ReservationStateManualReview {
			held += row.TotalCredits
		}
	}
	return held, nil
}
func (t memoryTx) HasActiveRefundFence(context.Context) (bool, error) { return t.m.fenced, nil }
func (t memoryTx) EstablishRefundFence(context.Context, string) (creditreservation.FenceResult, error) {
	if t.m.sequence == "" {
		t.m.sequence = "fence-1"
	}
	t.m.fenced = true
	return creditreservation.FenceResult{Sequence: t.m.sequence, Established: true}, nil
}
func (t memoryTx) ReleaseRefundFence(_ context.Context, _ string, sequence string) error {
	if sequence != t.m.sequence {
		return creditreservation.ErrFenceSequenceConflict
	}
	t.m.fenced = false
	return nil
}
func (t memoryTx) CreateReservation(_ context.Context, input reservationadapter.CreateReservationInput) (creditreservation.Reservation, bool, error) {
	if row, ok := t.m.rows[input.Reservation.ID]; ok {
		return row, false, nil
	}
	t.m.rows[input.Reservation.ID] = input.Reservation
	return input.Reservation, true, nil
}
func (t memoryTx) UpdateReservation(_ context.Context, input reservationadapter.UpdateReservationInput) (creditreservation.Reservation, error) {
	row, ok := t.m.rows[input.ID.ID]
	if !ok {
		return creditreservation.Reservation{}, fmt.Errorf("not found")
	}
	valid := false
	for _, state := range input.ExpectedStates {
		if row.State == state {
			valid = true
		}
	}
	if !valid {
		return creditreservation.Reservation{}, creditreservation.ErrStateConflict
	}
	if err := creditreservation.ValidateTransitionWithEvidence(row.State, input.State, input.Evidence); err != nil {
		return creditreservation.Reservation{}, err
	}
	row.State = input.State
	if input.ExecutionDeadline != nil {
		row.ExecutionDeadline = input.ExecutionDeadline
	}
	t.m.rows[row.ID] = row
	return row, nil
}
func (t memoryTx) CreateCharge(context.Context, reservationadapter.CreateChargeInput) (creditreservation.Charge, bool, error) {
	return creditreservation.Charge{}, false, fmt.Errorf("unused")
}
func (t memoryTx) AppendUsageEvent(context.Context, creditreservation.UsageEvent) error {
	t.m.outbox++
	return nil
}
