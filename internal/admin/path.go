package admin

import (
	"net/http"
	"strings"

	"gbootcms/internal/auth"
)

// Logical admin URL prefix used in code/templates/ACL (always /admin).
const logicalAdmin = "/admin"

var adminPath = logicalAdmin

// SetAdminPath sets the public admin URL prefix (e.g. /x7k2m9). Empty → /admin.
func SetAdminPath(p string) {
	p = strings.TrimSpace(p)
	if p == "" {
		p = logicalAdmin
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	adminPath = strings.TrimRight(p, "/")
	if adminPath == "" {
		adminPath = logicalAdmin
	}
	auth.SetAdminPathName(adminPath)
}

// SetCookieSalt randomizes cookie names derived from salt.
func SetCookieSalt(s string) {
	auth.SetCookieSalt(s)
}

// Path is the public admin prefix.
func Path() string { return adminPath }

// CookieName is the admin session cookie (non-branded).
func CookieName() string { return auth.AdminCookie() }

// FrontCookieName is the front/member session cookie (non-branded).
func FrontCookieName() string { return auth.FrontCookie() }

// LegacyCookieNames are branded names to expire on reset.
func LegacyCookieNames() []string { return auth.LegacyCookies() }

// mapPublicURL rewrites logical /admin… links to the public prefix in HTML/JSON.
func mapPublicURL(s string) string {
	if adminPath == logicalAdmin {
		return s
	}
	from, to := logicalAdmin, adminPath
	s = strings.ReplaceAll(s, `"`+from+`/`, `"`+to+`/`)
	s = strings.ReplaceAll(s, `'`+from+`/`, `'`+to+`/`)
	s = strings.ReplaceAll(s, `=`+from+`/`, `=`+to+`/`)
	s = strings.ReplaceAll(s, `>`+from+`/`, `>`+to+`/`)
	s = strings.ReplaceAll(s, `(`+from+`/`, `(`+to+`/`)
	s = strings.ReplaceAll(s, `"`+from+`"`, `"`+to+`"`)
	s = strings.ReplaceAll(s, `'`+from+`'`, `'`+to+`'`)
	return s
}

func mapPublicLoc(loc string) string {
	if adminPath == logicalAdmin {
		return loc
	}
	if loc == logicalAdmin {
		return adminPath
	}
	if strings.HasPrefix(loc, logicalAdmin+"/") {
		return adminPath + strings.TrimPrefix(loc, logicalAdmin)
	}
	return loc
}

// WrapPath rewrites Location + HTML/JSON bodies from /admin to the public prefix.
func WrapPath(w http.ResponseWriter) http.ResponseWriter {
	if adminPath == logicalAdmin {
		return w
	}
	return &pathRW{ResponseWriter: w}
}

type pathRW struct {
	http.ResponseWriter
	wrote bool
}

func (p *pathRW) WriteHeader(code int) {
	if loc := p.Header().Get("Location"); loc != "" {
		p.Header().Set("Location", mapPublicLoc(loc))
	}
	p.wrote = true
	p.ResponseWriter.WriteHeader(code)
}

func (p *pathRW) Write(b []byte) (int, error) {
	if !p.wrote {
		p.WriteHeader(http.StatusOK)
	}
	ct := p.Header().Get("Content-Type")
	if ct == "" || strings.Contains(ct, "html") || strings.Contains(ct, "json") || strings.Contains(ct, "javascript") {
		b = []byte(mapPublicURL(string(b)))
	}
	return p.ResponseWriter.Write(b)
}
