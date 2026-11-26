package pboot

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/zn49/pbootcms-go/internal/domain"
	"github.com/zn49/pbootcms-go/internal/store"
	"github.com/zn49/pbootcms-go/internal/urlx"
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
	if !strings.Contains(html, "PbootCMS-Go") {
		t.Fatal("missing site title")
	}
}

func TestMultiSortAndElseIsico(t *testing.T) {
	ctx := &Ctx{
		Rules: urlx.Rules{Type: 3, Suffix: ".html", Break: "_"},
		Sorts: []map[string]string{
			{"scode": "16", "pcode": "15", "name": "A", "status": "1", "filename": "fa1", "mcode": "1", "def1": "descA", "ico": "/a.jpg"},
			{"scode": "17", "pcode": "15", "name": "B", "status": "1", "filename": "fa2", "mcode": "1", "def1": "descB", "ico": "/b.jpg"},
		},
	}
	out := Render(`{pboot:sort scode=16,17 num=2}([sort:name]|[sort:def1]){/pboot:sort}`, ctx)
	if !strings.Contains(out, "(A|descA)") || !strings.Contains(out, "(B|descB)") {
		t.Fatalf("multi sort: %s", out)
	}
	item := contentItem(map[string]string{"id": "1", "scode": "1", "ico": "/x.png", "filename": ""}, ctx)
	if item["isico"] != "1" {
		t.Fatalf("isico want 1 got %q", item["isico"])
	}
	empty := contentItem(map[string]string{"id": "2", "scode": "1", "ico": "", "filename": ""}, ctx)
	if empty["isico"] != "0" || empty["ico"] != "/static/images/nopic.png" {
		t.Fatalf("empty ico: %#v", empty)
	}
	ifOut := Render(`{pboot:if('1'=='1')}YES{pboot:else}NO{/pboot:if}`, ctx)
	if ifOut != "YES" {
		t.Fatalf("pboot:else true branch: %q", ifOut)
	}
	ifOut = Render(`{pboot:if('0'=='1')}YES{pboot:else}NO{/pboot:if}`, ctx)
	if ifOut != "NO" {
		t.Fatalf("pboot:else false branch: %q", ifOut)
	}
}

func TestTwoNav(t *testing.T) {
	ctx := &Ctx{
		Rules: urlx.Rules{Type: 3, Suffix: ".html", Break: "_"},
		Sorts: []map[string]string{
			{"scode": "1", "pcode": "0", "name": "Top", "status": "1", "filename": "top", "mcode": "1"},
			{"scode": "15", "pcode": "1", "name": "ChildA", "status": "1", "filename": "a", "mcode": "1"},
			{"scode": "16", "pcode": "1", "name": "ChildB", "status": "1", "filename": "b", "mcode": "1"},
		},
	}
	out := Render(`{pboot:nav parent=0}<b>[nav:name]</b>{pboot:2nav parent=[nav:scode]}([2nav:name]){/pboot:2nav}{/pboot:nav}`, ctx)
	if strings.Contains(out, "{pboot:2nav") || strings.Contains(out, "[2nav:") {
		t.Fatalf("2nav not expanded: %s", out)
	}
	if !strings.Contains(out, "(ChildA)") || !strings.Contains(out, "(ChildB)") {
		t.Fatalf("missing children: %s", out)
	}
}

func openSQL(t *testing.T) *store.DB {
	t.Helper()
	db, err := store.Open("sqlite", filepath.Join("..", "..", "..", "site", "data", "pbootcms.db"), filepath.Join("..", "..", "..", "site"), "")
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
