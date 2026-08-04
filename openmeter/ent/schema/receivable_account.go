package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

// ReceivableAccount models an enterprise customer's credit account. The
// current_balance_cents may be negative (customer owes money) up to
// -credit_limit_cents. One account per (namespace, customer_id).
type ReceivableAccount struct {
	ent.Schema
}

func (ReceivableAccount) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.IDMixin{},
		entutils.NamespaceMixin{},
		entutils.AnnotationsMixin{},
		entutils.TimeMixin{},
	}
}

func (ReceivableAccount) Fields() []ent.Field {
	return []ent.Field{
		field.String("customer_id").NotEmpty().Immutable(),
		field.Int64("credit_limit_cents").Min(0).Default(0),
		field.Int64("current_balance_cents").Default(0),
		field.String("currency").Default("CNY").Immutable(),
	}
}

func (ReceivableAccount) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("periods", ReceivablePeriod.Type),
		edge.To("offline_payments", OfflinePayment.Type),
	}
}

func (ReceivableAccount) Indexes() []ent.Index {
	return []ent.Index{
		// One account per customer.
		index.Fields("namespace", "customer_id").Unique(),
	}
}
