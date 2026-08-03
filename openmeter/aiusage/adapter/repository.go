package adapter

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/openmeterio/openmeter/openmeter/aiusage"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ent/db/aiusagebatch"
)

type repository struct {
	db     *db.Client
	logger *slog.Logger
}

func NewRepository(client *db.Client, logger *slog.Logger) aiusage.Repository {
	return &repository{
		db:     client,
		logger: logger,
	}
}

func (r *repository) CreateBatch(ctx context.Context, batch aiusage.AIUsageBatch, snapshots []aiusage.RatingSnapshot) (*aiusage.BatchSettlementResult, error) {
	// Check idempotency first.
	existing, err := r.db.AIUsageBatch.Query().
		Where(aiusagebatch.Namespace(batch.Namespace), aiusagebatch.UsageBatchID(batch.UsageBatchID)).
		Only(ctx)
	if err != nil && !db.IsNotFound(err) {
		return nil, fmt.Errorf("failed to check existing batch: %w", err)
	}
	if existing != nil {
		if existing.PayloadHash == batch.PayloadHash {
			return r.mapBatchToResult(existing), nil
		}
		return nil, aiusage.ErrBatchPayloadConflict
	}

	var totalCredits int64
	for _, s := range snapshots {
		totalCredits += s.Credits
	}

	cmd := r.db.AIUsageBatch.Create().
		SetNamespace(batch.Namespace).
		SetCustomerID(batch.CustomerID).
		SetSubjectID(batch.SubjectID).
		SetUsageBatchID(batch.UsageBatchID).
		SetTenantSeq(batch.TenantSeq).
		SetOccurredAt(batch.OccurredAt).
		SetRateVersion(batch.RateVersion).
		SetBillingMode(string(batch.BillingMode)).
		SetPayloadHash(batch.PayloadHash).
		SetStatus(string(batch.Status)).
		SetTotalCredits(totalCredits).
		SetCoveredTenantSeq(batch.TenantSeq)

	if batch.ReservationID != nil {
		cmd = cmd.SetReservationID(*batch.ReservationID)
	}
	if batch.CeilingCredits != nil {
		cmd = cmd.SetCeilingCredits(*batch.CeilingCredits)
	}

	ent, err := cmd.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create batch: %w", err)
	}

	// Create line items.
	for _, item := range batch.LineItems {
		_, err := r.db.AIUsageLineItem.Create().
			SetNamespace(batch.Namespace).
			SetResourceCode(string(item.ResourceCode)).
			SetQuantity(item.Quantity).
			SetProvider(item.Provider).
			SetModel(item.Model).
			SetProviderManaged(item.ProviderManaged).
			SetBatchID(ent.ID).
			Save(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to create line item: %w", err)
		}
	}

	// Create rating snapshots.
	for _, snap := range snapshots {
		_, err := r.db.AIUsageRatingSnapshot.Create().
			SetNamespace(batch.Namespace).
			SetResourceCode(string(snap.ResourceCode)).
			SetCostCurrency(snap.CostSnapshot.Currency).
			SetCostAmount(snap.CostSnapshot.Amount).
			SetCostSource(snap.CostSnapshot.Source).
			SetSalesCurrency(snap.SalesSnapshot.Currency).
			SetSalesAmount(snap.SalesSnapshot.Amount).
			SetRateCardVersion(snap.SalesSnapshot.RateCardVersion).
			SetCredits(snap.Credits).
			SetBatchID(ent.ID).
			Save(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to create rating snapshot: %w", err)
		}
	}

	return r.mapBatchToResult(ent), nil
}

func (r *repository) GetBatchByBatchID(ctx context.Context, namespace, usageBatchID string) (*aiusage.AIUsageBatch, error) {
	ent, err := r.db.AIUsageBatch.Query().
		Where(aiusagebatch.Namespace(namespace), aiusagebatch.UsageBatchID(usageBatchID)).
		WithLineItems().
		WithRatingSnapshots().
		Only(ctx)
	if err != nil {
		if db.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}

	return r.mapEntityToBatch(ent), nil
}

func (r *repository) GetBatchResult(ctx context.Context, namespace, usageBatchID string) (*aiusage.BatchSettlementResult, error) {
	ent, err := r.db.AIUsageBatch.Query().
		Where(aiusagebatch.Namespace(namespace), aiusagebatch.UsageBatchID(usageBatchID)).
		Only(ctx)
	if err != nil {
		return nil, err
	}

	return r.mapBatchToResult(ent), nil
}

func (r *repository) GetCoveredSeq(ctx context.Context, namespace, customerID string) (int64, error) {
	batches, err := r.db.AIUsageBatch.Query().
		Where(
			aiusagebatch.Namespace(namespace),
			aiusagebatch.CustomerID(customerID),
			aiusagebatch.Status(string(aiusage.BatchStatusSettled)),
		).
		Select(aiusagebatch.FieldCoveredTenantSeq).
		All(ctx)
	if err != nil {
		return 0, err
	}

	var maxSeq int64
	for _, b := range batches {
		if b.CoveredTenantSeq > maxSeq {
			maxSeq = b.CoveredTenantSeq
		}
	}

	return maxSeq, nil
}

func (r *repository) mapBatchToResult(ent *db.AIUsageBatch) *aiusage.BatchSettlementResult {
	return &aiusage.BatchSettlementResult{
		BatchID:          ent.UsageBatchID,
		Status:           aiusage.BatchStatus(ent.Status),
		TotalCredits:     ent.TotalCredits,
		CoveredTenantSeq: ent.CoveredTenantSeq,
	}
}

func (r *repository) mapEntityToBatch(ent *db.AIUsageBatch) *aiusage.AIUsageBatch {
	batch := &aiusage.AIUsageBatch{
		Namespace:      ent.Namespace,
		CustomerID:     ent.CustomerID,
		SubjectID:      ent.SubjectID,
		UsageBatchID:   ent.UsageBatchID,
		TenantSeq:      ent.TenantSeq,
		OccurredAt:     ent.OccurredAt,
		RateVersion:    ent.RateVersion,
		BillingMode:    aiusage.BillingMode(ent.BillingMode),
		PayloadHash:    ent.PayloadHash,
		Status:         aiusage.BatchStatus(ent.Status),
		LineItems:      []aiusage.UsageLineItem{},
	}

	if ent.ReservationID != nil {
		batch.ReservationID = ent.ReservationID
	}
	if ent.CeilingCredits != nil {
		batch.CeilingCredits = ent.CeilingCredits
	}

	if ent.Edges.LineItems != nil {
		for _, li := range ent.Edges.LineItems {
			batch.LineItems = append(batch.LineItems, aiusage.UsageLineItem{
				ResourceCode:    aiusage.ResourceCode(li.ResourceCode),
				Quantity:        li.Quantity,
				Provider:        li.Provider,
				Model:           li.Model,
				ProviderManaged: li.ProviderManaged,
			})
		}
	}

	return batch
}

var _ aiusage.Repository = &repository{}
