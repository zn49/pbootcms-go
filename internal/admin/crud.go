package admin

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"entgo.io/ent/dialect/sql"
	entg "github.com/zn49/pbootcms-go/internal/ent"
	"github.com/zn49/pbootcms-go/internal/ent/content"
	"github.com/zn49/pbootcms-go/internal/ent/contentext"
	"github.com/zn49/pbootcms-go/internal/ent/contentsort"
	"github.com/zn49/pbootcms-go/internal/ent/extfield"
	"github.com/zn49/pbootcms-go/internal/ent/form"
	"github.com/zn49/pbootcms-go/internal/ent/formfield"
	"github.com/zn49/pbootcms-go/internal/ent/label"
	"github.com/zn49/pbootcms-go/internal/ent/menu"
	"github.com/zn49/pbootcms-go/internal/ent/message"
	"github.com/zn49/pbootcms-go/internal/ent/model"
	"github.com/zn49/pbootcms-go/internal/ent/site"
	"github.com/zn49/pbootcms-go/internal/ent/sysconfig"

	"github.com/zn49/pbootcms-go/internal/app"
	"github.com/zn49/pbootcms-go/internal/auth"
	"github.com/zn49/pbootcms-go/internal/domain"
	"github.com/zn49/pbootcms-go/internal/front"
	"github.com/zn49/pbootcms-go/internal/fsperm"
	"github.com/zn49/pbootcms-go/internal/store"
)

var bg = context.Background()

func sortPage(a *app.App, w http.ResponseWriter, r *http.Request, sess *auth.Session, acode string, kv map[string]string, action string) {
	switch action {
	case "add":
		if r.Method == http.MethodPost {
			sortAdd(a, w, r, sess, acode)
			return
		}
		http.Redirect(w, r, "/admin/ContentSort/index", http.StatusFound)
		return
	case "del":
		sortDel(a, w, r, acode, kv)
		return
	case "mod":
		if r.Method == http.MethodPost {
			sortMod(a, w, r, sess, acode, kv)
			return
		}
		sortModForm(a, w, r, sess, acode, kv)
		return
	}
	// index (and legacy POST create/update for older clients)
	if r.Method == http.MethodPost {
		if r.FormValue("id") != "" {
			kv["id"] = r.FormValue("id")
			sortMod(a, w, r, sess, acode, kv)
			return
		}
		sortAdd(a, w, r, sess, acode)
		return
	}
	d := sortBag(a, w, r, sess, acode, "", "")
	d.Vars["list"] = "1"
	writePHP(a, w, "content/contentsort.html", d)
}

func sortAdd(a *app.App, w http.ResponseWriter, r *http.Request, sess *auth.Session, acode string) {
	user := sess.Values["username"]
	now := domain.Now()
	pcode := or(r.FormValue("pcode"), "0")
	mcode := r.FormValue("mcode")
	typ := or(r.FormValue("type"), "1")
	status := or(r.FormValue("status"), "1")
	listtpl := filepath.Base(r.FormValue("listtpl"))
	contenttpl := filepath.Base(r.FormValue("contenttpl"))
	if listtpl == "." || listtpl == ".." {
		listtpl = ""
	}
	if contenttpl == "." || contenttpl == ".." {
		contenttpl = ""
	}
	if mcode == "" {
		sortFail(w, r, "栏目模型必须选择！")
		return
	}
	names := []string{}
	if multi := r.FormValue("multiplename"); multi != "" {
		multi = strings.ReplaceAll(multi, "，", ",")
		for _, n := range strings.Split(multi, ",") {
			n = strings.TrimSpace(n)
			if n != "" {
				names = append(names, n)
			}
		}
	} else if name := strings.TrimSpace(r.FormValue("name")); name != "" {
		names = append(names, name)
	}
	if len(names) == 0 {
		sortFail(w, r, "栏目名不能为空！")
		return
	}
	scode := nextScode(a)
	for _, name := range names {
		filename := strings.Trim(r.FormValue("filename"), "/")
		if len(names) > 1 {
			filename = ""
		}
		err := a.DB.Ent.ContentSort.Create().
			SetAcode(acode).SetMcode(mcode).SetPcode(pcode).SetScode(scode).
			SetName(name).SetListtpl(listtpl).SetContenttpl(contenttpl).SetStatus(status).
			SetOutlink(r.FormValue("outlink")).SetSubname(r.FormValue("subname")).
			SetDef1(r.FormValue("def1")).SetDef2(r.FormValue("def2")).SetDef3(r.FormValue("def3")).
			SetIco(r.FormValue("ico")).SetPic(r.FormValue("pic")).
			SetTitle(r.FormValue("title")).SetKeywords(r.FormValue("keywords")).SetDescription(r.FormValue("description")).
			SetFilename(filename).SetSorting(255).
			SetCreateUser(user).SetUpdateUser(user).SetCreateTime(now).SetUpdateTime(now).
			SetGtype(or(r.FormValue("gtype"), "4")).SetGid(r.FormValue("gid")).SetGnote(r.FormValue("gnote")).
			Exec(bg)
		if err != nil {
			sortFail(w, r, err.Error())
			return
		}
		if typ == "1" && r.FormValue("outlink") == "" && len(names) == 1 {
			_ = addSingleContent(a, sess, acode, scode, name)
		}
		if len(names) > 1 && typ == "1" {
			_ = addSingleContent(a, sess, acode, scode, name)
		}
		scode = incrScode(scode)
	}
	front.ClearPages(a)
	sortOK(w, r, "新增成功！", "/admin/ContentSort/index")
}

func sortMod(a *app.App, w http.ResponseWriter, r *http.Request, sess *auth.Session, acode string, kv map[string]string) {
	if r.FormValue("submit") == "sorting" {
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
			_, _ = a.DB.Ent.ContentSort.Update().Where(contentsort.IDEQ(domain.Atoi(id, 0)), contentsort.AcodeEQ(acode)).
				SetSorting(domain.Atoi(sv, 255)).SetUpdateTime(domain.Now()).Save(bg)
			out = append(out, sv)
		}
		front.JSON(w, 1, map[string]any{"sorting": out})
		return
	}
	scode := kv["scode"]
	if scode == "" && kv["id"] != "" {
		if one, err := a.DB.Ent.ContentSort.Get(bg, domain.Atoi(kv["id"], 0)); err == nil {
			scode = one.Scode
		}
	}
	if scode == "" {
		sortFail(w, r, "传递的参数值错误！")
		return
	}
	if f := r.FormValue("field"); f == "status" {
		_, err := a.DB.Ent.ContentSort.Update().Where(contentsort.ScodeEQ(scode), contentsort.AcodeEQ(acode)).
			SetStatus(r.FormValue("value")).SetUpdateUser(sess.Values["username"]).SetUpdateTime(domain.Now()).Save(bg)
		if err != nil {
			sortFail(w, r, "修改失败！")
			return
		}
		front.ClearPages(a)
		sortOK(w, r, "修改成功！", "/admin/ContentSort/index")
		return
	}
	if r.FormValue("name") == "" && r.FormValue("mcode") == "" {
		sortFail(w, r, "传递的参数值错误！")
		return
	}
	listtpl := filepath.Base(r.FormValue("listtpl"))
	contenttpl := filepath.Base(r.FormValue("contenttpl"))
	if listtpl == "." || listtpl == ".." {
		listtpl = ""
	}
	if contenttpl == "." || contenttpl == ".." {
		contenttpl = ""
	}
	_, err := a.DB.Ent.ContentSort.Update().Where(contentsort.ScodeEQ(scode), contentsort.AcodeEQ(acode)).
		SetName(r.FormValue("name")).SetFilename(strings.Trim(r.FormValue("filename"), "/")).
		SetPcode(or(r.FormValue("pcode"), "0")).SetMcode(r.FormValue("mcode")).
		SetListtpl(listtpl).SetContenttpl(contenttpl).
		SetStatus(or(r.FormValue("status"), "1")).SetSubname(r.FormValue("subname")).SetOutlink(r.FormValue("outlink")).
		SetDef1(r.FormValue("def1")).SetDef2(r.FormValue("def2")).SetDef3(r.FormValue("def3")).
		SetIco(r.FormValue("ico")).SetPic(r.FormValue("pic")).SetTitle(r.FormValue("title")).
		SetKeywords(r.FormValue("keywords")).SetDescription(r.FormValue("description")).
		SetGtype(or(r.FormValue("gtype"), "4")).SetGid(r.FormValue("gid")).SetGnote(r.FormValue("gnote")).
		SetUpdateUser(sess.Values["username"]).SetUpdateTime(domain.Now()).
		Save(bg)
	if err != nil {
		sortFail(w, r, err.Error())
		return
	}
	if r.FormValue("modsub") == "1" {
		for _, sub := range domain.Descendants(mustSorts(a, acode), scode) {
			if sub == scode {
				continue
			}
			_, _ = a.DB.Ent.ContentSort.Update().Where(contentsort.ScodeEQ(sub), contentsort.AcodeEQ(acode)).
				SetMcode(r.FormValue("mcode")).SetListtpl(listtpl).SetContenttpl(contenttpl).Save(bg)
		}
	}
	front.ClearPages(a)
	sortOK(w, r, "修改成功！", "/admin/ContentSort/index")
}

func sortDel(a *app.App, w http.ResponseWriter, r *http.Request, acode string, kv map[string]string) {
	if r.Method != http.MethodPost {
		sortFail(w, r, "请使用正确方式提交！")
		return
	}
	rows := mustSorts(a, acode)
	var codes []string
	if list := r.Form["list[]"]; len(list) > 0 {
		for _, c := range list {
			codes = append(codes, domain.Descendants(rows, c)...)
		}
	} else if list := r.Form["list"]; len(list) > 0 {
		for _, c := range list {
			codes = append(codes, domain.Descendants(rows, c)...)
		}
	} else if scode := kv["scode"]; scode != "" {
		codes = domain.Descendants(rows, scode)
	}
	if len(codes) == 0 {
		sortFail(w, r, "请选择要删除的栏目！")
		return
	}
	seen := map[string]bool{}
	var uniq []string
	for _, c := range codes {
		if !seen[c] {
			seen[c] = true
			uniq = append(uniq, c)
		}
	}
	_, _ = a.DB.Ent.Content.Delete().Where(content.ScodeIn(uniq...)).Exec(bg)
	_, err := a.DB.Ent.ContentSort.Delete().Where(contentsort.AcodeEQ(acode), contentsort.ScodeIn(uniq...)).Exec(bg)
	if err != nil {
		sortFail(w, r, "删除失败！")
		return
	}
	front.ClearPages(a)
	sortOK(w, r, "删除成功！", "/admin/ContentSort/index")
}

