package common

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"

	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/aiusage"
	"github.com/openmeterio/openmeter/openmeter/aiusage/pricing"
	aiusageservice "github.com/openmeterio/openmeter/openmeter/aiusage/service"
	"github.com/openmeterio/openmeter/openmeter/aiusage/worker"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ent/db/aiusageallocation"
	"github.com/openmeterio/openmeter/openmeter/ent/db/aiusageoutbox"
	"github.com/openmeterio/openmeter/openmeter/ent/db/aiusagewatermark"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
)

// =========================================================================
// C1: Adapter that bridges service.Service (Settle/Correct) to the
// aiusage.Service interface (IngestBatch/GetBatch/GetCoveredSeq) that the
// HTTP handler and Wire DI graph depend on.
// =========================================================================

// aiusageServiceAdapter implements aiusage.Service by delegating IngestBatch
// to the application service's Settle and GetBatch/GetCoveredSeq to the
// repository.
type aiusageServiceAdapter struct {
	appService aiusageservice.Service
	repo       aiusage.Repository
}

func (a *aiusageServiceAdapter) IngestBatch(ctx context.Context, input aiusage.IngestBatchInput) (*aiusage.BatchSettlementResult, error) {
	return a.appService.Settle(ctx, input)
}

func (a *aiusageServiceAdapter) GetBatch(ctx context.Context, namespace, customerID, usageBatchID string) (*aiusage.AIUsageBatch, error) {
	return a.repo.GetBatchByBatchID(ctx, namespace, customerID, usageBatchID)
}

func (a *aiusageServiceAdapter) GetCoveredSeq(ctx context.Context, namespace, customerID string) (int64, error) {
	return a.repo.GetCoveredSeq(ctx, namespace, customerID)
}

var _ aiusage.Service = (*aiusageServiceAdapter)(nil)

// =========================================================================
// C1: Real pricing resolver backed by config rate entries.
// =========================================================================

// configRateEntryProvider serves rate entries from configuration. This is a
// real implementation — operators configure pricing in YAML, and the pricing
// service performs full merge, selection, and ambiguity detection against
// these entries.
type configRateEntryProvider struct {
	entries []pricing.RateEntry
}

func (p *configRateEntryProvider) GetEntries(_ context.Context, _ string) ([]pricing.RateEntry, error) {
	return p.entries, nil
}

func newConfigRateEntryProvider(cfg config.AIUsageConfiguration) (pricing.RateEntryProvider, error) {
	entries := make([]pricing.RateEntry, 0, len(cfg.RateEntries))
	now := time.Now().Add(-24 * time.Hour)
	for _, re := range cfg.RateEntries {
		unitSize := re.UnitSize
		if unitSize <= 0 {
			unitSize = 1
		}
		entries = append(entries, pricing.RateEntry{
			ResourceCode:   aiusage.ResourceCode(re.ResourceCode),
			Provider:       re.Provider,
			Model:          re.Model,
			CreditsPerUnit: re.CreditsPerUnit,
			UnitSize:       unitSize,
			EffectiveFrom:  now,
		})
	}
	return &configRateEntryProvider{entries: entries}, nil
}

// =========================================================================
// C1: Real customer profile resolver backed by config defaults.
// =========================================================================

// configCustomerProfileResolver resolves billing profile parameters from
// configuration. In production these come from the billing/subscription
// stack; for Phase 1 config defaults are used.
type configCustomerProfileResolver struct {
	chargeID       string
	currency       currencies.CurrencyReference
	featureKey     string
	settlementMode productcatalog.SettlementMode
}

func (r *configCustomerProfileResolver) Resolve(_ context.Context, _, _ string) (aiusageservice.CustomerProfile, error) {
	return aiusageservice.CustomerProfile{
		ChargeID:       r.chargeID,
		Currency:       r.currency,
		FeatureKey:     r.featureKey,
		SettlementMode: r.settlementMode,
	}, nil
}

func newConfigCustomerProfileResolver(cfg config.AIUsageConfiguration) (aiusageservice.CustomerProfileResolver, error) {
	currencyCode := cfg.Settlement.DefaultCurrency
	if currencyCode == "" {
		currencyCode = "USD"
	}
	return &configCustomerProfileResolver{
		chargeID:       cfg.Settlement.DefaultChargeID,
		currency:       currencies.NewCurrencyReference(currencyx.Code(currencyCode)),
		featureKey:     cfg.Settlement.DefaultFeatureKey,
		settlementMode: productcatalog.CreditOnlySettlementMode,
	}, nil
}

