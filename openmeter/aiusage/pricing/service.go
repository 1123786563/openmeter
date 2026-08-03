package pricing

import (
	"context"
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/aiusage"
)

// RateEntryProvider supplies the rate entries relevant to a resolution.
// In production this would query the rate-card store scoped by namespace,
// customer, and rate-package version. For tests, a static slice is sufficient.
type RateEntryProvider interface {
	GetEntries(ctx context.Context, ratePackageVersion string) ([]RateEntry, error)
}

// Service resolves usage lines into separated cost (provider) and credit
// (customer-facing) charges.
type Service struct {
	rates RateEntryProvider
}

// NewService creates a pricing Service backed by the given rate provider.
func NewService(rates RateEntryProvider) *Service {
	return &Service{rates: rates}
}

// Resolve normalizes, rates, and accumulates usage lines into a ResolvedBatch.
//
// Behaviour:
//   - Homogeneous lines (same resource+provider+model) are merged before rating.
//   - Each line is rated via CeilCredits(quantity, creditsPerUnit, unitSize).
//   - BYOK provider-managed tokens: CustomerCredits=0 AND ProviderCost=0.
//   - Platform resources under BYOK retain their configured sales credit.
//   - Ambiguous rates (two equally specific active entries) → ErrAmbiguousRate.
//   - Zero quantity → 0 credits.
//   - Component and bundle modes are mutually exclusive within a batch.
func (s *Service) Resolve(ctx context.Context, input ResolveInput) (ResolvedBatch, error) {
	// Billing-mode mutual exclusivity: if the input carries both line items
	// (component) and a ceiling (bundle) we reject it.
	if input.BillingMode == aiusage.BillingModeBundle && len(input.Lines) > 0 {
		return ResolvedBatch{}, aiusage.ErrBillingModeConflict
	}

	if len(input.Lines) == 0 {
		return ResolvedBatch{BillingMode: input.BillingMode}, nil
	}

	// Validate each line's resource code up front.
	for i, line := range input.Lines {
		if err := line.ResourceCode.Validate(); err != nil {
			return ResolvedBatch{}, fmt.Errorf("pricing: line[%d] invalid resource code %q: %w", i, line.ResourceCode, err)
		}
	}

	// Merge homogeneous lines (same resource + provider + model).
	merged := mergeLines(input.Lines)

	entries, err := s.rates.GetEntries(ctx, input.RatePackageVersion)
	if err != nil {
		return ResolvedBatch{}, fmt.Errorf("pricing: failed to load rate entries: %w", err)
	}

	batch := ResolvedBatch{
		Lines:        make([]ResolvedLine, 0, len(merged)),
		BillingMode:  input.BillingMode,
		TotalCredits: 0,
	}

	for _, m := range merged {
		line, err := s.resolveLine(ctx, m, entries, input.OccurredAt, input.ProviderManaged)
		if err != nil {
			return ResolvedBatch{}, err
		}
		batch.Lines = append(batch.Lines, line)
		batch.TotalCredits += line.CustomerCredits
	}

	return batch, nil
}

// mergedLine is a homogeneous group of usage lines after merging.
type mergedLine struct {
	ResourceCode aiusage.ResourceCode
	Provider     string
	Model        string
	Quantity     int64
}

// mergeLines combines lines that share the same resource code, provider, and
// model into a single entry with summed quantity. Order of first appearance
// is preserved.
func mergeLines(lines []aiusage.UsageLineInput) []mergedLine {
	type key struct {
		resource aiusage.ResourceCode
		provider string
		model    string
	}

	order := make([]key, 0)
	merged := make(map[key]*mergedLine)

	for _, l := range lines {
		k := key{resource: l.ResourceCode, provider: l.Provider, model: l.Model}
		if existing, ok := merged[k]; ok {
			existing.Quantity += l.Quantity
		} else {
			merged[k] = &mergedLine{
				ResourceCode: l.ResourceCode,
				Provider:     l.Provider,
				Model:        l.Model,
				Quantity:     l.Quantity,
			}
			order = append(order, k)
		}
	}

	result := make([]mergedLine, 0, len(order))
	for _, k := range order {
		result = append(result, *merged[k])
	}
	return result
}

// resolveLine rates a single merged line, handling BYOK and ambiguity.
func (s *Service) resolveLine(_ context.Context, m mergedLine, entries []RateEntry, at time.Time, providerManaged bool) (ResolvedLine, error) {
	// BYOK + provider-managed resource: zero customer credits and zero provider cost.
	if !providerManaged && m.ResourceCode.IsProviderManaged() {
		return ResolvedLine{
			ResourceCode:    m.ResourceCode,
			CustomerCredits: 0,
			ProviderCost:    alpacadecimal.NewFromInt(0),
			CostSource:      "byok",
		}, nil
	}

	// Zero quantity always yields zero credits.
	if m.Quantity == 0 {
		return ResolvedLine{
			ResourceCode:    m.ResourceCode,
			CustomerCredits: 0,
			ProviderCost:    alpacadecimal.NewFromInt(0),
		}, nil
	}

	entry, err := selectRate(entries, m, at)
	if err != nil {
		return ResolvedLine{}, err
	}

	credits, err := aiusage.CeilCredits(m.Quantity, entry.CreditsPerUnit, entry.UnitSize)
	if err != nil {
		return ResolvedLine{}, fmt.Errorf("pricing: credit calculation for %s: %w", m.ResourceCode, err)
	}

	return ResolvedLine{
		ResourceCode:    m.ResourceCode,
		CustomerCredits: credits,
		ProviderCost:    alpacadecimal.NewFromInt(0), // provider cost is resolved by a separate cost resolver
		CostSource:      "rate_card",
	}, nil
}

// selectRate picks the single most specific active rate entry for a merged line.
// Two entries at the same top specificity → ErrAmbiguousRate.
// No match → ErrRateMissing.
func selectRate(entries []RateEntry, m mergedLine, at time.Time) (RateEntry, error) {
	var best *RateEntry
	bestSpec := -1
	bestCount := 0

	for i := range entries {
		e := &entries[i]
		if e.ResourceCode != m.ResourceCode {
			continue
		}
		if !e.IsActiveAt(at) {
			continue
		}
		if !e.matchesProvider(m.Provider) {
			continue
		}
		if !e.matchesModel(m.Model) {
			continue
		}

		spec := e.specificity()
		if spec > bestSpec {
			best = e
			bestSpec = spec
			bestCount = 1
		} else if spec == bestSpec {
			bestCount++
		}
	}

	if bestCount == 0 {
		return RateEntry{}, aiusage.ErrRateMissing
	}
	if bestCount > 1 {
		return RateEntry{}, aiusage.ErrAmbiguousRate
	}
	return *best, nil
}
