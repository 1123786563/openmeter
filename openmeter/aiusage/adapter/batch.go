package adapter

import (
	"context"
	"fmt"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/aiusage"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ent/db/aiusagebatch"
)

// GetBatchByIdempotencyKey looks up an existing batch by (namespace, customer_id, usage_batch_id).
// Returns nil (not an error) when no batch is found.
func (t *txAdapter) GetBatchByIdempotencyKey(ctx context.Context, namespace, customerID, key string) (*aiusage.AIUsageBatch, error) {
	ent, err := t.db.AIUsageBatch.Query().
		Where(
			aiusagebatch.Namespace(namespace),
			aiusagebatch.CustomerIDEQ(customerID),
			aiusagebatch.UsageBatchID(key),
		).
		WithLineItems().
		WithRatingSnapshots().
		Only(ctx)
	if err != nil {
		if entdb.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("query existing batch: %w", err)
	}

	return mapEntityToBatch(ent), nil
}

// CreateSettledBatch persists a fully rated and settled batch atomically.
//
// Returns (batch, created, error):
//   - If a batch with the same (namespace, customer_id, usage_batch_id) exists and
//     has the same payload_hash, the existing batch is returned with created=false.
//   - If the hash differs, ErrIdempotencyConflict is returned.
//   - Otherwise a new batch is created and returned with created=true.
//
// Side effects: batch row, line items, rating snapshots, allocations, outbox events,
// and a watermark advance — all in the caller's transaction.
func (t *txAdapter) CreateSettledBatch(ctx context.Context, in aiusage.SettledBatch) (*aiusage.AIUsageBatch, bool, error) {
	// ---- idempotency check ----
	existing, err := t.GetBatchByIdempotencyKey(ctx, in.Namespace, in.CustomerID, in.UsageBatchID)
	if err != nil {
		return nil, false, err
	}
	if existing != nil {
		if existing.PayloadHash == in.PayloadHash {
			return existing, false, nil
		}
		return nil, false, aiusage.ErrIdempotencyConflict
	}

	// ---- advance watermark (gap-aware) ----
	covered, err := t.AdvanceWatermark(ctx, in.Namespace, in.SubjectID, in.TenantSeq)
	if err != nil {
		return nil, false, fmt.Errorf("advance watermark: %w", err)
	}

	// ---- create batch row ----
	batchCreate := t.db.AIUsageBatch.Create().
		SetNamespace(in.Namespace).
		SetCustomerID(in.CustomerID).
		SetSubjectID(in.SubjectID).
		SetUsageBatchID(in.UsageBatchID).
		SetTenantSeq(in.TenantSeq).
		SetOccurredAt(in.OccurredAt).
		SetRateVersion(in.RateVersion).
		SetBillingMode(string(in.BillingMode)).
		SetPayloadHash(in.PayloadHash).
		SetStatus(string(in.Status)).
		SetTotalCredits(in.TotalCredits).
		SetCoveredTenantSeq(covered).
		SetSettlementScope(aiusagebatch.SettlementScope(string(in.SettlementScope)))

	if in.ReservationID != nil {
		batchCreate = batchCreate.SetReservationID(*in.ReservationID)
	}
	if in.CeilingCredits != nil {
		batchCreate = batchCreate.SetCeilingCredits(*in.CeilingCredits)
	}

	ent, err := batchCreate.Save(ctx)
	if err != nil {
		return nil, false, fmt.Errorf("create batch: %w", err)
	}

	// ---- line items ----
	for _, item := range in.LineItems {
		builder := t.db.AIUsageLineItem.Create().
			SetNamespace(in.Namespace).
			SetResourceCode(string(item.ResourceCode)).
			SetQuantity(item.Quantity).
			SetProvider(item.Provider).
			SetModel(item.Model).
			SetProviderManaged(item.ProviderManaged).
			SetBatchID(ent.ID)

		if item.Dimensions != nil {
			builder = builder.SetDimensions(item.Dimensions)
		}

		if _, err := builder.Save(ctx); err != nil {
			return nil, false, fmt.Errorf("create line item: %w", err)
		}
	}

	// ---- rating snapshots ----
	for _, snap := range in.RatingSnapshots {
		if _, err := t.db.AIUsageRatingSnapshot.Create().
			SetNamespace(in.Namespace).
			SetResourceCode(string(snap.ResourceCode)).
			SetCostCurrency(snap.CostSnapshot.Currency).
			SetCostAmount(snap.CostSnapshot.Amount).
			SetCostSource(snap.CostSnapshot.Source).
			SetSalesCurrency(snap.SalesSnapshot.Currency).
			SetSalesAmount(snap.SalesSnapshot.Amount).
			SetRateCardVersion(snap.SalesSnapshot.RateCardVersion).
			SetCredits(snap.Credits).
			SetBatchID(ent.ID).
			Save(ctx); err != nil {
			return nil, false, fmt.Errorf("create rating snapshot: %w", err)
		}
	}

	// ---- allocations ----
	for _, alloc := range in.Allocations {
		if _, err := t.db.AIUsageAllocation.Create().
			SetNamespace(in.Namespace).
			SetCustomerID(in.CustomerID).
			SetSubjectID(in.SubjectID).
			SetGrantID(alloc.GrantID).
			SetAmount(alpacadecimal.NewFromInt(alloc.Amount)).
			SetPriority(alloc.Priority).
			SetFundingSource(string(alloc.FundingSource)).
			SetBatchID(ent.ID).
			Save(ctx); err != nil {
			return nil, false, fmt.Errorf("create allocation: %w", err)
		}
	}

	// ---- outbox events ----
	if err := t.AppendOutbox(ctx, in.Namespace, in.CustomerID, in.SubjectID, in.OutboxEvents, ent.ID); err != nil {
		return nil, false, fmt.Errorf("append outbox: %w", err)
	}

	return mapEntityToBatch(ent), true, nil
}

// ---- mapping helpers ----

func mapEntityToBatch(ent *entdb.AIUsageBatch) *aiusage.AIUsageBatch {
	batch := &aiusage.AIUsageBatch{
		Namespace:      ent.Namespace,
		CustomerID:     ent.CustomerID,
		SubjectID:      ent.SubjectID,
		UsageBatchID:   ent.UsageBatchID,
		TenantSeq:      ent.TenantSeq,
		OccurredAt:     ent.OccurredAt,
		ReservationID:  ent.ReservationID,
		CeilingCredits: ent.CeilingCredits,
		RateVersion:    ent.RateVersion,
		BillingMode:    aiusage.BillingMode(ent.BillingMode),
		PayloadHash:    ent.PayloadHash,
		Status:         aiusage.BatchStatus(ent.Status),
		LineItems:      []aiusage.UsageLineItem{},
	}

	if ent.Edges.LineItems != nil {
		for _, li := range ent.Edges.LineItems {
			batch.LineItems = append(batch.LineItems, aiusage.UsageLineItem{
				ResourceCode:    aiusage.ResourceCode(li.ResourceCode),
				Quantity:        li.Quantity,
				Provider:        li.Provider,
				Model:           li.Model,
				ProviderManaged: li.ProviderManaged,
				Dimensions:      li.Dimensions,
			})
		}
	}

	return batch
}
