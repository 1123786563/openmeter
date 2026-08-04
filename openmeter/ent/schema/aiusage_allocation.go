package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

// AIUsageAllocation stores the Credit deduction from a single funding source
// (grant) during batch settlement. It is an immutable append-only fact.
type AIUsageAllocation struct {
	ent.Schema
}

func (AIUsageAllocation) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.IDMixin{},
		entutils.NamespaceMixin{},
	}
}

func (AIUsageAllocation) Fields() []ent.Field {
	return []ent.Field{
		field.Time("created_at").
			Default(clock.Now).
			Immutable(),
		field.String("customer_id").NotEmpty().Immutable(),
		field.String("subject_id").NotEmpty().Immutable(),
		field.String("grant_id").NotEmpty().Immutable().SchemaType(map[string]string{
			dialect.Postgres: "char(26)",
		}),
		field.Other("amount", alpacadecimal.Decimal{}).
			Immutable().
			SchemaType(map[string]string{
				dialect.Postgres: "numeric",
			}),
		field.Uint8("priority").Default(0).Immutable(),
		field.String("funding_source").Default("").Immutable(),
	}
}

func (AIUsageAllocation) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("batch", AIUsageBatch.Type).
			Ref("allocations").
			Required().
			Immutable().
			Unique(),
	}
}

func (AIUsageAllocation) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "customer_id"),
	}
}
