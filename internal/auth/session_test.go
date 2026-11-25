package auth

import "testing"

func TestValidSessionID(t *testing.T) {
	if !validSessionID("abcdef0123456789") {
		t.Fatal("hex ok")
	}
	if validSessionID("../etc/passwd") || validSessionID("ab/cd") || validSessionID("") {
		t.Fatal("reject path-like ids")
	}
}
