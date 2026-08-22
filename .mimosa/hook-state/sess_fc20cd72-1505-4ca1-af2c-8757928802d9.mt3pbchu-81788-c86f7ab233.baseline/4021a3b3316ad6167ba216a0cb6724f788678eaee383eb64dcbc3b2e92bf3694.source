package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

// ReceivablePeriod is a billing period within a ReceivableAccount. It
// aggregates charges and tracks payment status. period_start and period_end
// are immutable once set.
type ReceivablePeriod struct {
	ent.Schema
}

func (ReceivablePeriod) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.IDMixin{},
		entutils.NamespaceMixin{},
		entutils.TimeMixin{},
	}
}

func (ReceivablePeriod) Fields() []ent.Field {
	return []ent.Field{
		field.String("receivable_account_id").Immutable().SchemaType(map[string]string{
			dialect.Postgres: "char(26)",
		}),
		field.Enum("status").
			Values("open", "closed", "partially_paid", "paid", "overdue").
			Default("open"),
		field.Time("period_start").Immutable(),
		field.Time("period_end").Immutable(),
		field.Int64("total_cents").Min(0).Default(0),
		field.Int64("paid_cents").Min(0).Default(0),
		field.String("currency").Default("CNY").Immutable(),
	}
}

func (ReceivablePeriod) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("account", ReceivableAccount.Type).
			Ref("periods").
			Field("receivable_account_id").
			Required().
			Immutable().
			Unique(),
		edge.To("invoice_refs", ExternalInvoiceRef.Type),
	}
}

func (ReceivablePeriod) Indexes() []ent.Index {
	return []ent.Index{
		// One period per account per start date.
		index.Fields("namespace", "receivable_account_id", "period_start").Unique(),
		// Status lookup.
		index.Fields("namespace", "status"),
	}
}
