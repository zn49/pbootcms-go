package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
)

// Model is the existing ay_model table. Ent does not migrate it.
type Model struct{ ent.Schema }

func (Model) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "ay_model"}}
}

func (Model) Fields() []ent.Field {
	return []ent.Field{
		field.String("mcode"),
		field.String("name"),
		field.String("type"),
		field.String("urlname"),
		field.String("listtpl"),
		field.String("contenttpl"),
		field.String("status"),
		field.String("issystem"),
		field.String("create_user"),
		field.String("update_user"),
		field.String("create_time"),
		field.String("update_time"),
	}
}