func sortModForm(a *app.App, w http.ResponseWriter, r *http.Request, sess *auth.Session, acode string, kv map[string]string) {
	scode := kv["scode"]
	if scode == "" && kv["id"] != "" {
		if one, err := a.DB.Ent.ContentSort.Get(bg, domain.Atoi(kv["id"], 0)); err == nil {
			scode = one.Scode
		}
	}
	if scode == "" {
		http.Redirect(w, r, "/admin/ContentSort/index", http.StatusFound)
		return
	}
	d := sortBag(a, w, r, sess, acode, scode, scode)
	one, err := a.DB.Ent.ContentSort.Query().Where(contentsort.ScodeEQ(scode), contentsort.AcodeEQ(acode)).First(bg)
	if err != nil {
		http.Redirect(w, r, "/admin/ContentSort/index", http.StatusFound)
		return
	}
	row := store.Row(one)
	if m, err := a.DB.Ent.Model.Query().Where(model.McodeEQ(one.Mcode)).First(bg); err == nil {
		row["type"] = m.Type
	}
	d.Vars["mod"] = "1"
	d.Vars["sort"] = row
	d.Get["scode"] = scode
	writePHP(a, w, "content/contentsort.html", d)
}

func sortBag(a *app.App, w http.ResponseWriter, r *http.Request, sess *auth.Session, acode, selectScode, excludeScode string) Data {
	d := bag(a, w, r, sess)
	models := store.Rows(must(a.DB.Ent.Model.Query().Order(model.ByID()).All(bg)))
	active := []map[string]string{}
	for _, m := range models {
		if m["status"] == "1" || m["status"] == "" {
			active = append(active, m)
		}
	}
	rows := mustSorts(a, acode)
	byM := map[string]map[string]string{}
	for _, m := range models {
		byM[m["mcode"]] = m
	}
	kids := map[string][]map[string]string{}
	for _, row := range rows {
		if m := byM[row["mcode"]]; m != nil {
			row["type"] = m["type"]
			row["urlname"] = m["urlname"]
		}
		kids[row["pcode"]] = append(kids[row["pcode"]], row)
	}
	for _, row := range rows {
		if len(kids[row["scode"]]) > 0 {
			row["son"] = "1"
		} else {
			row["son"] = ""
		}
	}
	flat := flattenSorts(kids, "0")
	d.Vars["sorts"] = flat
	d.Vars["allmodels"] = models
	d.Vars["models"] = active
	d.Vars["groups"] = memberGroups(a)
	d.Vars["tpls"] = themeTpls(a, acode)
	d.Vars["sort_select"] = sortSelectHTML(kids, "0", "", selectScode, excludeScode)
	d.Vars["backurl"] = ""
	fc, _ := d.Vars["formcheck"].(string)
	d.Funs["admin_stay_input"] = func(args []string) string {
		return `<input type="hidden" name="stay" value="">`
	}
	d.Funs["get_btn_back"] = func(args []string) string {
		return `<a href="/admin/ContentSort/index" class="layui-btn layui-btn-primary">返 回</a>`
	}
	d.Funs["get_btn_mod"] = func(args []string) string {
		id, field := argAt(args, 0), or(argAt(args, 1), "id")
		return `<a href="/admin/ContentSort/mod/` + field + `/` + id + `" class="layui-btn layui-btn-xs">修改</a>`
	}
	d.Funs["get_btn_del_post"] = func(args []string) string {
		id, field := argAt(args, 0), or(argAt(args, 1), "id")
		return `<button type="button" class="layui-btn layui-btn-xs layui-btn-danger pboot-post" data-action="/admin/ContentSort/del/` + field + `/` + id + `" data-after="reload" data-confirm="您确定要删除么？" data-formcheck="` + fc + `">删除</button>`
	}
	d.Funs["admin_switch_btn"] = func(args []string) string {
		id, field, value, on, idName := argAt(args, 0), argAt(args, 1), argAt(args, 2), argAt(args, 3), or(argAt(args, 4), "id")
		icon := "fa-toggle-off"
		if on == "1" {
			icon = "fa-toggle-on"
		}
		return `<button type="button" class="pboot-post" data-action="/admin/ContentSort/mod/` + idName + `/` + id + `" data-after="reload" data-formcheck="` + fc + `" data-field="` + field + `" data-value="` + value + `" style="background:none;border:0;padding:0;cursor:pointer;color:inherit"><i class="fa ` + icon + `"></i></button>`
	}
	return d
}

func must(rows []*entg.Model, err error) []*entg.Model {
	if err != nil {
		return nil
	}
	return rows
}

func mustSorts(a *app.App, acode string) []map[string]string {
	rows, _ := domain.Sorts(a.DB, acode)
	return rows
}

func flattenSorts(kids map[string][]map[string]string, pcode string) []map[string]string {
	var out []map[string]string
	for _, row := range kids[pcode] {
		out = append(out, row)
		out = append(out, flattenSorts(kids, row["scode"])...)
	}
	return out
}

func sortSelectHTML(kids map[string][]map[string]string, pcode, blank, selected, exclude string) string {
	var b strings.Builder
	for _, row := range kids[pcode] {
		if row["scode"] == exclude {
			continue
		}
		sel := ""
		if row["scode"] == selected {
			sel = " selected"
		}
		b.WriteString("<option value='" + esc(row["scode"]) + "'" + sel + ">" + blank + esc(row["name"]) + "</option>")
		b.WriteString(sortSelectHTML(kids, row["scode"], blank+"　　", selected, exclude))
	}
	return b.String()
}

func themeTpls(a *app.App, acode string) []string {
	theme := "default"
	if one, err := a.DB.Ent.Site.Query().Where(site.AcodeEQ(acode)).First(bg); err == nil && one.Theme != "" {
		theme = one.Theme
	}
	dir := filepath.Join(a.Cfg.SiteRoot, "template", theme)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(strings.ToLower(e.Name()), ".html") {
			out = append(out, e.Name())
		}
	}
	return out
}

func nextScode(a *app.App) string {
	return incrScode(maxScode(a))
}

func incrScode(scode string) string {
	n := domain.Atoi(scode, 0)
	return strconv.Itoa(n + 1)
}

func addSingleContent(a *app.App, sess *auth.Session, acode, scode, title string) error {
	user := sess.Values["username"]
	now := domain.Now()
	return a.DB.Ent.Content.Create().
		SetAcode(acode).SetScode(scode).SetSubscode("").SetTitle(title).SetTitlecolor("#333333").
		SetSubtitle("").SetFilename("").SetAuthor(or(sess.Values["realname"], user)).SetSource("本站").
		SetOutlink("").SetDate(now).SetIco("").SetPics("").SetPicstitle("").SetContent("").
		SetTags("").SetEnclosure("").SetKeywords("").SetDescription("").SetSorting(255).
		SetStatus("1").SetIstop("0").SetIsrecommend("0").SetIsheadline("0").
		SetGid("").SetGtype("4").SetGnote("").SetVisits(0).SetLikes(0).SetOppose(0).
		SetCreateUser(user).SetUpdateUser(user).SetCreateTime(now).SetUpdateTime(now).
		Exec(bg)
}

func sortOK(w http.ResponseWriter, r *http.Request, msg, loc string) {
	if wantsJSON(r) {
		front.JSON(w, 1, msg)
		return
	}
	http.Redirect(w, r, loc, http.StatusFound)
}

func sortFail(w http.ResponseWriter, r *http.Request, msg string) {
	if wantsJSON(r) {
		front.JSON(w, 0, msg)
		return
	}
	alert(w, msg)
}

func argAt(args []string, i int) string {
	if i < len(args) {
		return args[i]
	}
	return ""
}

func baseTpl(s string) string {
	s = filepath.Base(s)
	if s == "." || s == ".." {
		return ""
	}
	return s
}

func contentPage(a *app.App, w http.ResponseWriter, r *http.Request, sess *auth.Session, acode string, kv map[string]string, single bool, action string) {
	ctrl := "Content"
	tpl := "content/content.html"
	if single {
		ctrl = "Single"
		tpl = "content/single.html"
	}
	switch action {
	case "add":
		if r.Method == http.MethodPost {
			saveContent(a, w, r, sess, acode, kv, single)
			return
		}
		http.Redirect(w, r, "/admin/"+ctrl+"/index", http.StatusFound)
		return
	case "del":
		contentDel(a, w, r, acode, kv, single)
		return
	case "mod":
		for _, kind := range []string{"baiduzz", "baiduks", "indexnow"} {
			if id := kv[kind]; id != "" {
				contentPushOne(a, w, r, sess, kind, id)
				return
			}
		}
		if r.Method == http.MethodPost {
			if r.FormValue("submit") == "sorting" {
				contentSorting(a, w, r, acode)
				return
			}
			if f := r.FormValue("field"); f != "" {
				contentField(a, w, r, sess, acode, kv, f)
				return
			}
			saveContent(a, w, r, sess, acode, kv, single)
			return
		}
		contentModForm(a, w, r, sess, acode, kv, single, tpl, ctrl)
		return
	}
	if r.Method == http.MethodPost {
		switch r.FormValue("submit") {
		case "baiduzz", "baiduks", "indexnow":
			contentPushBatch(a, w, r, sess, r.FormValue("submit"))
			return
		}
		saveContent(a, w, r, sess, acode, kv, single)
		return
	}
	if kv["del"] != "" {
		delContent(a, w, r, kv["del"])
		return
	}
	if kv["id"] != "" {
		contentModForm(a, w, r, sess, acode, kv, single, tpl, ctrl)
		return
	}
	mcode := kv["mcode"]
	d := contentBag(a, w, r, sess, acode, mcode, single, ctrl)
	d.Vars["list"] = "1"
	rows := contentRows(a, acode, mcode, single, kv)
	d.Vars["contents"] = rows
	d.Vars["model_name"] = modelName(a, mcode, single)
	writePHP(a, w, tpl, d)
}

