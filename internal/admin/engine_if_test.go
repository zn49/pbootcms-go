package admin

import (
	"strings"
	"testing"
)

func TestAiKeyOpensslIf(t *testing.T) {
	d := Data{Vars: map[string]any{"ai_key_openssl_warn": ""}}
	in := "{if([$ai_key_openssl_warn]==1)}\n<div>warn</div>\n{/if}"
	out := renderBody(in, d)
	if strings.Contains(out, "{if") || strings.Contains(out, "warn") {
		t.Fatalf("empty warn: %q", out)
	}
	d.Vars["ai_key_openssl_warn"] = "1"
	out = renderBody(in, d)
	if strings.Contains(out, "{if") || !strings.Contains(out, "warn") {
		t.Fatalf("warn=1: %q", out)
	}
}
