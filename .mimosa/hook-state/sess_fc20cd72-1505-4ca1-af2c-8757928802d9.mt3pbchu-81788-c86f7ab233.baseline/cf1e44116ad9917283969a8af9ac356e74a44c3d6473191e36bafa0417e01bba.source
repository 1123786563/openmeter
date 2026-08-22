package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

// Fulfillment records the grant or credit provisioning triggered by a paid
// order. At most one fulfillment per order may reach status='fulfilled'
// (enforced by a partial unique index).
type Fulfillment struct {
	ent.Schema
}

func (Fulfillment) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.IDMixin{},
		entutils.NamespaceMixin{},
		entutils.TimeMixin{},
	}
}

func (Fulfillment) Fields() []ent.Field {
	return []ent.Field{
		field.String("commerce_order_id").Immutable().SchemaType(map[string]string{
			dialect.Postgres: "char(26)",
		}),
		field.String("customer_id").NotEmpty().Immutable(),
		field.Enum("status").
			Values("pending", "processing", "fulfilled", "failed").
			Default("pending"),
		// claimed_at records when a worker started processing; records stuck
		// in processing past the lease timeout become eligible for re-claim.
		field.Time("claimed_at").Optional().Nillable(),
		// grant_id references the Grant created by this fulfillment.
		field.String("grant_id").Optional().Nillable().SchemaType(map[string]string{
			dialect.Postgres: "char(26)",
		}),
		field.Int64("credits_granted").Min(0).Default(0),
		field.Time("fulfilled_at").Optional().Nillable(),
		field.String("failure_reason").Optional().Nillable(),
	}
}

func (Fulfillment) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("order", CommerceOrder.Type).
			Ref("fulfillments").
			Field("commerce_order_id").
			Required().
			Immutable().
			Unique(),
	}
}

func (Fulfillment) Indexes() []ent.Index {
	return []ent.Index{
		// One successful fulfillment per order.
		index.Fields("namespace", "commerce_order_id").
			Annotations(entsql.IndexWhere("status = 'fulfilled'")).
			Unique(),
		// Customer lookup.
		index.Fields("namespace", "customer_id", "status"),
	}
}
