package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

// CustomerAIRatePackage groups rate card entries into a package for a customer.
// It is a mutable catalog record (supports soft delete).
type CustomerAIRatePackage struct {
	ent.Schema
}

func (CustomerAIRatePackage) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.IDMixin{},
		entutils.NamespaceMixin{},
		entutils.AnnotationsMixin{},
		entutils.TimeMixin{},
	}
}

func (CustomerAIRatePackage) Fields() []ent.Field {
	return []ent.Field{
		field.String("customer_id").NotEmpty(),
		// PackageCode is the human-readable identifier (e.g. "enterprise-2025").
		field.String("package_code").NotEmpty(),
		field.String("name").NotEmpty(),
		field.String("description").Optional().Nillable(),
		// Status controls whether the package is active for settlement.
		field.Enum("status").
			Values("draft", "active", "archived").
			Default("draft"),
		field.Time("effective_from"),
		field.Time("effective_to").Optional().Nillable(),
	}
}

func (CustomerAIRatePackage) Edges() []ent.Edge {
	return nil
}

func (CustomerAIRatePackage) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "customer_id", "package_code").
			Annotations(entsql.IndexWhere("deleted_at IS NULL")).
			Unique(),
		index.Fields("namespace", "customer_id", "status").
			Annotations(entsql.IndexWhere("deleted_at IS NULL")),
	}
}
