package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
)

// RoleLevel is the existing ay_role_level table. Ent does not migrate it.
type RoleLevel struct{ ent.Schema }

func (RoleLevel) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "ay_role_level"}}
}

func (RoleLevel) Fields() []ent.Field {
	return []ent.Field{
		field.String("rcode"),
		field.String("level"),
	}
}
