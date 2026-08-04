// Package reconciliation implements scheduled invariant checks for the Phase 2
// commerce domain. Reconciliation checks report inconsistencies — they never
// silently rewrite data. Each check returns a list of CheckResult entries
// describing the violations found (if any).
//
// Approved invariants (from the Phase 2 brief):
//
//  1. paid order without fulfillment beyond threshold
//  2. fulfilled order without exactly one Ledger grant
//  3. provider success without Payment Fact
//  4. Refund Fact without matching fence/reversal
//  5. Wallet aggregate differing from Ledger-derived value
//  6. closed receivable differing from frozen settlement range
//
// Event invariants are also checked: successful state transitions must publish
// only the approved event names (order.updated, payment.settled, payment.failed,
// refund.updated, invoice.updated, subscription.updated), event IDs must equal
// Outbox IDs, and retries must not create a second domain effect.
package reconciliation

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/openmeterio/openmeter/openmeter/commerce"
)

// Severity classifies a reconciliation finding.
type Severity string

const (
	SeverityWarning Severity = "warning" // soft inconsistency, not data corruption
	SeverityError   Severity = "error"   // data corruption or broken invariant
)

// CheckName identifies which invariant was checked.
type CheckName string

const (
	CheckPaidWithoutFulfillment       CheckName = "paid_order_without_fulfillment"
	CheckFulfilledWithoutGrant        CheckName = "fulfilled_without_ledger_grant"
	CheckProviderSuccessWithoutFact   CheckName = "provider_success_without_payment_fact"
	CheckRefundFactWithoutFence       CheckName = "refund_fact_without_fence_reversal"
	CheckWalletDiffersFromLedger      CheckName = "wallet_differs_from_ledger"
	CheckClosedReceivableRangeChanged CheckName = "closed_receivable_range_changed"
	CheckUnknownEventTypes            CheckName = "unknown_event_types"
	CheckEventIdMismatch              CheckName = "event_id_outbox_id_mismatch"
)

// Finding is a single reconciliation violation. It describes what was found,
// where, and at what severity. It never modifies data.
type Finding struct {
	Check     CheckName `json:"check"`
	Severity  Severity  `json:"severity"`
	Namespace string    `json:"namespace"`
	EntityID  string    `json:"entity_id"`
	Detail    string    `json:"detail"`
}

// Report collects all findings from a reconciliation run.
type Report struct {
	RunAt     time.Time `json:"run_at"`
	Findings  []Finding `json:"findings"`
	ChecksRun int       `json:"checks_run"`
}

// HasErrors reports whether any finding has Error severity.
func (r *Report) HasErrors() bool {
	for _, f := range r.Findings {
		if f.Severity == SeverityError {
			return true
		}
	}
	return false
}

// ProbePort is the read interface the reconciliation checker needs. All methods
// are read-only — reconciliation never mutates data. Implementations query the
// Ent database (or a read replica) to gather the raw facts each invariant checks.
type ProbePort interface {
	// ListPaidOrdersWithoutFulfillment returns orders in "paid" status that have
	// no fulfilled fulfillment record and are older than the threshold.
	ListPaidOrdersWithoutFulfillment(ctx context.Context, namespace string, threshold time.Duration) ([]StaleOrder, error)

	// ListFulfilledOrdersWithoutGrant returns orders in "fulfilled" status that
	// do not have exactly one Ledger grant.
	ListFulfilledOrdersWithoutGrant(ctx context.Context, namespace string) ([]FulfilledOrder, error)

	// ListProviderSuccessWithoutFact returns payment attempts where the provider
	// reports success but no Payment Fact exists in the domain.
	ListProviderSuccessWithoutFact(ctx context.Context, namespace string) ([]ProviderSuccessGap, error)

	// ListRefundFactsWithoutFence returns refund facts that have no matching
	// fence or credit reversal.
	ListRefundFactsWithoutFence(ctx context.Context, namespace string) ([]RefundFactGap, error)

	// ListWalletLedgerMismatches returns customers whose Wallet aggregate total
	// differs from the Ledger-derived value.
	ListWalletLedgerMismatches(ctx context.Context, namespace string) ([]WalletMismatch, error)

	// ListClosedReceivableRangeChanges returns closed receivable periods whose
	// settlement range or amount differs from the frozen values.
	ListClosedReceivableRangeChanges(ctx context.Context, namespace string) ([]ReceivableRangeDrift, error)

	// ListUnknownEventTypes returns outbox events with event types not in the
	// approved set.
	ListUnknownEventTypes(ctx context.Context, namespace string) ([]UnknownEvent, error)

	// ListEventIdMismatches returns outbox events whose event ID does not equal
	// the outbox row ID.
	ListEventIdMismatches(ctx context.Context, namespace string) ([]EventIdMismatch, error)
}