func contentModForm(a *app.App, w http.ResponseWriter, r *http.Request, sess *auth.Session, acode string, kv map[string]string, single bool, tpl, ctrl string) {
	id := domain.Atoi(kv["id"], 0)
	one, err := a.DB.Ent.Content.Get(bg, id)
	if err != nil {
		http.Redirect(w, r, "/admin/"+ctrl+"/index", http.StatusFound)
		return
	}
	cur := store.Row(one)
	if ext, err := a.DB.ExtByContent([]int{one.ID}); err == nil {
		for k, v := range ext[cur["id"]] {
			if strings.HasPrefix(k, "ext_") {
				cur[k] = v
			}
		}
	}
	mcode := kv["mcode"]
	if mcode == "" {
		if s, err := a.DB.Ent.ContentSort.Query().Where(contentsort.ScodeEQ(one.Scode), contentsort.AcodeEQ(acode)).First(bg); err == nil {
			mcode = s.Mcode
		}
	}
	d := contentBag(a, w, r, sess, acode, mcode, single, ctrl)
	d.Vars["mod"] = "1"
	d.Vars["content"] = cur
	d.Vars["model_name"] = modelName(a, mcode, single)
	d.Get["id"] = strconv.Itoa(id)
	if mcode != "" {
		d.Get["mcode"] = mcode
	}
	// preselect sort
	d.Vars["sort_select"] = contentSortSelect(a, acode, mcode, single, one.Scode)
	writePHP(a, w, tpl, d)
}

func contentBag(a *app.App, w http.ResponseWriter, r *http.Request, sess *auth.Session, acode, mcode string, single bool, ctrl string) Data {
	d := bag(a, w, r, sess)
	if mcode != "" {
		d.Get["mcode"] = mcode
	}
	d.Vars["C"] = ctrl
	d.Vars["pathinfo"] = ""
	d.Vars["btnqs"] = ""
	d.Vars["backurl"] = ""
	d.Vars["pagebar"] = ""
	d.Vars["baidu_zz_token"] = a.DB.Conf("baidu_zz_token")
	d.Vars["baidu_ks_token"] = a.DB.Conf("baidu_ks_token")
	d.Vars["indexnow_key"] = effectiveIndexnowKey(a)
	if d.Vars["indexnow_key"] == "" {
		d.Vars["indexnow_key"] = a.DB.Conf("indexnow_key")
	}
	d.Vars["url_break_char"] = or(a.DB.Conf("url_break_char"), "_")
	d.Vars["groups"] = memberGroups(a)
	d.Vars["ai_show_assistant"] = ""
	sel := contentSortSelect(a, acode, mcode, single, "")
	d.Vars["search_select"] = contentSortSelect(a, acode, mcode, single, d.Get["scode"])
	d.Vars["sort_select"] = sel
	d.Vars["subsort_select"] = sel
	d.Vars["extfield"] = extFields(a, mcode)
	fc, _ := d.Vars["formcheck"].(string)
	d.Funs["admin_stay_input"] = func(args []string) string { return `<input type="hidden" name="stay" value="">` }
	d.Funs["get_btn_back"] = func(args []string) string {
		href := "/admin/" + ctrl + "/index"
		if mcode != "" {
			href += "/mcode/" + mcode
		}
		return `<a href="` + href + `" class="layui-btn layui-btn-primary">返 回</a>`
	}
	d.Funs["get_btn_del_post"] = func(args []string) string {
		return `<button type="button" class="layui-btn layui-btn-xs layui-btn-danger pboot-post" data-action="/admin/` + ctrl + `/del/id/` + argAt(args, 0) + `" data-after="reload" data-confirm="您确定要删除么？" data-formcheck="` + fc + `">删除</button>`
	}
	d.Funs["admin_switch_btn"] = func(args []string) string {
		id, field, value, on, idName := argAt(args, 0), argAt(args, 1), argAt(args, 2), argAt(args, 3), or(argAt(args, 4), "id")
		icon := "fa-toggle-off"
		if on == "1" {
			icon = "fa-toggle-on"
		}
		return `<button type="button" class="pboot-post" data-action="/admin/` + ctrl + `/mod/` + idName + `/` + id + `" data-after="reload" data-formcheck="` + fc + `" data-field="` + field + `" data-value="` + value + `" style="background:none;border:0;padding:0;cursor:pointer;color:inherit"><i class="fa ` + icon + `"></i></button>`
	}
	d.Funs["substr_both"] = func(args []string) string {
		s := argAt(args, 0)
		start := domain.Atoi(argAt(args, 1), 0)
		n := domain.Atoi(argAt(args, 2), 15)
		rs := []rune(s)
		if start >= len(rs) {
			return ""
		}
		end := start + n
		if end > len(rs) {
			end = len(rs)
		}
		out := string(rs[start:end])
		if end < len(rs) {
			out += "…"
		}
		return esc(out)
	}
	d.Funs["date"] = func(args []string) string {
		layout := argAt(args, 0)
		if layout == "" {
			layout = "2006-01-02 15:04:05"
		}
		// PHP date format → Go, only the common admin case
		if layout == "Y-m-d H:i:s" {
			return time.Now().Format("2006-01-02 15:04:05")
		}
		return time.Now().Format("2006-01-02 15:04:05")
	}
	d.Funs["ueditor_catch_remote_enabled"] = func(args []string) string {
		if a.DB.Conf("ueditor_catch_remote") == "0" {
			return ""
		}
		return "1"
	}
	d.Funs["check_level"] = func(args []string) string {
		if sess != nil && (sess.Values["id"] == "1" || argAt(args, 0) == "mod") {
			return "1"
		}
		return ""
	}
	return d
}

func contentRows(a *app.App, acode, mcode string, single bool, kv map[string]string) []map[string]string {
	q := a.DB.Ent.Content.Query().Where(content.AcodeEQ(acode)).Order(content.ByID(sql.OrderDesc())).Limit(100)
	sorts, _ := a.DB.Ent.ContentSort.Query().Where(contentsort.AcodeEQ(acode)).All(bg)
	sortBy := map[string]*entg.ContentSort{}
	models, _ := a.DB.Ent.Model.Query().All(bg)
	modelBy := map[string]*entg.Model{}
	for _, m := range models {
		modelBy[m.Mcode] = m
	}
	for _, s := range sorts {
		sortBy[s.Scode] = s
	}
	var scodes []string
	if single {
		allow := map[string]bool{}
		for _, m := range models {
			if m.Type == "1" && (mcode == "" || m.Mcode == mcode) {
				allow[m.Mcode] = true
			}
		}
		for _, s := range sorts {
			if allow[s.Mcode] {
				scodes = append(scodes, s.Scode)
			}
		}
	} else if mcode != "" {
		for _, s := range sorts {
			if s.Mcode == mcode {
				scodes = append(scodes, s.Scode)
			}
		}
	}
	if kv["scode"] != "" {
		scodes = []string{kv["scode"]}
	}
	if len(scodes) > 0 {
		q = q.Where(content.ScodeIn(scodes...))
	}
	if kw := kv["keyword"]; kw != "" {
		q = q.Where(content.TitleContains(kw))
	}
	list, _ := q.All(bg)
	var rows []map[string]string
	rules := front.Rules(a.DB)
	for _, c := range list {
		m := store.Row(c)
		urlname, modeltype := "", ""
		if s := sortBy[c.Scode]; s != nil {
			m["sortname"] = s.Name
			m["sortfilename"] = s.Filename
			m["mcode"] = s.Mcode
			if mod := modelBy[s.Mcode]; mod != nil {
				urlname = mod.Urlname
				modeltype = mod.Type
				m["urlname"] = urlname
				m["modeltype"] = modeltype
			}
		}
		if m["outlink"] == "" {
			if modeltype == "1" {
				m["link"] = rules.Sort(m["sortfilename"], urlname, c.Scode)
			} else {
				m["link"] = rules.Content(m["sortfilename"], urlname, c.Scode, m["id"], m["filename"])
			}
		}
		rows = append(rows, m)
	}
	return rows
}

func contentSortSelect(a *app.App, acode, mcode string, single bool, selected string) string {
	rows := mustSorts(a, acode)
	models, _ := a.DB.Ent.Model.Query().All(bg)
	allow := map[string]bool{}
	for _, m := range models {
		if single && m.Type == "1" {
			if mcode == "" || m.Mcode == mcode {
				allow[m.Mcode] = true
			}
		} else if !single && m.Type == "2" {
			if mcode == "" || m.Mcode == mcode {
				allow[m.Mcode] = true
			}
		}
	}
	kids := map[string][]map[string]string{}
	for _, row := range rows {
		if !allow[row["mcode"]] {
			continue
		}
		kids[row["pcode"]] = append(kids[row["pcode"]], row)
	}
	return sortSelectHTML(kids, "0", "", selected, "")
}

func extFields(a *app.App, mcode string) []map[string]string {
	q := a.DB.Ent.ExtField.Query().Order(extfield.BySorting(), extfield.ByID())
	if mcode != "" {
		q = q.Where(extfield.McodeEQ(mcode))
	}
	list, _ := q.All(bg)
	return store.Rows(list)
}

func modelName(a *app.App, mcode string, single bool) string {
	if mcode != "" {
		if m, err := a.DB.Ent.Model.Query().Where(model.McodeEQ(mcode)).First(bg); err == nil {
			return m.Name
		}
	}
	if single {
		return "单页"
	}
	return "内容"
}

func contentSorting(a *app.App, w http.ResponseWriter, r *http.Request, acode string) {
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
		_, _ = a.DB.Ent.Content.Update().Where(content.IDEQ(domain.Atoi(id, 0)), content.AcodeEQ(acode)).
			SetSorting(domain.Atoi(sv, 255)).SetUpdateTime(domain.Now()).Save(bg)
		out = append(out, sv)
	}
	front.JSON(w, 1, map[string]any{"sorting": out})
}

func contentField(a *app.App, w http.ResponseWriter, r *http.Request, sess *auth.Session, acode string, kv map[string]string, field string) {
	id := domain.Atoi(kv["id"], 0)
	val := r.FormValue("value")
	u := a.DB.Ent.Content.Update().Where(content.IDEQ(id), content.AcodeEQ(acode)).SetUpdateUser(sess.Values["username"]).SetUpdateTime(domain.Now())
	switch field {
	case "status":
		u = u.SetStatus(val)
	case "istop":
		u = u.SetIstop(val)
	case "isrecommend":
		u = u.SetIsrecommend(val)
	case "isheadline":
		u = u.SetIsheadline(val)
	default:
		sortFail(w, r, "不支持的字段")
		return
	}
	if _, err := u.Save(bg); err != nil {
		sortFail(w, r, "修改失败！")
		return
	}
	front.ClearPages(a)
	sortOK(w, r, "修改成功！", r.URL.Path)
}

