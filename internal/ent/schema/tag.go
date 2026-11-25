package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
)

// Tag is the existing ay_tags table. Ent does not migrate it.
type Tag struct{ ent.Schema }

func (Tag) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "ay_tags"}}
}

func (Tag) Fields() []ent.Field {
	return []ent.Field{
		field.String("acode"),
		field.String("name"),
		field.String("link"),
		field.String("create_user"),
		field.String("update_user"),
		field.String("create_time"),
		field.String("update_time"),
	}
}
