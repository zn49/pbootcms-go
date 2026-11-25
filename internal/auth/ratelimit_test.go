package auth

import "testing"

func TestLoginRateLimit(t *testing.T) {
	key := "test-ip-rate"
	LoginOK(key)
	for i := 0; i < 5; i++ {
		if !LoginAllowed(key, 5, 900) {
			t.Fatalf("blocked early at %d", i)
		}
		LoginFail(key)
	}
	if LoginAllowed(key, 5, 900) {
		t.Fatal("expected lockout after 5 fails")
	}
	LoginOK(key)
	if !LoginAllowed(key, 5, 900) {
		t.Fatal("expected clear after OK")
	}
}
