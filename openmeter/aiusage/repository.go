package aiusage

import (
	"context"
)

// Repository persists AI Usage batches and related entities.
type Repository interface {
	// CreateBatch persists a batch with its line items and rating snapshots atomically.
	// Returns ErrBatchAlreadyExists if the same usage_batch_id + payload_hash exists.
	// Returns ErrBatchPayloadConflict if usage_batch_id exists with a different hash.
	CreateBatch(ctx context.Context, batch AIUsageBatch, snapshots []RatingSnapshot) (*BatchSettlementResult, error)

	// GetBatchByBatchID retrieves a batch by its client-generated usage_batch_id.
	GetBatchByBatchID(ctx context.Context, namespace, usageBatchID string) (*AIUsageBatch, error)

	// GetBatchResult retrieves the stored settlement result for a batch.
	GetBatchResult(ctx context.Context, namespace, usageBatchID string) (*BatchSettlementResult, error)

	// GetCoveredSeq returns the highest continuously settled tenant_seq for a customer.
	GetCoveredSeq(ctx context.Context, namespace, customerID string) (int64, error)
}

// CostResolver resolves provider costs for model tokens.
// This wraps the existing llmcost service.
type CostResolver interface {
	// ResolveCost returns the cost snapshot for a provider-managed resource.
	ResolveCost(ctx context.Context, namespace, provider, model string, resourceCode ResourceCode, quantity int64) (CostSnapshot, error)
}
