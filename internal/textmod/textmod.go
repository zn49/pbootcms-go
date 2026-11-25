package textmod

import (
	"html"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

var tagRe = regexp.MustCompile(`<[^>]*>`)

// Attrs parses `k=v k2='v 2'` attribute lists.
func Attrs(s string) map[string]string {
	out := map[string]string{}
	s = strings.TrimSpace(s)
	for len(s) > 0 {
		s = strings.TrimLeft(s, " \t\r\n")
		if s == "" {
			break
		}
		eq := strings.IndexByte(s, '=')
		sp := strings.IndexAny(s, " \t\r\n")
		if eq < 0 || (sp >= 0 && sp < eq) {
			key := s
			if sp >= 0 {
				key = s[:sp]
				s = s[sp:]
			} else {
				s = ""
			}
			out[key] = "1"
			continue
		}
		key := strings.TrimSpace(s[:eq])
		rest := strings.TrimLeft(s[eq+1:], " \t")
		var val string
		if rest == "" {
			s = ""
		} else if rest[0] == '\'' || rest[0] == '"' {
			q := rest[0]
			rest = rest[1:]
			i := strings.IndexByte(rest, q)
			if i < 0 {
				val = rest
				s = ""
			} else {
				val = rest[:i]
				s = rest[i+1:]
			}
		} else {
			i := strings.IndexAny(rest, " \t\r\n")
			if i < 0 {
				val = rest
				s = ""
			} else {
				val = rest[:i]
				s = rest[i:]
			}
		}
		out[key] = val
	}
	return out
}

// Apply runs pboot field modifiers.
func Apply(val, attr, keyword string) string {
	// PHP decode_string: content often stored entity-encoded
	val = html.UnescapeString(val)
	p := Attrs(attr)
	if p["drophtml"] == "1" {
		val = tagRe.ReplaceAllString(val, "")
	}
	if p["dropblank"] == "1" {
		val = strings.TrimSpace(stripSpace(val))
	}
	cut := false
	if n, ok := p["lencn"]; ok {
		val, cut = cutRunes(val, atoi(n))
	} else if n, ok := p["len"]; ok {
		val, cut = cutRunes(val, atoi(n))
	}
	if more, ok := p["more"]; ok && cut {
		val += more
	}
	if st, ok := p["style"]; ok {
		val = formatDate(val, st)
	}
	if op, ok := p["operate"]; ok {
		val = operate(val, op)
	}
	if p["mark"] == "1" && keyword != "" {
		val = strings.ReplaceAll(val, keyword, "<em>"+keyword+"</em>")
	}
	return val
}

func stripSpace(s string) string {
	s = strings.ReplaceAll(s, "&nbsp;", "")
	s = strings.ReplaceAll(s, "\n", "")
	s = strings.ReplaceAll(s, "\r", "")
	s = strings.ReplaceAll(s, "\t", "")
	return strings.Join(strings.Fields(s), "")
}

func cutRunes(s string, n int) (string, bool) {
	if n <= 0 || utf8.RuneCountInString(s) <= n {
		return s, false
	}
	r := []rune(s)
	return string(r[:n]), true
}

func atoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

func formatDate(val, style string) string {
	t, err := time.ParseInLocation("2006-01-02 15:04:05", val, time.Local)
	if err != nil {
		t, err = time.ParseInLocation("2006-01-02", val, time.Local)
		if err != nil {
			return val
		}
	}
	style = strings.ReplaceAll(style, "Y", "2006")
	style = strings.ReplaceAll(style, "m", "01")
	style = strings.ReplaceAll(style, "d", "02")
	style = strings.ReplaceAll(style, "H", "15")
	style = strings.ReplaceAll(style, "i", "04")
	style = strings.ReplaceAll(style, "s", "05")
	return t.Format(style)
}

func operate(val, op string) string {
	n, _ := strconv.Atoi(strings.TrimSpace(val))
	op = strings.TrimSpace(op)
	if strings.HasPrefix(op, "+") {
		d, _ := strconv.Atoi(op[1:])
		return strconv.Itoa(n + d)
	}
	if strings.HasPrefix(op, "-") {
		d, _ := strconv.Atoi(op[1:])
		return strconv.Itoa(n - d)
	}
	return val
}

var inArrayRe = regexp.MustCompile(`(?i)^in_array\(\s*(.+?)\s*,\s*(.+?)\s*\)$`)

// Truth evaluates a small PHP-like condition.
func Truth(expr string) bool {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return false
	}
	if strings.HasPrefix(expr, "!") {
		return !Truth(strings.TrimSpace(expr[1:]))
	}
	if i := indexTop(expr, "||"); i >= 0 {
		return Truth(expr[:i]) || Truth(expr[i+2:])
	}
	if i := indexTop(expr, "&&"); i >= 0 {
		return Truth(expr[:i]) && Truth(expr[i+2:])
	}
	if m := inArrayRe.FindStringSubmatch(expr); m != nil {
		needle := unquote(strings.TrimSpace(m[1]))
		hay := unquote(strings.TrimSpace(m[2]))
		for _, p := range strings.Split(hay, ",") {
			if strings.TrimSpace(p) == needle {
				return true
			}
		}
		return false
	}
	ops := []string{"!=", "==", ">=", "<=", ">", "<"}
	for _, op := range ops {
		if i := indexTop(expr, op); i >= 0 {
			l := unquote(strings.TrimSpace(expr[:i]))
			r := unquote(strings.TrimSpace(expr[i+len(op):]))
			return cmp(l, r, op)
		}
	}
	v := unquote(expr)
	return v != "" && v != "0"
}

func indexTop(s, op string) int {
	inQ := byte(0)
	for i := 0; i+len(op) <= len(s); i++ {
		c := s[i]
		if inQ != 0 {
			if c == inQ {
				inQ = 0
			}
			continue
		}
		if c == '\'' || c == '"' {
			inQ = c
			continue
		}
		if s[i:i+len(op)] == op {
			return i
		}
	}
	return -1
}

func unquote(s string) string {
	if len(s) >= 2 {
		if (s[0] == '\'' && s[len(s)-1] == '\'') || (s[0] == '"' && s[len(s)-1] == '"') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

func cmp(l, r, op string) bool {
	ln, lok := num(l)
	rn, rok := num(r)
	if lok && rok {
		switch op {
		case "==":
			return ln == rn
		case "!=":
			return ln != rn
		case ">":
			return ln > rn
		case "<":
			return ln < rn
		case ">=":
			return ln >= rn
		case "<=":
			return ln <= rn
		}
	}
	// PHP 7: 0 == ” is true. Non-numeric strings stay unequal to 0.
	if op == "==" && ((l == "0" && r == "") || (l == "" && r == "0")) {
		return true
	}
	if op == "!=" && ((l == "0" && r == "") || (l == "" && r == "0")) {
		return false
	}
	switch op {
	case "==":
		return l == r
	case "!=":
		return l != r
	default:
		return false
	}
}

func num(s string) (float64, bool) {
	if s == "" {
		return 0, false
	}
	n, err := strconv.ParseFloat(s, 64)
	return n, err == nil
}
