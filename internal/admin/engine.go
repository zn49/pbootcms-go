package admin

import (
	"embed"
	"fmt"
	"html"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"gbootcms/internal/textmod"
)

//go:embed view/*.html
var viewFS embed.FS

var (
	incAdm     = regexp.MustCompile(`\{include\s+file\s*=\s*['"]?([^'"}\s]+)['"]?\s*\}`)
	varRe      = regexp.MustCompile(`\{\$([a-zA-Z0-9_.>-]+)\}`)
	urlRe      = regexp.MustCompile(`\{url\./([^}]+)\}`)
	identRe    = regexp.MustCompile(`[A-Za-z_][A-Za-z0-9_]*`)
	bracketRe  = regexp.MustCompile(`\[\$([a-zA-Z0-9_.>-]+)\]`)
	bodyOpenRe = regexp.MustCompile(`(?i)<div\s+class="layui-body"\s*>`)
	phpRe      = regexp.MustCompile(`(?s)\{php\}.*?\{/php\}`)
	valueOnly  = regexp.MustCompile(`\[value\]`)
	sessCallRe = regexp.MustCompile(`session\(['"](\w+)['"]\)`)
	notLitRe   = regexp.MustCompile(`!(?:'([^']*)'|(\d+))`)
)

// Data is the view bag.
type Data struct {
	Vars    map[string]any
	Session map[string]string
	Get     map[string]string
	Funs    map[string]func(args []string) string
}

func Render(name string, d Data) (string, error) {
	b, err := fs.ReadFile(viewFS, "view/"+name)
	if err != nil {
		return "", err
	}
	s := string(b)
	for i := 0; i < 20; i++ {
		n := 0
		s = incAdm.ReplaceAllStringFunc(s, func(m string) string {
			n++
			fn := incAdm.FindStringSubmatch(m)[1]
			bb, err := fs.ReadFile(viewFS, "view/"+fn)
			if err != nil {
				return ""
			}
			return string(bb)
		})
		if n == 0 {
			break
		}
	}
	return renderBody(applyConsts(s), d), nil
}

// RenderPHP loads SiteRoot/apps/admin/view/default/<rel>, swaps common head/foot
// for the Go chrome, and renders PHP-style tokens used by the original admin views.
func RenderPHP(root, rel string, d Data) (string, error) {
	path := filepath.Join(root, "apps", "admin", "view", "default", filepath.FromSlash(rel))
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	s := string(b)
	relSlash := filepath.ToSlash(rel)
	if strings.HasSuffix(relSlash, "system/database.html") {
		s = patchDatabaseTpl(s)
	}
	if strings.HasSuffix(relSlash, "content/content.html") || strings.HasSuffix(relSlash, "content/single.html") {
		s = patchContentLinkTpl(s)
	}
	s = stripPHPChrome(s)
	s = resolvePHPIncludes(root, s)
	head, err := fs.ReadFile(viewFS, "view/head.html")
	if err != nil {
		return "", err
	}
	foot, err := fs.ReadFile(viewFS, "view/foot.html")
	if err != nil {
		return "", err
	}
	return renderBody(applyConsts(string(head)+s+string(foot)), d), nil
}

