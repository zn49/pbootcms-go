//go:build ignore

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
)

var want = map[string]string{
	"ay_area": "Area", "ay_company": "Company", "ay_config": "Config",
	"ay_content": "Content", "ay_content_ext": "ContentExt", "ay_content_sort": "ContentSort",
	"ay_extfield": "ExtField", "ay_form": "Form", "ay_form_field": "FormField",
	"ay_label": "Label", "ay_link": "Link", "ay_member": "Member",
	"ay_member_comment": "MemberComment", "ay_menu": "Menu", "ay_message": "Message",
	"ay_model": "Model", "ay_role": "Role", "ay_role_level": "RoleLevel",
	"ay_site": "Site", "ay_slide": "Slide", "ay_syslog": "Syslog",
	"ay_tags": "Tag", "ay_user": "User", "ay_user_role": "UserRole",
}

var colRe = regexp.MustCompile("^\\s*`([a-zA-Z0-9_]+)`\\s+([a-zA-Z]+)")

func main() {
	b, err := os.ReadFile(filepath.Join("..", "..", "site", "static", "backup", "sql", "pbootcms.sql"))
	if err != nil {
		panic(err)
	}
	blocks := strings.Split(string(b), "CREATE TABLE `")
	dir := "schema"
	os.MkdirAll(dir, 0o755)
	for _, block := range blocks[1:] {
		name, rest, _ := strings.Cut(block, "`")
		typ, ok := want[name]
		if !ok {
			continue
		}
		body, _, _ := strings.Cut(rest, "\n)")
		var fields []string
		for _, line := range strings.Split(body, "\n") {
			m := colRe.FindStringSubmatch(line)
			if m == nil || m[1] == "id" {
				continue
			}
			kind := "String"
			if strings.Contains(strings.ToLower(m[2]), "int") {
				kind = "Int"
			}
			f := fmt.Sprintf("field.%s(%q)", kind, m[1])
			if !strings.Contains(line, "NOT NULL") {
				f += ".Optional().Nillable()"
			}
			fields = append(fields, "\t\t"+f+",")
		}
		src := fmt.Sprintf(`package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
)

// %s is the existing %s table. Ent does not migrate it.
type %s struct{ ent.Schema }

func (%s) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: %q}}
}

func (%s) Fields() []ent.Field {
	return []ent.Field{
%s
	}
}
`, typ, name, typ, typ, name, typ, strings.Join(fields, "\n"))
		if err := os.WriteFile(filepath.Join(dir, snake(typ)+".go"), []byte(src), 0o644); err != nil {
			panic(err)
		}
	}
}

func snake(s string) string {
	var b strings.Builder
	for i, r := range s {
		if unicode.IsUpper(r) && i > 0 {
			b.WriteByte('_')
		}
		b.WriteRune(unicode.ToLower(r))
	}
	return b.String()
}