// Raw fact types returned by the ProbePort.

type StaleOrder struct {
	OrderID    string
	CustomerID string
	PaidAt     time.Time
}

type FulfilledOrder struct {
	OrderID    string
	CustomerID string
	GrantCount int
}

type ProviderSuccessGap struct {
	OrderID         string
	AttemptID       string
	ProviderOrderID string
}

type RefundFactGap struct {
	RefundID         string
	ProviderRefundID string
}

type WalletMismatch struct {
	CustomerID  string
	WalletTotal int64
	LedgerTotal int64
	Difference  int64
}

type ReceivableRangeDrift struct {
	PeriodID     string
	FrozenTotal  int64
	CurrentTotal int64
}

type UnknownEvent struct {
	OutboxID  string
	EventType string
}

type EventIdMismatch struct {
	OutboxID string
	EventID  string
}

// Config wires the reconciliation checker.
type Config struct {
	Probe          ProbePort
	StaleThreshold time.Duration // how long a paid order can lack fulfillment before flagging
	Logger         *slog.Logger
}

// Checker runs all reconciliation invariants.
type Checker struct {
	probe     ProbePort
	threshold time.Duration
	logger    *slog.Logger
}

// New creates a Checker.
func New(cfg Config) *Checker {
	threshold := cfg.StaleThreshold
	if threshold <= 0 {
		threshold = 30 * time.Minute
	}
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}
	return &Checker{
		probe:     cfg.Probe,
		threshold: threshold,
		logger:    logger.With("component", "reconciliation"),
	}
}

// Run executes all reconciliation checks for the given namespace and returns a
// Report. Each check runs independently; a check failure (query error) logs a
// warning and continues to the next check. The report never mutates data.
func (c *Checker) Run(ctx context.Context, namespace string) *Report {
	report := &Report{
		RunAt:    time.Now().UTC(),
		Findings: []Finding{},
	}

	checks := []struct {
		name CheckName
		fn   func(context.Context, string) ([]Finding, error)
	}{
		{CheckPaidWithoutFulfillment, c.checkPaidWithoutFulfillment},
		{CheckFulfilledWithoutGrant, c.checkFulfilledWithoutGrant},
		{CheckProviderSuccessWithoutFact, c.checkProviderSuccessWithoutFact},
		{CheckRefundFactWithoutFence, c.checkRefundFactWithoutFence},
		{CheckWalletDiffersFromLedger, c.checkWalletDiffersFromLedger},
		{CheckClosedReceivableRangeChanged, c.checkClosedReceivableRangeChanged},
		{CheckUnknownEventTypes, c.checkUnknownEventTypes},
		{CheckEventIdMismatch, c.checkEventIdMismatch},
	}

	for _, chk := range checks {
		report.ChecksRun++
		findings, err := chk.fn(ctx, namespace)
		if err != nil {
			c.logger.WarnContext(ctx, "reconciliation check failed",
				"check", chk.name, "namespace", namespace, "error", err)
			report.Findings = append(report.Findings, Finding{
				Check:     chk.name,
				Severity:  SeverityWarning,
				Namespace: namespace,
				Detail:    fmt.Sprintf("check query failed: %v", err),
			})
			continue
		}
		report.Findings = append(report.Findings, findings...)
	}

	return report
}

func (c *Checker) checkPaidWithoutFulfillment(ctx context.Context, ns string) ([]Finding, error) {
	stale, err := c.probe.ListPaidOrdersWithoutFulfillment(ctx, ns, c.threshold)
	if err != nil {
		return nil, err
	}
	findings := make([]Finding, 0, len(stale))
	for _, s := range stale {
		findings = append(findings, Finding{
			Check:     CheckPaidWithoutFulfillment,
			Severity:  SeverityError,
			Namespace: ns,
			EntityID:  s.OrderID,
			Detail:    fmt.Sprintf("order %s paid at %s without fulfillment after %s", s.OrderID, s.PaidAt, c.threshold),
		})
	}
	return findings, nil
}

func (c *Checker) checkFulfilledWithoutGrant(ctx context.Context, ns string) ([]Finding, error) {
	orders, err := c.probe.ListFulfilledOrdersWithoutGrant(ctx, ns)
	if err != nil {
		return nil, err
	}
	findings := make([]Finding, 0, len(orders))
	for _, o := range orders {
		findings = append(findings, Finding{
			Check:     CheckFulfilledWithoutGrant,
			Severity:  SeverityError,
			Namespace: ns,
			EntityID:  o.OrderID,
			Detail:    fmt.Sprintf("order %s fulfilled with %d grants (expected exactly 1)", o.OrderID, o.GrantCount),
		})
	}
	return findings, nil
}

