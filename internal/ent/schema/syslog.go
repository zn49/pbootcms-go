package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
)

// Syslog is the existing ay_syslog table. Ent does not migrate it.
type Syslog struct{ ent.Schema }

func (Syslog) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "ay_syslog"}}
}

func (Syslog) Fields() []ent.Field {
	return []ent.Field{
		field.String("level"),
		field.String("event"),
		field.String("user_ip"),
		field.String("user_os"),
		field.String("user_bs"),
		field.String("create_user"),
		field.String("create_time"),
	}
}
