package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

// PaymentFact stores a verified provider callback fact. Only the SHA-256 hash
// of the raw body (raw_hash) and verified signed fields (signed_payload) are
// persisted -- the raw body is never stored. Immutable append-only record.
type PaymentFact struct {
	ent.Schema
}

func (PaymentFact) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.IDMixin{},
		entutils.NamespaceMixin{},
	}
}

func (PaymentFact) Fields() []ent.Field {
	return []ent.Field{
		field.Time("created_at").
			Default(clock.Now).
			Immutable(),
		field.String("payment_attempt_id").Immutable().SchemaType(map[string]string{
			dialect.Postgres: "char(26)",
		}),
		// raw_hash is the SHA-256 of the provider callback raw body. Immutable.
		field.String("raw_hash").NotEmpty().Immutable(),
		field.Enum("provider").
			Values("wechat", "alipay", "offline").
			Immutable(),
		// signed_payload stores verified fields extracted from the callback, not the raw body.
		field.JSON("signed_payload", map[string]any{}).
			Immutable().
			SchemaType(map[string]string{
				dialect.Postgres: "jsonb",
			}),
		// timestamp is the event timestamp reported by the provider.
		field.Time("timestamp").Immutable(),
	}
}

func (PaymentFact) Edges() []ent.Edge {
	return []ent.Edge{
	edge.From("attempt", PaymentAttempt.Type).
		Ref("facts").
		Field("payment_attempt_id").
		Required().
		Immutable().
		Unique(),
	}
}

func (PaymentFact) Indexes() []ent.Index {
	return []ent.Index{
		// DB-enforced callback dedup: one fact per raw_hash within a namespace.
		index.Fields("namespace", "raw_hash").Unique(),
	}
}
