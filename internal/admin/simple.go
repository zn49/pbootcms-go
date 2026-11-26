package admin

import (
	"fmt"
	"net/http"

	"entgo.io/ent/dialect/sql"
	"github.com/zn49/pbootcms-go/internal/app"
	"github.com/zn49/pbootcms-go/internal/auth"
	"github.com/zn49/pbootcms-go/internal/domain"
	"github.com/zn49/pbootcms-go/internal/ent/link"
	"github.com/zn49/pbootcms-go/internal/ent/slide"
	"github.com/zn49/pbootcms-go/internal/ent/tag"
	"github.com/zn49/pbootcms-go/internal/front"
	"github.com/zn49/pbootcms-go/internal/store"
)

func crudDel(a *app.App, table, acode, id string) error {
	n := domain.Atoi(id, 0)
	switch table {
	case "ay_slide":
		_, err := a.DB.Ent.Slide.Delete().Where(slide.IDEQ(n), slide.AcodeEQ(acode)).Exec(bg)
		return err
	case "ay_link":
		_, err := a.DB.Ent.Link.Delete().Where(link.IDEQ(n), link.AcodeEQ(acode)).Exec(bg)
		return err
	case "ay_tags":
		_, err := a.DB.Ent.Tag.Delete().Where(tag.IDEQ(n), tag.AcodeEQ(acode)).Exec(bg)
		return err
	default:
		return fmt.Errorf("unknown table")
	}
}

func crudInsert(a *app.App, table, acode, user string, r *http.Request) error {
	now := domain.Now()
	switch table {
	case "ay_slide":
		gid := domain.Atoi(r.FormValue("gid"), 0)
		if r.FormValue("gid") == "" {
			gid = nextGid(a, acode, "slide")
		}
		return a.DB.Ent.Slide.Create().
			SetAcode(acode).SetGid(gid).
			SetPic(r.FormValue("pic")).SetLink(r.FormValue("link")).
			SetTitle(r.FormValue("title")).SetSubtitle(r.FormValue("subtitle")).
			SetSorting(domain.Atoi(r.FormValue("sorting"), 255)).
			SetCreateUser(user).SetUpdateUser(user).SetCreateTime(now).SetUpdateTime(now).
			Exec(bg)
	case "ay_link":
		gid := domain.Atoi(r.FormValue("gid"), 0)
		if r.FormValue("gid") == "" {
			gid = nextGid(a, acode, "link")
		}
		return a.DB.Ent.Link.Create().
			SetAcode(acode).SetGid(gid).
			SetName(r.FormValue("name")).SetLink(r.FormValue("link")).SetLogo(r.FormValue("logo")).
			SetSorting(domain.Atoi(r.FormValue("sorting"), 255)).
			SetCreateUser(user).SetUpdateUser(user).SetCreateTime(now).SetUpdateTime(now).
			Exec(bg)
	case "ay_tags":
		return a.DB.Ent.Tag.Create().
			SetAcode(acode).SetName(r.FormValue("name")).SetLink(r.FormValue("link")).
			SetCreateUser(user).SetUpdateUser(user).SetCreateTime(now).SetUpdateTime(now).
			Exec(bg)
	default:
		return fmt.Errorf("unknown table")
	}
}

func crudUpdate(a *app.App, table, acode, id string, r *http.Request) error {
	n := domain.Atoi(id, 0)
	now := domain.Now()
	switch table {
	case "ay_slide":
		_, err := a.DB.Ent.Slide.Update().Where(slide.IDEQ(n), slide.AcodeEQ(acode)).
			SetGid(domain.Atoi(r.FormValue("gid"), 0)).
			SetPic(r.FormValue("pic")).SetLink(r.FormValue("link")).
			SetTitle(r.FormValue("title")).SetSubtitle(r.FormValue("subtitle")).
			SetSorting(domain.Atoi(r.FormValue("sorting"), 255)).
			SetUpdateTime(now).Save(bg)
		return err
	case "ay_link":
		_, err := a.DB.Ent.Link.Update().Where(link.IDEQ(n), link.AcodeEQ(acode)).
			SetGid(domain.Atoi(r.FormValue("gid"), 0)).
			SetName(r.FormValue("name")).SetLink(r.FormValue("link")).SetLogo(r.FormValue("logo")).
			SetSorting(domain.Atoi(r.FormValue("sorting"), 255)).
			SetUpdateTime(now).Save(bg)
		return err
	case "ay_tags":
		_, err := a.DB.Ent.Tag.Update().Where(tag.IDEQ(n), tag.AcodeEQ(acode)).
			SetName(r.FormValue("name")).SetLink(r.FormValue("link")).
			SetUpdateTime(now).Save(bg)
		return err
	default:
		return fmt.Errorf("unknown table")
	}
}

