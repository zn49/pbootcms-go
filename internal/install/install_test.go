package install

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zn49/pbootcms-go/internal/app"
	"github.com/zn49/pbootcms-go/internal/auth"
	"github.com/zn49/pbootcms-go/internal/config"
	"github.com/zn49/pbootcms-go/internal/ent/site"
	entuser "github.com/zn49/pbootcms-go/internal/ent/user"
)

func TestCompleteFresh(t *testing.T) {
	repo := filepath.Join("..", "..")
	siteSrc := filepath.Join(repo, "site")
	sqlSrc := filepath.Join(siteSrc, "static", "backup", "sql", "pbootcms.sql")
	if _, err := os.Stat(sqlSrc); err != nil {
		t.Skip("no schema sql")
	}
	dir := t.TempDir()
	root := filepath.Join(dir, "site")
	mustCopyTree(t, filepath.Join(siteSrc, "static", "backup", "sql"), filepath.Join(root, "static", "backup", "sql"))
	cfgPath := filepath.Join(dir, "config.json")
	a := app.New(config.File{
		Addr:     ":0",
		Driver:   "sqlite",
		DSN:      filepath.Join(root, "data", "pbootcms.db"),
		SiteRoot: root,
	})
	a.CfgPath = cfgPath

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/install", nil)
	Serve(a, rec, req)
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), "安装 PbootCMS-Go") {
		t.Fatalf("GET install: %d %s", rec.Code, rec.Body.String())
	}

	path, plain, err := Complete(a, Input{
		Title:      "测试站",
		Username:   "boss",
		Password:   "Secret99",
		Confirm:    "Secret99",
		Addr:       ":8099",
		Driver:     "sqlite",
		SQLitePath: filepath.Join(root, "data", "pbootcms.db"),
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = a.ClearBootstrap() })
	if path == "" || path[0] != '/' {
		t.Fatalf("admin path %q", path)
	}
	if plain != "Secret99" {
		t.Fatalf("plain %q", plain)
	}
	if !Locked(root) {
		t.Fatal("expected lock")
	}
	if _, err := os.Stat(cfgPath); err != nil {
		t.Fatal(err)
	}
	one, err := a.DB.Ent.User.Query().Where(entuser.IDEQ(1)).Only(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if one.Username != "boss" || one.Password != auth.Encrypt("Secret99") {
		t.Fatalf("user %#v", one)
	}
	st, err := a.DB.Ent.Site.Query().Where(site.AcodeEQ("cn")).Only(context.Background())
	if err != nil || st.Title != "测试站" {
		t.Fatalf("site %#v %v", st, err)
	}
	// second complete fails
	if _, _, err := Complete(a, Input{Title: "x", Username: "boss", Password: "Secret99", Confirm: "Secret99", Driver: "sqlite"}); err == nil {
		t.Fatal("expected already installed")
	}
}

func TestGenPassword(t *testing.T) {
	for i := 0; i < 20; i++ {
		p := GenPassword()
		if !auth.Strong(p) || len(p) != 16 {
			t.Fatalf("%q", p)
		}
	}
}

func TestBuildDSN(t *testing.T) {
	wd := t.TempDir()
	root := filepath.Join(wd, "site")
	dsn, err := buildDSN(wd, root, Input{Driver: "mysql", DBHost: "127.0.0.1", DBPort: "3306", DBUser: "u", DBPass: "p@ss", DBName: "cms"}, "mysql")
	if err != nil || !strings.Contains(dsn, "tcp(127.0.0.1:3306)") || !strings.Contains(dsn, "/cms") {
		t.Fatalf("%q %v", dsn, err)
	}
	dsn, err = buildDSN(wd, root, Input{Driver: "pgsql", DBUser: "u", DBPass: "p", DBName: "cms"}, "pgsql")
	if err != nil || !strings.HasPrefix(dsn, "postgres://") {
		t.Fatalf("%q %v", dsn, err)
	}
}

func mustCopyTree(t *testing.T, src, dst string) {
	t.Helper()
	if err := os.MkdirAll(dst, 0o755); err != nil {
		t.Fatal(err)
	}
	ents, err := os.ReadDir(src)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range ents {
		sb, err := os.ReadFile(filepath.Join(src, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dst, e.Name()), sb, 0o644); err != nil {
			t.Fatal(err)
		}
	}
}
