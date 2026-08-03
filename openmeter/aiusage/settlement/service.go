// Package settlement orchestrates credit allocation through the ledger
// collector. It converts AI-usage int64 credit charges into collector
// CollectToAccrued calls and maps the resulting credit-realization
// allocations back into aiusage.Allocation records with full ledger provenance.
//
// The collector handles all source selection: querying live FBO balance,
// prioritising by credit-priority -> feature-restriction -> expiry -> cursor
// (the within-category burn order), locking accounts, managing breakage, and
// committing the ledger transaction group. The settlement service does NOT
// scan grant balances or synthesise provenance.
package settlement

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"go.opentelemetry.io/otel/trace"

	"github.com/openmeterio/openmeter/openmeter/aiusage"
	"github.com/openmeterio/openmeter/openmeter/aiusage/adapter"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/ledger/collector"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

// Service settles rated usage by delegating source selection to the ledger
// collector. Formal scope writes real ledger entries; shadow scope records the
// computation for visibility with zero ledger side effects.
type Service interface {
	AllocateAndBook(ctx context.Context, tx adapter.TxAdapter, in SettlementInput) ([]aiusage.Allocation, error)
	Correct(ctx context.Context, tx adapter.TxAdapter, in CorrectionInput) ([]aiusage.Allocation, error)
}

// SettlementInput carries everything needed to allocate credits for one batch.
type SettlementInput struct {
	Namespace       string
	CustomerID      string
	SubjectID       string
	TotalCredits    int64
	CeilingCredits  *int64
	SettlementScope aiusage.SettlementScope
	BatchID         string

	// Collector parameters sourced from the customer's billing profile.
	ChargeID       string
	Currency       currencies.CurrencyReference
	FeatureKey     string
	SettlementMode productcatalog.SettlementMode
	BookedAt       time.Time
	ServicePeriod  timeutil.ClosedPeriod

	// ReceivableHardLimit caps enterprise receivable absorption for this call.
	// When nil, the collector's CreditOnly settlement mode allows unlimited
	// advance-backed usage.
	ReceivableHardLimit *int64
}

// CorrectionInput reverses a previously settled batch.
type CorrectionInput struct {
	Namespace       string
	CustomerID      string
	SubjectID       string
	OriginalBatchID string
	BookedAt        time.Time

	// OriginalAllocations are the allocations from the original batch, used to
	// build the correction request for the collector.
	OriginalAllocations []aiusage.Allocation

	ChargeID   string
	Currency   currencies.CurrencyReference
	FeatureKey string
}

// ServiceConfig wires the settlement service.
type ServiceConfig struct {
	Collector collector.Service
	Logger    *slog.Logger
	Tracer    trace.Tracer
}

type service struct {
	collector collector.Service
	logger    *slog.Logger
	tracer    trace.Tracer
}

func New(cfg ServiceConfig) Service {
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}
	return &service{
		collector: cfg.Collector,
		logger:    logger,
		tracer:    cfg.Tracer,
	}
}

