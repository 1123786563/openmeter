package pricing

import (
	"time"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/aiusage"
)

// ResolvedLine is the resolved cost/sales price for one merged usage line.
// Cost (provider) and Credits (customer-facing) are separated so that BYOK
// usage can have zero customer credits while platform resources still charge.
type ResolvedLine struct {
	ResourceCode     aiusage.ResourceCode
	CustomerCredits  int64
	ProviderCost     alpacadecimal.Decimal
	ProviderCurrency string
	CostSource       string
}

// ResolvedBatch is the full resolution of a ResolveInput: one ResolvedLine per
// merged usage group, the total customer credit charge, and the billing mode.
type ResolvedBatch struct {
	Lines        []ResolvedLine
	TotalCredits int64
	BillingMode  aiusage.BillingMode
}

// ResolveInput is the input to the pricing Service. It carries the occurrence
// time (for effective-from/to filtering), the rate package version, the billing
// mode, the provider-managed flag (BYOK detection), and the raw usage lines.
type ResolveInput struct {
	OccurredAt         time.Time
	RatePackageVersion string
	BillingMode        aiusage.BillingMode
	ProviderManaged    bool
	Lines              []aiusage.UsageLineInput
}

// RateEntry is a single customer-facing rate: credits charged per unit for a
// resource scoped by provider and model. The pricing service selects the most
// specific active entry for each line and errors on ambiguity.
type RateEntry struct {
	ResourceCode   aiusage.ResourceCode
	Provider       string
	Model          string
	CreditsPerUnit int64
	UnitSize       int64
	EffectiveFrom  time.Time
	EffectiveTo    *time.Time
}

// IsActiveAt returns true when the entry is effective at the given time.
func (e RateEntry) IsActiveAt(at time.Time) bool {
	if at.Before(e.EffectiveFrom) {
		return false
	}
	if e.EffectiveTo != nil && at.After(*e.EffectiveTo) {
		return false
	}
	return true
}

// matchesProvider returns true when the entry's provider scope includes the
// given provider. An empty Provider means "all providers".
func (e RateEntry) matchesProvider(provider string) bool {
	return e.Provider == "" || e.Provider == provider
}

// matchesModel returns true when the entry's model scope includes the given
// model. An empty Model means "all models".
func (e RateEntry) matchesModel(model string) bool {
	return e.Model == "" || e.Model == model
}

// specificity returns a priority score for rate selection. Higher wins.
//
//	provider + model specific = 3 (most specific)
//	provider only             = 2
//	resource only             = 1 (least specific)
func (e RateEntry) specificity() int {
	switch {
	case e.Provider != "" && e.Model != "":
		return 3
	case e.Provider != "":
		return 2
	default:
		return 1
	}
}
