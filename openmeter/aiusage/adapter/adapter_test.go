package adapter_test

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/aiusage"
	aiusageadapter "github.com/openmeterio/openmeter/openmeter/aiusage/adapter"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ent/db/aiusagebatch"
	"github.com/openmeterio/openmeter/openmeter/testutils"
)

func setupAdapter(t *testing.T) (aiusageadapter.Adapter, *entdb.Client) {
	t.Helper()

	testDB := testutils.InitPostgresDB(t, testutils.PostgresDBStateAtlasMigrated)
	t.Cleanup(func() { testDB.Close(t) })

	dbClient := entdb.NewClient(entdb.Driver(testDB.EntDriver.Driver()))

	adp, err := aiusageadapter.New(aiusageadapter.Config{
		Client: dbClient,
		Logger: slog.Default(),
	})
	require.NoError(t, err)

	return adp, dbClient
}

func makeSettledBatch(namespace, customerID, subjectID, batchID string, seq int64, hash string) aiusage.SettledBatch {
	return aiusage.SettledBatch{
		Namespace:       namespace,
		CustomerID:      customerID,
		SubjectID:       subjectID,
		UsageBatchID:    batchID,
		TenantSeq:       seq,
		OccurredAt:      time.Now().UTC(),
		RateVersion:     "v1",
		BillingMode:     aiusage.BillingModeComponent,
		PayloadHash:     hash,
		SettlementScope: aiusage.SettlementScopeFormal,
		Status:          aiusage.BatchStatusSettled,
		TotalCredits:    10,
		LineItems: []aiusage.UsageLineItem{
			{
				ResourceCode:    aiusage.ResourceCode("llm.input_tokens"),
				Quantity:        1000,
				Provider:        "openai",
				Model:           "gpt-4",
				ProviderManaged: true,
			},
		},
		RatingSnapshots: []aiusage.RatingSnapshot{
			{
				ResourceCode: aiusage.ResourceCode("llm.input_tokens"),
				CostSnapshot: aiusage.CostSnapshot{
					Currency: "USD",
					Amount:   alpacadecimal.NewFromFloat(0.01),
					Source:   "ratecard",
				},
				SalesSnapshot: aiusage.SalesSnapshot{
					Currency:        "CNY",
					Amount:          alpacadecimal.NewFromFloat(0.07),
					RateCardVersion: "v1",
				},
				Credits: 10,
			},
		},
		Allocations: []aiusage.Allocation{
			{
				GrantID:       "01J00000000000000000000001",
				Amount:        10,
				Priority:      0,
				FundingSource: "grant",
			},
		},
		OutboxEvents: []aiusage.OutboxEvent{
			{
				EventType: "ai_usage.batch.settled",
				Payload:   map[string]any{"batch_id": batchID},
			},
		},
	}
}

func TestCreateSettledBatchIsIdempotent(t *testing.T) {
	adp, _ := setupAdapter(t)

	ctx := context.Background()
	ns := "test-idempotent"
	customerID := "cust-1"
	subjectID := "subj-1"

	in1 := makeSettledBatch(ns, customerID, subjectID, "batch-A", 1, "hash-aaa")

	err := adp.WithCustomerLock(ctx, ns, customerID, func(tx aiusageadapter.TxAdapter) error {
		b, err := tx.CreateSettledBatch(ctx, in1)
		require.NoError(t, err)
		require.NotNil(t, b)
		require.Equal(t, "batch-A", b.UsageBatchID)
		return nil
	})
	require.NoError(t, err)

	err = adp.WithCustomerLock(ctx, ns, customerID, func(tx aiusageadapter.TxAdapter) error {
		b, err := tx.CreateSettledBatch(ctx, in1)
		require.NoError(t, err)
		require.NotNil(t, b)
		require.Equal(t, "batch-A", b.UsageBatchID)
		return nil
	})
	require.NoError(t, err)

	in2 := makeSettledBatch(ns, customerID, subjectID, "batch-A", 1, "hash-bbb")
	err = adp.WithCustomerLock(ctx, ns, customerID, func(tx aiusageadapter.TxAdapter) error {
		_, err := tx.CreateSettledBatch(ctx, in2)
		return err
	})
	require.ErrorIs(t, err, aiusage.ErrIdempotencyConflict)
}

func TestWatermarkGap(t *testing.T) {
	adp, _ := setupAdapter(t)

	ctx := context.Background()
	ns := "test-watermark-gap"
	customerID := "cust-gap"
	subjectID := "subj-gap"

	mk := func(batchID string, seq int64) aiusage.SettledBatch {
		return makeSettledBatch(ns, customerID, subjectID, batchID, seq, fmt.Sprintf("hash-%d", seq))
	}

	checkWatermark := func(expected int64) {
		t.Helper()
		var covered int64
		err := adp.WithCustomerLock(ctx, ns, customerID, func(tx aiusageadapter.TxAdapter) error {
			// seq 0 can never advance the watermark, so this reads the current value.
			c, err := tx.AdvanceWatermark(ctx, ns, subjectID, 0)
			covered = c
			return err
		})
		require.NoError(t, err)
		require.Equal(t, expected, covered, "watermark mismatch")
	}

	// seq 1 → covered = 1
	err := adp.WithCustomerLock(ctx, ns, customerID, func(tx aiusageadapter.TxAdapter) error {
		_, err := tx.CreateSettledBatch(ctx, mk("gap-1", 1))
		return err
	})
	require.NoError(t, err)
	checkWatermark(1)

	// seq 3 → gap, covered stays 1
	err = adp.WithCustomerLock(ctx, ns, customerID, func(tx aiusageadapter.TxAdapter) error {
		_, err := tx.CreateSettledBatch(ctx, mk("gap-3", 3))
		return err
	})
	require.NoError(t, err)
	checkWatermark(1)

	// seq 2 → fills the gap, covered catches up to 3
	err = adp.WithCustomerLock(ctx, ns, customerID, func(tx aiusageadapter.TxAdapter) error {
		_, err := tx.CreateSettledBatch(ctx, mk("gap-2", 2))
		return err
	})
	require.NoError(t, err)
	checkWatermark(3)
}

func TestConcurrentIdempotency(t *testing.T) {
	adp, dbClient := setupAdapter(t)

	ctx := context.Background()
	ns := "test-concurrent"
	customerID := "cust-concurrent"
	subjectID := "subj-concurrent"

	in := makeSettledBatch(ns, customerID, subjectID, "batch-concurrent", 1, "hash-concurrent")

	var wg sync.WaitGroup
	var successCount int32
	var errCount int32

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			err := adp.WithCustomerLock(ctx, ns, customerID, func(tx aiusageadapter.TxAdapter) error {
				_, err := tx.CreateSettledBatch(ctx, in)
				return err
			})

			if err == nil {
				atomic.AddInt32(&successCount, 1)
			} else {
				atomic.AddInt32(&errCount, 1)
			}
		}()
	}

	wg.Wait()

	require.Equal(t, int32(20), successCount, "all goroutines should succeed")
	require.Equal(t, int32(0), errCount, "unexpected errors")

	// Verify exactly one batch row exists.
	batchCount, err := dbClient.AIUsageBatch.Query().
		Where(
			aiusagebatch.Namespace(ns),
			aiusagebatch.UsageBatchID("batch-concurrent"),
		).
		Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, batchCount, "exactly one batch row should exist")
}
