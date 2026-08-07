package adapter

import (
	"context"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/openmeter/aiusage"
	"github.com/openmeterio/openmeter/openmeter/aiusage/pricing"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ent/db/aiusageratecardentry"
)

// dbRateEntryProvider implements pricing.RateEntryProvider by reading active
// rate card entries from the ent database. It replaces the static
// configRateEntryProvider for production use.
type dbRateEntryProvider struct {
	db *db.Client
}

// NewDBRateEntryProvider creates a DB-backed pricing.RateEntryProvider.
func NewDBRateEntryProvider(client *db.Client) pricing.RateEntryProvider {
	return &dbRateEntryProvider{db: client}
}

func (p *dbRateEntryProvider) GetEntries(ctx context.Context, _ string) ([]pricing.RateEntry, error) {
	now := time.Now().UTC()
	ents, err := p.db.AIUsageRatecardEntry.Query().
		Where(
			aiusageratecardentry.EffectiveFromLTE(now),
			aiusageratecardentry.Or(
				aiusageratecardentry.EffectiveToIsNil(),
				aiusageratecardentry.EffectiveToGTE(now),
			),
		).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("db rate entry provider: query: %w", err)
	}

	entries := make([]pricing.RateEntry, 0, len(ents))
	for _, e := range ents {
		entry := pricing.RateEntry{
			ResourceCode:   aiusage.ResourceCode(e.ResourceCode),
			Provider:       derefStr(e.Provider),
			Model:          derefStr(e.Model),
			CreditsPerUnit: e.CreditsPerUnit,
			UnitSize:       e.UnitSize,
			EffectiveFrom:  e.EffectiveFrom,
		}
		if e.EffectiveTo != nil {
			entry.EffectiveTo = e.EffectiveTo
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

var _ pricing.RateEntryProvider = (*dbRateEntryProvider)(nil)

func derefStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
