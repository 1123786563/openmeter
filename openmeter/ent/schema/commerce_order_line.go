package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

// CommerceOrderLine is a single line item within an order. Immutable append-only
// record. product_id references CommerceProduct by ULID (stored as char(26));
// the reference is a loose FK because products may be soft-deleted.
type CommerceOrderLine struct {
	ent.Schema
}

func (CommerceOrderLine) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.IDMixin{},
		entutils.NamespaceMixin{},
	}
}

func (CommerceOrderLine) Fields() []ent.Field {
	return []ent.Field{
		field.String("commerce_order_id").Immutable().SchemaType(map[string]string{
			dialect.Postgres: "char(26)",
		}),
		field.String("product_id").Immutable().SchemaType(map[string]string{
			dialect.Postgres: "char(26)",
		}),
		field.String("product_sku").Default("").Immutable(),
		field.String("product_name").Default("").Immutable(),
		field.Int32("quantity").Min(0).Immutable(),
		field.Int64("unit_price_cents").Min(0).Immutable(),
		field.Int64("subtotal_cents").Min(0).Immutable(),
		field.JSON("snapshot_data", map[string]any{}).
			Optional().
			SchemaType(map[string]string{
				dialect.Postgres: "jsonb",
			}),
	}
}

func (CommerceOrderLine) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("order", CommerceOrder.Type).
			Ref("lines").
			Field("commerce_order_id").
			Required().
			Immutable().
			Unique(),
	}
}

func (CommerceOrderLine) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "product_id"),
	}
}
