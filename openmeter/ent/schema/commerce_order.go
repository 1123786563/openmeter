package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/oklog/ulid/v2"

	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

// CommerceOrder is the immutable checkout header. Once created, total_cents
// and currency are frozen; only status transitions are allowed. The public_id
// is the customer-facing order number (ULID). Idempotency is enforced by the
// (namespace, customer_id, idempotency_key) unique index.
type CommerceOrder struct {
	ent.Schema
}

func (CommerceOrder) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.IDMixin{},
		entutils.NamespaceMixin{},
		entutils.TimeMixin{},
	}
}

func (CommerceOrder) Fields() []ent.Field {
	return []ent.Field{
		field.String("public_id").
			DefaultFunc(func() string {
				return ulid.Make().String()
			}).
			Unique().
			Immutable().
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}),
		field.String("customer_id").NotEmpty().Immutable(),
		field.Enum("kind").
			Values("plan_purchase", "subscription_renewal", "wallet_top_up").
			Immutable(),
		field.Enum("status").
			Values("created", "awaiting_payment", "paid", "fulfilled", "cancelled", "expired", "refund_pending", "partially_refunded", "refunded").
			Default("created"),
		field.Int64("total_cents").Min(0).Immutable(),
		field.String("currency").Default("CNY").Immutable(),
		field.String("idempotency_key").NotEmpty().Immutable(),
		field.String("description").Optional().Nillable(),
	}
}

func (CommerceOrder) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("lines", CommerceOrderLine.Type),
		edge.To("payment_attempts", PaymentAttempt.Type),
		edge.To("fulfillments", Fulfillment.Type),
		edge.To("refund_requests", RefundRequest.Type),
	}
}

func (CommerceOrder) Indexes() []ent.Index {
	return []ent.Index{
		// Idempotency: one order per (namespace, customer, idempotency_key).
		index.Fields("namespace", "customer_id", "idempotency_key").Unique(),
		// Status lookup.
		index.Fields("namespace", "customer_id", "status"),
	}
}
