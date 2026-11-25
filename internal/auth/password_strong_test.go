package auth

import "testing"

func TestStrong(t *testing.T) {
	if Strong("123456") || Strong("abcdefgh") || Strong("Abcdefgh") || Strong("secret") {
		t.Fatal("weak accepted")
	}
	if !Strong("Abcd1234") {
		t.Fatal("good rejected")
	}
}
