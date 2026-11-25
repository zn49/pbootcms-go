package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
)

// FormField is the existing ay_form_field table. Ent does not migrate it.
type FormField struct{ ent.Schema }

func (FormField) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "ay_form_field"}}
}

func (FormField) Fields() []ent.Field {
	return []ent.Field{
		field.String("fcode"),
		field.String("name"),
		field.Int("length"),
		field.String("required"),
		field.String("description"),
		field.Int("sorting"),
		field.String("create_user"),
		field.String("update_user"),
		field.String("create_time"),
		field.String("update_time"),
	}
}
