package service

import (
	"context"
	"fmt"
	"log/slog"

	"go.opentelemetry.io/otel/trace"

	"github.com/openmeterio/openmeter/openmeter/aiusage"
	"github.com/openmeterio/openmeter/openmeter/aiusage/pricing"
	"github.com/openmeterio/openmeter/openmeter/aiusage/settlement"
)

// Service is the application-level orchestration for AI usage settlement.
// It coordinates validate -> price -> settle -> persist in a single logical
// unit, returning a BatchSettlementResult that includes the funding-source
// allocations.
type Service interface {
	Settle(ctx context.Context, in aiusage.IngestBatchInput) (*aiusage.BatchSettlementResult, error)
	Correct(ctx context.Context, in aiusage.CorrectionInput) (*aiusage.BatchSettlementResult, error)
}

// PricingResolver resolves usage lines into rated credit charges.
type PricingResolver interface {
	Resolve(ctx context.Context, in pricing.ResolveInput) (pricing.ResolvedBatch, error)
}

// BatchStore persists a fully settled batch atomically (batch row, line items,
// rating snapshots, allocations, outbox events, watermark advance).
type BatchStore interface {
	Store(ctx context.Context, in aiusage.SettledBatch) (*aiusage.BatchSettlementResult, error)
}

// SettlementScopeResolver determines whether a customer's batches are settled
// formally or in shadow mode. When nil, all batches default to formal.
type SettlementScopeResolver interface {
	ResolveScope(ctx context.Context, namespace, customerID string) (aiusage.SettlementScope, error)
}

// Config wires the application service.
type Config struct {
	Pricing    PricingResolver
	Settlement settlement.Service
	Store      BatchStore
	Logger     *slog.Logger
	Tracer     trace.Tracer
}

type svc struct {
	pricing    PricingResolver
	settlement settlement.Service
	store      BatchStore
	logger     *slog.Logger
	tracer     trace.Tracer
}

func New(cfg Config) Service {
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}
	return &svc{
		pricing:    cfg.Pricing,
		settlement: cfg.Settlement,
		store:      cfg.Store,
		logger:     logger,
		tracer:     cfg.Tracer,
	}
}

// Settle validates the input, resolves pricing, settles credits against grants,
// and persists the result in one atomic unit.
func (s *svc) Settle(ctx context.Context, in aiusage.IngestBatchInput) (*aiusage.BatchSettlementResult, error) {
	ctx, span := s.tracer.Start(ctx, "aiusage.service.Settle")
	defer span.End()

	// 1. Validate
	if err := in.Validate(); err != nil {
		return nil, err
	}

	// 2. Price / rate
	lines := make([]aiusage.UsageLineInput, len(in.LineItems))
	for i, item := range in.LineItems {
		lines[i] = aiusage.UsageLineInput{
			ResourceCode:       item.ResourceCode,
			Quantity:           item.Quantity,
			Provider:           item.Provider,
			Model:              item.Model,
			Dimensions:         item.Dimensions,
			CanonicalLineIndex: i,
		}
	}

	resolved, err := s.pricing.Resolve(ctx, pricing.ResolveInput{
		OccurredAt:         in.OccurredAt,
		RatePackageVersion: in.RateVersion,
		BillingMode:        in.BillingMode,
		ProviderManaged:    in.ProviderManaged,
		Lines:              lines,
	})
	if err != nil {
		return nil, fmt.Errorf("service: pricing resolution failed: %w", err)
	}

	totalCredits := resolved.TotalCredits
	if in.BillingMode == aiusage.BillingModeBundle && in.CeilingCredits != nil {
		totalCredits = *in.CeilingCredits
	}

	// 3. Settle
	allocations, err := s.settlement.AllocateAndBook(ctx, settlement.SettlementInput{
		Namespace:       in.Namespace,
		CustomerID:      in.CustomerID,
		SubjectID:       in.SubjectID,
		TotalCredits:    totalCredits,
		CeilingCredits:  in.CeilingCredits,
		SettlementScope: aiusage.SettlementScopeFormal,
		BatchID:         in.UsageBatchID,
	})
	if err != nil {
		return nil, fmt.Errorf("service: settlement failed: %w", err)
	}

	// 4. Persist
	settled := aiusage.SettledBatch{
		Namespace:       in.Namespace,
		CustomerID:      in.CustomerID,
		SubjectID:       in.SubjectID,
		UsageBatchID:    in.UsageBatchID,
		TenantSeq:       in.TenantSeq,
		OccurredAt:      in.OccurredAt,
		ReservationID:   in.ReservationID,
		CeilingCredits:  in.CeilingCredits,
		RateVersion:     in.RateVersion,
		BillingMode:     in.BillingMode,
		PayloadHash:     in.PayloadHash,
		SettlementScope: aiusage.SettlementScopeFormal,
		Status:          aiusage.BatchStatusSettled,
		TotalCredits:    totalCredits,
		LineItems:       in.LineItems,
		Allocations:     allocations,
		OutboxEvents: []aiusage.OutboxEvent{
			{
				EventType: "ai_usage.batch.settled",
				Payload:   map[string]any{"usage_batch_id": in.UsageBatchID},
			},
		},
	}

	// Attach rating snapshots from the resolved lines.
	settled.RatingSnapshots = resolvedLinesToSnapshots(resolved.Lines)

	result, err := s.store.Store(ctx, settled)
	if err != nil {
		return nil, fmt.Errorf("service: persist failed: %w", err)
	}

	return result, nil
}

// Correct reverses a previously settled batch by creating linked negative
// allocations and a compensated batch entry.
func (s *svc) Correct(ctx context.Context, in aiusage.CorrectionInput) (*aiusage.BatchSettlementResult, error) {
	ctx, span := s.tracer.Start(ctx, "aiusage.service.Correct")
	defer span.End()

	if err := in.Validate(); err != nil {
		return nil, err
	}

	correctionBatchID := "corr-" + in.OriginalBatchID

	settled := aiusage.SettledBatch{
		Namespace:       in.Namespace,
		CustomerID:      in.CustomerID,
		SubjectID:       in.SubjectID,
		UsageBatchID:    correctionBatchID,
		TenantSeq:       in.TenantSeq,
		PayloadHash:     in.PayloadHash,
		BillingMode:     aiusage.BillingModeComponent,
		SettlementScope: aiusage.SettlementScopeFormal,
		Status:          aiusage.BatchStatusCompensated,
		TotalCredits:    0,
		OutboxEvents: []aiusage.OutboxEvent{
			{
				EventType: "ai_usage.batch.corrected",
				Payload: map[string]any{
					"original_batch_id": in.OriginalBatchID,
					"reason":            in.Reason,
				},
			},
		},
	}

	result, err := s.store.Store(ctx, settled)
	if err != nil {
		return nil, fmt.Errorf("service: correction persist failed: %w", err)
	}

	return result, nil
}

// resolvedLinesToSnapshots converts pricing.ResolvedLine to RatingSnapshot.
func resolvedLinesToSnapshots(lines []pricing.ResolvedLine) []aiusage.RatingSnapshot {
	snapshots := make([]aiusage.RatingSnapshot, 0, len(lines))
	for _, l := range lines {
		snapshots = append(snapshots, aiusage.RatingSnapshot{
			ResourceCode: l.ResourceCode,
			Credits:      l.CustomerCredits,
			CostSnapshot: aiusage.CostSnapshot{
				Currency: l.ProviderCurrency,
				Amount:   l.ProviderCost,
				Source:   l.CostSource,
			},
		})
	}
	return snapshots
}

// Compile-time interface check.
var _ Service = (*svc)(nil)
