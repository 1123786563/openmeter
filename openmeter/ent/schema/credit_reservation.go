package schema

import (
	"encoding/json"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

// CreditReservation stores the lifecycle and immutable pricing snapshots of a
// synchronous authorization. It intentionally contains no derived balance or
// ledger entries; ledger group IDs are references to effects owned by Ledger.
type CreditReservation struct {
	ent.Schema
}

func (CreditReservation) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.IDMixin{},
		entutils.NamespaceMixin{},
		entutils.TimeMixin{},
	}
}

func (CreditReservation) Fields() []ent.Field {
	return []ent.Field{
		field.String("customer_id").NotEmpty().Immutable(),
		field.String("subject_id").NotEmpty().Immutable(),
		field.String("client_call_id").NotEmpty().Immutable(),
		field.String("operation").NotEmpty().Immutable(),
		field.String("idempotency_key").NotEmpty().Immutable(),
		field.String("payload_hash").NotEmpty().Immutable(),
		field.JSON("currency", currencies.CurrencyReference{}).
			SchemaType(map[string]string{dialect.Postgres: "jsonb"}).
			Immutable(),
		field.String("custom_currency_id").Optional().Nillable().Immutable(),
		field.JSON("estimated_lines", []json.RawMessage{}).
			SchemaType(map[string]string{dialect.Postgres: "jsonb"}),
		field.JSON("rated_lines", []json.RawMessage{}).
			SchemaType(map[string]string{dialect.Postgres: "jsonb"}),
		field.JSON("actual_lines", []json.RawMessage{}).
			Optional().
			SchemaType(map[string]string{dialect.Postgres: "jsonb"}),
		field.Int64("ceiling_credits").Default(0),
		field.Int64("prepaid_hold").Default(0),
		field.Int64("enterprise_hold").Default(0),
		field.Int64("settled_credits").Default(0),
		field.String("rate_version").Default(""),
		field.String("state").NotEmpty(),
		field.String("provider").Default(""),
		field.String("model").Default(""),
		field.String("request_id").Default(""),
		field.Time("authorization_expires_at").Optional().Nillable(),
		field.Time("execution_deadline").Optional().Nillable(),
		field.String("hold_ledger_group_id").Default(""),
		field.String("settlement_ledger_group_id").Default(""),
		field.String("release_ledger_group_id").Default(""),
		field.String("usage_event_id").Default(""),
	}
}

func (CreditReservation) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "idempotency_key").StorageKey("credit_reservation_idempotency").Unique(),
		index.Fields("namespace", "client_call_id").StorageKey("credit_reservation_call").Unique(),
		index.Fields("namespace", "customer_id", "custom_currency_id", "state").
			StorageKey("credit_reservation_active_holds").
			Annotations(entsql.IndexWhere("state IN ('ACTIVE', 'EXECUTING', 'UNKNOWN', 'MANUAL_REVIEW')")),
	}
}
