package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
)

// Link is the existing ay_link table. Ent does not migrate it.
type Link struct{ ent.Schema }

func (Link) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "ay_link"}}
}

func (Link) Fields() []ent.Field {
	return []ent.Field{
		field.String("acode"),
		field.Int("gid"),
		field.String("name"),
		field.String("link"),
		field.String("logo"),
		field.Int("sorting"),
		field.String("create_user"),
		field.String("update_user"),
		field.String("create_time"),
		field.String("update_time"),
	}
}
