package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

// ExternalInvoiceRef links a ReceivablePeriod to an external invoicing system.
// The status may change (issued -> paid -> void) but historical audit entries
// (issued_at, invoice_number) are immutable once set.
type ExternalInvoiceRef struct {
	ent.Schema
}

func (ExternalInvoiceRef) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.IDMixin{},
		entutils.NamespaceMixin{},
		entutils.TimeMixin{},
	}
}

func (ExternalInvoiceRef) Fields() []ent.Field {
	return []ent.Field{
		field.String("receivable_period_id").Immutable().SchemaType(map[string]string{
			dialect.Postgres: "char(26)",
		}),
		field.String("invoice_number").NotEmpty().Immutable(),
		field.String("invoice_url").Optional().Nillable(),
		field.Enum("status").
			Values("draft", "issued", "void", "paid").
			Default("draft"),
		field.Time("issued_at").Optional().Nillable().Immutable(),
	}
}

func (ExternalInvoiceRef) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("period", ReceivablePeriod.Type).
			Ref("invoice_refs").
			Field("receivable_period_id").
			Required().
			Immutable().
			Unique(),
	}
}

func (ExternalInvoiceRef) Indexes() []ent.Index {
	return []ent.Index{
		// One invoice number per namespace.
		index.Fields("namespace", "invoice_number").Unique(),
		index.Fields("namespace", "receivable_period_id"),
	}
}
