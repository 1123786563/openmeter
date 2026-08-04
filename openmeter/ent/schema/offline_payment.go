package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

// OfflinePayment records a manually-confirmed payment against a
// ReceivableAccount (e.g. bank transfer). Immutable once confirmed.
type OfflinePayment struct {
	ent.Schema
}

func (OfflinePayment) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.IDMixin{},
		entutils.NamespaceMixin{},
		entutils.TimeMixin{},
	}
}

func (OfflinePayment) Fields() []ent.Field {
	return []ent.Field{
		field.String("receivable_account_id").Immutable().SchemaType(map[string]string{
			dialect.Postgres: "char(26)",
		}),
		field.Int64("amount_cents").Min(0).Immutable(),
		field.String("currency").Default("CNY").Immutable(),
		field.String("confirmed_by").NotEmpty().Immutable(),
		field.Time("confirmed_at").Immutable(),
		field.String("reference").Optional().Nillable(),
		field.String("note").Optional().Nillable(),
	}
}

func (OfflinePayment) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("account", ReceivableAccount.Type).
			Ref("offline_payments").
			Field("receivable_account_id").
			Required().
			Immutable().
			Unique(),
	}
}

func (OfflinePayment) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "receivable_account_id"),
		index.Fields("namespace", "confirmed_at"),
	}
}
