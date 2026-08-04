package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

// CommerceOutbox stores transactional outbox events for the commerce domain.
// Events are written atomically within the same transaction as the state
// transition they describe (e.g. order.paid), so a relay can forward them
// reliably even if the process crashes immediately after commit.
type CommerceOutbox struct {
	ent.Schema
}

func (CommerceOutbox) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.IDMixin{},
		entutils.NamespaceMixin{},
	}
}

func (CommerceOutbox) Fields() []ent.Field {
	return []ent.Field{
		field.Time("created_at").
			Default(clock.Now).
			Immutable(),
		field.String("aggregate_type").NotEmpty().Immutable(),
		field.String("aggregate_id").NotEmpty().Immutable(),
		field.String("event_type").NotEmpty().Immutable(),
		field.JSON("payload", map[string]any{}).
			Immutable().
			SchemaType(map[string]string{
				dialect.Postgres: "jsonb",
			}),
		field.Bool("published").Default(false),
		field.Time("published_at").Optional().Nillable(),
	}
}

func (CommerceOutbox) Edges() []ent.Edge {
	return nil
}

func (CommerceOutbox) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "published"),
		index.Fields("namespace", "aggregate_id"),
	}
}
