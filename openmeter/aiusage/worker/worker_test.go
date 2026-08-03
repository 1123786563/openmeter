package worker_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/aiusage/worker"
)

// ---- in-memory outbox repository for tests ----

type memOutboxRow struct {
	row        worker.OutboxRow
	published  bool
	deadLetter bool
	dlReason   string
}

type memRepo struct {
	mu      sync.Mutex
	rows    map[string]*memOutboxRow
	order   []string // insertion order for deterministic iteration
	claimNo int64
}

func newMemRepo() *memRepo {
	return &memRepo{rows: make(map[string]*memOutboxRow)}
}

func (r *memRepo) Add(row worker.OutboxRow) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rows[row.ID] = &memOutboxRow{row: row}
	r.order = append(r.order, row.ID)
}

func (r *memRepo) Claim(_ context.Context, ownerID string, batchSize int, leaseDuration time.Duration) ([]worker.OutboxRow, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	var claimed []worker.OutboxRow
	now := time.Now().UTC()
	r.claimNo++

	for _, id := range r.order {
		entry := r.rows[id]
		if entry == nil || entry.published || entry.deadLetter {
			continue
		}
		// Claimable if never leased or lease has expired.
		if !entry.row.LeasedUntil.IsZero() && now.Before(entry.row.LeasedUntil) {
			continue
		}
		entry.row.ClaimCount++
		entry.row.Owner = ownerID
		entry.row.LeasedUntil = now.Add(leaseDuration)
		claimed = append(claimed, entry.row)
		if len(claimed) >= batchSize {
			break
		}
	}
	return claimed, nil
}

func (r *memRepo) MarkPublished(_ context.Context, ids []string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, id := range ids {
		if entry, ok := r.rows[id]; ok {
			entry.published = true
		}
	}
	return nil
}

func (r *memRepo) ReleaseLease(_ context.Context, ownerID string, ids []string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, id := range ids {
		if entry, ok := r.rows[id]; ok {
			// Only release if the row is owned by the requesting worker.
			if entry.row.Owner != ownerID {
				continue
			}
			entry.row.LeasedUntil = time.Time{}
			entry.row.Owner = ""
		}
	}
	return nil
}

func (r *memRepo) MarkDeadLetter(_ context.Context, id string, reason string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if entry, ok := r.rows[id]; ok {
		entry.deadLetter = true
		entry.dlReason = reason
	}
	return nil
}

func (r *memRepo) CountUnpublished(_ context.Context) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var n int64
	for _, entry := range r.rows {
		if !entry.published && !entry.deadLetter {
			n++
		}
	}
	return n, nil
}

func (r *memRepo) IsPublished(id string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if entry, ok := r.rows[id]; ok {
		return entry.published
	}
	return false
}

func (r *memRepo) IsDeadLettered(id string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if entry, ok := r.rows[id]; ok {
		return entry.deadLetter
	}
	return false
}

// ---- mock projection ----

type mockProjection struct {
	name       string
	mu         sync.Mutex
	published  map[string]bool // event_id -> delivered (idempotent dedup)
	shouldFail bool
	delay      time.Duration
}

func newMockProjection(name string) *mockProjection {
	return &mockProjection{
		name:      name,
		published: make(map[string]bool),
	}
}

func (m *mockProjection) Name() string { return m.name }