func contentDel(a *app.App, w http.ResponseWriter, r *http.Request, acode string, kv map[string]string, single bool) {
	if r.Method != http.MethodPost {
		sortFail(w, r, "请使用正确方式提交！")
		return
	}
	var ids []int
	if list := r.Form["list[]"]; len(list) > 0 {
		for _, x := range list {
			ids = append(ids, domain.Atoi(x, 0))
		}
	} else if list := r.Form["list"]; len(list) > 0 {
		for _, x := range list {
			ids = append(ids, domain.Atoi(x, 0))
		}
	} else if kv["id"] != "" {
		ids = append(ids, domain.Atoi(kv["id"], 0))
	}
	if len(ids) == 0 {
		sortFail(w, r, "请选择要删除的内容！")
		return
	}
	_, _ = a.DB.Ent.Content.Delete().Where(content.IDIn(ids...), content.AcodeEQ(acode)).Exec(bg)
	for _, id := range ids {
		_, _ = a.DB.Ent.ContentExt.Delete().Where(contentext.ContentidEQ(id)).Exec(bg)
	}
	front.ClearPages(a)
	back := "/admin/Content/index"
	if single {
		back = "/admin/Single/index"
	}
	sortOK(w, r, "删除成功！", back)
}

func saveContent(a *app.App, w http.ResponseWriter, r *http.Request, sess *auth.Session, acode string, kv map[string]string, single bool) {
	id := r.FormValue("id")
	if id == "" {
		id = kv["id"]
	}
	now := domain.Now()
	user := sess.Values["username"]
	cid := 0
	pics := r.FormValue("pics")
	if pics == "" {
		pics = strings.Join(r.Form["pics"], ",")
	}
	picstitle := r.FormValue("picstitle")
	if picstitle == "" {
		picstitle = strings.Join(r.Form["picstitle"], ",")
	}
	if id == "" {
		row, err := a.DB.Ent.Content.Create().
			SetAcode(acode).SetScode(r.FormValue("scode")).SetSubscode(r.FormValue("subscode")).SetTitle(r.FormValue("title")).
			SetTitlecolor(or(r.FormValue("titlecolor"), "#333333")).SetSubtitle(r.FormValue("subtitle")).
			SetFilename(r.FormValue("filename")).SetAuthor(or(r.FormValue("author"), user)).SetSource(or(r.FormValue("source"), "本站")).
			SetOutlink(r.FormValue("outlink")).
			SetDate(or(r.FormValue("date"), now)).SetIco(r.FormValue("ico")).SetPics(pics).SetPicstitle(picstitle).SetContent(r.FormValue("content")).
			SetTags(strings.ReplaceAll(r.FormValue("tags"), "，", ",")).SetEnclosure(r.FormValue("enclosure")).
			SetKeywords(r.FormValue("keywords")).SetDescription(r.FormValue("description")).
			SetSorting(domain.Atoi(r.FormValue("sorting"), 255)).SetStatus(or(r.FormValue("status"), "1")).
			SetIstop(yn(r, "istop")).SetIsrecommend(yn(r, "isrecommend")).SetIsheadline(yn(r, "isheadline")).
			SetVisits(0).SetLikes(0).SetOppose(0).
			SetCreateUser(user).SetUpdateUser(user).SetCreateTime(now).SetUpdateTime(now).
			SetGtype(or(r.FormValue("gtype"), "4")).SetGid(r.FormValue("gid")).SetGnote(r.FormValue("gnote")).
			Save(bg)
		if err != nil {
			sortFail(w, r, err.Error())
			return
		}
		cid = row.ID
	} else {
		cid = domain.Atoi(id, 0)
		if _, err := a.DB.Ent.Content.Update().
			Where(content.IDEQ(cid), content.AcodeEQ(acode)).
			SetScode(r.FormValue("scode")).SetSubscode(r.FormValue("subscode")).SetTitle(r.FormValue("title")).
			SetTitlecolor(or(r.FormValue("titlecolor"), "#333333")).SetSubtitle(r.FormValue("subtitle")).
			SetFilename(r.FormValue("filename")).SetAuthor(r.FormValue("author")).SetSource(r.FormValue("source")).
			SetOutlink(r.FormValue("outlink")).
			SetDate(r.FormValue("date")).SetIco(r.FormValue("ico")).SetPics(pics).SetPicstitle(picstitle).
			SetContent(r.FormValue("content")).SetTags(strings.ReplaceAll(r.FormValue("tags"), "，", ",")).
			SetEnclosure(r.FormValue("enclosure")).
			SetKeywords(r.FormValue("keywords")).SetDescription(r.FormValue("description")).
			SetSorting(domain.Atoi(r.FormValue("sorting"), 255)).SetStatus(or(r.FormValue("status"), "1")).
			SetIstop(yn(r, "istop")).SetIsrecommend(yn(r, "isrecommend")).SetIsheadline(yn(r, "isheadline")).
			SetGtype(or(r.FormValue("gtype"), "4")).SetGid(r.FormValue("gid")).SetGnote(r.FormValue("gnote")).
			SetUpdateUser(user).SetUpdateTime(now).
			Save(bg); err != nil {
			sortFail(w, r, err.Error())
			return
		}
	}
	if err := saveExt(a, r, cid); err != nil {
		sortFail(w, r, err.Error())
		return
	}
	front.ClearPages(a)
	back := r.URL.Path
	if mcode := kv["mcode"]; mcode != "" && !strings.Contains(back, "mcode") {
		if single {
			back = "/admin/Single/index/mcode/" + mcode
		} else {
			back = "/admin/Content/index/mcode/" + mcode
		}
	}
	sortOK(w, r, "保存成功！", back)
}

func yn(r *http.Request, name string) string {
	if r.FormValue(name) == "1" {
		return "1"
	}
	return "0"
}

func saveExt(a *app.App, r *http.Request, id int) error {
	if id == 0 {
		return nil
	}
	fields, err := a.DB.Ent.ExtField.Query().All(bg)
	if err != nil {
		return err
	}
	allow := map[string]bool{}
	for _, f := range fields {
		allow[f.Name] = true
	}
	cols := map[string]string{}
	for k, vs := range r.PostForm {
		if allow[k] && strings.HasPrefix(k, "ext_") && store.Ident(k) {
			cols[k] = strings.Join(vs, ",")
		}
	}
	return a.DB.UpsertExt(id, cols)
}

func delContent(a *app.App, w http.ResponseWriter, r *http.Request, id string) {
	n := domain.Atoi(id, 0)
	_, _ = a.DB.Ent.Content.Delete().Where(content.IDEQ(n)).Exec(bg)
	_, _ = a.DB.Ent.ContentExt.Delete().Where(contentext.ContentidEQ(n)).Exec(bg)
	front.ClearPages(a)
	http.Redirect(w, r, "/admin/Content/index", http.StatusFound)
}

func modelPage(a *app.App, w http.ResponseWriter, r *http.Request, sess *auth.Session, kv map[string]string, action string) {
	switch action {
	case "add":
		if r.Method == http.MethodPost {
			modelAdd(a, w, r, sess)
			return
		}
		http.Redirect(w, r, "/admin/Model/index", http.StatusFound)
		return
	case "del":
		modelDel(a, w, r, kv)
		return
	case "mod":
		if r.Method == http.MethodPost {
			modelMod(a, w, r, sess, kv)
			return
		}
		modelModForm(a, w, r, sess, kv)
		return
	}
	if r.Method == http.MethodPost {
		if r.FormValue("op") == "del" {
			kv["id"] = r.FormValue("id")
			modelDel(a, w, r, kv)
			return
		}
		if r.FormValue("id") != "" {
			kv["id"] = r.FormValue("id")
			modelMod(a, w, r, sess, kv)
			return
		}
		modelAdd(a, w, r, sess)
		return
	}
	d := modelBag(a, w, r, sess)
	d.Vars["list"] = "1"
	rows, _ := domain.Models(a.DB)
	d.Vars["models"] = rows
	d.Vars["pagebar"] = ""
	writePHP(a, w, "content/model.html", d)
}

func modelAdd(a *app.App, w http.ResponseWriter, r *http.Request, sess *auth.Session) {
	now := domain.Now()
	user := sess.Values["username"]
	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		sortFail(w, r, "模型名称不能为空！")
		return
	}
	typ := or(r.FormValue("type"), "2")
	urlname := r.FormValue("urlname")
	if urlname == "" {
		if typ == "1" {
			urlname = "about"
		} else {
			urlname = "list"
		}
	}
	mcode := strconv.Itoa(domain.Atoi(maxCode(a, "ay_model", "mcode"), 1) + 1)
	err := a.DB.Ent.Model.Create().
		SetMcode(mcode).SetName(name).SetType(typ).SetUrlname(urlname).
		SetListtpl(baseTpl(r.FormValue("listtpl"))).SetContenttpl(baseTpl(r.FormValue("contenttpl"))).
		SetStatus(or(r.FormValue("status"), "1")).SetIssystem("0").
		SetCreateUser(user).SetUpdateUser(user).SetCreateTime(now).SetUpdateTime(now).
		Exec(bg)
	if err != nil {
		sortFail(w, r, err.Error())
		return
	}
	sortOK(w, r, "新增成功！", "/admin/Model/index")
}

func modelMod(a *app.App, w http.ResponseWriter, r *http.Request, sess *auth.Session, kv map[string]string) {
	id := domain.Atoi(kv["id"], 0)
	if id == 0 {
		sortFail(w, r, "传递的参数值错误！")
		return
	}
	if f := r.FormValue("field"); f == "status" {
		_, err := a.DB.Ent.Model.Update().Where(model.IDEQ(id)).
			SetStatus(r.FormValue("value")).SetUpdateUser(sess.Values["username"]).SetUpdateTime(domain.Now()).Save(bg)
		if err != nil {
			sortFail(w, r, "修改失败！")
			return
		}
		sortOK(w, r, "修改成功！", "/admin/Model/index")
		return
	}
	typ := or(r.FormValue("type"), "2")
	urlname := r.FormValue("urlname")
	if urlname == "" {
		if typ == "1" {
			urlname = "about"
		} else {
			urlname = "list"
		}
	}
	_, err := a.DB.Ent.Model.Update().Where(model.IDEQ(id)).
		SetName(r.FormValue("name")).SetType(typ).SetUrlname(urlname).
		SetListtpl(baseTpl(r.FormValue("listtpl"))).SetContenttpl(baseTpl(r.FormValue("contenttpl"))).
		SetStatus(or(r.FormValue("status"), "1")).
		SetUpdateUser(sess.Values["username"]).SetUpdateTime(domain.Now()).Save(bg)
	if err != nil {
		sortFail(w, r, err.Error())
		return
	}
	sortOK(w, r, "修改成功！", "/admin/Model/index")
}

func modelDel(a *app.App, w http.ResponseWriter, r *http.Request, kv map[string]string) {
	if r.Method != http.MethodPost {
		sortFail(w, r, "请使用正确方式提交！")
		return
	}
	id := domain.Atoi(kv["id"], 0)
	if id == 0 {
		id = domain.Atoi(r.FormValue("id"), 0)
	}
	_, err := a.DB.Ent.Model.Delete().Where(model.IDEQ(id), model.IssystemNEQ("1")).Exec(bg)
	if err != nil {
		sortFail(w, r, "删除失败！")
		return
	}
	sortOK(w, r, "删除成功！", "/admin/Model/index")
}

