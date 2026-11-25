package auth

import (
	"net/http"
	"strings"
)

// CookieSecure is true when the request is HTTPS (direct TLS or reverse-proxy header).
func CookieSecure(r *http.Request) bool {
	return r != nil && (r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https"))
}

// SessionCookie sets a session id cookie: HttpOnly + SameSite=Lax; Secure when TLS.
func SessionCookie(w http.ResponseWriter, r *http.Request, name, value string) {
	c := &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   CookieSecure(r),
	}
	http.SetCookie(w, c)
}

// SetCookie sets a cookie with SameSite=Lax and Secure when TLS.
func SetCookie(w http.ResponseWriter, r *http.Request, c *http.Cookie) {
	if c.Path == "" {
		c.Path = "/"
	}
	if c.SameSite == 0 {
		c.SameSite = http.SameSiteLaxMode
	}
	c.Secure = CookieSecure(r)
	http.SetCookie(w, c)
}
