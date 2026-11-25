package urlx

import (
	"net/url"
	"strconv"
	"strings"
)

// Rules mirrors ay_config URL settings.
type Rules struct {
	Type        int // 1 normal, 2 rewrite, 3 compatible
	Suffix      string
	Break       string
	ContentPath bool
}

func (r Rules) norm() Rules {
	if r.Type == 0 {
		r.Type = 3
	}
	if r.Suffix == "" {
		r.Suffix = ".html"
	}
	if r.Break == "" {
		r.Break = "_"
	}
	return r
}

// Home builds a front link. suffixMode: 1 = .html, 0 = trailing slash, -1 = none.
func Home(r Rules, path string, suffixMode int) string {
	r = r.norm()
	path = strings.Trim(path, "/")
	suf := "/"
	switch suffixMode {
	case 1:
		suf = r.Suffix
	case -1:
		suf = ""
	}
	if path == "" {
		switch r.Type {
		case 1:
			return "/index.php/"
		case 2:
			return "/"
		default:
			return "/?"
		}
	}
	switch r.Type {
	case 1:
		return "/index.php/" + path + suf
	case 2:
		return "/" + path + suf
	default:
		return "/?" + path + suf
	}
}

func (r Rules) Sort(filename, urlname, scode string) string {
	r = r.norm()
	if filename != "" {
		return Home(r, filename, 0)
	}
	name := urlname
	if name == "" {
		name = "list"
	}
	return Home(r, name+r.Break+scode, 0)
}

func (r Rules) Content(sortFilename, urlname, scode, id, filename string) string {
	r = r.norm()
	if r.ContentPath {
		if filename != "" {
			return Home(r, filename, 1)
		}
		return Home(r, id, 1)
	}
	name := urlname
	if name == "" {
		name = "list"
	}
	leaf := filename
	if leaf == "" {
		leaf = id
	}
	if sortFilename != "" {
		return Home(r, sortFilename+"/"+leaf, 1)
	}
	return Home(r, name+r.Break+scode+"/"+leaf, 1)
}

func (r Rules) Page(base string, page int) string {
	r = r.norm()
	base = strings.Trim(base, "/")
	if page <= 1 {
		return Home(r, base, 0)
	}
	return Home(r, base+r.Break+strconv.Itoa(page), 0)
}

// AddQuery appends a query pair onto a generated link.
func AddQuery(link, key, val string) string {
	if key == "" {
		return link
	}
	enc := url.QueryEscape(key) + "=" + url.QueryEscape(val)
	sep := "?"
	if strings.Contains(link, "?") {
		sep = "&"
	}
	return link + sep + enc
}

// StripSuffix removes the configured suffix or a trailing slash from a path.
func StripSuffix(r Rules, path string) (string, bool) {
	r = r.norm()
	suf := strings.TrimPrefix(r.Suffix, ".")
	if suf != "" && strings.HasSuffix(path, "."+suf) {
		return strings.TrimSuffix(path, "."+suf), true
	}
	if strings.HasSuffix(path, "/") {
		return strings.TrimSuffix(path, "/"), false
	}
	return path, false
}
