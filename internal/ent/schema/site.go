package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
)

// Site is the existing ay_site table. Ent does not migrate it.
type Site struct{ ent.Schema }

func (Site) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "ay_site"}}
}

func (Site) Fields() []ent.Field {
	return []ent.Field{
		field.String("acode"),
		field.String("title"),
		field.String("subtitle"),
		field.String("domain"),
		field.String("logo"),
		field.String("keywords"),
		field.String("description"),
		field.String("icp"),
		field.String("theme"),
		field.String("statistical"),
		field.String("copyright"),
	}
}
