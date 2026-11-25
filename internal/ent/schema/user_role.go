package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
)

// UserRole is the existing ay_user_role table. Ent does not migrate it.
type UserRole struct{ ent.Schema }

func (UserRole) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "ay_user_role"}}
}

func (UserRole) Fields() []ent.Field {
	return []ent.Field{
		field.String("ucode"),
		field.String("rcode"),
	}
}
