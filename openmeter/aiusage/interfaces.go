package aiusage

import "context"

// Service is the integration-layer interface that the HTTP handler and Wire DI
// graph depend on. The production implementation is an adapter (see
// app/common/aiusage.go) that wraps the application-level service.Service
// (Settle/Correct) and exposes it through the original IngestBatch/GetBatch
// surface.
type Service interface {
	// IngestBatch receives a Canonical Usage Batch, validates it, resolves
	// rates, settles credits, and persists the result atomically. Idempotent:
	// same usage_batch_id + same payload_hash returns the original result.
	IngestBatch(ctx context.Context, input IngestBatchInput) (*BatchSettlementResult, error)

	// GetBatch retrieves a previously submitted batch, scoped to the billing
	// customer to prevent cross-tenant data leakage.
	GetBatch(ctx context.Context, namespace, customerID, usageBatchID string) (*AIUsageBatch, error)

	// GetCoveredSeq returns the highest continuously settled tenant_seq for a customer.
	GetCoveredSeq(ctx context.Context, namespace, customerID string) (int64, error)
}

// Repository persists AI Usage batches and related entities. The production
// implementation lives in openmeter/aiusage/adapter/repository.go.
type Repository interface {
	// CreateBatch persists a batch with its line items and rating snapshots atomically.
	// Returns ErrBatchAlreadyExists if the same usage_batch_id + payload_hash exists.
	// Returns ErrBatchPayloadConflict if usage_batch_id exists with a different hash.
	CreateBatch(ctx context.Context, batch AIUsageBatch, snapshots []RatingSnapshot) (*BatchSettlementResult, error)

	// GetBatchByBatchID retrieves a batch by its client-generated usage_batch_id,
	// scoped to the billing customer to prevent cross-tenant idempotency leakage.
	GetBatchByBatchID(ctx context.Context, namespace, customerID, usageBatchID string) (*AIUsageBatch, error)

	// GetBatchResult retrieves the stored settlement result for a batch.
	GetBatchResult(ctx context.Context, namespace, usageBatchID string) (*BatchSettlementResult, error)

	// GetCoveredSeq returns the highest continuously settled tenant_seq for a customer.
	GetCoveredSeq(ctx context.Context, namespace, customerID string) (int64, error)
}
