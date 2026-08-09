package worker

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestRunner_StartStop(t *testing.T) {
	var calls int32
	r, err := New(RunnerConfig{
		Name:     "test",
		Interval: 10 * time.Millisecond,
		Job: func(_ context.Context) (int, error) {
			atomic.AddInt32(&calls, 1)
			return 1, nil
		},
	})
	if err != nil {
		t.Fatalf("create runner: %v", err)
	}
	if r.Name() != "test" {
		t.Errorf("expected name 'test', got %s", r.Name())
	}
	if r.IsRunning() {
		t.Error("expected not running before start")
	}
	ctx := context.Background()
	r.Start(ctx)
	time.Sleep(55 * time.Millisecond)
	r.Stop()
	if atomic.LoadInt32(&calls) < 2 {
		t.Errorf("expected at least 2 calls, got %d", atomic.LoadInt32(&calls))
	}
	if r.IsRunning() {
		t.Error("expected not running after stop")
	}
}

func TestRunner_StopIdempotent(t *testing.T) {
	r, _ := New(RunnerConfig{
		Name: "test",
		Job:  func(context.Context) (int, error) { return 0, nil },
	})
	r.Stop() // should not panic
	r.Stop()
}

func TestRunner_StartIdempotent(t *testing.T) {
	var calls int32
	r, _ := New(RunnerConfig{
		Name:     "test",
		Interval: 10 * time.Millisecond,
		Job: func(_ context.Context) (int, error) {
			atomic.AddInt32(&calls, 1)
			return 1, nil
		},
	})
	ctx := context.Background()
	r.Start(ctx)
	r.Start(ctx) // second start is no-op
	time.Sleep(25 * time.Millisecond)
	r.Stop()
	// Should have roughly 2 calls, not double — second Start doesn't spawn another goroutine
	if atomic.LoadInt32(&calls) > 5 {
		t.Errorf("double start may have spawned extra goroutines: %d calls", atomic.LoadInt32(&calls))
	}
}

func TestRunner_ContextCancellation(t *testing.T) {
	r, _ := New(RunnerConfig{
		Name:     "test",
		Interval: 10 * time.Millisecond,
		Job:      func(context.Context) (int, error) { return 0, nil },
	})
	ctx, cancel := context.WithCancel(context.Background())
	r.Start(ctx)
	cancel()
	time.Sleep(30 * time.Millisecond)
	// After context cancel, the runner should have exited.
	// We can't check IsRunning directly because it's set by Stop, but
	// the doneCh should be closed. We test by checking that Stop doesn't hang.
	done := make(chan struct{})
	go func() {
		r.Stop()
		close(done)
	}()
	select {
	case <-done:
		// good
	case <-time.After(time.Second):
		t.Error("Stop timed out after context cancellation")
	}
}

func TestManager_StartStop(t *testing.T) {
	m := NewManager(nil)
	var calls int32
	for i := 0; i < 3; i++ {
		r, _ := New(RunnerConfig{
			Name:     "runner",
			Interval: 10 * time.Millisecond,
			Job: func(_ context.Context) (int, error) {
				atomic.AddInt32(&calls, 1)
				return 1, nil
			},
		})
		m.Register(r)
	}
	if len(m.RunnerNames()) != 3 {
		t.Fatalf("expected 3 runners, got %d", len(m.RunnerNames()))
	}
	ctx := context.Background()
	m.Start(ctx)
	time.Sleep(35 * time.Millisecond)
	m.Stop()
	total := atomic.LoadInt32(&calls)
	if total < 3 {
		t.Errorf("expected at least 3 total calls, got %d", total)
	}
}

