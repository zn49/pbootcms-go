package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNormalizeAdminPath(t *testing.T) {
	if NormalizeAdminPath("") != "/admin" {
		t.Fatal("empty")
	}
	if NormalizeAdminPath("x7k2") != "/x7k2" {
		t.Fatal("slash")
	}
	if NormalizeAdminPath("/api") != "/admin" {
		t.Fatal("reserved")
	}
}

func TestAutoAdminPathPersists(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "config.json")
	raw := map[string]any{
		"addr": ":0", "driver": "sqlite", "dsn": filepath.Join(dir, "t.db"),
		"site_root": dir, "admin_path": "auto",
	}
	b, _ := json.Marshal(raw)
	_ = os.WriteFile(p, b, 0o644)
	f, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if f.AdminPath == "/admin" || !strings.HasPrefix(f.AdminPath, "/") || len(f.AdminPath) < 5 {
		t.Fatalf("bad path %q", f.AdminPath)
	}
	nb, _ := os.ReadFile(p)
	if !strings.Contains(string(nb), f.AdminPath) {
		t.Fatalf("not persisted: %s", nb)
	}
}
