package schema

import (
	"encoding/json"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

// CreditCharge stores a direct settlement or reversal command. It records the
// command's rated snapshot and references to Ledger, but never a balance.
type CreditCharge struct {
	ent.Schema
}

func (CreditCharge) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.IDMixin{},
		entutils.NamespaceMixin{},
		entutils.TimeMixin{},
	}
}

func (CreditCharge) Fields() []ent.Field {
	return []ent.Field{
		field.String("reservation_id").Optional().Nillable().Immutable(),
		field.String("customer_id").NotEmpty().Immutable(),
		field.String("subject_id").NotEmpty().Immutable(),
		field.String("operation").NotEmpty().Immutable(),
		field.String("idempotency_key").NotEmpty().Immutable(),
		field.String("payload_hash").NotEmpty().Immutable(),
		field.JSON("currency", currencies.CurrencyReference{}).
			SchemaType(map[string]string{dialect.Postgres: "jsonb"}).
			Immutable(),
		field.String("custom_currency_id").Optional().Nillable().Immutable(),
		field.JSON("rated_lines", []json.RawMessage{}).
			SchemaType(map[string]string{dialect.Postgres: "jsonb"}),
		field.Int64("amount"),
		field.String("rate_version").Default(""),
		field.JSON("settlement_allocations", []json.RawMessage{}).
			SchemaType(map[string]string{dialect.Postgres: "jsonb"}),
		field.Enum("state").Values("SETTLED", "REVERSED"),
		field.String("settlement_ledger_group_id").Default(""),
		field.String("reversal_ledger_group_id").Default(""),
		field.String("reversal_idempotency_key").Default(""),
		field.String("reversal_payload_hash").Default(""),
		field.String("usage_event_id").Default(""),
	}
}

func (CreditCharge) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "idempotency_key").Unique(),
		index.Fields("namespace", "reservation_id"),
	}
}
