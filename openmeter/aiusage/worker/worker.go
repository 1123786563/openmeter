// Package worker implements the transactional outbox relay for AI Usage
// domain events. It claims unpublished outbox rows, publishes them to all
// registered projections (Kafka, ClickHouse), and marks them published only
// after every projection acknowledges.
//
// Key correctness guarantees:
//
//   - PostgreSQL facts (Batch, Ledger) are settled inside the original
//     transaction regardless of projection availability.
//   - The Event ID published to every projection equals the Outbox row ID,
//     ensuring exactly-once deduplication downstream.
//   - Expired leases are reclaimed once; a second expiry dead-letters the row.
//   - On restart the worker resumes processing from the oldest unpublished row.
//   - Dead-letter replay republishes the same Event ID without a second Ledger
//     effect (the Ledger was already committed).
package worker

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"go.opentelemetry.io/otel/trace"
)

// OutboxRow is a single unpublished outbox event claimed by the worker.
type OutboxRow struct {
	ID          string
	Namespace   string
	CustomerID  string
	SubjectID   string
	EventType   string
	Payload     map[string]any
	Owner       string    // worker instance ID that holds the current lease
	ClaimCount  int       // number of times this row has been claimed
	LeasedUntil time.Time // expiry of the current lease
	CreatedAt   time.Time
}

// PublishEvent is the event delivered to a projection. The EventID is always
// the Outbox row ID, which downstream consumers use for deduplication.
type PublishEvent struct {
	EventID    string
	EventType  string
	CustomerID string
	SubjectID  string
	Payload    map[string]any
}

// OutboxRepository is the storage abstraction the worker uses to claim and
// update outbox rows. Implementations must ensure claim is atomic
// (row-level lock or conditional update).
type OutboxRepository interface {
	// Claim atomically leases up to batchSize unpublished (or lease-expired)
	// rows, setting owner=ownerID and leased_until=now+leaseDuration. Returns
	// the claimed rows with Owner populated.
	Claim(ctx context.Context, ownerID string, batchSize int, leaseDuration time.Duration) ([]OutboxRow, error)

	// MarkPublished marks the given row IDs as successfully published.
	MarkPublished(ctx context.Context, ownerID string, ids []string) error

	// ReleaseLease clears the lease on the given row IDs, returning them to the
	// claimable pool. Only rows owned by ownerID are released, preventing
	// cross-worker lease clobbering.
	ReleaseLease(ctx context.Context, ownerID string, ids []string) error

	// MarkDeadLetter marks a row as permanently failed after the maximum claim
	// count is exceeded.
	MarkDeadLetter(ctx context.Context, id string, reason string) error

	// CountUnpublished returns the number of rows that are unpublished and not
	// currently leased.
	CountUnpublished(ctx context.Context) (int64, error)
}

// Projection is a downstream sink that consumes outbox events. The worker
// publishes to all registered projections and waits for every one to
// acknowledge before marking a row published.
type Projection interface {
	// Name returns the projection identifier (e.g. "kafka", "clickhouse").
	Name() string

	// Publish delivers events to the projection. Must be idempotent: replaying
	// the same EventID must not produce a duplicate side effect.
	Publish(ctx context.Context, events []PublishEvent) error
}

// DeadLetterHandler is invoked when a row exceeds the maximum claim count.
type DeadLetterHandler interface {
	Handle(ctx context.Context, row OutboxRow, reason string) error
}

// Config wires the outbox worker.
type Config struct {
	Repo          OutboxRepository
	Projections   []Projection
	DeadLetter    DeadLetterHandler
	OwnerID       string // unique worker instance ID for lease ownership
	BatchSize     int
	LeaseDuration time.Duration
	MaxClaimCount int
	PollInterval  time.Duration
	Logger        *slog.Logger
	Tracer        trace.Tracer
}

const (
	defaultBatchSize     = 50
	defaultLeaseDuration = 30 * time.Second
	defaultMaxClaimCount = 2 // original claim + one reclaim
	defaultPollInterval  = 5 * time.Second
)

// generateOwnerID returns a unique worker instance ID for lease ownership.
func generateOwnerID() string {
	return fmt.Sprintf("worker-%d", time.Now().UnixNano())
}

func (c Config) withDefaults() Config {
	out := c
	if out.BatchSize <= 0 {
		out.BatchSize = defaultBatchSize
	}
	if out.LeaseDuration <= 0 {
		out.LeaseDuration = defaultLeaseDuration
	}
	if out.MaxClaimCount <= 0 {
		out.MaxClaimCount = defaultMaxClaimCount
	}
	if out.PollInterval <= 0 {
		out.PollInterval = defaultPollInterval
	}
	if out.OwnerID == "" {
		out.OwnerID = generateOwnerID()
	}
	if out.Logger == nil {
		out.Logger = slog.Default()
	}
	return out
}

func (c Config) validate() error {
	if c.Repo == nil {
		return errors.New("worker: repository is required")
	}
	if len(c.Projections) == 0 {
		return errors.New("worker: at least one projection is required")
	}
	return nil
}

// Worker is the outbox relay loop.
type Worker struct {
	repo          OutboxRepository
	projections   []Projection
	deadLetter    DeadLetterHandler
	ownerID       string
	batchSize     int
	leaseDuration time.Duration
	maxClaimCount int
	pollInterval  time.Duration
	logger        *slog.Logger
	tracer        trace.Tracer

	mu      sync.Mutex
	stopCh  chan struct{}
	doneCh  chan struct{}
	running bool
}

