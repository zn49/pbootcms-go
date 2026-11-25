package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
)

// Label is the existing ay_label table. Ent does not migrate it.
type Label struct{ ent.Schema }

func (Label) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "ay_label"}}
}

func (Label) Fields() []ent.Field {
	return []ent.Field{
		field.String("name"),
		field.String("value"),
		field.String("type"),
		field.String("description"),
		field.String("create_user"),
		field.String("update_user"),
		field.String("create_time"),
		field.String("update_time"),
	}
}