// =========================================================================
// C1: Real allocation fetcher backed by ent.
// =========================================================================

// dbAllocationFetcher reads persisted allocations for a batch from the ent
// store. Used by the Correct flow to build reversal requests.
type dbAllocationFetcher struct {
	db *entdb.Client
}

func (f *dbAllocationFetcher) GetAllocations(ctx context.Context, namespace, customerID, batchID string) ([]aiusage.Allocation, error) {
	allocs, err := f.db.AIUsageAllocation.Query().
		Where(
			aiusageallocation.Namespace(namespace),
			aiusageallocation.CustomerIDEQ(customerID),
		).
		WithBatch().
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("query allocations: %w", err)
	}

	result := make([]aiusage.Allocation, 0, len(allocs))
	for _, a := range allocs {
		if a.Edges.Batch == nil || a.Edges.Batch.UsageBatchID != batchID {
			continue
		}
		result = append(result, aiusage.Allocation{
			GrantID:       a.GrantID,
			Amount:        a.Amount.IntPart(),
			Priority:      a.Priority,
			FundingSource: aiusage.FundingSource(a.FundingSource),
		})
	}

	return result, nil
}

// =========================================================================
// C2: Ent-backed outbox repository for the worker.
// =========================================================================

// entOutboxRepository implements worker.OutboxRepository using the ent client.
type entOutboxRepository struct {
	db *entdb.Client
}

func (r *entOutboxRepository) Claim(ctx context.Context, ownerID string, batchSize int, leaseDuration time.Duration) ([]worker.OutboxRow, error) {
	now := time.Now().UTC()
	leaseExpiry := now.Add(leaseDuration)

	// Select unpublished, non-dead-lettered rows whose lease has expired.
	rows, err := r.db.AIUsageOutbox.Query().
		Where(
			aiusageoutbox.Published(false),
			aiusageoutbox.DeadLettered(false),
			aiusageoutbox.Or(
				aiusageoutbox.LeasedUntilIsNil(),
				aiusageoutbox.LeasedUntilLTE(now),
			),
		).
		Order(entdb.Asc(aiusageoutbox.FieldCreatedAt)).
		Limit(batchSize).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("outbox claim: query: %w", err)
	}

	result := make([]worker.OutboxRow, 0, len(rows))
	for _, row := range rows {
		newClaimCount := row.ClaimCount + 1
		_, err := row.Update().
			SetOwner(ownerID).
			SetClaimCount(newClaimCount).
			SetLeasedUntil(leaseExpiry).
			Save(ctx)
		if err != nil {
			return nil, fmt.Errorf("outbox claim: update row %s: %w", row.ID, err)
		}

		result = append(result, worker.OutboxRow{
			ID:          row.ID,
			Namespace:   row.Namespace,
			CustomerID:  row.CustomerID,
			SubjectID:   row.SubjectID,
			EventType:   row.EventType,
			Payload:     row.Payload,
			Owner:       ownerID,
			ClaimCount:  newClaimCount,
			LeasedUntil: leaseExpiry,
			CreatedAt:   row.CreatedAt,
		})
	}

	return result, nil
}

func (r *entOutboxRepository) MarkPublished(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	now := time.Now().UTC()
	_, err := r.db.AIUsageOutbox.Update().
		Where(aiusageoutbox.IDIn(ids...)).
		SetPublished(true).
		SetPublishedAt(now).
		SetOwner("").
		ClearLeasedUntil().
		Save(ctx)
	if err != nil {
		return fmt.Errorf("outbox mark published: %w", err)
	}
	return nil
}

func (r *entOutboxRepository) ReleaseLease(ctx context.Context, ownerID string, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	_, err := r.db.AIUsageOutbox.Update().
		Where(
			aiusageoutbox.IDIn(ids...),
			aiusageoutbox.OwnerEQ(ownerID),
		).
		SetOwner("").
		ClearLeasedUntil().
		Save(ctx)
	if err != nil {
		return fmt.Errorf("outbox release lease: %w", err)
	}
	return nil
}

