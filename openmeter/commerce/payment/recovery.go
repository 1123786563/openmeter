package payment

import (
	"context"
	"errors"
	"time"

	"github.com/openmeterio/openmeter/pkg/clock"
)

const recoveryBatchSize = 100

// RecoveryRepository exposes the single ordered query needed for callback-lost
// payment recovery. Implementations must filter by namespace, pending status,
// and updated_at <= cutoff, then order by updated_at and ID ascending.
type RecoveryRepository interface {
	ListStalePendingAttempts(ctx context.Context, namespace string, cutoff time.Time, limit int) ([]PaymentAttempt, error)
}

type paymentConfirmationService interface {
	ConfirmPayment(ctx context.Context, namespace, attemptID string) (CallbackResult, error)
}

// Recovery adapts payment persistence and Service to the commerce worker's
// paymentConfirmer boundary.
type Recovery struct {
	repo       RecoveryRepository
	service    paymentConfirmationService
	staleAfter time.Duration
	now        func() time.Time
}

func NewRecovery(repo RecoveryRepository, service paymentConfirmationService, staleAfter time.Duration) *Recovery {
	return &Recovery{repo: repo, service: service, staleAfter: staleAfter, now: clock.Now}
}

// ListStalePending returns at most 100 attempt IDs in repository order.
func (r *Recovery) ListStalePending(ctx context.Context, namespace string) ([]string, error) {
	if r == nil || r.repo == nil {
		return nil, errors.New("payment recovery: repository is required")
	}
	if namespace == "" {
		return nil, errors.New("payment recovery: namespace is required")
	}
	if r.staleAfter <= 0 {
		return nil, errors.New("payment recovery: stale duration must be positive")
	}

	attempts, err := r.repo.ListStalePendingAttempts(ctx, namespace, r.now().Add(-r.staleAfter), recoveryBatchSize)
	if err != nil {
		return nil, err
	}
	ids := make([]string, len(attempts))
	for i, attempt := range attempts {
		ids[i] = attempt.ID
	}
	return ids, nil
}

// ConfirmPayment delegates to the payment service without reclassifying its
// error so provider timeouts remain retryable to the worker.
func (r *Recovery) ConfirmPayment(ctx context.Context, namespace, attemptID string) error {
	if r == nil || r.service == nil {
		return errors.New("payment recovery: payment service is required")
	}
	_, err := r.service.ConfirmPayment(ctx, namespace, attemptID)
	return err
}
