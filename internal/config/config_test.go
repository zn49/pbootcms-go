package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDriverWhitelist(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "c.json")
	_ = os.WriteFile(path, []byte(`{"driver":"oracle","dsn":"x","site_root":"."}`), 0o644)
	if _, err := Load(path); err == nil {
		t.Fatal("oracle should fail")
	}
	_ = os.WriteFile(path, []byte(`{"driver":"postgres","dsn":"host=localhost user=a dbname=b","site_root":"."}`), 0o644)
	f, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if f.Driver != "pgsql" {
		t.Fatalf("got %q", f.Driver)
	}
}
