package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
)

// ExtField is the existing ay_extfield table. Ent does not migrate it.
type ExtField struct{ ent.Schema }

func (ExtField) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "ay_extfield"}}
}

func (ExtField) Fields() []ent.Field {
	return []ent.Field{
		field.String("mcode"),
		field.String("name"),
		field.String("type"),
		field.String("value"),
		field.String("description"),
		field.Int("sorting"),
	}
}