func modelModForm(a *app.App, w http.ResponseWriter, r *http.Request, sess *auth.Session, kv map[string]string) {
	id := domain.Atoi(kv["id"], 0)
	one, err := a.DB.Ent.Model.Get(bg, id)
	if err != nil {
		http.Redirect(w, r, "/admin/Model/index", http.StatusFound)
		return
	}
	d := modelBag(a, w, r, sess)
	d.Vars["mod"] = "1"
	d.Vars["model"] = store.Row(one)
	d.Get["id"] = strconv.Itoa(id)
	writePHP(a, w, "content/model.html", d)
}

func modelBag(a *app.App, w http.ResponseWriter, r *http.Request, sess *auth.Session) Data {
	d := bag(a, w, r, sess)
	d.Vars["backurl"] = ""
	d.Vars["pagebar"] = ""
	fc, _ := d.Vars["formcheck"].(string)
	d.Funs["get_btn_back"] = func(args []string) string {
		return `<a href="/admin/Model/index" class="layui-btn layui-btn-primary">返 回</a>`
	}
	d.Funs["get_btn_mod"] = func(args []string) string {
		return `<a href="/admin/Model/mod/id/` + argAt(args, 0) + `" class="layui-btn layui-btn-xs">修改</a>`
	}
	d.Funs["get_btn_del_post"] = func(args []string) string {
		return `<button type="button" class="layui-btn layui-btn-xs layui-btn-danger pboot-post" data-action="/admin/Model/del/id/` + argAt(args, 0) + `" data-after="reload" data-confirm="您确定要删除么？" data-formcheck="` + fc + `">删除</button>`
	}
	d.Funs["admin_switch_btn"] = func(args []string) string {
		id, field, value, on, idName := argAt(args, 0), argAt(args, 1), argAt(args, 2), argAt(args, 3), or(argAt(args, 4), "id")
		icon := "fa-toggle-off"
		if on == "1" {
			icon = "fa-toggle-on"
		}
		return `<button type="button" class="pboot-post" data-action="/admin/Model/mod/` + idName + `/` + id + `" data-after="reload" data-formcheck="` + fc + `" data-field="` + field + `" data-value="` + value + `" style="background:none;border:0;padding:0;cursor:pointer;color:inherit"><i class="fa ` + icon + `"></i></button>`
	}
	return d
}

func extPage(a *app.App, w http.ResponseWriter, r *http.Request, sess *auth.Session, kv map[string]string, action string) {
	switch action {
	case "add":
		if r.Method == http.MethodPost {
			extAdd(a, w, r)
			return
		}
		http.Redirect(w, r, "/admin/ExtField/index", http.StatusFound)
		return
	case "del":
		extDel(a, w, r, kv)
		return
	case "mod":
		if r.Method == http.MethodPost {
			extMod(a, w, r, kv)
			return
		}
		extModForm(a, w, r, sess, kv)
		return
	}
	if r.Method == http.MethodPost {
		if r.FormValue("id") != "" {
			kv["id"] = r.FormValue("id")
			extMod(a, w, r, kv)
			return
		}
		extAdd(a, w, r)
		return
	}
	d := extBag(a, w, r, sess)
	d.Vars["list"] = "1"
	list, _ := a.DB.Ent.ExtField.Query().Order(extfield.ByMcode(), extfield.BySorting(), extfield.ByID()).All(bg)
	d.Vars["extfields"] = store.Rows(list)
	d.Vars["pagebar"] = ""
	writePHP(a, w, "content/extfield.html", d)
}

func extAdd(a *app.App, w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("name")
	if !strings.HasPrefix(name, "ext_") {
		name = "ext_" + name
	}
	if !store.Ident(name) {
		sortFail(w, r, "字段名需为 ext_ 开头的标识符")
		return
	}
	err := a.DB.Ent.ExtField.Create().
		SetMcode(r.FormValue("mcode")).SetName(name).SetType(or(r.FormValue("type"), "1")).
		SetValue(r.FormValue("value")).SetDescription(r.FormValue("description")).
		SetSorting(domain.Atoi(r.FormValue("sorting"), 255)).
		Exec(bg)
	if err != nil {
		sortFail(w, r, err.Error())
		return
	}
	_ = a.DB.AddColumn("ay_content_ext", name, "TEXT", "varchar(500) NOT NULL DEFAULT ''")
	sortOK(w, r, "新增成功！", "/admin/ExtField/index")
}

func extMod(a *app.App, w http.ResponseWriter, r *http.Request, kv map[string]string) {
	id := domain.Atoi(or(kv["id"], r.FormValue("id")), 0)
	_, err := a.DB.Ent.ExtField.Update().Where(extfield.IDEQ(id)).
		SetMcode(r.FormValue("mcode")).SetDescription(r.FormValue("description")).
		SetType(or(r.FormValue("type"), "1")).SetValue(r.FormValue("value")).
		SetSorting(domain.Atoi(r.FormValue("sorting"), 255)).
		Save(bg)
	if err != nil {
		sortFail(w, r, err.Error())
		return
	}
	sortOK(w, r, "修改成功！", "/admin/ExtField/index")
}

func extDel(a *app.App, w http.ResponseWriter, r *http.Request, kv map[string]string) {
	if r.Method != http.MethodPost {
		sortFail(w, r, "请使用正确方式提交！")
		return
	}
	id := domain.Atoi(or(kv["id"], r.FormValue("id")), 0)
	// shortcut: leave ay_content_ext column; drop when schema sync exists
	_, err := a.DB.Ent.ExtField.Delete().Where(extfield.IDEQ(id)).Exec(bg)
	if err != nil {
		sortFail(w, r, "删除失败！")
		return
	}
	sortOK(w, r, "删除成功！", "/admin/ExtField/index")
}

func extModForm(a *app.App, w http.ResponseWriter, r *http.Request, sess *auth.Session, kv map[string]string) {
	id := domain.Atoi(kv["id"], 0)
	one, err := a.DB.Ent.ExtField.Get(bg, id)
	if err != nil {
		http.Redirect(w, r, "/admin/ExtField/index", http.StatusFound)
		return
	}
	d := extBag(a, w, r, sess)
	d.Vars["mod"] = "1"
	d.Vars["extfield"] = store.Row(one)
	d.Get["id"] = strconv.Itoa(id)
	writePHP(a, w, "content/extfield.html", d)
}

func extBag(a *app.App, w http.ResponseWriter, r *http.Request, sess *auth.Session) Data {
	d := bag(a, w, r, sess)
	models, _ := domain.Models(a.DB)
	d.Vars["models"] = models
	d.Vars["pagebar"] = ""
	d.Vars["backurl"] = ""
	fc, _ := d.Vars["formcheck"].(string)
	d.Funs["get_btn_back"] = func(args []string) string {
		return `<a href="/admin/ExtField/index" class="layui-btn layui-btn-primary">返 回</a>`
	}
	d.Funs["get_btn_mod"] = func(args []string) string {
		return `<a href="/admin/ExtField/mod/id/` + argAt(args, 0) + `" class="layui-btn layui-btn-xs">修改</a>`
	}
	d.Funs["get_btn_del_post"] = func(args []string) string {
		return `<button type="button" class="layui-btn layui-btn-xs layui-btn-danger pboot-post" data-action="/admin/ExtField/del/id/` + argAt(args, 0) + `" data-after="reload" data-confirm="您确定要删除么？" data-formcheck="` + fc + `">删除</button>`
	}
	return d
}

func labelPage(a *app.App, w http.ResponseWriter, r *http.Request, sess *auth.Session, action string, kv map[string]string) {
	switch action {
	case "add":
		if r.Method == http.MethodPost {
			labelAdd(a, w, r, sess)
			return
		}
		http.Redirect(w, r, "/admin/Label/index", http.StatusFound)
		return
	case "del":
		labelDel(a, w, r, kv)
		return
	case "mod":
		if r.Method == http.MethodPost {
			labelModMeta(a, w, r, sess, kv)
			return
		}
		labelModForm(a, w, r, sess, kv)
		return
	}
	if r.Method == http.MethodPost {
		op := r.FormValue("op")
		switch op {
		case "del":
			labelDel(a, w, r, kv)
			return
		case "add":
			labelAdd(a, w, r, sess)
			return
		case "mod":
			labelModMeta(a, w, r, sess, kv)
			return
		}
		labelSaveValues(a, w, r, sess)
		return
	}
	if kv["op"] == "mod" && kv["id"] != "" {
		labelModForm(a, w, r, sess, kv)
		return
	}
	d := labelBag(a, w, r, sess)
	d.Vars["list"] = "1"
	list, _ := a.DB.Ent.Label.Query().Order(label.ByID()).All(bg)
	d.Vars["labels"] = store.Rows(list)
	writePHP(a, w, "content/label.html", d)
}

func labelAdd(a *app.App, w http.ResponseWriter, r *http.Request, sess *auth.Session) {
	name := r.FormValue("name")
	if name == "" || !labelNameOK(name) {
		sortFail(w, r, "标签名称只允许字母、数字、下划线")
		return
	}
	if n, _ := a.DB.Ent.Label.Query().Where(label.NameEQ(name)).Count(bg); n > 0 {
		sortFail(w, r, "该自定义标签名称已经存在")
		return
	}
	now := domain.Now()
	user := sess.Values["username"]
	if err := a.DB.Ent.Label.Create().
		SetName(name).SetValue("").SetType(or(r.FormValue("type"), "1")).SetDescription(r.FormValue("description")).
		SetCreateUser(user).SetUpdateUser(user).SetCreateTime(now).SetUpdateTime(now).
		Exec(bg); err != nil {
		sortFail(w, r, err.Error())
		return
	}
	front.ClearPages(a)
	sortOK(w, r, "新增成功！", "/admin/Label/index")
}

func labelModMeta(a *app.App, w http.ResponseWriter, r *http.Request, sess *auth.Session, kv map[string]string) {
	id := or(r.FormValue("id"), kv["id"])
	name := r.FormValue("name")
	if name == "" || !labelNameOK(name) {
		sortFail(w, r, "标签名称只允许字母、数字、下划线")
		return
	}
	if n, _ := a.DB.Ent.Label.Query().Where(label.NameEQ(name), label.IDNEQ(domain.Atoi(id, 0))).Count(bg); n > 0 {
		sortFail(w, r, "该自定义标签名称已经存在")
		return
	}
	if _, err := a.DB.Ent.Label.Update().Where(label.IDEQ(domain.Atoi(id, 0))).
		SetName(name).SetDescription(r.FormValue("description")).SetType(or(r.FormValue("type"), "1")).
		SetUpdateUser(sess.Values["username"]).SetUpdateTime(domain.Now()).
		Save(bg); err != nil {
		sortFail(w, r, err.Error())
		return
	}
	front.ClearPages(a)
	sortOK(w, r, "修改成功！", "/admin/Label/index")
}

