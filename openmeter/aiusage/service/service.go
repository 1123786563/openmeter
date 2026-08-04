// Package service is the application-level orchestration for AI usage
// settlement. It coordinates validate -> price -> settle -> persist in one
// atomic customer-locked transaction.
package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel/trace"

	"github.com/openmeterio/openmeter/openmeter/aiusage"
	"github.com/openmeterio/openmeter/openmeter/aiusage/adapter"
	"github.com/openmeterio/openmeter/openmeter/aiusage/pricing"
	"github.com/openmeterio/openmeter/openmeter/aiusage/settlement"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

// Service is the application-level orchestration for AI usage settlement.
type Service interface {
	Settle(ctx context.Context, in aiusage.IngestBatchInput) (*aiusage.BatchSettlementResult, error)
	Correct(ctx context.Context, in aiusage.CorrectionInput) (*aiusage.BatchSettlementResult, error)
}

// PricingResolver resolves usage lines into rated credit charges.
type PricingResolver interface {
	Resolve(ctx context.Context, in pricing.ResolveInput) (pricing.ResolvedBatch, error)
}

// ScopeResolver determines whether a customer's batches are settled formally
// or in shadow mode. When nil, all batches default to formal.
type ScopeResolver interface {
	ResolveScope(ctx context.Context, namespace, customerID string) (aiusage.SettlementScope, error)
}

// CustomerProfileResolver provides billing profile data (charge ID, currency,
// settlement mode, feature key) needed for collector integration.
type CustomerProfileResolver interface {
	Resolve(ctx context.Context, namespace, customerID string) (CustomerProfile, error)
}

// CustomerProfile carries billing configuration for collector integration.
type CustomerProfile struct {
	ChargeID       string
	Currency       currencies.CurrencyReference
	FeatureKey     string
	SettlementMode productcatalog.SettlementMode
}

// Config wires the application service.
type Config struct {
	Adapter           adapter.Adapter
	Pricing           PricingResolver
	Settlement        settlement.Service
	ProfileResolver   CustomerProfileResolver
	ScopeResolver     ScopeResolver
	AllocationFetcher AllocationFetcher
	Logger            *slog.Logger
	Tracer            trace.Tracer
}

// AllocationFetcher reads persisted allocations for a batch (used by Correct
// to build the reversal request).
type AllocationFetcher interface {
	GetAllocations(ctx context.Context, namespace, customerID, batchID string) ([]aiusage.Allocation, error)
}

type svc struct {
	adp             adapter.Adapter
	pricing         PricingResolver
	settlement      settlement.Service
	profileResolver CustomerProfileResolver
	scopeResolver   ScopeResolver
	allocFetcher    AllocationFetcher
	logger          *slog.Logger
	tracer          trace.Tracer
}

func New(cfg Config) Service {
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}
	return &svc{
		adp:             cfg.Adapter,
		pricing:         cfg.Pricing,
		settlement:      cfg.Settlement,
		profileResolver: cfg.ProfileResolver,
		scopeResolver:   cfg.ScopeResolver,
		allocFetcher:    cfg.AllocationFetcher,
		logger:          logger,
		tracer:          cfg.Tracer,
	}
}

