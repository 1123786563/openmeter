package worker_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cloudevents/sdk-go/v2/event"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/creditreservation/worker"
)

func TestProcessOncePublishesCreditUsageWithOutboxID(t *testing.T) {
	repo := &memoryRepo{rows: []worker.OutboxRow{{
		ID: "outbox-1", Namespace: "ns", EventType: "openmeter.credit.usage",
		Subject: "subject-1", OccurredAt: time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC),
		Payload: map[string]any{"customer_id": "customer-1", "credits": int64(7)},
	}}}
	collector := &recordingCollector{}
	w, err := worker.New(worker.Config{Repo: repo, Collector: collector, OwnerID: "worker-1"})
	require.NoError(t, err)

	require.NoError(t, w.ProcessOnce(t.Context()))
	require.True(t, repo.rows[0].Published)
	require.Len(t, collector.events, 1)
	require.Equal(t, "outbox-1", collector.events[0].ID())
	require.Equal(t, "openmeter.credit.usage", collector.events[0].Type())
	require.Equal(t, "openmeter://credit-reservation", collector.events[0].Source())
	require.Equal(t, "subject-1", collector.events[0].Subject())
}

func TestProcessOnceRetriesThenDeadLettersWithoutPublishing(t *testing.T) {
	repo := &memoryRepo{rows: []worker.OutboxRow{{ID: "outbox-1", Namespace: "ns", EventType: "openmeter.credit.usage"}}}
	collector := &recordingCollector{err: errors.New("broker unavailable")}
	w, err := worker.New(worker.Config{Repo: repo, Collector: collector, OwnerID: "worker-1", MaxClaimCount: 1})
	require.NoError(t, err)

	err = w.ProcessOnce(t.Context())
	require.Error(t, err)
	require.False(t, repo.rows[0].Published)
	require.False(t, repo.rows[0].DeadLettered)
	require.Equal(t, 1, repo.rows[0].ClaimCount)

	require.NoError(t, w.ProcessOnce(t.Context()))
	require.False(t, repo.rows[0].Published)
	require.True(t, repo.rows[0].DeadLettered)
}

type memoryRepo struct{ rows []worker.OutboxRow }

func (r *memoryRepo) Claim(_ context.Context, owner string, _ int, _ time.Duration) ([]worker.OutboxRow, error) {
	for i := range r.rows {
		if r.rows[i].Published || r.rows[i].DeadLettered || r.rows[i].Owner != "" {
			continue
		}
		r.rows[i].Owner = owner
		r.rows[i].ClaimCount++
		return []worker.OutboxRow{r.rows[i]}, nil
	}
	return nil, nil
}

func (r *memoryRepo) MarkPublished(_ context.Context, owner, id string) error {
	for i := range r.rows {
		if r.rows[i].ID == id && r.rows[i].Owner == owner {
			r.rows[i].Published = true
			r.rows[i].Owner = ""
		}
	}
	return nil
}

func (r *memoryRepo) Release(_ context.Context, owner, id string) error {
	for i := range r.rows {
		if r.rows[i].ID == id && r.rows[i].Owner == owner {
			r.rows[i].Owner = ""
		}
	}
	return nil
}

func (r *memoryRepo) MarkDeadLetter(_ context.Context, owner, id, _ string) error {
	for i := range r.rows {
		if r.rows[i].ID == id && r.rows[i].Owner == owner {
			r.rows[i].DeadLettered = true
			r.rows[i].Owner = ""
		}
	}
	return nil
}

type recordingCollector struct {
	events []event.Event
	err    error
}

func (c *recordingCollector) Ingest(_ context.Context, _ string, event event.Event) error {
	if c.err != nil {
		return c.err
	}
	c.events = append(c.events, event)
	return nil
}

func (c *recordingCollector) Close() {}
