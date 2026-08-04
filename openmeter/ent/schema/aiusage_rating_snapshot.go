package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

// AIUsageRatingSnapshot stores the cost and sales price resolution for one line item.
// Immutable append-only record; no soft delete.
type AIUsageRatingSnapshot struct {
	ent.Schema
}

func (AIUsageRatingSnapshot) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.IDMixin{},
		entutils.NamespaceMixin{},
		entutils.AnnotationsMixin{},
	}
}

func (AIUsageRatingSnapshot) Fields() []ent.Field {
	return []ent.Field{
		field.String("resource_code").NotEmpty(),
		field.String("cost_currency").Default("USD"),
		field.Other("cost_amount", alpacadecimal.Decimal{}).
			SchemaType(map[string]string{
				dialect.Postgres: "numeric",
			}),
		field.String("cost_source").Default(""),
		field.String("sales_currency").Default("CNY"),
		field.Other("sales_amount", alpacadecimal.Decimal{}).
			SchemaType(map[string]string{
				dialect.Postgres: "numeric",
			}),
		field.String("rate_card_version").Default(""),
		field.Int64("credits").Default(0),
	}
}

func (AIUsageRatingSnapshot) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("batch", AIUsageBatch.Type).
			Ref("rating_snapshots").
			Required().
			Unique(),
	}
}
