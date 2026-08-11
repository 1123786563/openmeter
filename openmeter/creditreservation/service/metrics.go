package service

import (
	"context"
	"errors"
	"fmt"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/openmeterio/openmeter/openmeter/creditreservation"
)

// lifecycleMetrics keeps reservation telemetry bounded: only fixed operation,
// outcome, and state enums are attributes; tenant and command identifiers are
// never labels.
type lifecycleMetrics struct {
	commands    metric.Int64Counter
	transitions metric.Int64Counter
	ceiling     metric.Int64Histogram
	receivable  metric.Int64Histogram
}

func newLifecycleMetrics(meter metric.Meter) (lifecycleMetrics, error) {
	commands, err := meter.Int64Counter("openmeter.credit_reservation.commands", metric.WithUnit("{command}"))
	if err != nil {
		return lifecycleMetrics{}, fmt.Errorf("create reservation command metric: %w", err)
	}
	transitions, err := meter.Int64Counter("openmeter.credit_reservation.transitions", metric.WithUnit("{transition}"))
	if err != nil {
		return lifecycleMetrics{}, fmt.Errorf("create reservation transition metric: %w", err)
	}
	ceiling, err := meter.Int64Histogram("openmeter.credit_reservation.ceiling_credits", metric.WithUnit("{credit}"))
	if err != nil {
		return lifecycleMetrics{}, fmt.Errorf("create reservation ceiling metric: %w", err)
	}
	receivable, err := meter.Int64Histogram("openmeter.credit_reservation.enterprise_hold_credits", metric.WithUnit("{credit}"))
	if err != nil {
		return lifecycleMetrics{}, fmt.Errorf("create reservation receivable metric: %w", err)
	}
	return lifecycleMetrics{commands: commands, transitions: transitions, ceiling: ceiling, receivable: receivable}, nil
}

func (m lifecycleMetrics) command(ctx context.Context, operation, outcome string) {
	m.commands.Add(ctx, 1, metric.WithAttributes(attribute.String("operation", operation), attribute.String("outcome", outcome)))
}

func (m lifecycleMetrics) transition(ctx context.Context, state string) {
	m.transitions.Add(ctx, 1, metric.WithAttributes(attribute.String("state", state)))
}

func (m lifecycleMetrics) commandOutcome(ctx context.Context, operation string, err error) {
	outcome := "success"
	switch {
	case errors.Is(err, creditreservation.ErrInsufficientFunds):
		outcome = "insufficient_funds"
	case errors.Is(err, creditreservation.ErrIdempotencyConflict), errors.Is(err, creditreservation.ErrStateConflict):
		outcome = "conflict"
	case err != nil:
		outcome = "error"
	}
	m.command(ctx, operation, outcome)
}