// New creates a Worker from Config.
func New(cfg Config) (*Worker, error) {
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	cfg = cfg.withDefaults()
	return &Worker{
		repo:          cfg.Repo,
		projections:   cfg.Projections,
		deadLetter:    cfg.DeadLetter,
		ownerID:       cfg.OwnerID,
		batchSize:     cfg.BatchSize,
		leaseDuration: cfg.LeaseDuration,
		maxClaimCount: cfg.MaxClaimCount,
		pollInterval:  cfg.PollInterval,
		logger:        cfg.Logger,
		tracer:        cfg.Tracer,
		stopCh:        make(chan struct{}),
		doneCh:        make(chan struct{}),
	}, nil
}

// OwnerID returns the worker's unique instance ID used for lease ownership.
func (w *Worker) OwnerID() string { return w.ownerID }

// Start launches the poll loop in a background goroutine. It returns immediately.
// Call Stop to shut down.
func (w *Worker) Start(ctx context.Context) {
	w.mu.Lock()
	if w.running {
		w.mu.Unlock()
		return
	}
	w.running = true
	w.stopCh = make(chan struct{})
	w.doneCh = make(chan struct{})
	w.mu.Unlock()

	go w.run(ctx)
}

// Stop signals the worker to shut down and waits for the current batch to finish.
func (w *Worker) Stop() {
	w.mu.Lock()
	if !w.running {
		w.mu.Unlock()
		return
	}
	w.mu.Unlock()

	close(w.stopCh)
	<-w.doneCh

	w.mu.Lock()
	w.running = false
	w.mu.Unlock()
}

// run is the main poll loop. It exits when ctx is canceled or Stop is called.
func (w *Worker) run(ctx context.Context) {
	defer close(w.doneCh)

	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-w.stopCh:
			return
		case <-ticker.C:
			if err := w.processBatch(ctx); err != nil {
				w.logger.WarnContext(ctx, "worker: process batch error", "err", err)
			}
		}
	}
}

// ProcessOnce claims and publishes one batch of outbox rows without entering
// the poll loop. Useful for testing and manual triggers.
func (w *Worker) ProcessOnce(ctx context.Context) error {
	return w.processBatch(ctx)
}

// processBatch claims rows, publishes to all projections, and marks them
// published or dead-letters them.
func (w *Worker) processBatch(ctx context.Context) error {
	ctx, span := w.startSpan(ctx, "worker.processBatch")
	defer span.End()

	rows, err := w.repo.Claim(ctx, w.ownerID, w.batchSize, w.leaseDuration)
	if err != nil {
		return fmt.Errorf("worker: claim rows: %w", err)
	}
	if len(rows) == 0 {
		return nil
	}

	// Partition rows into publishable and dead-letter candidates.
	var publishable []OutboxRow
	var deadLetterRows []OutboxRow

	for _, row := range rows {
		if row.ClaimCount > w.maxClaimCount {
			deadLetterRows = append(deadLetterRows, row)
		} else {
			publishable = append(publishable, row)
		}
	}

	// Dead-letter rows that exceeded the maximum claim count.
	for _, row := range deadLetterRows {
		reason := fmt.Sprintf("exceeded max claim count (%d)", w.maxClaimCount)
		if err := w.repo.MarkDeadLetter(ctx, row.ID, reason); err != nil {
			w.logger.ErrorContext(ctx, "worker: mark dead-letter",
				"row_id", row.ID, "err", err)
		}
		if w.deadLetter != nil {
			if err := w.deadLetter.Handle(ctx, row, reason); err != nil {
				w.logger.ErrorContext(ctx, "worker: dead-letter handler",
					"row_id", row.ID, "err", err)
			}
		}
	}

	if len(publishable) == 0 {
		return nil
	}

	// Convert rows to publish events. Event ID = Outbox row ID.
	events := make([]PublishEvent, len(publishable))
	for i, row := range publishable {
		events[i] = PublishEvent{
			EventID:    row.ID,
			EventType:  row.EventType,
			CustomerID: row.CustomerID,
			SubjectID:  row.SubjectID,
			Payload:    row.Payload,
		}
	}

	// Publish to every projection. If any projection fails, release the lease
	// so the rows can be retried on the next poll. The rows that already
	// succeeded will be re-published with the same Event ID, which projections
	// must deduplicate.
	failedIDs := make([]string, 0, len(publishable))
	var publishErrs []error
	for _, proj := range w.projections {
		if err := proj.Publish(ctx, events); err != nil {
			w.logger.WarnContext(ctx, "worker: projection publish failed",
				"projection", proj.Name(), "err", err)
			publishErrs = append(publishErrs, fmt.Errorf("projection %s: %w", proj.Name(), err))
		}
	}

	if len(publishErrs) > 0 {
		// Release all leases so rows are retried.
		for _, row := range publishable {
			failedIDs = append(failedIDs, row.ID)
		}
		if err := w.repo.ReleaseLease(ctx, w.ownerID, failedIDs); err != nil {
			w.logger.ErrorContext(ctx, "worker: release lease after publish failure",
				"err", err)
		}
		return errors.Join(publishErrs...)
	}

	// All projections acknowledged — mark published.
	publishedIDs := make([]string, len(publishable))
	for i, row := range publishable {
		publishedIDs[i] = row.ID
	}
	if err := w.repo.MarkPublished(ctx, w.ownerID, publishedIDs); err != nil {
		return fmt.Errorf("worker: mark published: %w", err)
	}

	w.logger.DebugContext(ctx, "worker: published batch",
		"count", len(publishable))

	return nil
}

func (w *Worker) startSpan(ctx context.Context, name string) (context.Context, trace.Span) {
	if w.tracer != nil {
		return w.tracer.Start(ctx, name)
	}
	return ctx, trace.SpanFromContext(ctx)
}
