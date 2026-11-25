package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
)

// User is the existing ay_user table. Ent does not migrate it.
type User struct{ ent.Schema }

func (User) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "ay_user"}}
}

func (User) Fields() []ent.Field {
	return []ent.Field{
		field.String("ucode"),
		field.String("username"),
		field.String("realname"),
		field.String("password"),
		field.String("status"),
		field.Int("login_count"),
		field.String("last_login_ip"),
		field.String("create_user"),
		field.String("update_user"),
		field.String("create_time"),
		field.String("update_time"),
	}
}
