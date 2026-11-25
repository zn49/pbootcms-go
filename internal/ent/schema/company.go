package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
)

// Company is the existing ay_company table. Ent does not migrate it.
type Company struct{ ent.Schema }

func (Company) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "ay_company"}}
}

func (Company) Fields() []ent.Field {
	return []ent.Field{
		field.String("acode"),
		field.String("name"),
		field.String("address"),
		field.String("postcode"),
		field.String("contact"),
		field.String("mobile"),
		field.String("phone"),
		field.String("fax"),
		field.String("email"),
		field.String("qq"),
		field.String("weixin"),
		field.String("blicense"),
		field.String("other"),
	}
}
