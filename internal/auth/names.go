package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

var (
	nameSalt  = "default"
	nameAdmin = "/admin"
)

// SetSessionNames sets non-branded cookie name material (salt + admin path).
func SetSessionNames(adminPath, cookieSalt string) {
	SetAdminPathName(adminPath)
	SetCookieSalt(cookieSalt)
}

// SetAdminPathName updates only the admin path used in cookie derivation.
func SetAdminPathName(adminPath string) {
	adminPath = strings.TrimSpace(adminPath)
	if adminPath == "" {
		adminPath = "/admin"
	}
	if !strings.HasPrefix(adminPath, "/") {
		adminPath = "/" + adminPath
	}
	nameAdmin = strings.TrimRight(adminPath, "/")
}

// SetCookieSalt updates only the cookie salt.
func SetCookieSalt(cookieSalt string) {
	cookieSalt = strings.TrimSpace(cookieSalt)
	if cookieSalt == "" {
		cookieSalt = "default"
	}
	nameSalt = cookieSalt
}

// AdminCookie is the admin session cookie name.
func AdminCookie() string {
	sum := sha256.Sum256([]byte("a|" + nameSalt + "|" + nameAdmin))
	return "s" + hex.EncodeToString(sum[:6])
}

// FrontCookie is the front/member session cookie name.
func FrontCookie() string {
	sum := sha256.Sum256([]byte("f|" + nameSalt))
	return "s" + hex.EncodeToString(sum[:6])
}

// LegacyCookies are branded names to expire after rotation.
func LegacyCookies() []string {
	return []string{"gboot_admin", "gboot_front", "pboot_admin", "pboot_front"}
}
