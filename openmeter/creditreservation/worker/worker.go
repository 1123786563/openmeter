// Package worker relays committed credit-usage outbox facts to the standard
// ingest collector. Ledger settlement intentionally happens before this relay:
// a temporary ingestion failure can only delay the event, never re-book it.
package worker

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/cloudevents/sdk-go/v2/event"
	"go.opentelemetry.io/otel/metric"
	metricnoop "go.opentelemetry.io/otel/metric/noop"

	"github.com/openmeterio/openmeter/openmeter/ingest"
)

// OutboxRow is the delivery-only record claimed by an implementation of Repo.
// ID, rather than an application supplied event id, is the downstream
// idempotency key.
type OutboxRow struct {
	ID           string
	Namespace    string
	EventType    string
	Subject      string
	OccurredAt   time.Time
	Payload      map[string]any
	Owner        string
	ClaimCount   int
	Published    bool
	DeadLettered bool
}

// Repository operations must be conditional on owner: a lease that expired or
// moved to another worker must never be acknowledged by the old worker.
type Repository interface {
	Claim(context.Context, string, int, time.Duration) ([]OutboxRow, error)
	MarkPublished(context.Context, string, string) error
	Release(context.Context, string, string) error
	MarkDeadLetter(context.Context, string, string, string) error
}

type Config struct {
	Repo          Repository
	Collector     ingest.Collector
	OwnerID       string
	BatchSize     int
	LeaseDuration time.Duration
	MaxClaimCount int
	Meter         metric.Meter
}

type Worker struct {
	repo          Repository
	collector     ingest.Collector
	ownerID       string
	batchSize     int
	leaseDuration time.Duration
	maxClaimCount int
	metrics       outboxMetrics
}

func New(config Config) (*Worker, error) {
	if config.Repo == nil {
		return nil, errors.New("credit reservation worker: repository is required")
	}
	if config.Collector == nil {
		return nil, errors.New("credit reservation worker: ingest collector is required")
	}
	if config.OwnerID == "" {
		return nil, errors.New("credit reservation worker: owner id is required")
	}
	if config.BatchSize <= 0 {
		config.BatchSize = 50
	}
	if config.LeaseDuration <= 0 {
		config.LeaseDuration = 30 * time.Second
	}
	if config.MaxClaimCount <= 0 {
		config.MaxClaimCount = 3
	}
	meter := config.Meter
	if meter == nil {
		meter = metricnoop.NewMeterProvider().Meter("openmeter.credit_reservation")
	}
	metrics, err := newOutboxMetrics(meter)
	if err != nil {
		return nil, err
	}
	return &Worker{repo: config.Repo, collector: config.Collector, ownerID: config.OwnerID, batchSize: config.BatchSize, leaseDuration: config.LeaseDuration, maxClaimCount: config.MaxClaimCount, metrics: metrics}, nil
}

// ProcessOnce processes one claimed batch. A publish failure releases only its
// lease; the underlying settlement remains immutable and will not run again.
func (w *Worker) ProcessOnce(ctx context.Context) error {
	rows, err := w.repo.Claim(ctx, w.ownerID, w.batchSize, w.leaseDuration)
	if err != nil {
		return fmt.Errorf("claim credit usage outbox: %w", err)
	}
	var errs []error
	for _, row := range rows {
		if row.ClaimCount > w.maxClaimCount {
			reason := fmt.Sprintf("exceeded max claim count (%d)", w.maxClaimCount)
			if err := w.repo.MarkDeadLetter(ctx, w.ownerID, row.ID, reason); err != nil {
				errs = append(errs, fmt.Errorf("dead-letter %s: %w", row.ID, err))
				w.metrics.record(ctx, outboxOutcomeDeadLetterFailed)
			} else {
				w.metrics.record(ctx, outboxOutcomeDeadLettered)
			}
			continue
		}
		if err := w.collector.Ingest(ctx, row.Namespace, usageEvent(row)); err != nil {
			if releaseErr := w.repo.Release(ctx, w.ownerID, row.ID); releaseErr != nil {
				errs = append(errs, fmt.Errorf("publish %s: %w (release lease: %v)", row.ID, err, releaseErr))
				w.metrics.record(ctx, outboxOutcomeReleaseFailed)
			} else {
				errs = append(errs, fmt.Errorf("publish %s: %w", row.ID, err))
				w.metrics.record(ctx, outboxOutcomeRetry)
			}
			continue
		}
		if err := w.repo.MarkPublished(ctx, w.ownerID, row.ID); err != nil {
			errs = append(errs, fmt.Errorf("mark published %s: %w", row.ID, err))
			w.metrics.record(ctx, outboxOutcomeAckFailed)
		} else {
			w.metrics.record(ctx, outboxOutcomePublished)
		}
	}
	return errors.Join(errs...)
}

func usageEvent(row OutboxRow) event.Event {
	e := event.New()
	e.SetID(row.ID)
	e.SetSource("openmeter://credit-reservation")
	e.SetType(row.EventType)
	e.SetSubject(row.Subject)
	if !row.OccurredAt.IsZero() {
		e.SetTime(row.OccurredAt)
	}
	_ = e.SetData("application/json", row.Payload)
	return e
}
