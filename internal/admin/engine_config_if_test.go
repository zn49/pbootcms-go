package admin

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigAIKeyIf(t *testing.T) {
	root := filepath.Join("..", "..", "site")
	if _, err := os.Stat(filepath.Join(root, "apps", "admin", "view", "default", "system", "config.html")); err != nil {
		t.Skip(err)
	}
	d := Data{
		Vars: map[string]any{
			"ai_key_openssl_warn": "",
			"formcheck":           "x",
			"configs":             map[string]map[string]string{},
			"llms_sort_checkbox":  "",
			"llms_url":            "/llms.txt",
			"is_sqlite":           "1",
			"ai_provider_json":    "{}",
			"LICENSE":             "3",
		},
		Session: map[string]string{"id": "1", "username": "admin"},
		Get:     map[string]string{},
		Funs:    map[string]func([]string) string{},
	}
	out, err := RenderPHP(root, "system/config.html", d)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "{if(==1)}") || strings.Contains(out, "{if([$ai_key_openssl_warn]") {
		i := strings.Index(out, "ai_api_key")
		if i < 0 {
			i = 0
		}
		end := i + 800
		if end > len(out) {
			end = len(out)
		}
		t.Fatalf("leftover if near key:\n%s", out[i:end])
	}
	d.Vars["ai_key_openssl_warn"] = "1"
	out, err = RenderPHP(root, "system/config.html", d)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "{if") && strings.Contains(out, "AES") {
		i := strings.Index(out, "AES")
		a := i - 80
		if a < 0 {
			a = 0
		}
		t.Fatalf("tags around AES:\n%s", out[a:i+120])
	}
	if !strings.Contains(out, "AES") {
		t.Fatal("expected AES warn")
	}
}
