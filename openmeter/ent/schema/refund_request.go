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
// The quantum fields (credit_quantum, refund_quantum_fen) persist the exact
// Credit-to-money ratio for every refund. reserved_credits is the Credits fenced
// for this refund; remainder_credits is the sub-quantum Credit left available.
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

		// Persisted quantum for every refund (10 Credit : 1 fen).
		field.Int64("credit_quantum").Default(10),
		field.Int64("refund_quantum_fen").Default(1),

		// Computed during the reserve step.
		field.Int64("reserved_credits").Default(0),
		field.Int64("refund_fen").Default(0),
		field.Int64("remainder_credits").Default(0),

		// Provider details.
		field.String("provider_name").Optional().Default(""),
		field.String("provider_refund_id").Optional().Default(""),

		// Fence details.
		field.String("fence_sequence").Optional().Default(""),

		// Snapshot version.
		field.String("snapshot_version").Optional().Default(""),

		// Failure detail.
		field.String("failure_reason").Optional().Nillable(),
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
