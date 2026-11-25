package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
)

// Area is the existing ay_area table. Ent does not migrate it.
type Area struct{ ent.Schema }

func (Area) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "ay_area"}}
}

func (Area) Fields() []ent.Field {
	return []ent.Field{
		field.String("acode"),
		field.String("pcode"),
		field.String("name"),
		field.String("domain"),
		field.String("is_default"),
		field.String("create_user"),
		field.String("update_user"),
		field.String("create_time"),
		field.String("update_time"),
	}
}
