package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"

	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

// RefundFact stores a verified provider refund callback fact. Only the SHA-256
// hash of the raw body and verified signed fields are persisted. Immutable.
type RefundFact struct {
	ent.Schema
}

func (RefundFact) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.IDMixin{},
		entutils.NamespaceMixin{},
	}
}

func (RefundFact) Fields() []ent.Field {
	return []ent.Field{
		field.Time("created_at").
			Default(clock.Now).
			Immutable(),
		field.String("raw_hash").NotEmpty().Immutable(),
		field.Enum("provider").
			Values("wechat", "alipay", "offline").
			Immutable(),
		field.JSON("signed_payload", map[string]any{}).
			Immutable().
			SchemaType(map[string]string{
				dialect.Postgres: "jsonb",
			}),
		field.Time("timestamp").Immutable(),
	}
}

func (RefundFact) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("refund_request", RefundRequest.Type).
			Ref("facts").
			Required().
			Immutable().
			Unique(),
	}
}
