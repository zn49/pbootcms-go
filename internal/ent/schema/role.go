package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
)

// Role is the existing ay_role table. Ent does not migrate it.
type Role struct{ ent.Schema }

func (Role) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "ay_role"}}
}

func (Role) Fields() []ent.Field {
	return []ent.Field{
		field.String("rcode"),
		field.String("name"),
		field.String("description"),
		field.String("create_user"),
		field.String("update_user"),
		field.String("create_time"),
		field.String("update_time"),
	}
}