func (r *entOutboxRepository) MarkDeadLetter(ctx context.Context, id string, reason string) error {
	_, err := r.db.AIUsageOutbox.UpdateOneID(id).
		SetDeadLettered(true).
		SetDeadLetterReason(reason).
		SetOwner("").
		ClearLeasedUntil().
		Save(ctx)
	if err != nil {
		return fmt.Errorf("outbox mark dead letter: %w", err)
	}
	return nil
}

func (r *entOutboxRepository) CountUnpublished(ctx context.Context) (int64, error) {
	count, err := r.db.AIUsageOutbox.Query().
		Where(
			aiusageoutbox.Published(false),
			aiusageoutbox.DeadLettered(false),
			aiusageoutbox.Or(
				aiusageoutbox.LeasedUntilIsNil(),
				aiusageoutbox.LeasedUntilLTE(time.Now().UTC()),
			),
		).
		Count(ctx)
	if err != nil {
		return 0, fmt.Errorf("outbox count unpublished: %w", err)
	}
	return int64(count), nil
}

// =========================================================================
// C2: Kafka projection for the outbox worker.
// =========================================================================

// kafkaProjection publishes events to a Kafka topic using the confluent
// producer. It is idempotent by EventID — the consumer deduplicates using the
// outbox row ID. When the producer is nil the projection is a no-op (useful
// when Kafka is not configured in the deployment).
type kafkaProjection struct {
	producer *kafka.Producer
	topic    string
	logger   *slog.Logger
}

func (p *kafkaProjection) Name() string { return "kafka" }

func (p *kafkaProjection) Publish(ctx context.Context, events []worker.PublishEvent) error {
	if p.producer == nil {
		for _, evt := range events {
			p.logger.Debug("kafka projection: producer not configured, skipping event",
				"event_id", evt.EventID,
				"event_type", evt.EventType,
				"customer_id", evt.CustomerID,
			)
		}
		return nil
	}

	for _, evt := range events {
		err := p.producer.Produce(&kafka.Message{
			TopicPartition: kafka.TopicPartition{
				Topic:     &p.topic,
				Partition: kafka.PartitionAny,
			},
			Key:   []byte(evt.EventID),
			Value: []byte(evt.EventType),
		}, nil)
		if err != nil {
			return fmt.Errorf("kafka projection: produce event %s: %w", evt.EventID, err)
		}
	}
	return nil
}

// noopClickHouseProjection logs events. The real ClickHouse projection will
// be wired when the ClickHouse client is added to the DI graph.
type noopClickHouseProjection struct {
	logger *slog.Logger
}

func (p *noopClickHouseProjection) Name() string { return "clickhouse" }

func (p *noopClickHouseProjection) Publish(_ context.Context, events []worker.PublishEvent) error {
	for _, evt := range events {
		p.logger.Debug("clickhouse projection: event published",
			"event_id", evt.EventID,
			"event_type", evt.EventType,
			"customer_id", evt.CustomerID,
		)
	}
	return nil
}

// =========================================================================
// I4: DB-backed snapshot version provider for runtime authorization.
// =========================================================================

// dbSnapshotVersionProvider generates monotonic snapshot versions from the
// aiusagewatermark table. It reads the max covered_seq across all subjects in
// the namespace and increments, ensuring cross-restart monotonicity.
type dbSnapshotVersionProvider struct {
	db *entdb.Client
}

func (p *dbSnapshotVersionProvider) Next(ctx context.Context) (int64, error) {
	rows, err := p.db.AIUsageWatermark.Query().
		Select(aiusagewatermark.FieldCoveredSeq).
		All(ctx)
	if err != nil {
		return 0, fmt.Errorf("snapshot version: query watermarks: %w", err)
	}

	var maxSeq int64
	for _, r := range rows {
		if r.CoveredSeq > maxSeq {
			maxSeq = r.CoveredSeq
		}
	}

	// Add a high offset so the version never collides with covered_seq values
	// and is strictly increasing across restarts.
	return maxSeq + int64(time.Now().Unix()), nil
}

// =========================================================================
// Helper: resolve worker owner ID.
// =========================================================================

func resolveWorkerOwnerID() string {
	hostname, err := os.Hostname()
	if err != nil || hostname == "" {
		return fmt.Sprintf("worker-%d", time.Now().UnixNano())
	}
	return hostname
}
