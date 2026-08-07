package adapter

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/openmeterio/openmeter/openmeter/aiusage"
	"github.com/openmeterio/openmeter/openmeter/aiusage/ratecard"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ent/db/aiusageratecardentry"
	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
)

// rateCardRepository implements ratecard.Service backed by the ent client.
type rateCardRepository struct {
	db     *db.Client
	logger *slog.Logger
}

// NewRateCardRepository creates an ent-backed rate card service.
func NewRateCardRepository(client *db.Client, logger *slog.Logger) ratecard.Service {
	return &rateCardRepository{db: client, logger: logger}
}

func (r *rateCardRepository) Create(ctx context.Context, input ratecard.RateCardEntryInput) (*ratecard.RateCardEntry, error) {
	cmd := r.db.AIUsageRatecardEntry.Create().
		SetNamespace(input.Namespace).
		SetResourceCode(string(input.ResourceCode)).
		SetCreditsPerUnit(input.CreditsPerUnit).
		SetUnitSize(input.UnitSize).
		SetEffectiveFrom(input.EffectiveFrom)

	if input.CustomerID != "" {
		cmd = cmd.SetCustomerID(input.CustomerID)
	}
	if input.Provider != "" {
		cmd = cmd.SetProvider(input.Provider)
	}
	if input.Model != "" {
		cmd = cmd.SetModel(input.Model)
	}
	if input.EffectiveTo != nil {
		cmd = cmd.SetEffectiveTo(*input.EffectiveTo)
	}

	ent, err := cmd.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("rate card repository: create: %w", err)
	}
	return r.mapEntToEntry(ent), nil
}

func (r *rateCardRepository) Get(ctx context.Context, namespace, id string) (*ratecard.RateCardEntry, error) {
	ent, err := r.db.AIUsageRatecardEntry.Query().
		Where(aiusageratecardentry.IDEQ(id), aiusageratecardentry.NamespaceEQ(namespace)).
		Only(ctx)
	if err != nil {
		if db.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("rate card repository: get: %w", err)
	}
	return r.mapEntToEntry(ent), nil
}

func (r *rateCardRepository) List(ctx context.Context, params ratecard.ListParams) ([]ratecard.RateCardEntry, error) {
	predicates := []predicate.AIUsageRatecardEntry{
		aiusageratecardentry.NamespaceEQ(params.Namespace),
	}
	if params.ResourceCode != "" {
		predicates = append(predicates, aiusageratecardentry.ResourceCodeEQ(params.ResourceCode))
	}
	if params.Provider != "" {
		predicates = append(predicates, aiusageratecardentry.ProviderEQ(params.Provider))
	}
	if params.Model != "" {
		predicates = append(predicates, aiusageratecardentry.ModelEQ(params.Model))
	}
	if params.ActiveOnly {
		now := time.Now().UTC()
		predicates = append(predicates, aiusageratecardentry.EffectiveFromLTE(now))
		predicates = append(predicates, aiusageratecardentry.Or(
			aiusageratecardentry.EffectiveToIsNil(),
			aiusageratecardentry.EffectiveToGTE(now),
		))
	}

	ents, err := r.db.AIUsageRatecardEntry.Query().
		Where(predicates...).
		Order(db.Asc(aiusageratecardentry.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("rate card repository: list: %w", err)
	}

	result := make([]ratecard.RateCardEntry, 0, len(ents))
	for _, e := range ents {
		result = append(result, *r.mapEntToEntry(e))
	}
	return result, nil
}

func (r *rateCardRepository) Update(ctx context.Context, namespace, id string, input ratecard.RateCardEntryInput) (*ratecard.RateCardEntry, error) {
	cmd := r.db.AIUsageRatecardEntry.UpdateOneID(id).
		SetResourceCode(string(input.ResourceCode)).
		SetCreditsPerUnit(input.CreditsPerUnit).
		SetUnitSize(input.UnitSize).
		SetEffectiveFrom(input.EffectiveFrom).
		ClearProvider().ClearModel().ClearCustomerID().ClearEffectiveTo()

	if input.CustomerID != "" {
		cmd = cmd.SetCustomerID(input.CustomerID)
	}
	if input.Provider != "" {
		cmd = cmd.SetProvider(input.Provider)
	}
	if input.Model != "" {
		cmd = cmd.SetModel(input.Model)
	}
	if input.EffectiveTo != nil {
		cmd = cmd.SetEffectiveTo(*input.EffectiveTo)
	}

	ent, err := cmd.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("rate card repository: update: %w", err)
	}
	return r.mapEntToEntry(ent), nil
}

func (r *rateCardRepository) Delete(ctx context.Context, namespace, id string) error {
	now := time.Now().UTC()
	_, err := r.db.AIUsageRatecardEntry.UpdateOneID(id).
		Where(aiusageratecardentry.NamespaceEQ(namespace)).
		SetEffectiveTo(now).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("rate card repository: delete (soft): %w", err)
	}
	return nil
}

func (r *rateCardRepository) BootstrapSeed(ctx context.Context, namespace string, entries []ratecard.RateCardEntryInput) error {
	count, err := r.db.AIUsageRatecardEntry.Query().
		Where(aiusageratecardentry.NamespaceEQ(namespace)).
		Count(ctx)
	if err != nil {
		return fmt.Errorf("rate card repository: seed count: %w", err)
	}
	if count > 0 {
		return nil // already seeded
	}

	bulk := make([]*db.AIUsageRatecardEntryCreate, 0, len(entries))
	for _, input := range entries {
		cmd := r.db.AIUsageRatecardEntry.Create().
			SetNamespace(namespace).
			SetResourceCode(string(input.ResourceCode)).
			SetCreditsPerUnit(input.CreditsPerUnit).
			SetUnitSize(input.UnitSize).
			SetEffectiveFrom(input.EffectiveFrom)
		if input.Provider != "" {
			cmd = cmd.SetProvider(input.Provider)
		}
		if input.Model != "" {
			cmd = cmd.SetModel(input.Model)
		}
		bulk = append(bulk, cmd)
	}

	if len(bulk) > 0 {
		if _, err := r.db.AIUsageRatecardEntry.CreateBulk(bulk...).Save(ctx); err != nil {
			return fmt.Errorf("rate card repository: seed bulk: %w", err)
		}
	}
	r.logger.InfoContext(ctx, "rate card bootstrap seed complete", "count", len(entries), "namespace", namespace)
	return nil
}

func (r *rateCardRepository) mapEntToEntry(e *db.AIUsageRatecardEntry) *ratecard.RateCardEntry {
	entry := &ratecard.RateCardEntry{
		ID:             e.ID,
		Namespace:      e.Namespace,
		ResourceCode:   aiusage.ResourceCode(e.ResourceCode),
		Provider:       derefStr(e.Provider),
		Model:          derefStr(e.Model),
		CreditsPerUnit: e.CreditsPerUnit,
		UnitSize:       e.UnitSize,
		EffectiveFrom:  e.EffectiveFrom,
		CreatedAt:      e.CreatedAt,
		UpdatedAt:      e.UpdatedAt,
	}
	if e.CustomerID != nil {
		entry.CustomerID = *e.CustomerID
	}
	if e.EffectiveTo != nil {
		entry.EffectiveTo = e.EffectiveTo
	}
	return entry
}

var _ ratecard.Service = (*rateCardRepository)(nil)