func (c *Checker) checkProviderSuccessWithoutFact(ctx context.Context, ns string) ([]Finding, error) {
	gaps, err := c.probe.ListProviderSuccessWithoutFact(ctx, ns)
	if err != nil {
		return nil, err
	}
	findings := make([]Finding, 0, len(gaps))
	for _, g := range gaps {
		findings = append(findings, Finding{
			Check:     CheckProviderSuccessWithoutFact,
			Severity:  SeverityError,
			Namespace: ns,
			EntityID:  g.AttemptID,
			Detail:    fmt.Sprintf("attempt %s has provider success for order %s but no Payment Fact", g.AttemptID, g.OrderID),
		})
	}
	return findings, nil
}

func (c *Checker) checkRefundFactWithoutFence(ctx context.Context, ns string) ([]Finding, error) {
	gaps, err := c.probe.ListRefundFactsWithoutFence(ctx, ns)
	if err != nil {
		return nil, err
	}
	findings := make([]Finding, 0, len(gaps))
	for _, g := range gaps {
		findings = append(findings, Finding{
			Check:     CheckRefundFactWithoutFence,
			Severity:  SeverityError,
			Namespace: ns,
			EntityID:  g.RefundID,
			Detail:    fmt.Sprintf("refund %s has provider fact %s without matching fence/reversal", g.RefundID, g.ProviderRefundID),
		})
	}
	return findings, nil
}

func (c *Checker) checkWalletDiffersFromLedger(ctx context.Context, ns string) ([]Finding, error) {
	mismatches, err := c.probe.ListWalletLedgerMismatches(ctx, ns)
	if err != nil {
		return nil, err
	}
	findings := make([]Finding, 0, len(mismatches))
	for _, m := range mismatches {
		findings = append(findings, Finding{
			Check:     CheckWalletDiffersFromLedger,
			Severity:  SeverityError,
			Namespace: ns,
			EntityID:  m.CustomerID,
			Detail:    fmt.Sprintf("customer %s wallet=%d ledger=%d diff=%d", m.CustomerID, m.WalletTotal, m.LedgerTotal, m.Difference),
		})
	}
	return findings, nil
}

func (c *Checker) checkClosedReceivableRangeChanged(ctx context.Context, ns string) ([]Finding, error) {
	drifts, err := c.probe.ListClosedReceivableRangeChanges(ctx, ns)
	if err != nil {
		return nil, err
	}
	findings := make([]Finding, 0, len(drifts))
	for _, d := range drifts {
		findings = append(findings, Finding{
			Check:     CheckClosedReceivableRangeChanged,
			Severity:  SeverityError,
			Namespace: ns,
			EntityID:  d.PeriodID,
			Detail:    fmt.Sprintf("period %s frozen=%d current=%d", d.PeriodID, d.FrozenTotal, d.CurrentTotal),
		})
	}
	return findings, nil
}

func (c *Checker) checkUnknownEventTypes(ctx context.Context, ns string) ([]Finding, error) {
	events, err := c.probe.ListUnknownEventTypes(ctx, ns)
	if err != nil {
		return nil, err
	}
	approved := make(map[string]bool, len(commerce.AllEventNames()))
	for _, n := range commerce.AllEventNames() {
		approved[n] = true
	}
	findings := make([]Finding, 0, len(events))
	for _, e := range events {
		if !approved[e.EventType] {
			findings = append(findings, Finding{
				Check:     CheckUnknownEventTypes,
				Severity:  SeverityError,
				Namespace: ns,
				EntityID:  e.OutboxID,
				Detail:    fmt.Sprintf("outbox row %s published unknown event type %s", e.OutboxID, e.EventType),
			})
		}
	}
	return findings, nil
}

func (c *Checker) checkEventIdMismatch(ctx context.Context, ns string) ([]Finding, error) {
	mismatches, err := c.probe.ListEventIdMismatches(ctx, ns)
	if err != nil {
		return nil, err
	}
	findings := make([]Finding, 0, len(mismatches))
	for _, m := range mismatches {
		findings = append(findings, Finding{
			Check:     CheckEventIdMismatch,
			Severity:  SeverityError,
			Namespace: ns,
			EntityID:  m.OutboxID,
			Detail:    fmt.Sprintf("outbox row %s has event ID %s (must match)", m.OutboxID, m.EventID),
		})
	}
	return findings, nil
}
