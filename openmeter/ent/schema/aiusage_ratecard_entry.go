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

// AIUsageRatecardEntry stores customer rate card entries mapping resources to prices.
type AIUsageRatecardEntry struct {
	ent.Schema
}

func (AIUsageRatecardEntry) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.IDMixin{},
		entutils.NamespaceMixin{},
		entutils.AnnotationsMixin{},
		entutils.TimeMixin{},
	}
}

func (AIUsageRatecardEntry) Fields() []ent.Field {
	return []ent.Field{
		field.String("customer_id").Optional().Nillable(),
		field.String("resource_code").NotEmpty(),
		field.String("provider").Optional().Nillable(),
		field.String("model").Optional().Nillable(),
		field.Other("price_per_unit_cny", alpacadecimal.Decimal{}).
			SchemaType(map[string]string{
				dialect.Postgres: "numeric",
			}),
		field.Int64("credit_rate").Default(1000),
		field.Time("effective_from"),
		field.Time("effective_to").Optional().Nillable(),
	}
}

func (AIUsageRatecardEntry) Indexes() []ent.Index {
	return []ent.Index{
		// Unique rate card entry per scope.
		index.Fields("namespace", "customer_id", "resource_code", "provider", "model", "effective_from").
			Annotations(entsql.IndexWhere("deleted_at IS NULL")).
			Unique(),
		// Default lookup.
		index.Fields("namespace", "resource_code").
			Annotations(entsql.IndexWhere("deleted_at IS NULL")),
	}
}
