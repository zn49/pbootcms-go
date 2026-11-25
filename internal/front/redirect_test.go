package front

import (
	"net/http"
	"testing"
)

func TestSafeRefererPath(t *testing.T) {
	r, _ := http.NewRequest("GET", "http://ex.com/x", nil)
	r.Host = "ex.com"
	r.Header.Set("Referer", "http://evil.com/phish")
	if safeRefererPath(r) != "/" {
		t.Fatal("cross-host referer")
	}
	r.Header.Set("Referer", "http://ex.com/a/b?c=1")
	if got := safeRefererPath(r); got != "/a/b?c=1" {
		t.Fatalf("same host got %q", got)
	}
	if safeMemberBack("//evil.com", "http://x", "/ok") != "/ok" {
		t.Fatal("member back")
	}
}
