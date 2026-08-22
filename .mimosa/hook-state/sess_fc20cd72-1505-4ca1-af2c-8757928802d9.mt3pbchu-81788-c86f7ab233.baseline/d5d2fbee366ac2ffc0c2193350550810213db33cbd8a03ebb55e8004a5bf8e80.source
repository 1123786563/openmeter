package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

// CreditReservationOutbox is a transactional relay for standard usage events.
// It stores delivery bookkeeping only and deliberately has no balance or
// ledger-specific columns.
type CreditReservationOutbox struct {
	ent.Schema
}

func (CreditReservationOutbox) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.IDMixin{},
		entutils.NamespaceMixin{},
	}
}

func (CreditReservationOutbox) Fields() []ent.Field {
	return []ent.Field{
		field.Time("created_at").Default(clock.Now).Immutable(),
		field.String("event_id").NotEmpty().Immutable(),
		field.String("aggregate_type").NotEmpty().Immutable(),
		field.String("aggregate_id").NotEmpty().Immutable(),
		field.String("event_type").NotEmpty().Immutable(),
		field.JSON("payload", map[string]any{}).
			Immutable().
			SchemaType(map[string]string{dialect.Postgres: "jsonb"}),
		field.Bool("published").Default(false),
		field.Time("published_at").Optional().Nillable(),
		field.String("owner").Default(""),
		field.Int("claim_count").Default(0),
		field.Time("leased_until").Optional().Nillable(),
		field.Bool("dead_lettered").Default(false),
		field.String("dead_letter_reason").Default(""),
	}
}

func (CreditReservationOutbox) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "event_id").Unique(),
		index.Fields("namespace", "published", "dead_lettered", "leased_until"),
		index.Fields("namespace", "aggregate_id"),
	}
}
