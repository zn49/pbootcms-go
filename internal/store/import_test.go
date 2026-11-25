package store

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestImportEmptySQLite(t *testing.T) {
	root := filepath.Join("..", "..", "site")
	sqlPath := filepath.Join(root, "static", "backup", "sql", "pbootcms.sql")
	if _, err := os.Stat(sqlPath); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	db, err := Open("sqlite", filepath.Join(dir, "empty.db"), root, sqlPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.SQL.Close()
	if db.Conf("url_rule_type") == "" {
		t.Fatal("ay_config was not loaded")
	}
	n, err := db.Ent.Content.Query().Count(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 { t.Fatalf("expected empty content after demo strip, got %d", n) }
}
