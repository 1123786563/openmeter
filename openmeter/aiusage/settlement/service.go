package settlement

import (
	"context"
	"fmt"
	"log/slog"
	"sort"

	"go.opentelemetry.io/otel/trace"

	"github.com/openmeterio/openmeter/openmeter/aiusage"
)

// Service settles rated usage against credit grants in the approved burn order:
// plan → promotional → paid_top_up → enterprise_receivable.
//
// Each call produces a slice of Allocations recording which grant was consumed
// and by how much. In formal scope the deductions are also written to the Ledger;
// in shadow scope the computation runs but no ledger side effects occur.
type Service interface {
	AllocateAndBook(ctx context.Context, in SettlementInput) ([]aiusage.Allocation, error)
}

// SettlementInput carries everything the settlement engine needs to burn credits.
type SettlementInput struct {
	Namespace       string
	CustomerID      string
	SubjectID       string
	TotalCredits    int64
	CeilingCredits  *int64
	SettlementScope aiusage.SettlementScope

	// ReceivableHardLimit caps the enterprise receivable absorption for this
	// settlement. When nil the receivable grant absorbs any remainder (legacy
	// behaviour). When set, exceeding it returns ErrCreditLimitExceeded.
	ReceivableHardLimit *int64

	// BatchID is passed through to the LedgerRecorder for provenance.
	BatchID string
}

// ServiceConfig wires the settlement service dependencies.
type ServiceConfig struct {
	GrantReader aiusage.GrantBalanceReader
	Ledger      aiusage.LedgerRecorder
	Logger      *slog.Logger
	Tracer      trace.Tracer
}

type service struct {
	grantReader aiusage.GrantBalanceReader
	ledger      aiusage.LedgerRecorder
	logger      *slog.Logger
	tracer      trace.Tracer
}

// NewService creates a settlement Service.
func NewService(cfg ServiceConfig) Service {
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}
	return &service{
		grantReader: cfg.GrantReader,
		ledger:      cfg.Ledger,
		logger:      logger,
		tracer:      cfg.Tracer,
	}
}

// AllocateAndBook burns TotalCredits (capped by CeilingCredits) against the
// customer's grants in priority order, records formal deductions, and returns
// the per-grant allocations.
func (s *service) AllocateAndBook(ctx context.Context, in SettlementInput) ([]aiusage.Allocation, error) {
	ctx, span := s.tracer.Start(ctx, "settlement.AllocateAndBook")
	defer span.End()

	// --- Ceiling enforcement ---------------------------------------------------
	charged := in.TotalCredits
	if in.CeilingCredits != nil && charged > *in.CeilingCredits {
		charged = *in.CeilingCredits
		s.logger.InfoContext(ctx, "ceiling applied",
			"batch_id", in.BatchID,
			"total_credits", in.TotalCredits,
			"ceiling", *in.CeilingCredits)
	}

	// Zero-credit batch (e.g. BYOK only) — nothing to burn.
	if charged <= 0 {
		return []aiusage.Allocation{}, nil
	}

	// --- Fetch grants ----------------------------------------------------------
	grants, err := s.grantReader.GetGrants(ctx, in.Namespace, in.CustomerID)
	if err != nil {
		return nil, fmt.Errorf("settlement: read grants: %w", err)
	}

	allocations := burnGrants(grants, charged)

	// --- Validate coverage -----------------------------------------------------
	covered := int64(0)
	hasReceivable := false
	receivableBurned := float64(0)
	for _, a := range allocations {
		covered += int64(a.Amount)
		if a.FundingSource == aiusage.FundingSourceEnterpriseReceivable {
			hasReceivable = true
			receivableBurned += a.Amount
		}
	}

	if covered < charged {
		// Prepaid grants ran out and there is no enterprise receivable.
		if !hasReceivable {
			return nil, aiusage.ErrCreditInsufficient
		}
		// Receivable exists but even it couldn't cover — this shouldn't happen
		// (receivable absorbs the full remainder), but guard anyway.
		return nil, aiusage.ErrCreditInsufficient
	}

	// --- Receivable hard-limit check -------------------------------------------
	if hasReceivable && in.ReceivableHardLimit != nil {
		if int64(receivableBurned) > *in.ReceivableHardLimit {
			return nil, aiusage.ErrCreditLimitExceeded
		}
	}

	// --- Shadow scope: no ledger writes ---------------------------------------
	if in.SettlementScope == aiusage.SettlementScopeShadow {
		return allocations, nil
	}

	// --- Formal scope: record deductions --------------------------------------
	if s.ledger != nil && len(allocations) > 0 {
		refs := allocationsToLedgerRefs(allocations)
		if err := s.ledger.RecordDeductions(ctx, in.Namespace, in.CustomerID, refs, in.BatchID); err != nil {
			return nil, fmt.Errorf("settlement: record deductions: %w", err)
		}
	}

	return allocations, nil
}

// burnGrants deducts credits from grants in priority order, mapping each grant
// source to its FundingSource enum value. Receivable grants absorb any remainder
// (up to their Amount limit when non-zero).
func burnGrants(grants []aiusage.SettlementGrant, amount int64) []aiusage.Allocation {
	if amount <= 0 || len(grants) == 0 {
		return []aiusage.Allocation{}
	}

	sorted := make([]aiusage.SettlementGrant, len(grants))
	copy(sorted, grants)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].Priority < sorted[j].Priority
	})

	allocations := make([]aiusage.Allocation, 0)
	remaining := float64(amount)

	for _, grant := range sorted {
		if remaining <= 0 {
			break
		}

		source := mapFundingSource(grant.Source)
		var burned float64

		if source == aiusage.FundingSourceEnterpriseReceivable {
			// Enterprise receivable absorbs the full remainder. When the grant
			// carries a non-zero Amount it acts as a hard credit limit.
			burned = remaining
			remaining = 0
		} else if grant.Amount <= 0 {
			continue
		} else if grant.Amount >= remaining {
			burned = remaining
			remaining = 0
		} else {
			burned = grant.Amount
			remaining -= grant.Amount
		}

		allocations = append(allocations, aiusage.Allocation{
			GrantID:       grant.GrantID,
			Amount:        burned,
			Priority:      grant.Priority,
			FundingSource: source,
		})
	}

	return allocations
}

// mapFundingSource converts the legacy grant Source strings to the typed
// FundingSource enum. Unknown sources default to paid_top_up.
func mapFundingSource(source string) aiusage.FundingSource {
	switch source {
	case "plan":
		return aiusage.FundingSourcePlan
	case "gift", "promotional":
		return aiusage.FundingSourcePromotional
	case "recharge", "paid_top_up":
		return aiusage.FundingSourcePaidTopUp
	case "receivable", "enterprise_receivable":
		return aiusage.FundingSourceEnterpriseReceivable
	default:
		return aiusage.FundingSourcePaidTopUp
	}
}

// allocationsToLedgerRefs converts typed allocations back to the legacy
// LedgerEntryRef representation for the LedgerRecorder.
func allocationsToLedgerRefs(allocs []aiusage.Allocation) []aiusage.LedgerEntryRef {
	refs := make([]aiusage.LedgerEntryRef, len(allocs))
	for i, a := range allocs {
		refs[i] = aiusage.LedgerEntryRef{
			GrantID:  a.GrantID,
			Amount:   a.Amount,
			Priority: a.Priority,
		}
	}
	return refs
}

// Compile-time interface check.
var _ Service = (*service)(nil)