func TestRunner_ProcessOnce(t *testing.T) {
	var calls int32
	r, _ := New(RunnerConfig{
		Name: "test",
		Job: func(_ context.Context) (int, error) {
			atomic.AddInt32(&calls, 1)
			return 1, nil
		},
	})
	n, err := r.ProcessOnce(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 1 {
		t.Errorf("expected 1 processed, got %d", n)
	}
	if atomic.LoadInt32(&calls) != 1 {
		t.Errorf("expected 1 call, got %d", atomic.LoadInt32(&calls))
	}
}

func TestNew_MissingName(t *testing.T) {
	_, err := New(RunnerConfig{Job: func(context.Context) (int, error) { return 0, nil }})
	if err == nil {
		t.Error("expected error for missing name")
	}
}

func TestNew_MissingJob(t *testing.T) {
	_, err := New(RunnerConfig{Name: "test"})
	if err == nil {
		t.Error("expected error for missing job")
	}
}

type countingLeaseRecoverer struct {
	calls int32
	err   error
}

func (c *countingLeaseRecoverer) RecoverExpiredLeases(context.Context) (int, error) {
	atomic.AddInt32(&c.calls, 1)
	if c.err != nil {
		return 0, c.err
	}
	return 5, nil
}

func TestManager_LeaseRecoveryOnStart(t *testing.T) {
	lr := &countingLeaseRecoverer{}
	m := NewManager(nil)
	m.leaseRecovery = lr

	r, _ := New(RunnerConfig{
		Name: "test",
		Job:  func(context.Context) (int, error) { return 0, nil },
	})
	m.Register(r)

	ctx := context.Background()
	m.Start(ctx)
	time.Sleep(20 * time.Millisecond)
	m.Stop()

	if atomic.LoadInt32(&lr.calls) != 1 {
		t.Errorf("expected lease recovery called once, got %d", atomic.LoadInt32(&lr.calls))
	}
}

func TestManager_LeaseRecoveryErrorDoesNotBlockStart(t *testing.T) {
	lr := &countingLeaseRecoverer{err: errors.New("db unavailable")}
	m := NewManager(nil)
	m.leaseRecovery = lr

	r, _ := New(RunnerConfig{
		Name: "test",
		Job:  func(context.Context) (int, error) { return 0, nil },
	})
	m.Register(r)

	ctx := context.Background()
	m.Start(ctx)
	m.Stop()
	// Should not panic or hang — runners start even if recovery fails
}

func TestRegisterCommerceWorkers_FulfillmentOnly(t *testing.T) {
	var calls int32
	mgr, err := RegisterCommerceWorkers(CommerceWorkerDeps{
		Namespace: "test-ns",
		Fulfillment: &mockFulfillmentProcessor{
			fn: func(_ context.Context, _ string, _ int) (int, error) {
				atomic.AddInt32(&calls, 1)
				return 1, nil
			},
		},
	})
	if err != nil {
		t.Fatalf("create manager: %v", err)
	}
	if len(mgr.RunnerNames()) != 1 {
		t.Fatalf("expected 1 runner, got %d: %v", len(mgr.RunnerNames()), mgr.RunnerNames())
	}
	if mgr.RunnerNames()[0] != "fulfillment" {
		t.Errorf("expected fulfillment, got %s", mgr.RunnerNames()[0])
	}
}

func TestRegisterCommerceWorkers_AllRunners(t *testing.T) {
	mgr, err := RegisterCommerceWorkers(CommerceWorkerDeps{
		Namespace:      "test-ns",
		Fulfillment:    &mockFulfillmentProcessor{},
		Refund:         &mockRefundProcessor{},
		Payment:        &mockPaymentConfirmer{},
		Reconciliation: &mockReconRunner{},
		Enterprise:     &mockEnterpriseCloser{},
	})
	if err != nil {
		t.Fatalf("create manager: %v", err)
	}
	names := mgr.RunnerNames()
	if len(names) != 5 {
		t.Fatalf("expected 5 runners, got %d: %v", len(names), names)
	}
}

// Mock types for RegisterCommerceWorkers

type mockFulfillmentProcessor struct {
	fn func(ctx context.Context, ns string, limit int) (int, error)
}

func (m *mockFulfillmentProcessor) ProcessPending(ctx context.Context, ns string, limit int) (int, error) {
	if m.fn != nil {
		return m.fn(ctx, ns, limit)
	}
	return 0, nil
}

type mockRefundProcessor struct {
	listIDs   []string
	listErr   error
	procErrs  map[string]error
	processed []string
}

func (m *mockRefundProcessor) ListProviderProcessing(_ context.Context, _ string) ([]string, error) {
	return m.listIDs, m.listErr
}

func (m *mockRefundProcessor) ProcessOne(_ context.Context, _, refundID string) error {
	if m.procErrs != nil {
		if err, ok := m.procErrs[refundID]; ok {
			return err
		}
	}
	m.processed = append(m.processed, refundID)
	return nil
}

type mockPaymentConfirmer struct {
	listIDs   []string
	listErr   error
	procErrs  map[string]error
	confirmed []string
}

func (m *mockPaymentConfirmer) ListStalePending(_ context.Context, _ string) ([]string, error) {
	return m.listIDs, m.listErr
}

func (m *mockPaymentConfirmer) ConfirmPayment(_ context.Context, _, attemptID string) error {
	if m.procErrs != nil {
		if err, ok := m.procErrs[attemptID]; ok {
			return err
		}
	}
	m.confirmed = append(m.confirmed, attemptID)
	return nil
}

type mockReconRunner struct {
	findings int
	err      error
}

func (m *mockReconRunner) Run(_ context.Context, _ string) (int, error) {
	return m.findings, m.err
}

type mockEnterpriseCloser struct {
	accountIDs []string
	listErr    error
	evalErrs   map[string]error
	evaluated  []string
}

func (m *mockEnterpriseCloser) ListAccountsForEvaluation(_ context.Context, _ string) ([]string, error) {
	return m.accountIDs, m.listErr
}

func (m *mockEnterpriseCloser) EvaluateCollection(_ context.Context, _, accountID string) error {
	if m.evalErrs != nil {
		if err, ok := m.evalErrs[accountID]; ok {
			return err
		}
	}
	m.evaluated = append(m.evaluated, accountID)
	return nil
}

// ---------------------------------------------------------------------------
// Runner JobFunc integration tests — verify runners actually call services
// ---------------------------------------------------------------------------

func TestRegisterCommerceWorkers_RefundQueryProcesses(t *testing.T) {
	refundMock := &mockRefundProcessor{
		listIDs:  []string{"ref-1", "ref-2", "ref-3"},
		procErrs: map[string]error{"ref-2": errors.New("provider timeout")},
	}
	mgr, err := RegisterCommerceWorkers(CommerceWorkerDeps{
		Namespace: "test-ns",
		Refund:    refundMock,
	})
	if err != nil {
		t.Fatalf("create manager: %v", err)
	}
	if len(mgr.RunnerNames()) != 1 {
		t.Fatalf("expected 1 runner, got %d", len(mgr.RunnerNames()))
	}

	// Run the job once via ProcessOnce on the registered runner
	runner := mgr.runners[0]
	n, err := runner.ProcessOnce(context.Background())
	if err == nil || !errors.Is(err, refundMock.procErrs["ref-2"]) {
		t.Fatalf("expected provider timeout error, got %v", err)
	}
	// ref-1 and ref-3 succeed, ref-2 fails — 2 processed
	if n != 2 {
		t.Errorf("expected 2 processed, got %d", n)
	}
	if len(refundMock.processed) != 2 {
		t.Errorf("expected 2 processed in mock, got %d", len(refundMock.processed))
	}
}

func TestRegisterCommerceWorkers_PaymentQueryReturnsItemErrorsAfterBatch(t *testing.T) {
	wantErr := errors.New("provider timeout")
	payMock := &mockPaymentConfirmer{
		listIDs:  []string{"att-1", "att-2", "att-3"},
		procErrs: map[string]error{"att-2": wantErr},
	}
	mgr, err := RegisterCommerceWorkers(CommerceWorkerDeps{Namespace: "test-ns", Payment: payMock})
	if err != nil {
		t.Fatalf("create manager: %v", err)
	}

	n, err := mgr.runners[0].ProcessOnce(context.Background())
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected provider timeout error, got %v", err)
	}
	if n != 2 {
		t.Fatalf("expected 2 confirmed, got %d", n)
	}
	if len(payMock.confirmed) != 2 {
		t.Fatalf("expected remaining attempts to continue, got %v", payMock.confirmed)
	}
}

