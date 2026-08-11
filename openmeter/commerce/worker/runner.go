// Package worker implements lifecycle-managed background runners for the Phase 2
// commerce domain. Each runner is a NamedRunner that ticks on a configurable
// interval and invokes a domain-specific Process function.
//
// Runners registered by the server composition layer:
//
//   - payment-query-recovery: polls payment providers for callback-lost orders
//     (attempts stuck in "pending" beyond the callback window).
//   - fulfillment: processes pending fulfillment records toward "fulfilled".
//   - refund-query: polls payment providers for refund status on refunds stuck
//     in "provider_processing".
//   - receivable-close: closes enterprise receivable periods that have ended.
//   - reconciliation: runs the scheduled reconciliation checks.
//
// Lifecycle contract:
//
//   - Startup recovers expired processing leases (records stuck in "processing"
//     beyond the lease timeout become eligible for re-claim).
//   - Shutdown stops intake, drains the current in-flight batch, and closes
//     dependencies in reverse order.
//   - Each runner is independently startable/stoppable.
package worker

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// JobFunc is the domain-specific work each runner performs on each tick. It
// receives a context and returns the number of items processed and an error.
// The runner logs errors but never crashes; a non-nil error is retried on the
// next tick.
type JobFunc func(ctx context.Context) (int, error)

// Runner is a single background loop. It ticks on an interval and calls the
// configured JobFunc. It is safe to Start and Stop concurrently, but each
// instance runs at most one loop at a time.
type Runner struct {
	name     string
	interval time.Duration
	job      JobFunc
	logger   *slog.Logger

	mu      sync.Mutex
	stopCh  chan struct{}
	doneCh  chan struct{}
	running bool
}

// RunnerConfig wires a Runner.
type RunnerConfig struct {
	Name     string
	Interval time.Duration
	Job      JobFunc
	Logger   *slog.Logger
}

const defaultInterval = 10 * time.Second

// New creates a Runner from the config. The interval defaults to 10s if unset.
func New(cfg RunnerConfig) (*Runner, error) {
	if cfg.Name == "" {
		return nil, errors.New("worker: runner name is required")
	}
	if cfg.Job == nil {
		return nil, fmt.Errorf("worker: runner %s: job is required", cfg.Name)
	}
	interval := cfg.Interval
	if interval <= 0 {
		interval = defaultInterval
	}
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}
	return &Runner{
		name:     cfg.Name,
		interval: interval,
		job:      cfg.Job,
		logger:   logger.With("runner", cfg.Name),
	}, nil
}

// Name returns the runner name.
func (r *Runner) Name() string { return r.name }

// Start launches the poll loop in a background goroutine. It returns
// immediately. Calling Start on an already-running runner is a no-op.
func (r *Runner) Start(ctx context.Context) {
	r.mu.Lock()
	if r.running {
		r.mu.Unlock()
		return
	}
	r.running = true
	r.stopCh = make(chan struct{})
	r.doneCh = make(chan struct{})
	r.mu.Unlock()

	go r.run(ctx)
}

// Stop signals the runner to shut down and waits for the current tick to
// finish. It is idempotent — calling Stop on a non-running runner returns
// immediately.
func (r *Runner) Stop() {
	r.mu.Lock()
	if !r.running {
		r.mu.Unlock()
		return
	}
	r.mu.Unlock()

	close(r.stopCh)
	<-r.doneCh

	r.mu.Lock()
	r.running = false
	r.mu.Unlock()
}

// IsRunning reports whether the runner is currently active.
func (r *Runner) IsRunning() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.running
}

// run is the main poll loop. It exits when ctx is canceled or Stop is called.
func (r *Runner) run(ctx context.Context) {
	defer close(r.doneCh)

	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			r.logger.DebugContext(ctx, "runner stopped: context canceled")
			return
		case <-r.stopCh:
			r.logger.DebugContext(ctx, "runner stopped: stop signal")
			return
		case <-ticker.C:
			processed, err := r.job(ctx)
			if err != nil {
				r.logger.WarnContext(ctx, "runner tick error", "error", err, "processed", processed)
			} else if processed > 0 {
				r.logger.DebugContext(ctx, "runner tick", "processed", processed)
			}
		}
	}
}

