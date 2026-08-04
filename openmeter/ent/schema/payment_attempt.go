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

// PaymentAttempt tracks a single attempt to pay an order through a provider.
// provider + provider_order_id and provider + provider_payment_id are unique
// when present (partial unique index). The idempotency_key is unique per
// (namespace, customer_id).
type PaymentAttempt struct {
	ent.Schema
}

func (PaymentAttempt) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.IDMixin{},
		entutils.NamespaceMixin{},
		entutils.TimeMixin{},
	}
}

func (PaymentAttempt) Fields() []ent.Field {
	return []ent.Field{
		field.String("commerce_order_id").Immutable().SchemaType(map[string]string{
			dialect.Postgres: "char(26)",
		}),
		field.String("customer_id").NotEmpty().Immutable(),
		field.Enum("provider").
			Values("wechat", "alipay", "offline").
			Immutable(),
		// provider_order_id is the order identifier assigned by the payment provider.
		field.String("provider_order_id").Optional().Nillable(),
		// provider_payment_id is the payment identifier assigned by the provider.
		field.String("provider_payment_id").Optional().Nillable(),
		field.Enum("status").
			Values("pending", "succeeded", "failed", "cancelled", "expired").
			Default("pending"),
		// provider_session_id carries provider-specific session data (e.g. WeChat prepay_id).
		field.String("provider_session_id").Optional().Nillable(),
		field.String("idempotency_key").NotEmpty().Immutable(),
		field.Int64("amount_cents").Min(0).Immutable(),
		field.String("currency").Default("CNY").Immutable(),
	}
}

func (PaymentAttempt) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("order", CommerceOrder.Type).
			Ref("payment_attempts").
			Field("commerce_order_id").
			Required().
			Immutable().
			Unique(),
		edge.To("facts", PaymentFact.Type),
	}
}

func (PaymentAttempt) Indexes() []ent.Index {
	return []ent.Index{
		// Idempotency: one attempt per (namespace, customer, idempotency_key).
		index.Fields("namespace", "customer_id", "idempotency_key").Unique(),
		// Provider order id uniqueness (when present).
		index.Fields("namespace", "provider", "provider_order_id").
			Annotations(entsql.IndexWhere("provider_order_id IS NOT NULL")).
			Unique(),
		// Provider payment id uniqueness (when present).
		index.Fields("namespace", "provider", "provider_payment_id").
			Annotations(entsql.IndexWhere("provider_payment_id IS NOT NULL")).
			Unique(),
		// Order lookup.
		index.Fields("namespace", "commerce_order_id"),
		// Status queries.
		index.Fields("namespace", "customer_id", "status"),
	}
}