func crudList(a *app.App, table, acode string) ([]map[string]string, error) {
	switch table {
	case "ay_slide":
		rows, err := a.DB.Ent.Slide.Query().Where(slide.AcodeEQ(acode)).Order(slide.BySorting(), slide.ByID()).All(bg)
		return store.Rows(rows), err
	case "ay_link":
		rows, err := a.DB.Ent.Link.Query().Where(link.AcodeEQ(acode)).Order(link.BySorting(), link.ByID()).All(bg)
		return store.Rows(rows), err
	case "ay_tags":
		rows, err := a.DB.Ent.Tag.Query().Where(tag.AcodeEQ(acode)).Order(tag.ByID(sql.OrderDesc())).All(bg)
		return store.Rows(rows), err
	default:
		return nil, fmt.Errorf("unknown table")
	}
}

func nextGid(a *app.App, acode, kind string) int {
	max := 0
	switch kind {
	case "slide":
		rows, _ := a.DB.Ent.Slide.Query().Where(slide.AcodeEQ(acode)).All(bg)
		for _, r := range rows {
			if r.Gid > max {
				max = r.Gid
			}
		}
	case "link":
		rows, _ := a.DB.Ent.Link.Query().Where(link.AcodeEQ(acode)).All(bg)
		for _, r := range rows {
			if r.Gid > max {
				max = r.Gid
			}
		}
	}
	return max + 1
}

func collectGids(rows []map[string]string) []string {
	seen := map[string]bool{}
	var out []string
	for _, r := range rows {
		g := r["gid"]
		if g == "" || seen[g] {
			continue
		}
		seen[g] = true
		out = append(out, g)
	}
	if len(out) == 0 {
		out = []string{"1"}
	}
	return out
}

func slidePage(a *app.App, w http.ResponseWriter, r *http.Request, sess *auth.Session, acode string, kv map[string]string, action string) {
	simpleExtPage(a, w, r, sess, acode, kv, action, "ay_slide", "Slide", "content/slide.html", "slides", "slide")
}

func linkPage(a *app.App, w http.ResponseWriter, r *http.Request, sess *auth.Session, acode string, kv map[string]string, action string) {
	simpleExtPage(a, w, r, sess, acode, kv, action, "ay_link", "Link", "content/link.html", "links", "link")
}

func tagsPage(a *app.App, w http.ResponseWriter, r *http.Request, sess *auth.Session, acode string, kv map[string]string, action string) {
	simpleExtPage(a, w, r, sess, acode, kv, action, "ay_tags", "Tags", "content/tags.html", "tags", "tags")
}

func simpleExtPage(a *app.App, w http.ResponseWriter, r *http.Request, sess *auth.Session, acode string, kv map[string]string, action, table, ctrl, tpl, listKey, oneKey string) {
	back := "/admin/" + ctrl + "/index"
	switch action {
	case "add":
		if r.Method == http.MethodPost {
			if err := crudInsert(a, table, acode, sess.Values["username"], r); err != nil {
				sortFail(w, r, err.Error())
				return
			}
			front.ClearPages(a)
			sortOK(w, r, "新增成功！", back)
			return
		}
		http.Redirect(w, r, back, http.StatusFound)
		return
	case "del":
		if r.Method != http.MethodPost {
			sortFail(w, r, "请使用正确方式提交！")
			return
		}
		if err := crudDel(a, table, acode, or(kv["id"], r.FormValue("id"))); err != nil {
			sortFail(w, r, err.Error())
			return
		}
		front.ClearPages(a)
		sortOK(w, r, "删除成功！", back)
		return
	case "mod":
		if r.Method == http.MethodPost {
			if r.FormValue("submit") == "sorting" {
				simpleSorting(a, w, r, acode, table)
				return
			}
			id := or(kv["id"], r.FormValue("id"))
			if err := crudUpdate(a, table, acode, id, r); err != nil {
				sortFail(w, r, err.Error())
				return
			}
			front.ClearPages(a)
			sortOK(w, r, "修改成功！", back)
			return
		}
		simpleModForm(a, w, r, sess, acode, kv, table, ctrl, tpl, oneKey)
		return
	}
	if r.Method == http.MethodPost {
		if r.FormValue("op") == "del" {
			_ = crudDel(a, table, acode, r.FormValue("id"))
			front.ClearPages(a)
			http.Redirect(w, r, back, http.StatusFound)
			return
		}
		if r.FormValue("id") != "" {
			_ = crudUpdate(a, table, acode, r.FormValue("id"), r)
		} else {
			_ = crudInsert(a, table, acode, sess.Values["username"], r)
		}
		front.ClearPages(a)
		http.Redirect(w, r, back, http.StatusFound)
		return
	}
	if kv["id"] != "" {
		simpleModForm(a, w, r, sess, acode, kv, table, ctrl, tpl, oneKey)
		return
	}
	rows, _ := crudList(a, table, acode)
	d := simpleBag(a, w, r, sess, ctrl, rows)
	d.Vars["list"] = "1"
	d.Vars[listKey] = rows
	d.Vars["pagebar"] = ""
	writePHP(a, w, tpl, d)
}