// ProcessOnce runs the job once without entering the poll loop. Useful for
// testing and manual triggers.
func (r *Runner) ProcessOnce(ctx context.Context) (int, error) {
	return r.job(ctx)
}

// Manager owns a set of named runners. It provides coordinated Start/Stop for
// the full lifecycle: startup starts all runners in registration order;
// shutdown stops them in reverse order, draining the current tick of each.
type Manager struct {
	runners []*Runner
	logger  *slog.Logger

	leaseRecovery LeaseRecoverer
}

// NewManager creates a Manager.
func NewManager(logger *slog.Logger) *Manager {
	if logger == nil {
		logger = slog.Default()
	}
	return &Manager{logger: logger.With("component", "commerce-worker-manager")}
}

// Register adds a Runner to the manager. Runners must be registered before
// Start is called.
func (m *Manager) Register(r *Runner) {
	m.runners = append(m.runners, r)
}

// Start starts all registered runners. Each runner starts in its own goroutine.
// Before starting, if a LeaseRecoverer is set, it runs once to reclaim expired
// processing leases — this is the startup recovery requirement.
func (m *Manager) Start(ctx context.Context) {
	if m.leaseRecovery != nil {
		recovered, err := m.leaseRecovery.RecoverExpiredLeases(ctx)
		if err != nil {
			m.logger.WarnContext(ctx, "lease recovery failed", "error", err)
		} else if recovered > 0 {
			m.logger.InfoContext(ctx, "recovered expired leases", "count", recovered)
		}
	}
	for _, r := range m.runners {
		r.Start(ctx)
		m.logger.InfoContext(ctx, "started runner", "name", r.Name())
	}
}

// Stop stops all registered runners in reverse registration order and waits
// for each to drain its current tick. This ensures dependencies are closed in
// the correct order.
func (m *Manager) Stop() {
	for i := len(m.runners) - 1; i >= 0; i-- {
		m.runners[i].Stop()
		m.logger.InfoContext(context.Background(), "stopped runner", "name", m.runners[i].Name())
	}
}

// RunnerNames returns the names of all registered runners, in registration
// order. Useful for health checks and startup diagnostics.
func (m *Manager) RunnerNames() []string {
	names := make([]string, len(m.runners))
	for i, r := range m.runners {
		names[i] = r.Name()
	}
	return names
}

// ---------------------------------------------------------------------------
// Lease recovery
// ---------------------------------------------------------------------------

// LeaseRecoverer is called once during Manager.Start before runners begin. It
// reclaims records stuck in "processing" past their lease timeout so the first
// tick of the fulfillment worker picks them up. Implementations query the
// database for stale processing records and reset them to a reclaimable state.
type LeaseRecoverer interface {
	RecoverExpiredLeases(ctx context.Context) (int, error)
}

// noopLeaseRecoverer is a safe no-op implementation for testing.
type noopLeaseRecoverer struct{}

func (noopLeaseRecoverer) RecoverExpiredLeases(context.Context) (int, error) {
	return 0, nil
}

// ---------------------------------------------------------------------------
// Commerce worker registration
// ---------------------------------------------------------------------------

