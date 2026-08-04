package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

// CommerceProduct is a catalog entry for purchasable items: plans, renewals,
// and wallet top-ups. Soft-deletable; price and kind are immutable once set.
type CommerceProduct struct {
	ent.Schema
}

func (CommerceProduct) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.IDMixin{},
		entutils.NamespaceMixin{},
		entutils.AnnotationsMixin{},
		entutils.TimeMixin{},
	}
}

func (CommerceProduct) Fields() []ent.Field {
	return []ent.Field{
		field.String("sku").NotEmpty(),
		field.String("name").NotEmpty(),
		field.Enum("kind").
			Values("plan_purchase", "subscription_renewal", "wallet_top_up"),
		field.Int64("price_cents").Min(0),
		field.String("currency").Default("CNY"),
		field.String("description").Optional().Nillable(),
		field.JSON("metadata", map[string]any{}).
			Optional().
			SchemaType(map[string]string{
				dialect.Postgres: "jsonb",
			}),
	}
}

func (CommerceProduct) Edges() []ent.Edge {
	return nil
}

func (CommerceProduct) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "sku").
			Unique(),
		index.Fields("namespace", "kind"),
	}
}
