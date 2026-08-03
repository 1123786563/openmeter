package aiusage

import (
	"context"
	"fmt"
	"log/slog"

	"go.opentelemetry.io/otel/trace"
)

// SettlementGrant represents a grant that can be burned during settlement.
// This is a read-model projection of the credit system's grant.
type SettlementGrant struct {
	GrantID   string  `json:"grant_id"`
	Amount    float64 `json:"amount"`    // remaining balance
	Priority  uint8   `json:"priority"`  // lower = consumed first
	Source    string  `json:"source"`    // "plan", "gift", "recharge", "receivable"
}

// GrantBalanceReader provides the current grant balances for a billing customer.
type GrantBalanceReader interface {
	// GetGrants returns all active grants for the billing customer, sorted by priority.
	GetGrants(ctx context.Context, namespace string, customerID string) ([]SettlementGrant, error)
}

// LedgerRecorder records the settlement deductions into the credit ledger.
type LedgerRecorder interface {
	// RecordDeductions writes the burn-down results as ledger entries.
	// Each entry represents a grant that was consumed.
	RecordDeductions(ctx context.Context, namespace string, customerID string, deductions []LedgerEntryRef, batchID string) error
}

// SettlementEngine settles rated usage against the Credit Ledger in strict priority order.
type SettlementEngine interface {
	// Settle processes a rated batch: deducts Credits in priority order,
	// creates ledger entries, and applies the batch ceiling.
	Settle(ctx context.Context, batch AIUsageBatch, snapshots []RatingSnapshot,
		ceiling *int64) (*BatchSettlementResult, error)
}

type settlementEngine struct {
	BalanceReader GrantBalanceReader
	Ledger        LedgerRecorder
	Logger        *slog.Logger
	Tracer        trace.Tracer
}

func NewSettlementEngine(cfg SettlementEngineConfig) SettlementEngine {
	return &settlementEngine{
		BalanceReader: cfg.GrantReader,
		Ledger:        cfg.Ledger,
		Logger:        cfg.Logger,
		Tracer:        cfg.Tracer,
	}
}

type SettlementEngineConfig struct {
	GrantReader GrantBalanceReader
	Ledger      LedgerRecorder
	Logger      *slog.Logger
	Tracer      trace.Tracer
}

// Settle implements SettlementEngine.
func (e *settlementEngine) Settle(ctx context.Context, batch AIUsageBatch, snapshots []RatingSnapshot, ceiling *int64) (*BatchSettlementResult, error) {
	ctx, span := e.Tracer.Start(ctx, "aiusage.Settle")
	defer span.End()

	// Step 1: Calculate total credits from rating snapshots or ceiling (bundle mode).
	totalCredits := int64(0)
	for _, s := range snapshots {
		totalCredits += s.Credits
	}
	// Bundle mode: ceiling IS the total charge.
	if batch.BillingMode == BillingModeBundle && ceiling != nil {
		totalCredits = *ceiling
	}

	// Step 2: Apply ceiling if set (component mode).
	ceilingApplied := false
	if ceiling != nil && totalCredits > *ceiling {
		totalCredits = *ceiling
		ceilingApplied = true
		e.Logger.InfoContext(ctx, "batch ceiling applied",
			"batch_id", batch.UsageBatchID,
			"original", totalCredits,
			"ceiling", *ceiling)
	}

	// Step 3: Zero-credit batch (e.g. BYOK only) — settle immediately.
	if totalCredits == 0 {
		return &BatchSettlementResult{
			BatchID:         batch.UsageBatchID,
			Status:          BatchStatusSettled,
			TotalCredits:    0,
			RatingSnapshots: snapshots,
			LedgerEntries:   []LedgerEntryRef{},
			CoveredTenantSeq: batch.TenantSeq,
		}, nil
	}

	// Step 4: Get current grants for the billing customer.
	grants, err := e.BalanceReader.GetGrants(ctx, batch.Namespace, batch.CustomerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get grants: %w", err)
	}

	// Step 5: Burn grants in priority order.
	deductions, remainder := burnGrants(grants, totalCredits)

	// Step 6: Handle remainder.
	if remainder > 0 {
		// Check if customer has enterprise receivable grant (priority 30).
		hasReceivable := false
		for _, g := range grants {
			if g.Source == "receivable" {
				hasReceivable = true
				break
			}
		}

		if !hasReceivable {
			// Non-enterprise: fail-closed, no partial deduction.
			return nil, ErrInsufficientCredits
		}

		// Enterprise: remainder was already consumed by the receivable grant in burnGrants.
		// If it wasn't, the receivable grant didn't have enough limit.
		// For Enterprise with credit terms, we allow it to go negative (receivable).
		if remainder > 0 && !ceilingApplied {
			e.Logger.WarnContext(ctx, "enterprise receivable exceeded",
				"batch_id", batch.UsageBatchID,
				"remainder", remainder)
		}
	}

	// Step 7: Record ledger entries.
	if len(deductions) > 0 {
		err = e.Ledger.RecordDeductions(ctx, batch.Namespace, batch.CustomerID, deductions, batch.UsageBatchID)
		if err != nil {
			return nil, fmt.Errorf("failed to record ledger entries: %w", err)
		}
	}

	return &BatchSettlementResult{
		BatchID:          batch.UsageBatchID,
		Status:           BatchStatusSettled,
		TotalCredits:     totalCredits,
		RatingSnapshots:  snapshots,
		LedgerEntries:    deductions,
		CoveredTenantSeq: batch.TenantSeq,
	}, nil
}

// burnGrants deducts credits from grants in priority order.
// Returns the deductions made and any remainder that couldn't be covered.
// Grants with source "receivable" can go negative to cover enterprise usage.
func burnGrants(grants []SettlementGrant, amount int64) ([]LedgerEntryRef, int64) {
	if amount <= 0 || len(grants) == 0 {
		return nil, amount
	}

	// Sort grants by priority (ascending — lower priority number = consumed first).
	// Grants are assumed pre-sorted by GetGrants, but we ensure it here.
	sorted := make([]SettlementGrant, len(grants))
	copy(sorted, grants)

	// Simple insertion sort by priority.
	for i := 1; i < len(sorted); i++ {
		for j := i; j > 0 && sorted[j-1].Priority > sorted[j].Priority; j-- {
			sorted[j-1], sorted[j] = sorted[j], sorted[j-1]
		}
	}

	deductions := make([]LedgerEntryRef, 0)
	remaining := float64(amount)

	for _, grant := range sorted {
		if remaining <= 0 {
			break
		}

		if grant.Amount <= 0 && grant.Source != "receivable" {
			continue
		}

		var burned float64
		if grant.Source == "receivable" {
			// Enterprise receivable: always covers the remainder (can go negative).
			burned = remaining
			remaining = 0
		} else if grant.Amount >= remaining {
			burned = remaining
			remaining = 0
		} else {
			burned = grant.Amount
			remaining -= grant.Amount
		}

		deductions = append(deductions, LedgerEntryRef{
			GrantID:  grant.GrantID,
			Amount:   burned,
			Priority: grant.Priority,
		})
	}

	return deductions, int64(remaining)
}

// Compile-time interface check.
var _ SettlementEngine = &settlementEngine{}