// CommerceWorkerDeps bundles the domain services each runner needs. Each field
// is optional — if nil, the corresponding runner is skipped. This keeps the
// factory usable in tests and partial deployments.
type CommerceWorkerDeps struct {
	// Namespace is the single-tenant namespace for worker queries. Multi-tenant
	// deployments iterate over namespaces; this covers the common single-ns case.
	Namespace string

	// Fulfillment processing: calls ProcessPending for all pending fulfillments.
	Fulfillment fulfillmentProcessor

	// Refund processing: advances initial, provider-pending, and ledger-recovery states.
	Refund refundProcessor

	// Payment query recovery: confirms payment attempts stuck in pending.
	Payment paymentConfirmer

	// Reconciliation: runs all invariant checks.
	Reconciliation reconRunner

	// Receivable close: closes ended enterprise periods.
	Enterprise enterpriseCloser

	// LeaseRecoverer reclaims stuck processing records at startup.
	LeaseRecovery LeaseRecoverer

	Logger *slog.Logger
}

// fulfillmentProcessor is the narrow interface for fulfillment processing.
// The fulfillment.Service satisfies this.
type fulfillmentProcessor interface {
	ProcessPending(ctx context.Context, namespace string, limit int) (int, error)
}

// refundProcessor advances every non-terminal state handled by ProcessOne,
// including initial submission, provider polling, and ledger recovery.
type refundProcessor interface {
	ListProcessable(ctx context.Context, namespace string) ([]string, error)
	ProcessOne(ctx context.Context, namespace, refundID string) error
}

// paymentConfirmer confirms payment attempts stuck in pending. The runner
// lists stale pending attempts and confirms each one via the provider.
type paymentConfirmer interface {
	ListStalePending(ctx context.Context, namespace string) ([]string, error)
	ConfirmPayment(ctx context.Context, namespace, attemptID string) error
}

// reconRunner runs the reconciliation check suite and returns the count of
// findings. Service errors are returned to the runner for retry/logging.
type reconRunner interface {
	Run(ctx context.Context, namespace string) (int, error)
}

// enterpriseCloser evaluates collection policy for all receivable accounts
// in the namespace. The runner lists account IDs and evaluates each one.
type enterpriseCloser interface {
	ListAccountsForEvaluation(ctx context.Context, namespace string) ([]string, error)
	EvaluateCollection(ctx context.Context, namespace, accountID string) error
}

// RegisterCommerceWorkers creates all five Phase 2 runners from the given deps
// and registers them on the Manager. Only runners whose dependency is non-nil
// are registered. The returned Manager is ready to Start/Stop.
//
// Runner intervals:
//   - fulfillment: 10s (time-critical for user experience)
//   - refund-query: 15s (fence held, so moderately urgent)
//   - payment-query-recovery: 30s (callback-lost recovery)
//   - receivable-close: 1h (periodic, not urgent)
//   - reconciliation: 5m (monitoring, not urgent)
func RegisterCommerceWorkers(deps CommerceWorkerDeps) (*Manager, error) {
	logger := deps.Logger
	if logger == nil {
		logger = slog.Default()
	}
	mgr := NewManager(logger)

	if deps.LeaseRecovery == nil {
		deps.LeaseRecovery = noopLeaseRecoverer{}
	}

	if deps.Fulfillment != nil {
		r, err := New(RunnerConfig{
			Name:     "fulfillment",
			Interval: 10 * time.Second,
			Job: func(ctx context.Context) (int, error) {
				return deps.Fulfillment.ProcessPending(ctx, deps.Namespace, 50)
			},
			Logger: logger,
		})
		if err != nil {
			return nil, fmt.Errorf("commerce worker: create fulfillment runner: %w", err)
		}
		mgr.Register(r)
	}

	if deps.Refund != nil {
		r, err := New(RunnerConfig{
			Name:     "refund-query",
			Interval: 15 * time.Second,
			Job: func(ctx context.Context) (int, error) {
				// The refund ProcessOne is per-refund; the worker loop would
				// list pending refunds and process each. For the factory we
				// delegate to a wrapper that handles batch iteration.
				return processRefundBatch(ctx, deps.Refund, deps.Namespace)
			},
			Logger: logger,
		})
		if err != nil {
			return nil, fmt.Errorf("commerce worker: create refund-query runner: %w", err)
		}
		mgr.Register(r)
	}

	if deps.Payment != nil {
		r, err := New(RunnerConfig{
			Name:     "payment-query-recovery",
			Interval: 30 * time.Second,
			Job: func(ctx context.Context) (int, error) {
				return confirmPendingPayments(ctx, deps.Payment, deps.Namespace)
			},
			Logger: logger,
		})
		if err != nil {
			return nil, fmt.Errorf("commerce worker: create payment-query-recovery runner: %w", err)
		}
		mgr.Register(r)
	}

	if deps.Enterprise != nil {
		r, err := New(RunnerConfig{
			Name:     "receivable-close",
			Interval: 1 * time.Hour,
			Job: func(ctx context.Context) (int, error) {
				return evaluateAllAccounts(ctx, deps.Enterprise, deps.Namespace, logger)
			},
			Logger: logger,
		})
		if err != nil {
			return nil, fmt.Errorf("commerce worker: create receivable-close runner: %w", err)
		}
		mgr.Register(r)
	}

	if deps.Reconciliation != nil {
		r, err := New(RunnerConfig{
			Name:     "reconciliation",
			Interval: 5 * time.Minute,
			Job: func(ctx context.Context) (int, error) {
				findings, err := deps.Reconciliation.Run(ctx, deps.Namespace)
				if err != nil {
					return 0, err
				}
				if findings > 0 {
					logger.WarnContext(ctx, "reconciliation found issues", "findings", findings)
				}
				return 1, nil
			},
			Logger: logger,
		})
		if err != nil {
			return nil, fmt.Errorf("commerce worker: create reconciliation runner: %w", err)
		}
		mgr.Register(r)
	}

	mgr.leaseRecovery = deps.LeaseRecovery
	return mgr, nil
}

