package payment

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type fakeRecoveryRepository struct {
	attempts  []PaymentAttempt
	namespace string
	cutoff    time.Time
	limit     int
	err       error
}

func (f *fakeRecoveryRepository) ListStalePendingAttempts(_ context.Context, namespace string, cutoff time.Time, limit int) ([]PaymentAttempt, error) {
	f.namespace = namespace
	f.cutoff = cutoff
	f.limit = limit
	return f.attempts, f.err
}

type fakePaymentConfirmationService struct {
	namespace string
	attemptID string
	err       error
}

func (f *fakePaymentConfirmationService) ConfirmPayment(_ context.Context, namespace, attemptID string) (CallbackResult, error) {
	f.namespace = namespace
	f.attemptID = attemptID
	return CallbackResult{}, f.err
}

func TestRecoveryListsStalePendingInFixedBatch(t *testing.T) {
	now := time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC)
	repo := &fakeRecoveryRepository{attempts: []PaymentAttempt{{ID: "older"}, {ID: "newer"}}}
	recovery := NewRecovery(repo, &fakePaymentConfirmationService{}, 30*time.Second)
	recovery.now = func() time.Time { return now }

	ids, err := recovery.ListStalePending(t.Context(), "default")
	require.NoError(t, err)
	require.Equal(t, []string{"older", "newer"}, ids)
	require.Equal(t, "default", repo.namespace)
	require.Equal(t, now.Add(-30*time.Second), repo.cutoff)
	require.Equal(t, 100, repo.limit)
}

func TestRecoveryConfirmPaymentPassesThroughServiceError(t *testing.T) {
	wantErr := errors.New("provider timeout")
	service := &fakePaymentConfirmationService{err: wantErr}
	recovery := NewRecovery(&fakeRecoveryRepository{}, service, 30*time.Second)

	err := recovery.ConfirmPayment(t.Context(), "default", "attempt-1")
	require.ErrorIs(t, err, wantErr)
	require.Equal(t, "default", service.namespace)
	require.Equal(t, "attempt-1", service.attemptID)
}