// AllocateAndBook settles TotalCredits (capped by CeilingCredits) through the
// ledger collector. The collector performs source selection, account locking,
// breakage management, and ledger commit — all within the caller's transaction
// (the collector joins the existing transaction via context).
//
// Shadow scope: the collector is NOT called; zero ledger rows are written.
//
// Within-category burn ordering (earliest expiry, earliest issuance, stable
// ledger cursor) is handled entirely by the collector's
// fboCollectionSource.Compare method. See ledger/collector/types.go.
func (s *service) AllocateAndBook(ctx context.Context, _ adapter.TxAdapter, in SettlementInput) ([]aiusage.Allocation, error) {
	ctx, span := s.tracer.Start(ctx, "settlement.AllocateAndBook")
	defer span.End()

	// Ceiling enforcement: charged credits never exceed the reservation ceiling.
	charged := in.TotalCredits
	if in.CeilingCredits != nil && charged > *in.CeilingCredits {
		charged = *in.CeilingCredits
		s.logger.InfoContext(ctx, "ceiling applied",
			"batch_id", in.BatchID,
			"total_credits", in.TotalCredits,
			"ceiling", *in.CeilingCredits)
	}

	// Zero-credit batch (e.g. BYOK only).
	if charged <= 0 {
		return []aiusage.Allocation{}, nil
	}

	// Shadow scope: persist for visibility but write zero ledger rows.
	if in.SettlementScope == aiusage.SettlementScopeShadow {
		return []aiusage.Allocation{}, nil
	}

	// Formal scope: delegate to the collector for source selection + commit.
	amount := alpacadecimal.NewFromInt(charged)

	allocInputs, err := s.collector.CollectToAccrued(ctx, collector.CollectToAccruedInput{
		Namespace:         in.Namespace,
		ChargeID:          in.ChargeID,
		CustomerID:        in.CustomerID,
		BookedAt:          in.BookedAt,
		SourceBalanceAsOf: in.BookedAt,
		Currency:          in.Currency,
		FeatureKey:        in.FeatureKey,
		SettlementMode:    in.SettlementMode,
		ServicePeriod:     in.ServicePeriod,
		Amount:            amount,
		Annotations: models.Annotations{
			"ai_usage.batch_id": in.BatchID,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("settlement: collector CollectToAccrued: %w", err)
	}

	allocations := mapCollectorAllocations(allocInputs)

	// Enforce receivable hard limit when set.
	if in.ReceivableHardLimit != nil {
		totalAllocated := int64(0)
		for _, a := range allocations {
			totalAllocated += a.Amount
		}
		receivable := charged - totalAllocated
		if receivable > *in.ReceivableHardLimit {
			return nil, aiusage.ErrCreditLimitExceeded
		}
	}

	return allocations, nil
}

// Correct reverses a previously settled batch by calling the collector's
// CorrectCollectedAccrued with the original allocation provenance. This
// unwinds the actual original ledger entries rather than synthesising reversals.
func (s *service) Correct(ctx context.Context, _ adapter.TxAdapter, in CorrectionInput) ([]aiusage.Allocation, error) {
	ctx, span := s.tracer.Start(ctx, "settlement.Correct")
	defer span.End()

	if len(in.OriginalAllocations) == 0 {
		return []aiusage.Allocation{}, nil
	}

	// Build the correction request from the original allocations.
	corrections := make(creditrealization.CorrectionRequest, 0, len(in.OriginalAllocations))
	for _, orig := range in.OriginalAllocations {
		if orig.Ledger.RealizationID == "" || orig.Ledger.TransactionGroupID == "" {
			continue
		}
		corrections = append(corrections, creditrealization.CorrectionRequestItem{
			Allocation: creditrealization.Realization{
				CreateInput: creditrealization.CreateInput{
					ID:   orig.Ledger.RealizationID,
					Type: creditrealization.TypeAllocation,
					LedgerTransaction: ledgertransaction.GroupReference{
						TransactionGroupID: orig.Ledger.TransactionGroupID,
					},
				},
				SortHint: orig.Ledger.SortHint,
			},
			Amount: alpacadecimal.NewFromInt(-orig.Amount),
		})
	}

	if len(corrections) == 0 {
		return []aiusage.Allocation{}, nil
	}

	correctionInputs, err := s.collector.CorrectCollectedAccrued(ctx, collector.CorrectCollectedAccruedInput{
		Namespace:   in.Namespace,
		ChargeID:    in.ChargeID,
		CustomerID:  in.CustomerID,
		AllocateAt:  in.BookedAt,
		Corrections: corrections,
	})
	if err != nil {
		return nil, fmt.Errorf("settlement: collector CorrectCollectedAccrued: %w", err)
	}

	// Map corrections to reversing allocations.
	reversing := make([]aiusage.Allocation, 0, len(correctionInputs))
	for _, c := range correctionInputs {
		reversing = append(reversing, aiusage.Allocation{
			GrantID: c.CorrectsRealizationID,
			Amount:  c.Amount.IntPart(),
			Ledger: aiusage.LedgerProvenance{
				TransactionGroupID: c.LedgerTransaction.TransactionGroupID,
				RealizationID:      c.CorrectsRealizationID,
			},
		})
	}

	return reversing, nil
}

// mapCollectorAllocations converts creditrealization.CreateAllocationInputs
// (Decimal amounts, ledger group provenance) to aiusage.Allocation (int64
// amounts, typed funding source, ledger provenance). The SortHint is the
// index in the returned slice, matching the collector's ordering.
func mapCollectorAllocations(inputs creditrealization.CreateAllocationInputs) []aiusage.Allocation {
	if len(inputs) == 0 {
		return []aiusage.Allocation{}
	}
	allocations := make([]aiusage.Allocation, 0, len(inputs))
	for i, input := range inputs {
		allocations = append(allocations, aiusage.Allocation{
			Amount: input.Amount.IntPart(),
			Ledger: aiusage.LedgerProvenance{
				TransactionGroupID: input.LedgerTransaction.TransactionGroupID,
				SortHint:           i,
			},
			FundingSource: inferFundingSource(input),
		})
	}
	return allocations
}

// inferFundingSource derives the funding source from the collector allocation's
// lineage annotations. The collector tags each allocation with an origin kind
// (real_credit or advance).
func inferFundingSource(input creditrealization.CreateAllocationInput) aiusage.FundingSource {
	originKind, ok := input.Annotations.GetString(creditrealization.AnnotationLineageOriginKind)
	if !ok {
		return aiusage.FundingSourcePaidTopUp
	}
	switch creditrealization.LineageOriginKind(originKind) {
	case creditrealization.LineageOriginKindAdvance:
		return aiusage.FundingSourceEnterpriseReceivable
	default:
		return aiusage.FundingSourcePaidTopUp
	}
}

// Compile-time interface check.
var _ Service = (*service)(nil)