// Settle validates the input, resolves pricing, settles credits through the
// collector, and persists the result in one atomic customer-locked transaction.
func (s *svc) Settle(ctx context.Context, in aiusage.IngestBatchInput) (*aiusage.BatchSettlementResult, error) {
	ctx, span := s.tracer.Start(ctx, "aiusage.service.Settle")
	defer span.End()

	if err := in.Validate(); err != nil {
		return nil, err
	}

	// Resolve customer billing profile.
	profile, err := s.profileResolver.Resolve(ctx, in.Namespace, in.CustomerID)
	if err != nil {
		return nil, fmt.Errorf("service: resolve customer profile: %w", err)
	}

	// Resolve settlement scope.
	scope := aiusage.SettlementScopeFormal
	if s.scopeResolver != nil {
		scope, err = s.scopeResolver.ResolveScope(ctx, in.Namespace, in.CustomerID)
		if err != nil {
			return nil, fmt.Errorf("service: resolve scope: %w", err)
		}
	}

	// Price / rate.
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

	servicePeriod := timeutil.ClosedPeriod{
		From: in.OccurredAt,
		To:   in.OccurredAt,
	}

	var result *aiusage.BatchSettlementResult

	err = s.adp.WithCustomerLock(ctx, in.Namespace, in.CustomerID, func(ctx context.Context, tx adapter.TxAdapter) error {
		// Idempotency short-circuit (Finding B): if a batch with this
		// UsageBatchID already exists, return it without calling the
		// collector. This prevents a second collector invocation on replay.
		existing, err := tx.GetBatchByIdempotencyKey(ctx, in.Namespace, in.CustomerID, in.UsageBatchID)
		if err != nil {
			return fmt.Errorf("idempotency check: %w", err)
		}
		if existing != nil {
			result = &aiusage.BatchSettlementResult{
				BatchID:          existing.UsageBatchID,
				Status:           existing.Status,
				CoveredTenantSeq: existing.TenantSeq,
			}
			return nil
		}

		// Settle through the collector inside the locked transaction.
		allocations, err := s.settlement.AllocateAndBook(ctx, tx, settlement.SettlementInput{
			Namespace:       in.Namespace,
			CustomerID:      in.CustomerID,
			SubjectID:       in.SubjectID,
			TotalCredits:    totalCredits,
			CeilingCredits:  in.CeilingCredits,
			SettlementScope: scope,
			BatchID:         in.UsageBatchID,
			ChargeID:        profile.ChargeID,
			Currency:        profile.Currency,
			FeatureKey:      profile.FeatureKey,
			SettlementMode:  profile.SettlementMode,
			BookedAt:        in.OccurredAt,
			ServicePeriod:   servicePeriod,
		})
		if err != nil {
			return err
		}

		// Persist atomically within the same transaction.
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
			SettlementScope: scope,
			Status:          aiusage.BatchStatusSettled,
			TotalCredits:    totalCredits,
			LineItems:       in.LineItems,
			Allocations:     allocations,
			OutboxEvents: []aiusage.OutboxEvent{
				{
					EventType: aiusage.EventBatchSettled,
					Payload:   map[string]any{"usage_batch_id": in.UsageBatchID},
				},
				{
					EventType: aiusage.EventCreditBalanceChanged,
					Payload: map[string]any{
						"usage_batch_id": in.UsageBatchID,
						"customer_id":    in.CustomerID,
						"credits_burned": totalCredits,
					},
				},
			},
		}
		settled.RatingSnapshots = resolvedLinesToSnapshots(resolved.Lines)

		batch, created, err := tx.CreateSettledBatch(ctx, settled)
		if err != nil {
			return fmt.Errorf("persist batch: %w", err)
		}
		if !created {
			// Idempotent re-submission.
			result = &aiusage.BatchSettlementResult{
				BatchID:          batch.UsageBatchID,
				Status:           batch.Status,
				CoveredTenantSeq: batch.TenantSeq,
			}
			return nil
		}

		result = &aiusage.BatchSettlementResult{
			BatchID:          batch.UsageBatchID,
			Status:           aiusage.BatchStatusSettled,
			TotalCredits:     totalCredits,
			CoveredTenantSeq: batch.TenantSeq,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

// Correct reverses a previously settled batch by fetching the original
// allocations, calling the collector's CorrectCollectedAccrued, and persisting
// the corrected batch — all within one customer-locked transaction.
//
// TenantSeq contract (I5): the CorrectionInput.TenantSeq MUST be unique per
// (namespace, subject_id) and MUST NOT reuse the original batch's seq. The
// database enforces this via a UNIQUE index on (namespace, subject_id,
// tenant_seq) — a collision will fail with a constraint error at persist time.
// Callers are responsible for allocating a fresh, monotonic seq for the
// correction batch. The service rejects seq <= 0 in CorrectionInput.Validate;
// the DB constraint provides the hard uniqueness guarantee.
func (s *svc) Correct(ctx context.Context, in aiusage.CorrectionInput) (*aiusage.BatchSettlementResult, error) {
	ctx, span := s.tracer.Start(ctx, "aiusage.service.Correct")
	defer span.End()

	if err := in.Validate(); err != nil {
		return nil, err
	}

	profile, err := s.profileResolver.Resolve(ctx, in.Namespace, in.CustomerID)
	if err != nil {
		return nil, fmt.Errorf("service: resolve customer profile: %w", err)
	}

	var result *aiusage.BatchSettlementResult

	err = s.adp.WithCustomerLock(ctx, in.Namespace, in.CustomerID, func(ctx context.Context, tx adapter.TxAdapter) error {
		// Idempotency short-circuit (Finding B): skip the collector on replay.
		correctionBatchID := "corr-" + in.OriginalBatchID
		existing, err := tx.GetBatchByIdempotencyKey(ctx, in.Namespace, in.CustomerID, correctionBatchID)
		if err != nil {
			return fmt.Errorf("idempotency check: %w", err)
		}
		if existing != nil {
			result = &aiusage.BatchSettlementResult{
				BatchID:          existing.UsageBatchID,
				Status:           existing.Status,
				CoveredTenantSeq: existing.TenantSeq,
			}
			return nil
		}

		// Fetch original allocations.
		originalAllocs, err := s.allocFetcher.GetAllocations(ctx, in.Namespace, in.CustomerID, in.OriginalBatchID)
		if err != nil {
			return fmt.Errorf("fetch original allocations: %w", err)
		}

		// Reverse through the collector.
		reversing, err := s.settlement.Correct(ctx, tx, settlement.CorrectionInput{
			Namespace:           in.Namespace,
			CustomerID:          in.CustomerID,
			SubjectID:           in.SubjectID,
			OriginalBatchID:     in.OriginalBatchID,
			BookedAt:            time.Now().UTC(),
			OriginalAllocations: originalAllocs,
			ChargeID:            profile.ChargeID,
			Currency:            profile.Currency,
			FeatureKey:          profile.FeatureKey,
		})
		if err != nil {
			return err
		}


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
			Allocations:     reversing,
			OutboxEvents: []aiusage.OutboxEvent{
				{
					EventType: aiusage.EventBatchCorrected,
					Payload: map[string]any{
						"original_batch_id": in.OriginalBatchID,
						"reason":            in.Reason,
					},
				},
			},
		}

		batch, _, err := tx.CreateSettledBatch(ctx, settled)
		if err != nil {
			return fmt.Errorf("persist correction batch: %w", err)
		}

		result = &aiusage.BatchSettlementResult{
			BatchID:          batch.UsageBatchID,
			Status:           aiusage.BatchStatusCompensated,
			CoveredTenantSeq: batch.TenantSeq,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

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

var _ Service = (*svc)(nil)
