package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

// CreditReservationCommand durably binds each lifecycle command's
// idempotency key to its payload. It is distinct from the reserve command so
// Execute, Release, and Unknown can safely use their own identities.
type CreditReservationCommand struct{ ent.Schema }

func (CreditReservationCommand) Mixin() []ent.Mixin {
	return []ent.Mixin{entutils.IDMixin{}, entutils.NamespaceMixin{}}
}

func (CreditReservationCommand) Fields() []ent.Field {
	return []ent.Field{
		field.Time("created_at").Default(clock.Now).Immutable(),
		field.String("reservation_id").NotEmpty().Immutable(),
		field.String("command_kind").NotEmpty().Immutable(),
		field.String("idempotency_key").NotEmpty().Immutable(),
		field.String("payload_hash").NotEmpty().Immutable(),
	}
}

func (CreditReservationCommand) Indexes() []ent.Index {
	return []ent.Index{index.Fields("namespace", "reservation_id", "command_kind", "idempotency_key").Unique()}
}