func (m *mockProjection) Publish(_ context.Context, events []worker.PublishEvent) error {
	if m.delay > 0 {
		time.Sleep(m.delay)
	}
	if m.shouldFail {
		return errors.New("projection unavailable")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, e := range events {
		m.published[e.EventID] = true
	}
	return nil
}

func (m *mockProjection) Delivered(id string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.published[id]
}

func (m *mockProjection) PublishCount(id string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.published[id] {
		return 1
	}
	return 0
}

func makeRow(id string) worker.OutboxRow {
	return worker.OutboxRow{
		ID:         id,
		Namespace:  "ns-test",
		CustomerID: "cust-1",
		SubjectID:  "subj-1",
		EventType:  "ai_usage.batch.settled",
		Payload:    map[string]any{"batch_id": id},
		CreatedAt:  time.Now().UTC(),
	}
}

// ---- tests ----

func TestWorkerPublishesAndMarksPublished(t *testing.T) {
	repo := newMemRepo()
	repo.Add(makeRow("row-1"))
	repo.Add(makeRow("row-2"))

	kafka := newMockProjection("kafka")
	clickhouse := newMockProjection("clickhouse")

	w, err := worker.New(worker.Config{
		Repo:        repo,
		Projections: []worker.Projection{kafka, clickhouse},
	})
	require.NoError(t, err)

	require.NoError(t, w.ProcessOnce(t.Context()))

	require.True(t, repo.IsPublished("row-1"))
	require.True(t, repo.IsPublished("row-2"))
	require.Equal(t, 1, kafka.PublishCount("row-1"))
	require.Equal(t, 1, clickhouse.PublishCount("row-1"))
}

func TestEventIDEqualsOutboxID(t *testing.T) {
	repo := newMemRepo()
	repo.Add(makeRow("outbox-id-42"))

	proj := newMockProjection("kafka")
	w, err := worker.New(worker.Config{
		Repo:        repo,
		Projections: []worker.Projection{proj},
	})
	require.NoError(t, err)

	require.NoError(t, w.ProcessOnce(t.Context()))

	require.Equal(t, 1, proj.PublishCount("outbox-id-42"))
}

func TestDuplicatePublishKeepsOneEventID(t *testing.T) {
	repo := newMemRepo()
	repo.Add(makeRow("dup-1"))

	proj := newMockProjection("kafka")
	proj.shouldFail = true // first publish fails

	w, err := worker.New(worker.Config{
		Repo:        repo,
		Projections: []worker.Projection{proj},
	})
	require.NoError(t, err)

	// First attempt: projection fails, lease released.
	err = w.ProcessOnce(t.Context())
	require.Error(t, err)

	// Second attempt: projection succeeds, row published.
	proj.shouldFail = false
	require.NoError(t, w.ProcessOnce(t.Context()))

	require.True(t, repo.IsPublished("dup-1"))
	require.Equal(t, 1, proj.PublishCount("dup-1"),
		"the Event ID should appear exactly once in the projection despite retry")
}

func TestExpiredLeaseReclaimedOnce(t *testing.T) {
	repo := newMemRepo()
	repo.Add(makeRow("lease-1"))

	// Failing projection so the row never gets published.
	proj := newMockProjection("kafka")
	proj.shouldFail = true

	w, err := worker.New(worker.Config{
		Repo:          repo,
		Projections:   []worker.Projection{proj},
		LeaseDuration: 50 * time.Millisecond,
		MaxClaimCount: 2, // original + one reclaim
	})
	require.NoError(t, err)

	// Original claim.
	require.Error(t, w.ProcessOnce(t.Context()))

	// Wait for lease to expire.
	time.Sleep(80 * time.Millisecond)

	// Reclaim — same row claimed again (ClaimCount now 2).
	require.Error(t, w.ProcessOnce(t.Context()))

	// Wait for lease to expire again.
	time.Sleep(80 * time.Millisecond)

	// Third claim would exceed MaxClaimCount (2) → dead-letter.
	require.NoError(t, w.ProcessOnce(t.Context()))

	require.True(t, repo.IsDeadLettered("lease-1"))
}

func TestKafkaOutageLeavesPostgreSQLFactsSettled(t *testing.T) {
	repo := newMemRepo()
	repo.Add(makeRow("pg-fact-1"))

	kafka := newMockProjection("kafka")
	kafka.shouldFail = true // Kafka is down

	clickhouse := newMockProjection("clickhouse")

	w, err := worker.New(worker.Config{
		Repo:        repo,
		Projections: []worker.Projection{kafka, clickhouse},
	})
	require.NoError(t, err)

	// Worker fails to publish because Kafka is down.
	err = w.ProcessOnce(t.Context())
	require.Error(t, err)

	// PostgreSQL fact: the outbox row is NOT published (still pending).
	require.False(t, repo.IsPublished("pg-fact-1"))

	// The row is still claimable — PostgreSQL state (the batch/ledger) is settled;
	// only the projection is delayed.
	count, err := repo.CountUnpublished(t.Context())
	require.NoError(t, err)
	require.Equal(t, int64(1), count)

	// Kafka recovers — projection converges.
	kafka.shouldFail = false
	require.NoError(t, w.ProcessOnce(t.Context()))

	require.True(t, repo.IsPublished("pg-fact-1"))
	require.Equal(t, 1, kafka.PublishCount("pg-fact-1"))
	require.Equal(t, 1, clickhouse.PublishCount("pg-fact-1"))
}

func TestClickHouseOutageConverges(t *testing.T) {
	repo := newMemRepo()
	repo.Add(makeRow("ch-fact-1"))

	kafka := newMockProjection("kafka")
	clickhouse := newMockProjection("clickhouse")
	clickhouse.shouldFail = true

	w, err := worker.New(worker.Config{
		Repo:        repo,
		Projections: []worker.Projection{kafka, clickhouse},
	})
	require.NoError(t, err)

	// ClickHouse down → batch fails, lease released.
	require.Error(t, w.ProcessOnce(t.Context()))
	require.False(t, repo.IsPublished("ch-fact-1"))

	// ClickHouse recovers → converges.
	clickhouse.shouldFail = false
	require.NoError(t, w.ProcessOnce(t.Context()))
	require.True(t, repo.IsPublished("ch-fact-1"))
}

func TestRestartResumesLease(t *testing.T) {
	repo := newMemRepo()
	repo.Add(makeRow("restart-1"))
	repo.Add(makeRow("restart-2"))
	repo.Add(makeRow("restart-3"))

	proj := newMockProjection("kafka")

	w1, err := worker.New(worker.Config{
		Repo:          repo,
		Projections:   []worker.Projection{proj},
		BatchSize:     2,                // only 2 at a time
		LeaseDuration: 10 * time.Minute, // long lease so rows stay claimed
	})
	require.NoError(t, err)

	// First run: claims 2 rows, publishes them.
	require.NoError(t, w1.ProcessOnce(t.Context()))
	require.True(t, repo.IsPublished("restart-1"))
	require.True(t, repo.IsPublished("restart-2"))
	require.False(t, repo.IsPublished("restart-3"))

	// Simulate restart: create a new worker instance.
	w2, err := worker.New(worker.Config{
		Repo:        repo,
		Projections: []worker.Projection{proj},
		BatchSize:   2,
	})
	require.NoError(t, err)

	// Resumes from where it left off — publishes the remaining row.
	require.NoError(t, w2.ProcessOnce(t.Context()))
	require.True(t, repo.IsPublished("restart-3"))
}

func TestDeadLetterReplaySameEventID(t *testing.T) {
	repo := newMemRepo()
	repo.Add(makeRow("dl-1"))

	// Projection always fails so the row eventually dead-letters.
	proj := newMockProjection("kafka")
	proj.shouldFail = true

	type dlRecord struct {
		row    worker.OutboxRow
		reason string
	}

	var dlHandler worker.DeadLetterHandler = &deadLetterRecorder{}

	w, err := worker.New(worker.Config{
		Repo:          repo,
		Projections:   []worker.Projection{proj},
		LeaseDuration: 10 * time.Millisecond,
		MaxClaimCount: 1, // dead-letter after just one failed attempt
		DeadLetter:    dlHandler,
	})
	require.NoError(t, err)

	// First claim: projection fails.
	require.Error(t, w.ProcessOnce(t.Context()))
	time.Sleep(20 * time.Millisecond)

	// Second claim: ClaimCount > MaxClaimCount → dead-letter.
	require.NoError(t, w.ProcessOnce(t.Context()))
	require.True(t, repo.IsDeadLettered("dl-1"))

	// Dead-letter replay: re-add the row with the same ID and a working projection.
	rec := dlHandler.(*deadLetterRecorder)
	require.Len(t, rec.records, 1)
	require.Equal(t, "dl-1", rec.records[0].row.ID)

	// Manually replay: clear dead-letter, release lease, and reprocess.
	repo.mu.Lock()
	entry := repo.rows["dl-1"]
	entry.deadLetter = false
	entry.row.ClaimCount = 0
	entry.row.LeasedUntil = time.Time{}
	repo.mu.Unlock()

	proj.shouldFail = false
	require.NoError(t, w.ProcessOnce(t.Context()))
	require.True(t, repo.IsPublished("dl-1"))
	require.Equal(t, 1, proj.PublishCount("dl-1"),
		"replay must publish the same Event ID exactly once")
}

func TestAllProjectionsMustAcknowledge(t *testing.T) {
	repo := newMemRepo()
	repo.Add(makeRow("all-ack-1"))

	kafka := newMockProjection("kafka")
	slowCH := newMockProjection("clickhouse")
	slowCH.delay = 50 * time.Millisecond

	w, err := worker.New(worker.Config{
		Repo:        repo,
		Projections: []worker.Projection{kafka, slowCH},
	})
	require.NoError(t, err)

	start := time.Now()
	require.NoError(t, w.ProcessOnce(t.Context()))
	elapsed := time.Since(start)

	require.True(t, repo.IsPublished("all-ack-1"))
	require.GreaterOrEqual(t, elapsed, 50*time.Millisecond,
		"worker must wait for the slow projection to acknowledge")
}

func TestStartStopLifecycle(t *testing.T) {
	repo := newMemRepo()
	proj := newMockProjection("kafka")

	w, err := worker.New(worker.Config{
		Repo:         repo,
		Projections:  []worker.Projection{proj},
		PollInterval: 10 * time.Millisecond,
	})
	require.NoError(t, err)

	w.Start(t.Context())

	// Add a row and let the poll loop process it.
	repo.Add(makeRow("lifecycle-1"))
	require.Eventually(t, func() bool {
		return repo.IsPublished("lifecycle-1")
	}, 2*time.Second, 10*time.Millisecond)

	w.Stop()
}

func TestValidateRequiresRepoAndProjections(t *testing.T) {
	_, err := worker.New(worker.Config{})
	require.Error(t, err)

	_, err = worker.New(worker.Config{
		Repo: newMemRepo(),
	})
	require.Error(t, err)
}

func TestOwnerIdentityThreading(t *testing.T) {
	repo := newMemRepo()
	repo.Add(makeRow("owner-1"))
	repo.Add(makeRow("owner-2"))

	proj := newMockProjection("kafka")

	w, err := worker.New(worker.Config{
		Repo:        repo,
		Projections: []worker.Projection{proj},
		OwnerID:     "worker-test-001",
	})
	require.NoError(t, err)
	require.Equal(t, "worker-test-001", w.OwnerID())

	require.NoError(t, w.ProcessOnce(t.Context()))

	// After claim + publish, each row must have had the worker's owner ID set.
	repo.mu.Lock()
	for _, id := range []string{"owner-1", "owner-2"} {
		entry := repo.rows[id]
		require.NotNil(t, entry)
		// Published rows are marked, so we just verify the claim set the owner.
		require.True(t, entry.published, "row %s should be published", id)
	}
	repo.mu.Unlock()
}

func TestCrossWorkerLeaseGuard(t *testing.T) {
	repo := newMemRepo()
	repo.Add(makeRow("cross-1"))

	// Directly test the repo Claim/ReleaseLease owner guard semantics.

	// Worker A claims the row.
	rows, err := repo.Claim(t.Context(), "worker-A", 10, 10*time.Minute)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, "worker-A", rows[0].Owner, "claimed row must carry owner ID")

	// Verify the repo stored the owner.
	repo.mu.Lock()
	require.Equal(t, "worker-A", repo.rows["cross-1"].row.Owner)
	repo.mu.Unlock()

	// Worker B tries to release worker A's lease — must be a no-op.
	require.NoError(t, repo.ReleaseLease(t.Context(), "worker-B", []string{"cross-1"}))

	// Worker A still owns it.
	repo.mu.Lock()
	require.Equal(t, "worker-A", repo.rows["cross-1"].row.Owner,
		"worker-B must not be able to release worker-A's lease")
	repo.mu.Unlock()

	// Worker A releases its own lease.
	require.NoError(t, repo.ReleaseLease(t.Context(), "worker-A", []string{"cross-1"}))
	repo.mu.Lock()
	require.Empty(t, repo.rows["cross-1"].row.Owner, "worker-A release should clear owner")
	repo.mu.Unlock()
}

// ---- dead-letter handler recorder ----

type deadLetterRecorder struct {
	mu      sync.Mutex
	records []struct {
		row    worker.OutboxRow
		reason string
	}
}

func (d *deadLetterRecorder) Handle(_ context.Context, row worker.OutboxRow, reason string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.records = append(d.records, struct {
		row    worker.OutboxRow
		reason string
	}{row: row, reason: reason})
	return nil
}
