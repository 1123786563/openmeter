package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

// AIUsageOutbox stores transactional outbox events written atomically with the
// batch they belong to. A relay reads published=false rows and publishes them.
type AIUsageOutbox struct {
	ent.Schema
}

func (AIUsageOutbox) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.IDMixin{},
		entutils.NamespaceMixin{},
	}
}

func (AIUsageOutbox) Fields() []ent.Field {
	return []ent.Field{
		field.Time("created_at").
			Default(clock.Now).
			Immutable(),
		field.String("customer_id").NotEmpty().Immutable(),
		field.String("subject_id").NotEmpty().Immutable(),
		field.String("event_type").NotEmpty().Immutable(),
		// Payload carries the event body as JSON.
		field.JSON("payload", map[string]any{}).
			Immutable().
			SchemaType(map[string]string{
				dialect.Postgres: "jsonb",
			}),
		// Published tracks whether a relay has forwarded the event.
		field.Bool("published").Default(false),
		field.Time("published_at").Optional().Nillable(),
		// Worker lease fields for the outbox relay.
		field.String("owner").Default(""),
		field.Int("claim_count").Default(0),
		field.Time("leased_until").Optional().Nillable(),
		field.Bool("dead_lettered").Default(false),
		field.String("dead_letter_reason").Default(""),
	}
}

func (AIUsageOutbox) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("batch", AIUsageBatch.Type).
			Ref("outbox_events").
			Required().
			Immutable().
			Unique(),
	}
}

func (AIUsageOutbox) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "published"),
		index.Fields("namespace", "customer_id"),
		// Relay claim index: unpublished, not dead-lettered, lease expired.
		index.Fields("namespace", "published", "dead_lettered", "leased_until"),
	}
}
