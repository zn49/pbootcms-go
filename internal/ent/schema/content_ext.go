package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
)

// ContentExt is the existing ay_content_ext table. Ent does not migrate it.
type ContentExt struct{ ent.Schema }

func (ContentExt) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "ay_content_ext"}}
}

func (ContentExt) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id").StorageKey("extid"),
		field.Int("contentid"),
		field.String("ext_price").Optional().Nillable(),
		field.String("ext_type").Optional().Nillable(),
		field.String("ext_color").Optional().Nillable(),
	}
}