func labelDel(a *app.App, w http.ResponseWriter, r *http.Request, kv map[string]string) {
	if r.Method != http.MethodPost {
		sortFail(w, r, "请使用正确方式提交！")
		return
	}
	id := or(r.FormValue("id"), kv["id"])
	if _, err := a.DB.Ent.Label.Delete().Where(label.IDEQ(domain.Atoi(id, 0))).Exec(bg); err != nil {
		sortFail(w, r, err.Error())
		return
	}
	front.ClearPages(a)
	sortOK(w, r, "删除成功！", "/admin/Label/index")
}

func labelSaveValues(a *app.App, w http.ResponseWriter, r *http.Request, sess *auth.Session) {
	rows, _ := a.DB.Ent.Label.Query().All(bg)
	for _, row := range rows {
		if !r.Form.Has(row.Name) {
			continue
		}
		val := strings.ReplaceAll(r.FormValue(row.Name), "\r\n", "<br>")
		_, _ = a.DB.Ent.Label.Update().Where(label.NameEQ(row.Name)).
			SetValue(scrubHTML(val)).SetUpdateUser(sess.Values["username"]).SetUpdateTime(domain.Now()).
			Save(bg)
	}
	front.ClearPages(a)
	sortOK(w, r, "修改成功！", "/admin/Label/index")
}

func labelModForm(a *app.App, w http.ResponseWriter, r *http.Request, sess *auth.Session, kv map[string]string) {
	id := domain.Atoi(kv["id"], 0)
	one, err := a.DB.Ent.Label.Get(bg, id)
	if err != nil {
		http.Redirect(w, r, "/admin/Label/index", http.StatusFound)
		return
	}
	d := labelBag(a, w, r, sess)
	d.Vars["mod"] = "1"
	d.Vars["label"] = store.Row(one)
	d.Get["id"] = strconv.Itoa(id)
	writePHP(a, w, "content/label.html", d)
}

func labelBag(a *app.App, w http.ResponseWriter, r *http.Request, sess *auth.Session) Data {
	d := bag(a, w, r, sess)
	d.Vars["backurl"] = ""
	fc, _ := d.Vars["formcheck"].(string)
	d.Funs["get_btn_back"] = func(args []string) string {
		return `<a href="/admin/Label/index" class="layui-btn layui-btn-primary">返 回</a>`
	}
	d.Funs["get_btn_mod"] = func(args []string) string {
		return `<a href="/admin/Label/mod/id/` + argAt(args, 0) + `" class="layui-btn layui-btn-xs">修改</a>`
	}
	d.Funs["get_btn_del_post"] = func(args []string) string {
		return `<button type="button" class="layui-btn layui-btn-xs layui-btn-danger pboot-post" data-action="/admin/Label/del/id/` + argAt(args, 0) + `" data-after="reload" data-confirm="您确定要删除么？" data-formcheck="` + fc + `">删除</button>`
	}
	return d
}

func labelNameOK(name string) bool {
	for _, c := range name {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' {
			continue
		}
		return false
	}
	return name != ""
}

func messagePage(a *app.App, w http.ResponseWriter, r *http.Request, sess *auth.Session, acode string, kv map[string]string, action string) {
	switch action {
	case "del":
		if r.Method != http.MethodPost {
			sortFail(w, r, "请使用正确方式提交！")
			return
		}
		id := domain.Atoi(or(kv["id"], r.FormValue("id")), 0)
		_, err := a.DB.Ent.Message.Delete().Where(message.IDEQ(id), message.AcodeEQ(acode)).Exec(bg)
		if err != nil {
			sortFail(w, r, "删除失败！")
			return
		}
		sortOK(w, r, "删除成功！", "/admin/Message/index")
		return
	case "mod":
		if r.Method == http.MethodPost {
			id := domain.Atoi(or(kv["id"], r.FormValue("id")), 0)
			if f := r.FormValue("field"); f == "status" {
				_, _ = a.DB.Ent.Message.Update().Where(message.IDEQ(id), message.AcodeEQ(acode)).
					SetStatus(r.FormValue("value")).SetUpdateUser(sess.Values["username"]).SetUpdateTime(domain.Now()).Save(bg)
				sortOK(w, r, "修改成功！", "/admin/Message/index")
				return
			}
			_, err := a.DB.Ent.Message.Update().Where(message.IDEQ(id), message.AcodeEQ(acode)).
				SetRecontent(r.FormValue("recontent")).SetStatus(or(r.FormValue("status"), "1")).
				SetUpdateUser(sess.Values["username"]).SetUpdateTime(domain.Now()).Save(bg)
			if err != nil {
				sortFail(w, r, err.Error())
				return
			}
			sortOK(w, r, "修改成功！", "/admin/Message/index")
			return
		}
		messageModForm(a, w, r, sess, acode, kv)
		return
	}
	if r.Method == http.MethodPost {
		_, _ = a.DB.Ent.Message.Update().Where(message.IDEQ(domain.Atoi(r.FormValue("id"), 0))).
			SetRecontent(r.FormValue("recontent")).SetStatus(or(r.FormValue("status"), "1")).SetUpdateTime(domain.Now()).
			Save(bg)
		http.Redirect(w, r, "/admin/Message/index", http.StatusFound)
		return
	}
	if kv["id"] != "" {
		messageModForm(a, w, r, sess, acode, kv)
		return
	}
	list, _ := a.DB.Ent.Message.Query().Where(message.AcodeEQ(acode)).Order(message.ByID(sql.OrderDesc())).Limit(100).All(bg)
	rows := store.Rows(list)
	fields := messageFields(a)
	d := messageBag(a, w, r, sess)
	d.Vars["list"] = "1"
	d.Vars["messages"] = rows
	d.Vars["fields"] = fields
	d.Vars["pagebar"] = ""
	writePHP(a, w, "content/message.html", d)
}

func messageModForm(a *app.App, w http.ResponseWriter, r *http.Request, sess *auth.Session, acode string, kv map[string]string) {
	id := domain.Atoi(kv["id"], 0)
	one, err := a.DB.Ent.Message.Query().Where(message.IDEQ(id), message.AcodeEQ(acode)).First(bg)
	if err != nil {
		http.Redirect(w, r, "/admin/Message/index", http.StatusFound)
		return
	}
	d := messageBag(a, w, r, sess)
	d.Vars["mod"] = "1"
	d.Vars["message"] = store.Row(one)
	d.Get["id"] = strconv.Itoa(id)
	writePHP(a, w, "content/message.html", d)
}

func messageFields(a *app.App) []map[string]string {
	// message form is fcode=1 in Pboot; fall back to common columns
	fields, err := a.DB.Ent.FormField.Query().Where(formfield.FcodeEQ("1")).Order(formfield.BySorting(), formfield.ByID()).All(bg)
	if err == nil && len(fields) > 0 {
		return store.Rows(fields)
	}
	return []map[string]string{
		{"name": "contacts", "description": "联系人"},
		{"name": "mobile", "description": "手机"},
		{"name": "content", "description": "内容"},
	}
}

func messageBag(a *app.App, w http.ResponseWriter, r *http.Request, sess *auth.Session) Data {
	d := bag(a, w, r, sess)
	d.Vars["backurl"] = ""
	d.Vars["pagebar"] = ""
	fc, _ := d.Vars["formcheck"].(string)
	d.Funs["get_btn_back"] = func(args []string) string {
		return `<a href="/admin/Message/index" class="layui-btn layui-btn-primary">返 回</a>`
	}
	d.Funs["get_btn_mod"] = func(args []string) string {
		label := or(argAt(args, 2), "修改")
		return `<a href="/admin/Message/mod/id/` + argAt(args, 0) + `" class="layui-btn layui-btn-xs">` + label + `</a>`
	}
	d.Funs["get_btn_del_post"] = func(args []string) string {
		return `<button type="button" class="layui-btn layui-btn-xs layui-btn-danger pboot-post" data-action="/admin/Message/del/id/` + argAt(args, 0) + `" data-after="reload" data-confirm="您确定要删除么？" data-formcheck="` + fc + `">删除</button>`
	}
	d.Funs["admin_switch_btn"] = func(args []string) string {
		id, field, value, on, idName := argAt(args, 0), argAt(args, 1), argAt(args, 2), argAt(args, 3), or(argAt(args, 4), "id")
		icon := "fa-toggle-off"
		if on == "1" {
			icon = "fa-toggle-on"
		}
		return `<button type="button" class="pboot-post" data-action="/admin/Message/mod/` + idName + `/` + id + `" data-after="reload" data-formcheck="` + fc + `" data-field="` + field + `" data-value="` + value + `" style="background:none;border:0;padding:0;cursor:pointer;color:inherit"><i class="fa ` + icon + `"></i></button>`
	}
	d.Funs["long2ip_safe"] = func(args []string) string {
		n := domain.Atoi(argAt(args, 0), 0)
		if n == 0 {
			return argAt(args, 0)
		}
		return fmt.Sprintf("%d.%d.%d.%d", byte(n>>24), byte(n>>16), byte(n>>8), byte(n))
	}
	return d
}

func formPage(a *app.App, w http.ResponseWriter, r *http.Request, sess *auth.Session, kv map[string]string, action string) {
	act := or(kv["action"], "")
	switch action {
	case "add":
		if r.Method == http.MethodPost {
			formAdd(a, w, r, sess, act)
			return
		}
		http.Redirect(w, r, "/admin/Form/index", http.StatusFound)
		return
	case "del":
		formDel(a, w, r, kv, act)
		return
	case "mod":
		if r.Method == http.MethodPost {
			if act == "" && r.FormValue("form_name") == "" {
				act = "modfield"
			}
			if act == "" {
				act = "modform"
			}
			formMod(a, w, r, sess, kv, act)
			return
		}
		if act == "modform" || act == "" {
			formModForm(a, w, r, sess, kv)
			return
		}
		http.Redirect(w, r, "/admin/Form/index", http.StatusFound)
		return
	case "clear":
		formClear(a, w, r, kv)
		return
	}
	// index
	if r.Method == http.MethodPost {
		if r.FormValue("op") == "del" {
			kv["id"] = r.FormValue("id")
			formDel(a, w, r, kv, "delform")
			return
		}
		if r.FormValue("id") != "" && r.FormValue("form_name") != "" {
			kv["id"] = r.FormValue("id")
			formMod(a, w, r, sess, kv, "modform")
			return
		}
		if r.FormValue("form_name") != "" {
			formAdd(a, w, r, sess, "addform")
			return
		}
		if r.FormValue("name") != "" || r.FormValue("field") != "" {
			formAddField(a, w, r, sess, kv)
			return
		}
	}
	switch act {
	case "showdata":
		formShowData(a, w, r, sess, kv)
		return
	case "showfield":
		formShowField(a, w, r, sess, kv)
		return
	}
	if kv["id"] != "" {
		formModForm(a, w, r, sess, kv)
		return
	}
	d := formBag(a, w, r, sess)
	d.Vars["list"] = "1"
	list, _ := a.DB.Ent.Form.Query().Order(form.ByID()).All(bg)
	d.Vars["forms"] = store.Rows(list)
	d.Vars["pagebar"] = ""
	writePHP(a, w, "content/form.html", d)
}