func TestRegisterCommerceWorkers_PaymentQueryProcesses(t *testing.T) {
	payMock := &mockPaymentConfirmer{
		listIDs: []string{"att-1", "att-2"},
	}
	mgr, err := RegisterCommerceWorkers(CommerceWorkerDeps{
		Namespace: "test-ns",
		Payment:   payMock,
	})
	if err != nil {
		t.Fatalf("create manager: %v", err)
	}
	if len(mgr.RunnerNames()) != 1 {
		t.Fatalf("expected 1 runner, got %d", len(mgr.RunnerNames()))
	}

	runner := mgr.runners[0]
	n, err := runner.ProcessOnce(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 2 {
		t.Errorf("expected 2 confirmed, got %d", n)
	}
	if len(payMock.confirmed) != 2 {
		t.Errorf("expected 2 confirmed in mock, got %d", len(payMock.confirmed))
	}
}

func TestRegisterCommerceWorkers_ReceivableCloseIterates(t *testing.T) {
	entMock := &mockEnterpriseCloser{
		accountIDs: []string{"acc-1", "acc-2", "acc-3"},
		evalErrs:   map[string]error{"acc-2": errors.New("db error")},
	}
	mgr, err := RegisterCommerceWorkers(CommerceWorkerDeps{
		Namespace:  "test-ns",
		Enterprise: entMock,
	})
	if err != nil {
		t.Fatalf("create manager: %v", err)
	}
	if len(mgr.RunnerNames()) != 1 {
		t.Fatalf("expected 1 runner, got %d", len(mgr.RunnerNames()))
	}

	runner := mgr.runners[0]
	n, err := runner.ProcessOnce(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// acc-1 and acc-3 succeed, acc-2 fails — 2 processed
	if n != 2 {
		t.Errorf("expected 2 evaluated, got %d", n)
	}
}

func TestRegisterCommerceWorkers_ReconciliationReportsFindings(t *testing.T) {
	mgr, err := RegisterCommerceWorkers(CommerceWorkerDeps{
		Namespace:      "test-ns",
		Reconciliation: &mockReconRunner{findings: 3},
	})
	if err != nil {
		t.Fatalf("create manager: %v", err)
	}
	runner := mgr.runners[0]
	n, err := runner.ProcessOnce(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 1 {
		t.Errorf("expected 1 (recon ran), got %d", n)
	}
}

func TestRegisterCommerceWorkers_ReconciliationReturnsServiceError(t *testing.T) {
	wantErr := errors.New("reconciliation unavailable")
	mgr, err := RegisterCommerceWorkers(CommerceWorkerDeps{
		Namespace:      "test-ns",
		Reconciliation: &mockReconRunner{err: wantErr},
	})
	if err != nil {
		t.Fatalf("create manager: %v", err)
	}

	n, err := mgr.runners[0].ProcessOnce(context.Background())
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected reconciliation error, got %v", err)
	}
	if n != 0 {
		t.Fatalf("expected no completed reconciliation run, got %d", n)
	}
}