func simpleModForm(a *app.App, w http.ResponseWriter, r *http.Request, sess *auth.Session, acode string, kv map[string]string, table, ctrl, tpl, oneKey string) {
	rows, _ := crudList(a, table, acode)
	id := kv["id"]
	var cur map[string]string
	for _, row := range rows {
		if row["id"] == id {
			cur = row
			break
		}
	}
	if cur == nil {
		http.Redirect(w, r, "/admin/"+ctrl+"/index", http.StatusFound)
		return
	}
	d := simpleBag(a, w, r, sess, ctrl, rows)
	d.Vars["mod"] = "1"
	d.Vars[oneKey] = cur
	// tags template uses $tags for one row
	if oneKey == "tags" {
		d.Vars["tags"] = cur
	}
	d.Get["id"] = id
	writePHP(a, w, tpl, d)
}

func simpleBag(a *app.App, w http.ResponseWriter, r *http.Request, sess *auth.Session, ctrl string, rows []map[string]string) Data {
	d := bag(a, w, r, sess)
	d.Vars["backurl"] = ""
	d.Vars["pagebar"] = ""
	d.Vars["gids"] = collectGids(rows)
	fc, _ := d.Vars["formcheck"].(string)
	d.Funs["admin_stay_input"] = func(args []string) string { return `<input type="hidden" name="stay" value="">` }
	d.Funs["get_btn_back"] = func(args []string) string {
		return `<a href="/admin/` + ctrl + `/index" class="layui-btn layui-btn-primary">返 回</a>`
	}
	d.Funs["get_btn_mod"] = func(args []string) string {
		return `<a href="/admin/` + ctrl + `/mod/id/` + argAt(args, 0) + `" class="layui-btn layui-btn-xs">修改</a>`
	}
	d.Funs["get_btn_del_post"] = func(args []string) string {
		return `<button type="button" class="layui-btn layui-btn-xs layui-btn-danger pboot-post" data-action="/admin/` + ctrl + `/del/id/` + argAt(args, 0) + `" data-after="reload" data-confirm="您确定要删除么？" data-formcheck="` + fc + `">删除</button>`
	}
	return d
}

func simpleSorting(a *app.App, w http.ResponseWriter, r *http.Request, acode, table string) {
	ids := r.Form["listall[]"]
	if len(ids) == 0 {
		ids = r.Form["listall"]
	}
	sortings := r.Form["sorting[]"]
	if len(sortings) == 0 {
		sortings = r.Form["sorting"]
	}
	out := make([]string, 0, len(ids))
	for i, id := range ids {
		sv := "255"
		if i < len(sortings) && sortings[i] != "" {
			sv = sortings[i]
		}
		n := domain.Atoi(id, 0)
		switch table {
		case "ay_slide":
			_, _ = a.DB.Ent.Slide.Update().Where(slide.IDEQ(n), slide.AcodeEQ(acode)).SetSorting(domain.Atoi(sv, 255)).Save(bg)
		case "ay_link":
			_, _ = a.DB.Ent.Link.Update().Where(link.IDEQ(n), link.AcodeEQ(acode)).SetSorting(domain.Atoi(sv, 255)).Save(bg)
		}
		out = append(out, sv)
	}
	front.JSON(w, 1, map[string]any{"sorting": out})
}
