package httpx

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSecurityHeaders(t *testing.T) {
	h := securityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(204)
	}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	h.ServeHTTP(rr, req)
	for _, k := range []string{"X-Content-Type-Options", "X-Frame-Options", "Referrer-Policy"} {
		if rr.Header().Get(k) == "" {
			t.Fatalf("missing header %s", k)
		}
	}
	if rr.Header().Get("Strict-Transport-Security") != "" {
		t.Fatal("HSTS should not set on plain HTTP")
	}
	rr2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.Header.Set("X-Forwarded-Proto", "https")
	h.ServeHTTP(rr2, req2)
	if rr2.Header().Get("Strict-Transport-Security") == "" {
		t.Fatal("expected HSTS behind https proxy")
	}
}
