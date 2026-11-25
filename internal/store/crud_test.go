package store

import (
	"context"
	"path/filepath"
	"testing"

	"gbootcms/internal/ent/content"
	"gbootcms/internal/ent/contentsort"
	"gbootcms/internal/ent/label"
)

func TestCRUDHelpersSQLite(t *testing.T) {
	root := filepath.Join("..", "..", "site")
	sqlPath := filepath.Join(root, "static", "backup", "sql", "pbootcms.sql")
	dir := t.TempDir()
	db, err := Open("sqlite", filepath.Join(dir, "crud.db"), root, sqlPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.SQL.Close()

	ok, err := db.TableExists("ay_config")
	if err != nil || !ok {
		t.Fatalf("TableExists: %v %v", ok, err)
	}
	ok, err = db.TableExists("no_such_table_xyz")
	if err != nil || ok {
		t.Fatalf("missing table: %v %v", ok, err)
	}

	ctx := context.Background()
	if err := db.Ent.Label.Create().
		SetName("t_label").SetValue("v1").SetType("1").SetDescription("test").
		SetCreateUser("t").SetUpdateUser("t").SetCreateTime("2026-01-01 00:00:00").SetUpdateTime("2026-01-01 00:00:00").
		Exec(ctx); err != nil {
		t.Fatal(err)
	}
	one, err := db.Ent.Label.Query().Where(label.NameEQ("t_label")).First(ctx)
	if err != nil || one.Value != "v1" {
		t.Fatalf("select: %+v %v", one, err)
	}
	if _, err := db.Ent.Label.Update().Where(label.NameEQ("t_label")).SetValue("v2").Save(ctx); err != nil {
		t.Fatal(err)
	}
	one, _ = db.Ent.Label.Query().Where(label.NameEQ("t_label")).First(ctx)
	if one.Value != "v2" {
		t.Fatalf("update got %v", one.Value)
	}
	if _, err := db.Ent.Label.Delete().Where(label.NameEQ("t_label")).Exec(ctx); err != nil {
		t.Fatal(err)
	}
	n, _ := db.Ent.Label.Query().Where(label.NameEQ("t_label")).Count(ctx)
	if n != 0 {
		t.Fatal("delete failed")
	}
	if err := db.CreateDiy("ay_diy_demo", []string{"field_a"}); err != nil {
		t.Fatal(err)
	}
	ok, _ = db.TableExists("ay_diy_demo")
	if !ok {
		t.Fatal("diy table missing")
	}
	now := "2026-01-01 00:00:00"
	_, err = db.Ent.ContentSort.Create().
		SetAcode("cn").SetMcode("2").SetPcode("0").SetScode("1").
		SetName("新闻").SetFilename("news").SetStatus("1").
		SetListtpl("newslist.html").SetContenttpl("news.html").
		SetOutlink("").SetSubname("").SetDef1("").SetDef2("").SetDef3("").
		SetIco("").SetPic("").SetTitle("").SetKeywords("").SetDescription("").
		SetSorting(255).SetGtype("4").SetGid("").SetGnote("").
		SetCreateUser("t").SetUpdateUser("t").SetCreateTime(now).SetUpdateTime(now).
		Save(ctx)
	if err != nil {
		t.Fatal(err)
	}
	row, err := db.Ent.Content.Create().
		SetAcode("cn").SetScode("1").SetSubscode("").
		SetTitle("t").SetTitlecolor("#333").SetSubtitle("").SetFilename("").
		SetAuthor("t").SetSource("").SetOutlink("").SetDate(now).
		SetIco("").SetPics("").SetPicstitle("").SetContent("x").
		SetTags("").SetEnclosure("").SetKeywords("").SetDescription("").
		SetSorting(255).SetStatus("1").SetIstop("0").SetIsrecommend("0").SetIsheadline("0").
		SetGid("").SetGtype("4").SetGnote("").SetVisits(0).SetLikes(0).SetOppose(0).
		SetCreateUser("t").SetUpdateUser("t").SetCreateTime(now).SetUpdateTime(now).
		Save(ctx)
	if err != nil {
		t.Fatal(err)
	}
	sort, err := db.Ent.ContentSort.Query().Where(contentsort.ScodeEQ(row.Scode)).First(ctx)
	if err != nil || sort.Name == "" {
		t.Fatalf("sort: %+v %v", sort, err)
	}
	before := row.Visits
	if _, err := db.Ent.Content.Update().Where(content.IDEQ(row.ID)).AddVisits(1).Save(ctx); err != nil {
		t.Fatal(err)
	}
	after, _ := db.Ent.Content.Get(ctx, row.ID)
	if after.Visits == before {
		t.Fatalf("incr stuck at %d", before)
	}
}

func TestOpenRejectsBadDriver(t *testing.T) {
	_, err := Open("oracle", "x", t.TempDir(), "")
	if err == nil {
		t.Fatal("expected error")
	}
}