func formAdd(a *app.App, w http.ResponseWriter, r *http.Request, sess *auth.Session, act string) {
	if act == "addfield" {
		formAddField(a, w, r, sess, map[string]string{"fcode": r.FormValue("fcode")})
		return
	}
	suffix := or(r.FormValue("table_name"), r.FormValue("table"))
	suffix = strings.TrimPrefix(strings.ToLower(suffix), "ay_diy_")
	tableName := "ay_diy_" + suffix
	if !strings.HasPrefix(tableName, "ay_diy_") || !store.Ident(tableName) || suffix == "" {
		sortFail(w, r, "表名非法")
		return
	}
	fcode := strconv.Itoa(domain.Atoi(maxCode(a, "ay_form", "fcode"), 1) + 1)
	now := domain.Now()
	user := sess.Values["username"]
	if err := a.DB.Ent.Form.Create().
		SetFcode(fcode).SetFormName(r.FormValue("form_name")).SetTableName(tableName).
		SetCreateUser(user).SetUpdateUser(user).SetCreateTime(now).SetUpdateTime(now).
		Exec(bg); err != nil {
		sortFail(w, r, err.Error())
		return
	}
	_ = a.DB.CreateDiy(tableName, nil)
	sortOK(w, r, "新增成功！", "/admin/Form/index")
}

func formMod(a *app.App, w http.ResponseWriter, r *http.Request, sess *auth.Session, kv map[string]string, act string) {
	id := domain.Atoi(or(kv["id"], r.FormValue("id")), 0)
	switch act {
	case "addmenu":
		if err := addFormMenu(a, id, sess.Values["username"]); err != nil {
			sortFail(w, r, err.Error())
			return
		}
		sortOK(w, r, "添加成功！", "/admin/Form/index")
		return
	case "modfield":
		fid := id
		_, err := a.DB.Ent.FormField.Update().Where(formfield.IDEQ(fid)).
			SetDescription(r.FormValue("description")).SetLength(domain.Atoi(r.FormValue("length"), 100)).
			SetRequired(or(r.FormValue("required"), "0")).SetSorting(domain.Atoi(r.FormValue("sorting"), 255)).
			SetUpdateUser(sess.Values["username"]).SetUpdateTime(domain.Now()).
			Save(bg)
		if err != nil {
			sortFail(w, r, err.Error())
			return
		}
		sortOK(w, r, "修改成功！", "/admin/Form/index/fcode/"+or(kv["fcode"], r.FormValue("fcode"))+"/action/showfield")
		return
	}
	if f := r.FormValue("field"); f == "required" {
		_, _ = a.DB.Ent.FormField.Update().Where(formfield.IDEQ(id)).
			SetRequired(r.FormValue("value")).SetUpdateTime(domain.Now()).Save(bg)
		sortOK(w, r, "修改成功！", r.URL.Path)
		return
	}
	_, err := a.DB.Ent.Form.Update().Where(form.IDEQ(id)).
		SetFormName(r.FormValue("form_name")).SetUpdateUser(sess.Values["username"]).SetUpdateTime(domain.Now()).
		Save(bg)
	if err != nil {
		sortFail(w, r, err.Error())
		return
	}
	sortOK(w, r, "修改成功！", "/admin/Form/index")
}

func formDel(a *app.App, w http.ResponseWriter, r *http.Request, kv map[string]string, act string) {
	if r.Method != http.MethodPost {
		sortFail(w, r, "请使用正确方式提交！")
		return
	}
	id := domain.Atoi(or(kv["id"], r.FormValue("id")), 0)
	switch act {
	case "deldata":
		fcode := or(kv["fcode"], r.FormValue("fcode"))
		if one, err := a.DB.Ent.Form.Query().Where(form.FcodeEQ(fcode)).First(bg); err == nil {
			_ = a.DB.DeleteDiyRow(one.TableName, id)
		}
		sortOK(w, r, "删除成功！", "/admin/Form/index/fcode/"+fcode+"/action/showdata")
		return
	case "delform", "":
		if one, err := a.DB.Ent.Form.Get(bg, id); err == nil {
			_, _ = a.DB.Ent.FormField.Delete().Where(formfield.FcodeEQ(one.Fcode)).Exec(bg)
			_, _ = a.DB.Ent.Form.Delete().Where(form.IDEQ(one.ID)).Exec(bg)
			if strings.HasPrefix(one.TableName, "ay_diy_") {
				_ = a.DB.DropDiy(one.TableName)
			}
		}
		sortOK(w, r, "删除成功！", "/admin/Form/index")
		return
	default:
		// delete field
		fcode := or(kv["fcode"], r.FormValue("fcode"))
		if one, err := a.DB.Ent.FormField.Get(bg, id); err == nil {
			_, _ = a.DB.Ent.FormField.Delete().Where(formfield.IDEQ(id)).Exec(bg)
			if f, err := a.DB.Ent.Form.Query().Where(form.FcodeEQ(one.Fcode)).First(bg); err == nil {
				_ = a.DB.DropColumn(f.TableName, one.Name)
			}
		}
		sortOK(w, r, "删除成功！", "/admin/Form/index/fcode/"+fcode+"/action/showfield")
	}
}

func formClear(a *app.App, w http.ResponseWriter, r *http.Request, kv map[string]string) {
	fcode := kv["fcode"]
	if one, err := a.DB.Ent.Form.Query().Where(form.FcodeEQ(fcode)).First(bg); err == nil {
		_ = a.DB.ClearDiy(one.TableName)
	}
	sortOK(w, r, "清空成功！", "/admin/Form/index/fcode/"+fcode+"/action/showdata")
}

func formAddField(a *app.App, w http.ResponseWriter, r *http.Request, sess *auth.Session, kv map[string]string) {
	name := or(r.FormValue("name"), r.FormValue("field"))
	fcode := or(r.FormValue("fcode"), kv["fcode"])
	if !store.Ident(name) {
		sortFail(w, r, "字段非法")
		return
	}
	now := domain.Now()
	user := sess.Values["username"]
	if err := a.DB.Ent.FormField.Create().
		SetFcode(fcode).SetName(name).SetLength(domain.Atoi(r.FormValue("length"), 100)).
		SetRequired(or(r.FormValue("required"), "0")).SetDescription(r.FormValue("description")).
		SetSorting(domain.Atoi(r.FormValue("sorting"), 255)).
		SetCreateUser(user).SetUpdateUser(user).SetCreateTime(now).SetUpdateTime(now).
		Exec(bg); err != nil {
		sortFail(w, r, err.Error())
		return
	}
	if one, err := a.DB.Ent.Form.Query().Where(form.FcodeEQ(fcode)).First(bg); err == nil {
		_ = a.DB.AddColumn(one.TableName, name, "TEXT", "varchar(500) NULL")
	}
	sortOK(w, r, "新增成功！", "/admin/Form/index/fcode/"+fcode+"/action/showfield")
}

func formModForm(a *app.App, w http.ResponseWriter, r *http.Request, sess *auth.Session, kv map[string]string) {
	id := domain.Atoi(kv["id"], 0)
	one, err := a.DB.Ent.Form.Get(bg, id)
	if err != nil {
		http.Redirect(w, r, "/admin/Form/index", http.StatusFound)
		return
	}
	d := formBag(a, w, r, sess)
	d.Vars["mod"] = "1"
	d.Vars["form"] = store.Row(one)
	d.Get["id"] = strconv.Itoa(id)
	writePHP(a, w, "content/form.html", d)
}

func formShowField(a *app.App, w http.ResponseWriter, r *http.Request, sess *auth.Session, kv map[string]string) {
	fcode := kv["fcode"]
	one, err := a.DB.Ent.Form.Query().Where(form.FcodeEQ(fcode)).First(bg)
	if err != nil {
		http.Redirect(w, r, "/admin/Form/index", http.StatusFound)
		return
	}
	if kv["id"] != "" {
		fld, err := a.DB.Ent.FormField.Get(bg, domain.Atoi(kv["id"], 0))
		if err == nil {
			d := formBag(a, w, r, sess)
			d.Vars["mod"] = "1"
			d.Vars["field"] = store.Row(fld)
			d.Get["fcode"] = fcode
			d.Get["id"] = kv["id"]
			writePHP(a, w, "content/form.html", d)
			return
		}
	}
	fields, _ := a.DB.Ent.FormField.Query().Where(formfield.FcodeEQ(fcode)).Order(formfield.BySorting(), formfield.ByID()).All(bg)
	d := formBag(a, w, r, sess)
	d.Vars["showfield"] = "1"
	d.Vars["form"] = store.Row(one)
	d.Vars["fields"] = store.Rows(fields)
	d.Get["fcode"] = fcode
	writePHP(a, w, "content/form.html", d)
}

func formShowData(a *app.App, w http.ResponseWriter, r *http.Request, sess *auth.Session, kv map[string]string) {
	fcode := kv["fcode"]
	one, err := a.DB.Ent.Form.Query().Where(form.FcodeEQ(fcode)).First(bg)
	if err != nil {
		http.Redirect(w, r, "/admin/Form/index", http.StatusFound)
		return
	}
	fields, _ := a.DB.Ent.FormField.Query().Where(formfield.FcodeEQ(fcode)).Order(formfield.BySorting(), formfield.ByID()).All(bg)
	rows, _ := a.DB.ListDiy(one.TableName, 100)
	d := formBag(a, w, r, sess)
	d.Vars["showdata"] = "1"
	d.Vars["form"] = store.Row(one)
	d.Vars["fields"] = store.Rows(fields)
	d.Vars["formdata"] = rows
	d.Get["fcode"] = fcode
	writePHP(a, w, "content/form.html", d)
}

