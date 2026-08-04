package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

// RefundRequest tracks a customer-initiated refund. The status flows through
// a state machine: pending_fence -> provider_processing -> ledger_reversing ->
// fulfilled (or failed at any stage). amount_cents and currency are immutable.
type RefundRequest struct {
	ent.Schema
}

func (RefundRequest) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.IDMixin{},
		entutils.NamespaceMixin{},
		entutils.TimeMixin{},
	}
}

func (RefundRequest) Fields() []ent.Field {
	return []ent.Field{
		field.String("commerce_order_id").Immutable().SchemaType(map[string]string{
			dialect.Postgres: "char(26)",
		}),
		field.String("customer_id").NotEmpty().Immutable(),
		field.Int64("amount_cents").Min(0).Immutable(),
		field.String("currency").Default("CNY").Immutable(),
		field.Enum("status").
			Values("pending_fence", "provider_processing", "ledger_reversing", "fulfilled", "failed").
			Default("pending_fence"),
		field.String("reason").Optional().Nillable(),
		field.String("idempotency_key").NotEmpty().Immutable(),
	}
}

func (RefundRequest) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("order", CommerceOrder.Type).
			Ref("refund_requests").
			Field("commerce_order_id").
			Required().
			Immutable().
			Unique(),
		edge.To("facts", RefundFact.Type),
	}
}

func (RefundRequest) Indexes() []ent.Index {
	return []ent.Index{
		// Idempotency: one refund per (namespace, customer, idempotency_key).
		index.Fields("namespace", "customer_id", "idempotency_key").Unique(),
		// Order lookup.
		index.Fields("namespace", "commerce_order_id"),
		// Status queries.
		index.Fields("namespace", "customer_id", "status"),
	}
}
