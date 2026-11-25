package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
)

// MemberComment is the existing ay_member_comment table. Ent does not migrate it.
type MemberComment struct{ ent.Schema }

func (MemberComment) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "ay_member_comment"}}
}

func (MemberComment) Fields() []ent.Field {
	return []ent.Field{
		field.Int("pid"),
		field.Int("contentid"),
		field.String("comment"),
		field.Int("uid"),
		field.Int("puid"),
		field.Int("likes"),
		field.Int("oppose"),
		field.String("status"),
		field.String("user_ip"),
		field.String("user_os"),
		field.String("user_bs"),
		field.String("create_time"),
		field.String("update_user"),
		field.String("update_time"),
	}
}
