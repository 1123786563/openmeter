package worker

import (
	"context"
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
