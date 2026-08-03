package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

// AIUsageWatermark stores the continuous watermark per subject — the highest
// tenant_seq with no gaps below. There is exactly one row per (namespace, subject_id).
// The row is locked with SELECT FOR UPDATE during batch persistence.
type AIUsageWatermark struct {
	ent.Schema
}

func (AIUsageWatermark) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.IDMixin{},
		entutils.NamespaceMixin{},
	}
}

func (AIUsageWatermark) Fields() []ent.Field {
	return []ent.Field{
		field.Time("created_at").
			Default(clock.Now).
			Immutable(),
		field.Time("updated_at").
			Default(clock.Now).
			UpdateDefault(clock.Now),
		field.String("subject_id").NotEmpty().Immutable(),
		field.String("customer_id").NotEmpty(),
		field.Int64("covered_seq").Default(0),
	}
}

func (AIUsageWatermark) Edges() []ent.Edge {
	return nil
}

func (AIUsageWatermark) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "subject_id").Unique(),
		index.Fields("namespace", "customer_id"),
	}
}
