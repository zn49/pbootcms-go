package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
)

// Message is the existing ay_message table. Ent does not migrate it.
type Message struct{ ent.Schema }

func (Message) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "ay_message"}}
}

func (Message) Fields() []ent.Field {
	return []ent.Field{
		field.String("acode"),
		field.String("contacts").Optional().Nillable(),
		field.String("mobile").Optional().Nillable(),
		field.String("content").Optional().Nillable(),
		field.String("user_ip"),
		field.String("user_os"),
		field.String("user_bs"),
		field.String("recontent"),
		field.String("status"),
		field.String("create_user"),
		field.String("update_user"),
		field.String("create_time"),
		field.String("update_time"),
		field.Int("uid"),
	}
}
