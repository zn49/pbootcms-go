package auth

import (
	"strings"
	"unicode"
)

// Strong reports whether plain meets admin strong-password rules:
// ≥8 chars, upper+lower+digit. Special char recommended but not required
// (keeps Chinese-admin UX workable).
func Strong(plain string) bool {
	if len(plain) < 8 {
		return false
	}
	var upper, lower, digit bool
	for _, r := range plain {
		switch {
		case unicode.IsUpper(r):
			upper = true
		case unicode.IsLower(r):
			lower = true
		case unicode.IsDigit(r):
			digit = true
		}
	}
	if !upper || !lower || !digit {
		return false
	}
	low := strings.ToLower(plain)
	for _, bad := range []string{"12345678", "password", "qwertyui", "admin123", "pbootcms-go", "pbootcms", "gbootcms"} {
		if low == bad || strings.Contains(low, bad) && len(bad) >= 8 {
			return false
		}
	}
	return true
}

// StrongMsg is the user-facing reason when Strong fails.
func StrongMsg() string {
	return "密码至少8位，且包含大小写字母和数字"
}
