package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

// CustomerCreditLimit is a time-bounded enterprise receivable allowance. The
// ledger remains the source of truth for consumed balance; this entity stores
// only policy, never a mutable used amount.
type CustomerCreditLimit struct {
	ent.Schema
}

func (CustomerCreditLimit) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.IDMixin{},
		entutils.NamespaceMixin{},
		entutils.TimeMixin{},
	}
}

func (CustomerCreditLimit) Fields() []ent.Field {
	return []ent.Field{
		field.String("customer_id").NotEmpty().Immutable(),
		// currency stores CurrencyReference.MarshalText(), preserving a managed
		// custom currency's identity and precision snapshot.
		field.String("currency").NotEmpty().Immutable(),
		field.String("custom_currency_id").NotEmpty().Immutable(),
		field.Other("amount", alpacadecimal.Decimal{}).
			SchemaType(map[string]string{dialect.Postgres: "numeric"}),
		field.Time("effective_from").Immutable(),
		field.Time("effective_to").Optional().Nillable(),
		field.Bool("enabled").Default(true),
	}
}

func (CustomerCreditLimit) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "customer_id", "custom_currency_id", "effective_from").
			Annotations(entsql.IndexWhere("deleted_at IS NULL")).
			Unique(),
	}
}
