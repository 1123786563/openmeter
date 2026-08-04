package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

// ManualResourceCost stores manually entered provider cost overrides for a
// resource/model combination. It is a mutable catalog record (supports soft delete).
type ManualResourceCost struct {
	ent.Schema
}

func (ManualResourceCost) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.IDMixin{},
		entutils.NamespaceMixin{},
		entutils.AnnotationsMixin{},
		entutils.TimeMixin{},
	}
}

func (ManualResourceCost) Fields() []ent.Field {
	return []ent.Field{
		field.String("resource_code").NotEmpty(),
		field.String("provider").Optional().Nillable(),
		field.String("model").Optional().Nillable(),
		field.String("cost_currency").Default("USD"),
		field.Other("cost_amount", alpacadecimal.Decimal{}).
			SchemaType(map[string]string{
				dialect.Postgres: "numeric",
			}),
		// Source records who entered the cost (e.g. "admin", "import").
		field.String("source").Default("manual"),
		field.Time("effective_from"),
		field.Time("effective_to").Optional().Nillable(),
	}
}

func (ManualResourceCost) Edges() []ent.Edge {
	return nil
}

func (ManualResourceCost) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "resource_code", "provider", "model", "effective_from").
			Annotations(entsql.IndexWhere("deleted_at IS NULL")).
			Unique(),
		index.Fields("namespace", "resource_code").
			Annotations(entsql.IndexWhere("deleted_at IS NULL")),
	}
}
