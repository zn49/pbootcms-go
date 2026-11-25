package pboot

import (
	"path/filepath"
	"strings"
	"testing"

	"gbootcms/internal/domain"
	"gbootcms/internal/store"
	"gbootcms/internal/urlx"
)

func TestRenderSnippet(t *testing.T) {
	db := openSQL(t)
	ctx := &Ctx{
		DB: db, Acode: "cn", Rules: urlx.Rules{Type: 3, Suffix: ".html", Break: "_"},
		SchemeHost: "http://127.0.0.1",
	}
	site, err := domain.Site(db, "cn")
	if err != nil || site == nil {
		t.Fatal(err)
	}
	ctx.Site = site
	ctx.Company, _ = domain.Company(db, "cn")
	ctx.Sorts, _ = domain.Sorts(db, "cn")
	ctx.Labels, _ = domain.Labels(db)
	ctx.TplRoot = filepath.Join("..", "..", "..", "site", "template", "default", "html")
	html, err := RenderFile(filepath.Join(ctx.TplRoot, "index.html"), ctx)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(html, "{pboot:slide") || strings.Contains(html, "{pboot:list") || strings.Contains(html, "{pboot:nav") {
		t.Fatalf("unexpanded tag:\n%s", html[:min(500, len(html))])
	}
	if !strings.Contains(html, "GbootCMS") {
		t.Fatal("missing site title")
	}
}

func openSQL(t *testing.T) *store.DB {
	t.Helper()
	db, err := store.Open("sqlite", filepath.Join("..", "..", "..", "site", "data", "gbootcms.db"), filepath.Join("..", "..", "..", "site"), "")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.SQL.Close() })
	return db
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
