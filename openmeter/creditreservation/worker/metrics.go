package worker

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

const outboxProcessedMetricName = "openmeter.credit_reservation.outbox.processed"

type outboxOutcome string

const (
	outboxOutcomePublished        outboxOutcome = "published"
	outboxOutcomeRetry            outboxOutcome = "retry"
	outboxOutcomeReleaseFailed    outboxOutcome = "release_failed"
	outboxOutcomeAckFailed        outboxOutcome = "ack_failed"
	outboxOutcomeDeadLettered     outboxOutcome = "dead_lettered"
	outboxOutcomeDeadLetterFailed outboxOutcome = "dead_letter_failed"
)

// outboxMetrics deliberately uses only a fixed outcome enum. Reservation IDs,
// customer IDs, and namespaces are request data and would create unbounded
// time-series cardinality if used as metric attributes.
type outboxMetrics struct {
	processed metric.Int64Counter
}

func newOutboxMetrics(meter metric.Meter) (outboxMetrics, error) {
	processed, err := meter.Int64Counter(
		outboxProcessedMetricName,
		metric.WithDescription("Number of credit reservation outbox rows processed"),
		metric.WithUnit("{outbox_record}"),
	)
	if err != nil {
		return outboxMetrics{}, fmt.Errorf("create credit reservation outbox processed counter: %w", err)
	}

	return outboxMetrics{processed: processed}, nil
}

func (m outboxMetrics) record(ctx context.Context, outcome outboxOutcome) {
	m.processed.Add(ctx, 1, metric.WithAttributes(attribute.String("outcome", string(outcome))))
}
