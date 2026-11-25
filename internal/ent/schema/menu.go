package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
)

// Menu is the existing ay_menu table. Ent does not migrate it.
type Menu struct{ ent.Schema }

func (Menu) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "ay_menu"}}
}

func (Menu) Fields() []ent.Field {
	return []ent.Field{
		field.String("mcode"),
		field.String("pcode"),
		field.String("name"),
		field.String("url"),
		field.Int("sorting"),
		field.String("status"),
		field.String("shortcut"),
		field.String("ico"),
		field.String("create_user"),
		field.String("update_user"),
		field.String("create_time"),
		field.String("update_time"),
	}
}
