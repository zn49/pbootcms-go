package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
)

// ContentSort is the existing ay_content_sort table. Ent does not migrate it.
type ContentSort struct{ ent.Schema }

func (ContentSort) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "ay_content_sort"}}
}

func (ContentSort) Fields() []ent.Field {
	return []ent.Field{
		field.String("acode"),
		field.String("mcode"),
		field.String("pcode"),
		field.String("scode"),
		field.String("name"),
		field.String("listtpl"),
		field.String("contenttpl"),
		field.String("status"),
		field.String("outlink"),
		field.String("subname"),
		field.String("def1"),
		field.String("def2"),
		field.String("def3"),
		field.String("ico"),
		field.String("pic"),
		field.String("title"),
		field.String("keywords"),
		field.String("description"),
		field.String("filename"),
		field.Int("sorting"),
		field.String("create_user"),
		field.String("update_user"),
		field.String("create_time"),
		field.String("update_time"),
		field.String("gtype"),
		field.String("gid"),
		field.String("gnote"),
	}
}