// processRefundBatch lists all processable refunds and advances each one. It
// continues after individual failures, then returns the joined errors so the
// runner reports the incomplete batch and retries later.
func processRefundBatch(ctx context.Context, svc refundProcessor, namespace string) (int, error) {
	ids, err := svc.ListProcessable(ctx, namespace)
	if err != nil {
		return 0, fmt.Errorf("list processable refunds: %w", err)
	}
	processed := 0
	var processErrors []error
	for _, id := range ids {
		if err := svc.ProcessOne(ctx, namespace, id); err != nil {
			processErrors = append(processErrors, fmt.Errorf("refund %s: %w", id, err))
			continue
		}
		processed++
	}
	return processed, errors.Join(processErrors...)
}

// confirmPendingPayments lists payment attempts stuck in pending status past
// the callback window and queries the provider for each one. This is the
// callback-lost recovery path.
func confirmPendingPayments(ctx context.Context, confirmer paymentConfirmer, namespace string) (int, error) {
	ids, err := confirmer.ListStalePending(ctx, namespace)
	if err != nil {
		return 0, fmt.Errorf("list stale pending attempts: %w", err)
	}
	processed := 0
	var confirmErrors []error
	for _, id := range ids {
		if err := confirmer.ConfirmPayment(ctx, namespace, id); err != nil {
			confirmErrors = append(confirmErrors, fmt.Errorf("payment attempt %s: %w", id, err))
			continue
		}
		processed++
	}
	return processed, errors.Join(confirmErrors...)
}

// evaluateAllAccounts lists all receivable accounts in the namespace and
// evaluates the collection policy for each one.
func evaluateAllAccounts(ctx context.Context, closer enterpriseCloser, namespace string, logger *slog.Logger) (int, error) {
	accountIDs, err := closer.ListAccountsForEvaluation(ctx, namespace)
	if err != nil {
		return 0, fmt.Errorf("list accounts for evaluation: %w", err)
	}
	processed := 0
	for _, accountID := range accountIDs {
		if err := closer.EvaluateCollection(ctx, namespace, accountID); err != nil {
			logger.WarnContext(ctx, "receivable-close: evaluate failed", "account_id", accountID, "error", err)
			continue
		}
		processed++
	}
	return processed, nil
}
