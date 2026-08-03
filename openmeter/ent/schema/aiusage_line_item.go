package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"

	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

// AIUsageLineItem stores one billable resource within a batch.
// Immutable append-only record; no soft delete.
type AIUsageLineItem struct {
	ent.Schema
}

func (AIUsageLineItem) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.IDMixin{},
		entutils.NamespaceMixin{},
		entutils.AnnotationsMixin{},
	}
}

func (AIUsageLineItem) Fields() []ent.Field {
	return []ent.Field{
		field.String("resource_code").NotEmpty(),
		field.Int64("quantity"),
		field.String("provider").Default(""),
		field.String("model").Default(""),
		field.Bool("provider_managed").Default(true),
		field.JSON("dimensions", map[string]string{}).
			Optional().
			SchemaType(map[string]string{
				dialect.Postgres: "jsonb",
			}),
	}
}

func (AIUsageLineItem) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("batch", AIUsageBatch.Type).
			Ref("line_items").
			Required().
			Unique(),
	}
}
