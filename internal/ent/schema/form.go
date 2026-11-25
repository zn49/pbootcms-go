package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
)

// Form is the existing ay_form table. Ent does not migrate it.
type Form struct{ ent.Schema }

func (Form) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "ay_form"}}
}

func (Form) Fields() []ent.Field {
	return []ent.Field{
		field.String("fcode"),
		field.String("form_name"),
		field.String("table_name"),
		field.String("create_user"),
		field.String("update_user"),
		field.String("create_time"),
		field.String("update_time"),
	}
}