func resolvePHPIncludes(root, s string) string {
	for i := 0; i < 10; i++ {
		n := 0
		s = incAdm.ReplaceAllStringFunc(s, func(m string) string {
			n++
			fn := strings.ReplaceAll(incAdm.FindStringSubmatch(m)[1], `\`, `/`)
			if strings.Contains(fn, "head.html") || strings.Contains(fn, "foot.html") {
				return ""
			}
			p := filepath.Join(root, "apps", "admin", "view", "default", filepath.FromSlash(fn))
			b, err := os.ReadFile(p)
			if err != nil {
				return ""
			}
			return string(b)
		})
		if n == 0 {
			break
		}
	}
	return s
}

func stripPHPChrome(s string) string {
	s = incAdm.ReplaceAllStringFunc(s, func(m string) string {
		fn := strings.ReplaceAll(incAdm.FindStringSubmatch(m)[1], `\`, `/`)
		if strings.Contains(fn, "head.html") || strings.Contains(fn, "foot.html") {
			return ""
		}
		return m
	})
	if loc := bodyOpenRe.FindStringIndex(s); loc != nil {
		s = s[:loc[0]] + s[loc[1]:]
	}
	return s
}

var (
	dbSizePHP      = regexp.MustCompile(`(?s)\{php\}\s*\$size\s*=\s*round.*?\{/php\}\s*<td>\{php\}echo\s*\$size\{/php\}\s*KB</td>`)
	dbFreePHP      = regexp.MustCompile(`(?s)<td>\s*\{if\(\$value->Data_free\)\}.*?\{else\}\s*-\s*\{/if\}\s*</td>`)
	contentLinkPHP = regexp.MustCompile(`(?s)\{php\}\s*\$Parser\s*=\s*new\s+app\\home\\controller\\ParserController\(\);.*?\{/php\}`)
	echoLinkPHP    = regexp.MustCompile(`\{php\}echo\s*\$link\{/php\}`)
)

func patchDatabaseTpl(s string) string {
	s = strings.ReplaceAll(s, "{php}$sum1 = 0; $sum2 = 0;{/php}", "")
	s = dbSizePHP.ReplaceAllString(s, `<td>[value->Size] KB</td>`)
	s = dbFreePHP.ReplaceAllString(s, `<td>[value->Free]</td>`)
	s = strings.ReplaceAll(s, "{php}echo $sum1{/php} KB", "{$db_sum_size} KB")
	s = strings.ReplaceAll(s, "{php}echo $sum2?:0{/php} KB", "{$db_sum_free} KB")
	s = strings.ReplaceAll(s, "<td>[num] 张表</td>", "<td>{$db_table_count} 张表</td>")
	return s
}

func patchContentLinkTpl(s string) string {
	s = contentLinkPHP.ReplaceAllString(s, "")
	s = echoLinkPHP.ReplaceAllString(s, "[value->link]")
	return s
}

var siteDirConst string

// SetSiteDirConst sets {SITE_DIR} for admin PHP views (subdir deploy).
func SetSiteDirConst(dir string) { siteDirConst = dir }

func applyConsts(s string) string {
	s = phpRe.ReplaceAllString(s, "#16b777")
	theme := "/apps/admin/view/default"
	repl := map[string]string{
		"{APP_THEME_DIR}": theme,
		"{CMSNAME}":       "GbootCMS",
		"{APP_VERSION}":   "3.2.33",
		"{RELEASE_TIME}":  "20251120",
		"{SITE_DIR}":      siteDirConst,
		"{CORE_DIR}":      "/core",
		"{LICENSE}":       "3",
		"{C}":             "",
		"{URL}":           "",
	}
	for k, v := range repl {
		s = strings.ReplaceAll(s, k, v)
	}
	return s
}

var (
	getCallRe    = regexp.MustCompile(`get\(['"](\w+)['"]\)`)
	contentDynRe = regexp.MustCompile(`\{\$content->\{([a-zA-Z0-9_]+)\}\}`)
)

func renderBody(s string, d Data) string {
	s = expandAllForeach(s, d)
	s = expandGetCalls(s, d)
	s = expandBrackets(s, d)
	s = expandURL(s)
	s = expandFuns(s, d)
	s = renderIf(s, d, nil)
	s = replaceVars(s, d)
	s = replaceVars(s, d)
	return s
}

func expandGetCalls(s string, d Data) string {
	return getCallRe.ReplaceAllStringFunc(s, func(m string) string {
		return d.Get[getCallRe.FindStringSubmatch(m)[1]]
	})
}

func expandFuns(s string, d Data) string {
	for i := 0; i < 80; i++ {
		start := strings.Index(s, "{fun=")
		if start < 0 {
			break
		}
		nameStart := start + len("{fun=")
		paren := strings.IndexByte(s[nameStart:], '(')
		if paren < 0 {
			break
		}
		name := s[nameStart : nameStart+paren]
		argStart := nameStart + paren + 1
		depth, end := 1, argStart
		for end < len(s) && depth > 0 {
			switch s[end] {
			case '(':
				depth++
			case ')':
				depth--
			}
			end++
		}
		if depth != 0 || end >= len(s) || s[end] != '}' {
			// skip broken tag
			s = s[:start] + s[start+5:]
			continue
		}
		argsRaw := s[argStart : end-1]
		close := end + 1 // past }
		argsRaw = expandURLCalls(argsRaw, d)
		var out string
		switch name {
		case "ueditor_holder_html":
			out = ueditorHolder(argsRaw, d)
		case "str_replace":
			out = funStrReplace(argsRaw)
		case "html_entity_decode":
			out = html.UnescapeString(strings.Trim(strings.TrimSpace(argsRaw), "'\""))
		default:
			fn := d.Funs[name]
			if fn != nil {
				out = fn(splitArgs(argsRaw))
			}
		}
		s = s[:start] + out + s[close:]
	}
	return s
}

func expandURLCalls(s string, d Data) string {
	for {
		i := strings.Index(s, "url(")
		if i < 0 {
			return s
		}
		start := i + 4
		depth, end := 1, start
		for end < len(s) && depth > 0 {
			switch s[end] {
			case '(':
				depth++
			case ')':
				depth--
			}
			end++
		}
		inner := s[start : end-1]
		var b strings.Builder
		for _, part := range strings.Split(inner, ".") {
			part = strings.TrimSpace(part)
			part = strings.Trim(part, "'\"")
			if part == "C" {
				if c, ok := d.Vars["C"].(string); ok {
					part = c
				}
			}
			b.WriteString(part)
		}
		s = s[:i] + b.String() + s[end:]
	}
}

func ueditorHolder(arg string, d Data) string {
	arg = strings.TrimSpace(arg)
	arg = strings.Trim(arg, "'\"")
	if strings.HasPrefix(arg, "$content->") {
		if cm, ok := d.Vars["content"].(map[string]string); ok {
			return cm[strings.TrimPrefix(arg, "$content->")]
		}
	}
	if cm, ok := d.Vars["content"].(map[string]string); ok {
		if v, ok := cm[arg]; ok {
			return v
		}
	}
	return arg
}

func funStrReplace(argsRaw string) string {
	// str_replace("a","b",subject) — subject may contain commas
	parts := splitArgsLoose(argsRaw)
	if len(parts) < 3 {
		return ""
	}
	return strings.ReplaceAll(parts[2], parts[0], parts[1])
}

func splitArgsLoose(s string) []string {
	var parts []string
	var cur strings.Builder
	inQ := byte(0)
	for i := 0; i < len(s); i++ {
		c := s[i]
		if inQ != 0 {
			if c == inQ {
				inQ = 0
			} else {
				cur.WriteByte(c)
			}
			continue
		}
		if c == '\'' || c == '"' {
			inQ = c
			continue
		}
		if c == ',' {
			parts = append(parts, strings.TrimSpace(cur.String()))
			cur.Reset()
			continue
		}
		cur.WriteByte(c)
	}
	parts = append(parts, strings.TrimSpace(cur.String()))
	return parts
}

func expandAllForeach(s string, d Data) string {
	return expandAllForeachParent(s, d, nil)
}

func expandAllForeachParent(s string, d Data, parent map[string]string) string {
	for i := 0; i < 40; i++ {
		loc := matchForeach(s)
		if loc == nil {
			break
		}
		key, alias, inner := loc.key, loc.alias, loc.body
		var b strings.Builder
		if rows, ok := d.Vars[key].([]map[string]string); ok {
			for i, row := range rows {
				r := mapsClone(row)
				r["num"] = strconv.Itoa(i + 1)
				chunk := expandRow(inner, alias, r)
				chunk = strings.ReplaceAll(chunk, "[num]", r["num"])
				if n := r["name"]; n != "" {
					if cm, ok := d.Vars["content"].(map[string]string); ok {
						chunk = strings.ReplaceAll(chunk, "{$content->{"+n+"}}", html.EscapeString(cm[n]))
						chunk = strings.ReplaceAll(chunk, "[$content->"+n+"]", html.EscapeString(cm[n]))
						chunk = strings.ReplaceAll(chunk, "[$content->$name]", html.EscapeString(cm[n]))
					}
					if parent != nil {
						chunk = strings.ReplaceAll(chunk, "[value->$field]", html.EscapeString(parent[n]))
						chunk = strings.ReplaceAll(chunk, "[value->"+n+"]", html.EscapeString(parent[n]))
					}
					chunk = strings.ReplaceAll(chunk, "[$content]->$name", n)
					chunk = strings.ReplaceAll(chunk, "$name", n)
				}
				chunk = expandAllForeachParent(chunk, d, r)
				chunk = renderIf(chunk, d, r)
				b.WriteString(chunk)
			}
		} else if strs, ok := d.Vars[key].([]string); ok {
			for i, v := range strs {
				chunk := valueOnly.ReplaceAllString(inner, html.EscapeString(v))
				chunk = strings.ReplaceAll(chunk, "["+alias+"]", html.EscapeString(v))
				chunk = strings.ReplaceAll(chunk, "[num]", strconv.Itoa(i+1))
				chunk = strings.ReplaceAll(chunk, "$"+alias, lit(v))
				chunk = expandAllForeachParent(chunk, d, parent)
				chunk = renderIf(chunk, d, map[string]string{"value": v})
				b.WriteString(chunk)
			}
		}
		s = s[:loc.start] + b.String() + s[loc.end:]
	}
	return s
}

type forMatch struct {
	start, end       int
	key, alias, body string
}

var forOpenRe = regexp.MustCompile(`\{foreach\s+\$([a-zA-Z0-9_]+)\((\w+),(\w+)(?:,\w+)?\)\}`)

func matchForeach(s string) *forMatch {
	loc := forOpenRe.FindStringSubmatchIndex(s)
	if loc == nil {
		return nil
	}
	depth := 1
	i := loc[1]
	for i < len(s) {
		if strings.HasPrefix(s[i:], "{foreach") {
			depth++
			i++
			continue
		}
		if strings.HasPrefix(s[i:], "{/foreach}") {
			depth--
			if depth == 0 {
				return &forMatch{
					start: loc[0], end: i + len("{/foreach}"),
					key: s[loc[2]:loc[3]], alias: s[loc[6]:loc[7]],
					body: s[loc[1]:i],
				}
			}
			i += len("{/foreach}")
			continue
		}
		i++
	}
	return nil
}

func mapsClone(m map[string]string) map[string]string {
	out := make(map[string]string, len(m)+1)
	for k, v := range m {
		out[k] = v
	}
	return out
}

func expandRow(chunk, alias string, row map[string]string) string {
	fieldRe := regexp.MustCompile(`\[` + regexp.QuoteMeta(alias) + `->([a-zA-Z0-9_]+)\]`)
	chunk = fieldRe.ReplaceAllStringFunc(chunk, func(m string) string {
		k := fieldRe.FindStringSubmatch(m)[1]
		return html.EscapeString(row[k])
	})
	for k, v := range row {
		chunk = strings.ReplaceAll(chunk, "'.$"+alias+"->"+k+".'", v)
		chunk = strings.ReplaceAll(chunk, `".$`+alias+`->`+k+`."`, v)
		chunk = strings.ReplaceAll(chunk, "$"+alias+"->"+k, v)
	}
	return chunk
}

func expandBrackets(s string, d Data) string {
	return bracketRe.ReplaceAllStringFunc(s, func(m string) string {
		name := strings.ReplaceAll(bracketRe.FindStringSubmatch(m)[1], "->", ".")
		return lookup(d, name)
	})
}

func expandURL(s string) string {
	return urlRe.ReplaceAllStringFunc(s, func(m string) string {
		p := strings.Trim(urlRe.FindStringSubmatch(m)[1], "/")
		p = strings.ReplaceAll(p, "'", "")
		p = strings.ReplaceAll(p, `"`, "")
		// PHP concat leftovers: /admin/.../scode/'.X.' → .../scode/.X.
		p = strings.ReplaceAll(p, "/.", "/")
		p = strings.Trim(p, ".")
		p = strings.TrimRight(p, "/")
		return "/" + p
	})
}

func replaceVars(s string, d Data) string {
	s = contentDynRe.ReplaceAllStringFunc(s, func(m string) string {
		k := contentDynRe.FindStringSubmatch(m)[1]
		if cm, ok := d.Vars["content"].(map[string]string); ok {
			return cm[k]
		}
		return ""
	})
	return varRe.ReplaceAllStringFunc(s, func(m string) string {
		name := varRe.FindStringSubmatch(m)[1]
		return lookup(d, name)
	})
}

func renderIf(s string, d Data, row map[string]string) string {
	// config.html alone has 100+ ifs; keep going until none left (cap guards runaway)
	for i := 0; i < 500; i++ {
		m := matchIf(s)
		if m == nil {
			return s
		}
		body := m.els
		if truth(m.cond, d, row) {
			body = m.inner
		}
		s = s[:m.start] + body + s[m.end:]
	}
	return s
}

type ifMatch struct {
	start, end       int
	cond, inner, els string
}

func matchIf(s string) *ifMatch {
	const open = "{if("
	idx := strings.Index(s, open)
	if idx < 0 {
		return nil
	}
	// find matching ) } after condition
	depth := 1
	i := idx + len(open)
	for i < len(s) && depth > 0 {
		switch s[i] {
		case '(':
			depth++
		case ')':
			depth--
		}
		i++
	}
	if depth != 0 || i >= len(s) || s[i] != '}' {
		return nil
	}
	cond := s[idx+len(open) : i-1]
	bodyStart := i + 1
	depth = 1
	elseAt := -1
	j := bodyStart
	for j < len(s) {
		if strings.HasPrefix(s[j:], "{if(") {
			depth++
			j += 4
			continue
		}
		if depth == 1 && strings.HasPrefix(s[j:], "{else}") {
			elseAt = j
			j += len("{else}")
			continue
		}
		if strings.HasPrefix(s[j:], "{/if}") {
			depth--
			if depth == 0 {
				inner, els := s[bodyStart:j], ""
				if elseAt >= 0 {
					inner = s[bodyStart:elseAt]
					els = s[elseAt+len("{else}") : j]
				}
				return &ifMatch{start: idx, end: j + len("{/if}"), cond: cond, inner: inner, els: els}
			}
			j += len("{/if}")
			continue
		}
		j++
	}
	return nil
}

func truth(cond string, d Data, row map[string]string) bool {
	cond = strings.TrimSpace(cond)
	if strings.HasPrefix(cond, "check_level(") {
		arg := strings.Trim(cond[len("check_level("):], " )'\"")
		if d.Funs["check_level"] != nil {
			return d.Funs["check_level"]([]string{arg}) != ""
		}
		return false
	}
	if strings.Contains(cond, "ueditor_catch_remote_enabled()") {
		on := d.Funs["ueditor_catch_remote_enabled"] != nil && d.Funs["ueditor_catch_remote_enabled"](nil) != ""
		if strings.Contains(cond, "!") {
			return !on
		}
		return on
	}
	cond = regexp.MustCompile(`\$value\d*->`).ReplaceAllString(cond, "")
	cond = strings.ReplaceAll(cond, "@", "")
	cond = sessCallRe.ReplaceAllStringFunc(cond, func(m string) string {
		return lit(d.Session[sessCallRe.FindStringSubmatch(m)[1]])
	})
	cond = bracketRe.ReplaceAllStringFunc(cond, func(m string) string {
		name := strings.ReplaceAll(bracketRe.FindStringSubmatch(m)[1], "->", ".")
		return lit(lookup(d, name))
	})
	cond = notLitRe.ReplaceAllStringFunc(cond, func(m string) string {
		sub := notLitRe.FindStringSubmatch(m)
		v := sub[1]
		if sub[2] != "" {
			v = sub[2]
		}
		if v == "" || v == "0" {
			return "1"
		}
		return "0"
	})
	cond = identRe.ReplaceAllStringFunc(cond, func(name string) string {
		if row != nil {
			if v, ok := row[name]; ok {
				return lit(v)
			}
		}
		if v, ok := d.Vars[name]; ok {
			return lit(fmt.Sprint(v))
		}
		if name == "LICENSE" || name == "CMSNAME" {
			if name == "CMSNAME" {
				return lit("GbootCMS")
			}
			return "3"
		}
		return name
	})
	return textmod.Truth(cond)
}

func lit(v string) string {
	if v == "" {
		return "''"
	}
	for _, r := range v {
		if r < '0' || r > '9' {
			return "'" + strings.ReplaceAll(v, "'", "") + "'"
		}
	}
	return v
}

func lookup(d Data, name string) string {
	name = strings.ReplaceAll(name, "->", ".")
	if strings.HasPrefix(name, "session.") {
		return d.Session[strings.TrimPrefix(name, "session.")]
	}
	if strings.HasPrefix(name, "get.") {
		return d.Get[strings.TrimPrefix(name, "get.")]
	}
	if strings.HasPrefix(name, "configs.") && strings.HasSuffix(name, ".value") {
		key := strings.TrimSuffix(strings.TrimPrefix(name, "configs."), ".value")
		if m, ok := d.Vars["configs"].(map[string]map[string]string); ok {
			if row := m[key]; row != nil {
				return row["value"]
			}
		}
		return ""
	}
	if i := strings.IndexByte(name, '.'); i > 0 {
		head, tail := name[:i], name[i+1:]
		if m, ok := d.Vars[head].(map[string]string); ok {
			return m[tail]
		}
	}
	if v, ok := d.Vars[name]; ok {
		switch t := v.(type) {
		case string:
			return t
		case fmt.Stringer:
			return t.String()
		default:
			return fmt.Sprint(t)
		}
	}
	return ""
}

func splitArgs(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	for i, p := range parts {
		parts[i] = strings.Trim(strings.TrimSpace(p), "'\"")
	}
	return parts
}
