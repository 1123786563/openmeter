package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

// AIUsageBatch stores the canonical AI usage batch — one business action.
type AIUsageBatch struct {
	ent.Schema
}

func (AIUsageBatch) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.IDMixin{},
		entutils.NamespaceMixin{},
		entutils.AnnotationsMixin{},
		entutils.TimeMixin{},
	}
}

func (AIUsageBatch) Fields() []ent.Field {
	return []ent.Field{
		field.String("customer_id").NotEmpty(),
		field.String("subject_id").NotEmpty(),
		field.String("usage_batch_id").NotEmpty(),
		field.Int64("tenant_seq"),
		field.Time("occurred_at"),
		field.String("reservation_id").Optional().Nillable(),
		field.Int64("ceiling_credits").Optional().Nillable(),
		field.String("rate_version").Default(""),
		field.String("billing_mode").NotEmpty(),
		field.String("payload_hash").NotEmpty(),
		field.String("status").Default("pending"),
		field.Int64("total_credits").Default(0),
		field.Int64("covered_tenant_seq").Default(0),
		// SettlementScope controls shadow (visibility only) vs formal (deduct grants) persistence.
		field.Enum("settlement_scope").
			Values("shadow", "formal").
			Default("formal").
			Immutable(),
	}
}

func (AIUsageBatch) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("line_items", AIUsageLineItem.Type),
		edge.To("rating_snapshots", AIUsageRatingSnapshot.Type),
		edge.To("allocations", AIUsageAllocation.Type),
		edge.To("outbox_events", AIUsageOutbox.Type),
	}
}

func (AIUsageBatch) Indexes() []ent.Index {
	return []ent.Index{
		// Idempotency key: one batch per usage_batch_id per namespace.
		index.Fields("namespace", "usage_batch_id").
			Annotations(entsql.IndexWhere("deleted_at IS NULL")).
			Unique(),
		// Subject monotonic seq: one batch per subject per tenant_seq.
		index.Fields("namespace", "subject_id", "tenant_seq").
			Annotations(entsql.IndexWhere("deleted_at IS NULL")).
			Unique(),
		// Watermark lookup: find highest settled tenant_seq.
		index.Fields("namespace", "customer_id", "tenant_seq").
			Annotations(entsql.IndexWhere("deleted_at IS NULL")),
		// Settlement queries.
		index.Fields("namespace", "customer_id", "status").
			Annotations(entsql.IndexWhere("deleted_at IS NULL")),
	}
}
