package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
)

// Content is the existing ay_content table. Ent does not migrate it.
type Content struct{ ent.Schema }

func (Content) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "ay_content"}}
}

func (Content) Fields() []ent.Field {
	return []ent.Field{
		field.String("acode"),
		field.String("scode"),
		field.String("subscode"),
		field.String("title"),
		field.String("titlecolor"),
		field.String("subtitle"),
		field.String("filename"),
		field.String("author"),
		field.String("source"),
		field.String("outlink"),
		field.String("date"),
		field.String("ico"),
		field.String("pics"),
		field.String("picstitle"),
		field.String("content"),
		field.String("tags"),
		field.String("enclosure"),
		field.String("keywords"),
		field.String("description"),
		field.Int("sorting"),
		field.String("status"),
		field.String("istop"),
		field.String("isrecommend"),
		field.String("isheadline"),
		field.Int("visits"),
		field.Int("likes"),
		field.Int("oppose"),
		field.String("create_user"),
		field.String("update_user"),
		field.String("create_time"),
		field.String("update_time"),
		field.String("gtype"),
		field.String("gid"),
		field.String("gnote"),
	}
}