func formBag(a *app.App, w http.ResponseWriter, r *http.Request, sess *auth.Session) Data {
	d := bag(a, w, r, sess)
	d.Vars["btnqs"] = ""
	d.Vars["backurl"] = ""
	d.Vars["pagebar"] = ""
	fc, _ := d.Vars["formcheck"].(string)
	d.Funs["get_btn_back"] = func(args []string) string {
		return `<a href="/admin/Form/index" class="layui-btn layui-btn-primary">返 回</a>`
	}
	d.Funs["get_btn_mod"] = func(args []string) string {
		id := argAt(args, 0)
		fcode := d.Get["fcode"]
		return `<a href="/admin/Form/index/fcode/` + fcode + `/action/showfield/id/` + id + `" class="layui-btn layui-btn-xs">修改</a>`
	}
	d.Funs["admin_post_link"] = func(args []string) string {
		action, label := argAt(args, 0), argAt(args, 1)
		after := or(argAt(args, 2), "stay")
		class := or(argAt(args, 3), "layui-btn layui-btn-xs layui-btn-primary")
		confirm := argAt(args, 4)
		conf := ""
		if confirm != "" {
			conf = ` data-confirm="` + confirm + `"`
		}
		return `<button type="button" class="` + class + ` pboot-post" data-action="` + action + `" data-after="` + after + `"` + conf + ` data-formcheck="` + fc + `">` + label + `</button>`
	}
	d.Funs["admin_switch_btn"] = func(args []string) string {
		id, field, value, on, idName := argAt(args, 0), argAt(args, 1), argAt(args, 2), argAt(args, 3), or(argAt(args, 4), "id")
		icon := "fa-toggle-off"
		if on == "1" {
			icon = "fa-toggle-on"
		}
		return `<button type="button" class="pboot-post" data-action="/admin/Form/mod/` + idName + `/` + id + `" data-after="reload" data-formcheck="` + fc + `" data-field="` + field + `" data-value="` + value + `" style="background:none;border:0;padding:0;cursor:pointer;color:inherit"><i class="fa ` + icon + `"></i></button>`
	}
	return d
}

func configPage(a *app.App, w http.ResponseWriter, r *http.Request, sess *auth.Session) {
	if r.Method == http.MethodPost {
		_ = r.ParseForm()
		indexnowKey := strings.TrimSpace(r.FormValue("indexnow_key"))
		indexnowLoc := strings.TrimSpace(r.FormValue("indexnow_key_location"))
		if indexnowKey != "" && !checkIndexnowKey(indexnowKey) {
			sortFail(w, r, "IndexNow密钥格式不正确，须为8-128位字母、数字或连字符")
			return
		}
		for name, vs := range r.Form {
			if name == "formcheck" || name == "submit" || len(vs) == 0 {
				continue
			}
			val := vs[0]
			if name == "ai_api_key" {
				if val == "" || isMaskedSecret(val) {
					continue
				}
				enc := aesEncrypt(a.Cfg.SiteRoot, val)
				if enc == "" {
					sortFail(w, r, "服务端不支持 AES 加密")
					return
				}
				val = enc
			}
			_ = a.DB.SetConf(name, val)
		}
		if indexnowKey != "" {
			if err := saveIndexnowKeyFiles(a, indexnowKey, indexnowLoc); err != nil {
				sortFail(w, r, "IndexNow密钥文件写入失败，请检查网站根目录写入权限！")
				return
			}
		}
		front.ClearPages(a)
		if wantsJSON(r) {
			front.JSON(w, 1, "修改成功！")
			return
		}
		http.Redirect(w, r, "/admin/Config/index", http.StatusFound)
		return
	}
	list, _ := a.DB.Ent.SysConfig.Query().Order(sysconfig.BySorting(), sysconfig.ByID()).All(bg)
	cfgs := map[string]map[string]string{}
	for _, row := range store.Rows(list) {
		cfgs[row["name"]] = row
	}
	if row := cfgs["ai_api_key"]; row != nil && row["value"] != "" {
		if plain := aesDecrypt(a.Cfg.SiteRoot, row["value"]); plain != "" {
			row["value"] = maskSecret(plain)
		} else if plain := aiKeyPlain(a); plain != "" {
			row["value"] = maskSecret(plain)
		} else {
			row["value"] = ""
		}
	}
	ensureConfig := func(name, def string) {
		if cfgs[name] == nil {
			cfgs[name] = map[string]string{"name": name, "value": def}
		}
	}
	for _, k := range []string{
		"close_site", "close_site_note", "session_in_sitepath", "lgautosw", "spiderlog",
		"to_https", "to_main_domain", "main_domain", "pagenum", "debug", "sn", "sn_user",
		"indexnow_key", "indexnow_key_location", "llms_open", "llms_sort_num", "llms_desc_len", "llms_intro",
		"tpl_html_cache", "tpl_html_cache_time", "url_type",
	} {
		ensureConfig(k, "")
	}
	d := bag(a, w, r, sess)
	d.Vars["configs"] = cfgs
	d.Vars["llms_sort_checkbox"] = ""
	d.Vars["llms_url"] = "/llms.txt"
	d.Vars["is_sqlite"] = "0"
	if a.DB.Driver == "sqlite" {
		d.Vars["is_sqlite"] = "1"
	}
	d.Vars["ai_provider_json"] = `{"deepseek":{"name":"DeepSeek","base":"https://api.deepseek.com","model":"deepseek-chat"},"openai":{"name":"OpenAI","base":"https://api.openai.com","model":"gpt-4o-mini"}}`
	d.Vars["ai_key_openssl_warn"] = ""
	if probe := aesEncrypt(a.Cfg.SiteRoot, "probe"); probe == "" {
		d.Vars["ai_key_openssl_warn"] = "1"
	}
	d.Vars["LICENSE"] = "3"
	writePHP(a, w, "system/config.html", d)
}

func addFormMenu(a *app.App, id int, user string) error {
	one, err := a.DB.Ent.Form.Get(bg, id)
	if err != nil {
		return fmt.Errorf("添加失败！")
	}
	needle := "Form/index/fcode/" + one.Fcode + "/action/showdata"
	if found, err := a.DB.Ent.Menu.Query().Where(menu.URLContains(needle)).First(bg); err == nil {
		if found.Name != one.FormName {
			_, _ = a.DB.Ent.Menu.UpdateOneID(found.ID).
				SetName(one.FormName).SetUpdateUser(user).SetUpdateTime(domain.Now()).Save(bg)
		}
		return fmt.Errorf("添加失败！")
	}
	now := domain.Now()
	return a.DB.Ent.Menu.Create().
		SetMcode(nextMcode(a)).SetPcode("M157").SetName(one.FormName).
		SetURL("/admin/" + needle).SetSorting(599).SetStatus("1").SetShortcut("0").
		SetIco("fa-plus-square-o").SetCreateUser(user).SetUpdateUser(user).
		SetCreateTime(now).SetUpdateTime(now).Exec(bg)
}

func nextMcode(a *app.App) string {
	rows, _ := a.DB.Ent.Menu.Query().All(bg)
	max := 100
	for _, r := range rows {
		n := 0
		for _, c := range r.Mcode {
			if c >= '0' && c <= '9' {
				n = n*10 + int(c-'0')
			}
		}
		if n > max {
			max = n
		}
	}
	return "M" + strconv.Itoa(max+1)
}

func sqlitePath(dsn string) string {
	if i := strings.Index(dsn, "?"); i >= 0 {
		dsn = dsn[:i]
	}
	return strings.TrimPrefix(dsn, "file:")
}

func backupSQLite(a *app.App) (string, error) {
	if a.DB.Driver != "sqlite" {
		return "", fmt.Errorf("当前只支持 sqlite 文件备份")
	}
	src := sqlitePath(a.Cfg.DSN)
	if src == "" {
		return "", fmt.Errorf("找不到数据库文件")
	}
	dir := filepath.Join(filepath.Dir(src), "backup")
	if err := fsperm.MkdirAll(dir, fsperm.DirPrivate); err != nil {
		return "", err
	}
	name := "pbootcms-go-" + time.Now().Format("20060102-150405") + ".db"
	dst := filepath.Join(dir, name)
	in, err := os.Open(src)
	if err != nil {
		return "", err
	}
	defer in.Close()
	out, err := fsperm.Create(dst, fsperm.FilePrivate)
	if err != nil {
		return "", err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return "", err
	}
	return name, nil
}

func esc(s string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;").Replace(s)
}

func or(a, b string) string {
	if a == "" {
		return b
	}
	return a
}

func maxScode(a *app.App) string {
	rows, _ := a.DB.Ent.ContentSort.Query().All(bg)
	max := 0
	for _, r := range rows {
		if n := domain.Atoi(r.Scode, 0); n > max {
			max = n
		}
	}
	return strconv.Itoa(max)
}

func maxCode(a *app.App, table, col string) string {
	if !store.Ident(table) || !store.Ident(col) {
		return "0"
	}
	var codes []string
	switch table {
	case "ay_model":
		rows, _ := a.DB.Ent.Model.Query().All(bg)
		for _, r := range rows {
			codes = append(codes, r.Mcode)
		}
	case "ay_form":
		rows, _ := a.DB.Ent.Form.Query().All(bg)
		for _, r := range rows {
			codes = append(codes, r.Fcode)
		}
	case "ay_role":
		rows, _ := a.DB.Ent.Role.Query().All(bg)
		for _, r := range rows {
			codes = append(codes, r.Rcode)
		}
	case "ay_user":
		rows, _ := a.DB.Ent.User.Query().All(bg)
		for _, r := range rows {
			codes = append(codes, r.Ucode)
		}
	}
	max := 0
	for _, code := range codes {
		digits := ""
		for _, c := range code {
			if c >= '0' && c <= '9' {
				digits += string(c)
			}
		}
		if n := domain.Atoi(digits, 0); n > max {
			max = n
		}
	}
	return strconv.Itoa(max)
}


func scrubHTML(s string) string {
	low := strings.ToLower(s)
	for _, tag := range []string{"script", "iframe", "object", "embed", "svg"} {
		for {
			a := strings.Index(low, "<"+tag)
			if a < 0 {
				break
			}
			b := strings.Index(low[a:], ">")
			if b < 0 {
				s = s[:a]
				low = strings.ToLower(s)
				break
			}
			end := a + b + 1
			close := strings.Index(low[end:], "</"+tag)
			if close >= 0 {
				c2 := strings.Index(low[end+close:], ">")
				if c2 >= 0 {
					end = end + close + c2 + 1
				}
			}
			s = s[:a] + s[end:]
			low = strings.ToLower(s)
		}
	}
	return s
}