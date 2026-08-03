package aiusage

import (
	"context"
	"fmt"
	"log/slog"

	"go.opentelemetry.io/otel/trace"
)

// Service orchestrates the full AI Usage batch settlement flow.
type Service interface {
	// IngestBatch receives a Canonical Usage Batch, validates it, resolves rates,
	// rates all line items, applies ceiling, settles Credits, and returns the result.
	// Idempotent: same usage_batch_id + same payload_hash returns the original result.
	IngestBatch(ctx context.Context, input IngestBatchInput) (*BatchSettlementResult, error)

	// GetBatch retrieves a previously submitted batch.
	GetBatch(ctx context.Context, namespace, usageBatchID string) (*AIUsageBatch, error)

	// GetCoveredSeq returns the highest continuously settled tenant_seq for a customer.
	GetCoveredSeq(ctx context.Context, namespace, customerID string) (int64, error)
}

type service struct {
	Repo             Repository
	RateCardResolver RateCardResolver
	CostResolver     CostResolver
	SettlementEngine SettlementEngine
	Logger           *slog.Logger
	Tracer           trace.Tracer
}

type ServiceConfig struct {
	Repo             Repository
	RateCardResolver RateCardResolver
	CostResolver     CostResolver
	SettlementEngine SettlementEngine
	Logger           *slog.Logger
	Tracer           trace.Tracer
}

func NewService(cfg ServiceConfig) Service {
	return &service{
		Repo:             cfg.Repo,
		RateCardResolver: cfg.RateCardResolver,
		CostResolver:     cfg.CostResolver,
		SettlementEngine: cfg.SettlementEngine,
		Logger:           cfg.Logger,
		Tracer:           cfg.Tracer,
	}
}

func (s *service) IngestBatch(ctx context.Context, input IngestBatchInput) (*BatchSettlementResult, error) {
	ctx, span := s.Tracer.Start(ctx, "aiusage.IngestBatch")
	defer span.End()

	// Step 1: Validate input.
	if err := input.Validate(); err != nil {
		return nil, err
	}

	// Step 2: Check idempotency — if batch exists with same hash, return stored result.
	existing, err := s.Repo.GetBatchByBatchID(ctx, input.Namespace, input.UsageBatchID)
	if err != nil {
		// Only proceed if it's a not-found error.
		s.Logger.DebugContext(ctx, "batch lookup", "err", err)
	}
	if existing != nil {
		if existing.PayloadHash == input.PayloadHash {
			result, err := s.Repo.GetBatchResult(ctx, input.Namespace, input.UsageBatchID)
			if err != nil {
				return nil, fmt.Errorf("failed to get existing batch result: %w", err)
			}
			return result, nil
		}
		// Same batch_id, different hash = conflict.
		return nil, ErrBatchPayloadConflict
	}

	// Step 3: Rate all line items.
	var snapshots []RatingSnapshot
	if input.BillingMode == BillingModeComponent {
		snapshots, err = s.rateLineItems(ctx, input)
		if err != nil {
			return nil, err
		}
	}

	// Step 4: Construct the domain batch.
	batch := AIUsageBatch{
		Namespace:      input.Namespace,
		CustomerID:     input.CustomerID,
		SubjectID:      input.SubjectID,
		UsageBatchID:   input.UsageBatchID,
		TenantSeq:      input.TenantSeq,
		OccurredAt:     input.OccurredAt,
		ReservationID:  input.ReservationID,
		CeilingCredits: input.CeilingCredits,
		RateVersion:    input.RateVersion,
		BillingMode:    input.BillingMode,
		PayloadHash:    input.PayloadHash,
		Status:         BatchStatusPending,
		LineItems:      input.LineItems,
	}

	// Step 5: Settle via settlement engine.
	result, err := s.SettlementEngine.Settle(ctx, batch, snapshots, input.CeilingCredits)
	if err != nil {
		return nil, err
	}

	// Step 6: Persist atomically.
	batch.Status = result.Status
	persistedResult, err := s.Repo.CreateBatch(ctx, batch, snapshots)
	if err != nil {
		return nil, fmt.Errorf("failed to persist batch: %w", err)
	}

	return persistedResult, nil
}

func (s *service) GetBatch(ctx context.Context, namespace, usageBatchID string) (*AIUsageBatch, error) {
	return s.Repo.GetBatchByBatchID(ctx, namespace, usageBatchID)
}

func (s *service) GetCoveredSeq(ctx context.Context, namespace, customerID string) (int64, error) {
	return s.Repo.GetCoveredSeq(ctx, namespace, customerID)
}

// rateLineItems resolves cost and sales prices for each line item and calculates Credits.
func (s *service) rateLineItems(ctx context.Context, input IngestBatchInput) ([]RatingSnapshot, error) {
	snapshots := make([]RatingSnapshot, 0, len(input.LineItems))

	for _, item := range input.LineItems {
		snapshot := RatingSnapshot{
			ResourceCode: item.ResourceCode,
		}

		if item.ProviderManaged {
			// Provider-managed resource: resolve cost from llmcost.
			cost, err := s.CostResolver.ResolveCost(ctx, input.Namespace, item.Provider, item.Model, item.ResourceCode, item.Quantity)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve cost for %s: %w", item.ResourceCode, err)
			}
			snapshot.CostSnapshot = cost
		} else {
			// BYOK resource: cost is zero.
			snapshot.CostSnapshot = CostSnapshot{
				Currency: "USD",
				Amount:   zeroDecimal(),
				Source:   "byok",
			}
		}

		// Resolve sales price from rate card.
		rate, err := s.RateCardResolver.Resolve(ctx, input.Namespace, input.CustomerID,
			item.ResourceCode, item.Provider, item.Model, input.OccurredAt)
		if err != nil {
			// BYOK resources can skip rate card if not found.
			if !item.ProviderManaged {
				snapshot.SalesSnapshot = SalesSnapshot{
					Currency:        "CNY",
					Amount:          zeroDecimal(),
					RateCardVersion: "byok",
				}
				snapshot.Credits = 0
				snapshots = append(snapshots, snapshot)
				continue
			}
			return nil, fmt.Errorf("failed to resolve rate for %s: %w", item.ResourceCode, err)
		}

		snapshot.SalesSnapshot = SalesSnapshot{
			Currency:        "CNY",
			Amount:          rate.PricePerUnitCNY,
			RateCardVersion: input.RateVersion,
		}
		snapshot.Credits = CalculateLineCredits(item.Quantity, rate.PricePerUnitCNY, rate.CreditRate)

		snapshots = append(snapshots, snapshot)
	}

	return snapshots, nil
}
