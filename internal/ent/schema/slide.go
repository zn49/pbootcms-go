package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
)

// Slide is the existing ay_slide table. Ent does not migrate it.
type Slide struct{ ent.Schema }

func (Slide) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "ay_slide"}}
}

func (Slide) Fields() []ent.Field {
	return []ent.Field{
		field.String("acode"),
		field.Int("gid"),
		field.String("pic"),
		field.String("link"),
		field.String("title"),
		field.String("subtitle"),
		field.Int("sorting"),
		field.String("create_user"),
		field.String("update_user"),
		field.String("create_time"),
		field.String("update_time"),
	}
}
