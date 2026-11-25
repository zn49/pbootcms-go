package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
)

// SysConfig is the existing ay_config table. Ent does not migrate it.
type SysConfig struct{ ent.Schema }

func (SysConfig) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "ay_config"}}
}

func (SysConfig) Fields() []ent.Field {
	return []ent.Field{
		field.String("name"),
		field.String("value"),
		field.String("type"),
		field.Int("sorting"),
		field.String("description"),
	}
}
