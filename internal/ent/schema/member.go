package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
)

// Member is the existing ay_member table. Ent does not migrate it.
type Member struct{ ent.Schema }

func (Member) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "ay_member"}}
}

func (Member) Fields() []ent.Field {
	return []ent.Field{
		field.String("ucode"),
		field.String("username"),
		field.String("useremail"),
		field.String("usermobile"),
		field.String("nickname"),
		field.String("password"),
		field.String("headpic"),
		field.String("status"),
		field.String("activation"),
		field.String("gid"),
		field.String("wxid"),
		field.String("qqid"),
		field.String("wbid"),
		field.Int("score"),
		field.String("register_time"),
		field.Int("login_count"),
		field.String("last_login_ip"),
		field.String("last_login_time"),
		field.String("sex"),
		field.String("birthday"),
		field.String("qq"),
	}
}
