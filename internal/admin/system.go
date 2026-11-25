package admin

import (
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"entgo.io/ent/dialect/sql"
	"gbootcms/internal/app"
	"gbootcms/internal/auth"
	"gbootcms/internal/domain"
	"gbootcms/internal/ent/area"
	"gbootcms/internal/ent/menu"
	"gbootcms/internal/ent/role"
	"gbootcms/internal/ent/rolelevel"
	"gbootcms/internal/ent/syslog"
	"gbootcms/internal/ent/user"
	"gbootcms/internal/ent/userrole"
	"gbootcms/internal/front"
	"gbootcms/internal/store"
)

func systemBag(a *app.App, w http.ResponseWriter, r *http.Request, sess *auth.Session, ctrl string) Data {
	d := bag(a, w, r, sess)
	d.Vars["backurl"] = ""
	d.Vars["pathinfo"] = ""
	d.Vars["pagebar"] = ""
	fc, _ := d.Vars["formcheck"].(string)
	attachSystemFuns(d, ctrl, fc)
	return d
}

func attachSystemFuns(d Data, ctrl, fc string) {
	d.Funs["admin_stay_input"] = func(args []string) string { return `<input type="hidden" name="stay" value="">` }
	d.Funs["get_btn_back"] = func(args []string) string {
		return `<a href="/admin/` + ctrl + `/index" class="layui-btn layui-btn-primary">返 回</a>`
	}
	d.Funs["get_btn_mod"] = func(args []string) string {
		id, field := argAt(args, 0), or(argAt(args, 1), "id")
		return `<a href="/admin/` + ctrl + `/mod/` + field + `/` + id + `" class="layui-btn layui-btn-xs">修改</a>`
	}
	d.Funs["get_btn_del_post"] = func(args []string) string {
		id, field := argAt(args, 0), or(argAt(args, 1), "id")
		return `<button type="button" class="layui-btn layui-btn-xs layui-btn-danger pboot-post" data-action="/admin/` + ctrl + `/del/` + field + `/` + id + `" data-after="reload" data-confirm="您确定要删除么？" data-formcheck="` + fc + `">删除</button>`
	}
	d.Funs["admin_switch_btn"] = func(args []string) string {
		id, field, value, on, idName := argAt(args, 0), argAt(args, 1), argAt(args, 2), argAt(args, 3), or(argAt(args, 4), "id")
		icon := "fa-toggle-off"
		if on == "1" {
			icon = "fa-toggle-on"
		}
		return `<button type="button" class="pboot-post" data-action="/admin/` + ctrl + `/mod/` + idName + `/` + id + `" data-after="reload" data-formcheck="` + fc + `" data-field="` + field + `" data-value="` + value + `" style="background:none;border:0;padding:0;cursor:pointer;color:inherit"><i class="fa ` + icon + `"></i></button>`
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
	d.Funs["long2ip_safe"] = func(args []string) string {
		if len(args) == 0 {
			return ""
		}
		return esc(args[0])
	}
	d.Funs["check_level"] = func(args []string) string { return "1" }
}

func rolePage(a *app.App, w http.ResponseWriter, r *http.Request, sess *auth.Session, kv map[string]string, action string) {
	switch action {
	case "add":
		if r.Method == http.MethodPost {
			roleSave(a, w, r, sess, "")
			return
		}
		http.Redirect(w, r, "/admin/Role/index", http.StatusFound)
		return
	case "del":
		roleDel(a, w, r, kv)
		return
	case "mod":
		if r.Method == http.MethodPost {
			rcode := or(kv["rcode"], r.FormValue("rcode"))
			roleSave(a, w, r, sess, rcode)
			return
		}
		roleModForm(a, w, r, sess, kv)
		return
	}
	if r.Method == http.MethodPost {
		if r.FormValue("op") == "del" {
			roleDelByID(a, w, r)
			return
		}
		if r.FormValue("id") != "" || r.FormValue("name") != "" {
			rcode := ""
			if id := r.FormValue("id"); id != "" {
				if one, err := a.DB.Ent.Role.Get(bg, domain.Atoi(id, 0)); err == nil {
					rcode = one.Rcode
				}
			}
			roleSave(a, w, r, sess, rcode)
			return
		}
		roleSave(a, w, r, sess, "")
		return
	}
	if kv["rcode"] != "" {
		roleModForm(a, w, r, sess, kv)
		return
	}
	if kv["id"] != "" {
		if one, err := a.DB.Ent.Role.Get(bg, domain.Atoi(kv["id"], 0)); err == nil {
			kv["rcode"] = one.Rcode
			roleModForm(a, w, r, sess, kv)
			return
		}
	}
	d := systemBag(a, w, r, sess, "Role")
	d.Vars["list"] = "1"
	d.Vars["roles"] = roleRows(a)
	d.Vars["area_checkbox"] = roleAreaCheckbox(a, nil)
	d.Vars["menu_list"] = roleMenuList(a, nil)
	writePHP(a, w, "system/role.html", d)
}

func roleRows(a *app.App) []map[string]string {
	list, _ := a.DB.Ent.Role.Query().Order(role.ByID()).All(bg)
	return store.Rows(list)
}

func roleSave(a *app.App, w http.ResponseWriter, r *http.Request, sess *auth.Session, rcode string) {
	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		sortFail(w, r, "角色名不能为空！")
		return
	}
	now := domain.Now()
	user := sess.Values["username"]
	levels := r.Form["levels"]
	if len(levels) == 0 {
		levels = r.Form["level"]
	}
	acodes := r.Form["acodes"]
	if rcode == "" {
		rcode = "R" + strconv.Itoa(domain.Atoi(maxCode(a, "ay_role", "rcode"), 100)+1)
		if err := a.DB.Ent.Role.Create().
			SetRcode(rcode).SetName(name).SetDescription(r.FormValue("description")).
			SetCreateUser(user).SetUpdateUser(user).SetCreateTime(now).SetUpdateTime(now).
			Exec(bg); err != nil {
			sortFail(w, r, err.Error())
			return
		}
		sortOK(w, r, "新增成功！", "/admin/Role/index")
	} else {
		_, err := a.DB.Ent.Role.Update().Where(role.RcodeEQ(rcode)).
			SetName(name).SetDescription(r.FormValue("description")).
			SetUpdateUser(user).SetUpdateTime(now).Save(bg)
		if err != nil {
			sortFail(w, r, err.Error())
			return
		}
		sortOK(w, r, "修改成功！", "/admin/Role/index")
	}
	saveRoleLevels(a, rcode, levels)
	saveRoleAreas(a, rcode, acodes)
}

func saveRoleLevels(a *app.App, rcode string, levels []string) {
	_, _ = a.DB.Ent.RoleLevel.Delete().Where(rolelevel.RcodeEQ(rcode)).Exec(bg)
	for _, lv := range levels {
		if lv == "" {
			continue
		}
		_ = a.DB.Ent.RoleLevel.Create().SetRcode(rcode).SetLevel(lv).Exec(bg)
	}
}

func saveRoleAreas(a *app.App, rcode string, acodes []string) {
	_ = a.DB.Exec("DELETE FROM ay_role_area WHERE rcode=?", rcode)
	for _, ac := range acodes {
		_ = a.DB.Insert("ay_role_area", map[string]any{"rcode": rcode, "acode": ac})
	}
}

func roleDel(a *app.App, w http.ResponseWriter, r *http.Request, kv map[string]string) {
	rcode := or(kv["rcode"], r.FormValue("rcode"))
	if rcode == "" && kv["id"] != "" {
		if one, err := a.DB.Ent.Role.Get(bg, domain.Atoi(kv["id"], 0)); err == nil {
			rcode = one.Rcode
		}
	}
	if rcode == "" {
		sortFail(w, r, "传递的参数值错误！")
		return
	}
	roleDelRcode(a, w, r, rcode)
}

func roleDelByID(a *app.App, w http.ResponseWriter, r *http.Request) {
	if one, err := a.DB.Ent.Role.Get(bg, domain.Atoi(r.FormValue("id"), 0)); err == nil {
		roleDelRcode(a, w, r, one.Rcode)
		return
	}
	http.Redirect(w, r, "/admin/Role/index", http.StatusFound)
}

func roleDelRcode(a *app.App, w http.ResponseWriter, r *http.Request, rcode string) {
	_, _ = a.DB.Ent.RoleLevel.Delete().Where(rolelevel.RcodeEQ(rcode)).Exec(bg)
	_ = a.DB.Exec("DELETE FROM ay_role_area WHERE rcode=?", rcode)
	_, _ = a.DB.Ent.Role.Delete().Where(role.RcodeEQ(rcode)).Exec(bg)
	sortOK(w, r, "删除成功！", "/admin/Role/index")
}

func roleModForm(a *app.App, w http.ResponseWriter, r *http.Request, sess *auth.Session, kv map[string]string) {
	rcode := or(kv["rcode"], r.URL.Query().Get("rcode"))
	if rcode == "" && kv["id"] != "" {
		if one, err := a.DB.Ent.Role.Get(bg, domain.Atoi(kv["id"], 0)); err == nil {
			rcode = one.Rcode
		}
	}
	one, err := a.DB.Ent.Role.Query().Where(role.RcodeEQ(rcode)).First(bg)
	if err != nil {
		http.Redirect(w, r, "/admin/Role/index", http.StatusFound)
		return
	}
	row := store.Row(one)
	checked := roleLevelSet(a, rcode)
	acodes := roleAreaCodes(a, rcode)
	d := systemBag(a, w, r, sess, "Role")
	d.Vars["mod"] = "1"
	d.Vars["role"] = row
	d.Vars["area_checkbox"] = roleAreaCheckbox(a, acodes)
	d.Vars["menu_list"] = roleMenuList(a, checked)
	d.Get["rcode"] = rcode
	writePHP(a, w, "system/role.html", d)
}

func roleLevelSet(a *app.App, rcode string) map[string]bool {
	out := map[string]bool{}
	rows, _ := a.DB.Ent.RoleLevel.Query().Where(rolelevel.RcodeEQ(rcode)).All(bg)
	for _, r := range rows {
		out[r.Level] = true
	}
	return out
}

func roleAreaCodes(a *app.App, rcode string) map[string]bool {
	out := map[string]bool{}
	rows, _ := a.DB.Maps("SELECT acode FROM ay_role_area WHERE rcode=?", rcode)
	for _, r := range rows {
		out[r["acode"]] = true
	}
	return out
}

func roleAreaCheckbox(a *app.App, checked map[string]bool) string {
	tree := areaTree(a)
	return areaCheckboxHTML(tree, checked)
}

func areaTree(a *app.App) map[string][]map[string]string {
	rows, _ := domain.Areas(a.DB)
	kids := map[string][]map[string]string{}
	for _, row := range rows {
		p := or(row["pcode"], "0")
		kids[p] = append(kids[p], row)
	}
	for _, list := range kids {
		for _, row := range list {
			if len(kids[row["acode"]]) > 0 {
				row["son"] = "1"
			}
		}
	}
	return kids
}

func areaCheckboxHTML(kids map[string][]map[string]string, checked map[string]bool) string {
	var walk func(pcode string) string
	walk = func(pcode string) string {
		var b strings.Builder
		for _, row := range kids[pcode] {
			if len(kids[row["acode"]]) > 0 {
				b.WriteString(walk(row["acode"]))
				continue
			}
			ch := ""
			if checked != nil && checked[row["acode"]] {
				ch = ` checked="checked"`
			}
			b.WriteString(`<input type="checkbox" name="acodes[]" value="` + esc(row["acode"]) + `" title="` + esc(row["acode"]+"-"+row["name"]) + `"` + ch + `>`)
		}
		return b.String()
	}
	return walk("0")
}

func menuLevelActions(a *app.App) map[string][]map[string]string {
	out := map[string][]map[string]string{}
	rows, err := a.DB.Maps(`SELECT ma.mcode, t.item, t.value FROM ay_menu_action ma
		JOIN ay_type t ON t.value=ma.action AND t.tcode='T101'`)
	if err != nil || len(rows) == 0 {
		return out
	}
	for _, row := range rows {
		out[row["mcode"]] = append(out[row["mcode"]], map[string]string{"item": row["item"], "value": row["value"]})
	}
	return out
}

func menuAdminTree(a *app.App) map[string][]map[string]string {
	list, _ := a.DB.Ent.Menu.Query().Where(menu.StatusEQ("1"), menu.URLHasPrefix("/admin/")).Order(menu.BySorting(), menu.ByID()).All(bg)
	rows := store.Rows(list)
	kids := map[string][]map[string]string{}
	for _, row := range rows {
		if strings.Contains(row["url"], "/admin/M") {
			continue
		}
		p := or(row["pcode"], "0")
		kids[p] = append(kids[p], row)
	}
	return kids
}

func roleMenuList(a *app.App, checked map[string]bool) string {
	kids := menuAdminTree(a)
	actions := menuLevelActions(a)
	var walk func(pcode, blank string) string
	walk = func(pcode, blank string) string {
		var b strings.Builder
		for _, m := range kids[pcode] {
			ico := "fa-folder-o"
			if len(kids[m["mcode"]]) > 0 {
				ico = "fa-folder-open-o"
			}
			url := m["url"]
			if url == "" {
				continue
			}
			pre := menuPreURL(url)
			ch := ""
			if checked != nil && checked[url] {
				ch = ` checked="checked"`
			}
			boxes := `<input type="checkbox" class="checkbox" lay-skin="primary" name="levels[]" value="` + esc(url) + `" title="浏览"` + ch + `>`
			if acts := actions[m["mcode"]]; len(acts) > 0 {
				for _, act := range acts {
					u := pre + act["value"]
					ch2 := ""
					if checked != nil && checked[u] {
						ch2 = ` checked="checked"`
					}
					boxes += `<input type="checkbox" class="checkbox" lay-skin="primary" name="levels[]" value="` + esc(u) + `" title="` + esc(act["item"]) + `"` + ch2 + `>`
				}
			}
			b.WriteString(`<div class="layui-row"><div class="layui-col-md3 layui-col-lg2" style="margin-top:10px;">` + blank + ` <i class="fa ` + ico + `"></i> ` + esc(m["name"]) + `</div><div class="layui-col-md9">` + boxes + `</div></div>`)
			if sub := kids[m["mcode"]]; len(sub) > 0 {
				b.WriteString(walk(m["mcode"], blank+"　　"))
			}
		}
		return b.String()
	}
	return walk("0", "")
}

func menuPreURL(url string) string {
	n := 0
	for i, c := range url {
		if c == '/' {
			n++
			if n == 3 {
				return url[:i+1]
			}
		}
	}
	return url
}

func userPage(a *app.App, w http.ResponseWriter, r *http.Request, sess *auth.Session, kv map[string]string, action string) {
	switch action {
	case "add":
		if r.Method == http.MethodPost {
			userSave(a, w, r, sess, "")
			return
		}
		http.Redirect(w, r, "/admin/User/index", http.StatusFound)
		return
	case "del":
		userDel(a, w, r, kv)
		return
	case "mod":
		if r.Method == http.MethodPost {
			userSave(a, w, r, sess, or(kv["ucode"], r.FormValue("ucode")))
			return
		}
		userModForm(a, w, r, sess, kv)
		return
	}
	if r.Method == http.MethodPost {
		if r.FormValue("op") == "del" {
			userDelByID(a, w, r)
			return
		}
		if r.FormValue("field") == "status" {
			ucode := r.FormValue("ucode")
			if ucode == "10001" {
				sortFail(w, r, "不能修改超级管理员")
				return
			}
			_, _ = a.DB.Ent.User.Update().Where(user.UcodeEQ(ucode)).SetStatus(r.FormValue("value")).SetUpdateTime(domain.Now()).Save(bg)
			sortOK(w, r, "修改成功！", "/admin/User/index")
			return
		}
		ucode := ""
		if id := r.FormValue("id"); id != "" {
			if one, err := a.DB.Ent.User.Get(bg, domain.Atoi(id, 0)); err == nil {
				ucode = one.Ucode
			}
		}
		if r.FormValue("username") != "" {
			userSave(a, w, r, sess, ucode)
			return
		}
		userSave(a, w, r, sess, "")
		return
	}
	if kv["ucode"] != "" || kv["id"] != "" {
		userModForm(a, w, r, sess, kv)
		return
	}
	d := systemBag(a, w, r, sess, "User")
	d.Vars["list"] = "1"
	d.Vars["users"] = userRows(a, kv)
	d.Vars["roles"] = roleRows(a)
	writePHP(a, w, "system/user.html", d)
}

func userRows(a *app.App, kv map[string]string) []map[string]string {
	q := a.DB.Ent.User.Query().Order(user.ByID())
	if kw := kv["keyword"]; kw != "" {
		switch kv["field"] {
		case "ucode":
			q = q.Where(user.UcodeContains(kw))
		default:
			q = q.Where(user.UsernameContains(kw))
		}
	}
	list, _ := q.All(bg)
	rows := store.Rows(list)
	rnames := map[string]string{}
	for _, ro := range roleRows(a) {
		rnames[ro["rcode"]] = ro["name"]
	}
	for _, row := range rows {
		rels, _ := a.DB.Ent.UserRole.Query().Where(userrole.UcodeEQ(row["ucode"])).All(bg)
		var names []string
		for _, rel := range rels {
			if n := rnames[rel.Rcode]; n != "" {
				names = append(names, n)
			}
		}
		row["rolename"] = strings.Join(names, ",")
	}
	return rows
}

func userRoles(r *http.Request) []string {
	if rs := r.Form["roles[]"]; len(rs) > 0 {
		return rs
	}
	if rc := r.FormValue("rcode"); rc != "" {
		return []string{rc}
	}
	return nil
}

func userSave(a *app.App, w http.ResponseWriter, r *http.Request, sess *auth.Session, ucode string) {
	name := strings.TrimSpace(r.FormValue("username"))
	if name == "" {
		sortFail(w, r, "请填写账号")
		return
	}
	now := domain.Now()
	actor := sess.Values["username"]
	roles := userRoles(r)
	if ucode == "" {
		pwd := r.FormValue("password")
		if pwd == "" {
			sortFail(w, r, "请填写密码")
			return
		}
		if rp := r.FormValue("rpassword"); rp != "" && rp != pwd {
			sortFail(w, r, "两次密码不一致")
			return
		}
		if a.DB.Conf("password_strong") != "0" && !auth.Strong(pwd) {
			sortFail(w, r, auth.StrongMsg())
			return
		}
		ucode = "U" + strconv.FormatInt(int64(domain.Atoi(maxCode(a, "ay_user", "ucode"), 10001)+1), 10)
		if err := a.DB.Ent.User.Create().
			SetUcode(ucode).SetUsername(name).SetRealname(r.FormValue("realname")).
			SetPassword(auth.Encrypt(pwd)).SetStatus(or(r.FormValue("status"), "1")).SetLoginCount(0).SetLastLoginIP("").
			SetCreateUser(actor).SetUpdateUser(actor).SetCreateTime(now).SetUpdateTime(now).
			Exec(bg); err != nil {
			sortFail(w, r, err.Error())
			return
		}
		_, _ = a.DB.Ent.UserRole.Delete().Where(userrole.UcodeEQ(ucode)).Exec(bg)
		for _, rc := range roles {
			if rc != "" {
				_ = a.DB.Ent.UserRole.Create().SetUcode(ucode).SetRcode(rc).Exec(bg)
			}
		}
		sortOK(w, r, "新增成功！", "/admin/User/index")
		return
	}
	if ucode == "10001" && sess.Values["ucode"] != "10001" {
		sortFail(w, r, "不能修改超级管理员")
		return
	}
	id := 0
	if one, err := a.DB.Ent.User.Query().Where(user.UcodeEQ(ucode)).First(bg); err == nil {
		id = one.ID
	}
	if taken, _ := a.DB.Ent.User.Query().Where(user.UsernameEQ(name), user.IDNEQ(id)).Count(bg); taken > 0 {
		sortFail(w, r, "账号已存在")
		return
	}
	upd := a.DB.Ent.User.Update().Where(user.UcodeEQ(ucode)).
		SetUsername(name).SetRealname(r.FormValue("realname")).SetStatus(or(r.FormValue("status"), "1")).
		SetUpdateUser(actor).SetUpdateTime(now)
	if p := r.FormValue("password"); p != "" {
		if rp := r.FormValue("rpassword"); rp != "" && rp != p {
			sortFail(w, r, "两次密码不一致")
			return
		}
		if a.DB.Conf("password_strong") != "0" && !auth.Strong(p) {
			sortFail(w, r, auth.StrongMsg())
			return
		}
		upd.SetPassword(auth.Encrypt(p))
	}
	if _, err := upd.Save(bg); err != nil {
		sortFail(w, r, err.Error())
		return
	}
	_, _ = a.DB.Ent.UserRole.Delete().Where(userrole.UcodeEQ(ucode)).Exec(bg)
	for _, rc := range roles {
		if rc != "" {
			_ = a.DB.Ent.UserRole.Create().SetUcode(ucode).SetRcode(rc).Exec(bg)
		}
	}
	sortOK(w, r, "修改成功！", "/admin/User/index")
}

func userModForm(a *app.App, w http.ResponseWriter, r *http.Request, sess *auth.Session, kv map[string]string) {
	ucode := kv["ucode"]
	if ucode == "" && kv["id"] != "" {
		if one, err := a.DB.Ent.User.Get(bg, domain.Atoi(kv["id"], 0)); err == nil {
			ucode = one.Ucode
		}
	}
	one, err := a.DB.Ent.User.Query().Where(user.UcodeEQ(ucode)).First(bg)
	if err != nil {
		http.Redirect(w, r, "/admin/User/index", http.StatusFound)
		return
	}
	row := store.Row(one)
	var rcodes []string
	rels, _ := a.DB.Ent.UserRole.Query().Where(userrole.UcodeEQ(ucode)).All(bg)
	for _, rel := range rels {
		rcodes = append(rcodes, rel.Rcode)
	}
	row["rcodes"] = strings.Join(rcodes, ",")
	d := systemBag(a, w, r, sess, "User")
	d.Vars["mod"] = "1"
	d.Vars["user"] = row
	d.Vars["roles"] = roleRows(a)
	d.Get["ucode"] = ucode
	writePHP(a, w, "system/user.html", d)
}

func userDel(a *app.App, w http.ResponseWriter, r *http.Request, kv map[string]string) {
	ucode := or(kv["ucode"], r.FormValue("ucode"))
	if ucode == "10001" {
		sortFail(w, r, "不能删除超级管理员")
		return
	}
	_, _ = a.DB.Ent.UserRole.Delete().Where(userrole.UcodeEQ(ucode)).Exec(bg)
	_, _ = a.DB.Ent.User.Delete().Where(user.UcodeEQ(ucode)).Exec(bg)
	sortOK(w, r, "删除成功！", "/admin/User/index")
}

func userDelByID(a *app.App, w http.ResponseWriter, r *http.Request) {
	id := domain.Atoi(r.FormValue("id"), 0)
	if id == 1 {
		alert(w, "不能删除初始用户")
		return
	}
	if one, err := a.DB.Ent.User.Get(bg, id); err == nil {
		if one.Ucode == "10001" {
			alert(w, "不能删除超级管理员")
			return
		}
		_, _ = a.DB.Ent.UserRole.Delete().Where(userrole.UcodeEQ(one.Ucode)).Exec(bg)
		_, _ = a.DB.Ent.User.Delete().Where(user.IDEQ(one.ID)).Exec(bg)
	}
	http.Redirect(w, r, "/admin/User/index", http.StatusFound)
}

func areaPage(a *app.App, w http.ResponseWriter, r *http.Request, sess *auth.Session, kv map[string]string, action string) {
	switch action {
	case "add":
		if r.Method == http.MethodPost {
			areaCreate(a, w, r, sess)
			return
		}
		http.Redirect(w, r, "/admin/Area/index", http.StatusFound)
		return
	case "del":
		areaDel(a, w, r, kv)
		return
	case "mod":
		if r.Method == http.MethodPost {
			areaUpdate(a, w, r, sess, kv)
			return
		}
		areaModForm(a, w, r, sess, kv)
		return
	}
	if r.Method == http.MethodPost {
		if r.FormValue("op") == "del" {
			areaDelByID(a, w, r)
			return
		}
		if r.FormValue("id") != "" {
			areaUpdate(a, w, r, sess, map[string]string{"id": r.FormValue("id")})
			return
		}
		if r.FormValue("acode") != "" {
			areaCreate(a, w, r, sess)
			return
		}
	}
	if kv["acode"] != "" || kv["id"] != "" {
		areaModForm(a, w, r, sess, kv)
		return
	}
	d := systemBag(a, w, r, sess, "Area")
	d.Vars["list"] = "1"
	kids := areaTree(a)
	d.Vars["areas"] = flattenAreas(kids, "0", "")
	d.Vars["area_select"] = areaSelectHTML(kids, "0", "", "", "")
	writePHP(a, w, "system/area.html", d)
}

func flattenAreas(kids map[string][]map[string]string, pcode, blank string) []map[string]string {
	var out []map[string]string
	for _, row := range kids[pcode] {
		row["blank"] = blank
		if len(kids[row["acode"]]) == 0 {
			row["son"] = ""
		} else {
			row["son"] = "1"
		}
		out = append(out, row)
		out = append(out, flattenAreas(kids, row["acode"], blank+"　　")...)
	}
	return out
}

func areaSelectHTML(kids map[string][]map[string]string, pcode, blank, selected, exclude string) string {
	var b strings.Builder
	for _, row := range kids[pcode] {
		if row["acode"] == exclude {
			continue
		}
		sel := ""
		if row["acode"] == selected {
			sel = " selected"
		}
		b.WriteString("<option value='" + esc(row["acode"]) + "'" + sel + ">" + blank + esc(row["name"]) + "</option>")
		b.WriteString(areaSelectHTML(kids, row["acode"], blank+"　　", selected, exclude))
	}
	return b.String()
}

func areaCreate(a *app.App, w http.ResponseWriter, r *http.Request, sess *auth.Session) {
	now := domain.Now()
	user := sess.Values["username"]
	if err := a.DB.Ent.Area.Create().
		SetAcode(r.FormValue("acode")).SetPcode(or(r.FormValue("pcode"), "0")).SetName(r.FormValue("name")).SetDomain(r.FormValue("domain")).SetIsDefault("0").
		SetCreateUser(user).SetUpdateUser(user).SetCreateTime(now).SetUpdateTime(now).
		Exec(bg); err != nil {
		sortFail(w, r, err.Error())
		return
	}
	sortOK(w, r, "新增成功！", "/admin/Area/index")
}

func areaUpdate(a *app.App, w http.ResponseWriter, r *http.Request, sess *auth.Session, kv map[string]string) {
	id := domain.Atoi(or(kv["id"], r.FormValue("id")), 0)
	_, err := a.DB.Ent.Area.Update().Where(area.IDEQ(id)).
		SetName(r.FormValue("name")).SetDomain(r.FormValue("domain")).SetPcode(or(r.FormValue("pcode"), "0")).
		SetUpdateUser(sess.Values["username"]).SetUpdateTime(domain.Now()).Save(bg)
	if err != nil {
		sortFail(w, r, err.Error())
		return
	}
	sortOK(w, r, "修改成功！", "/admin/Area/index")
}

func areaModForm(a *app.App, w http.ResponseWriter, r *http.Request, sess *auth.Session, kv map[string]string) {
	acode := kv["acode"]
	if acode == "" && kv["id"] != "" {
		if one, err := a.DB.Ent.Area.Get(bg, domain.Atoi(kv["id"], 0)); err == nil {
			acode = one.Acode
		}
	}
	one, err := a.DB.Ent.Area.Query().Where(area.AcodeEQ(acode)).First(bg)
	if err != nil {
		http.Redirect(w, r, "/admin/Area/index", http.StatusFound)
		return
	}
	row := store.Row(one)
	kids := areaTree(a)
	d := systemBag(a, w, r, sess, "Area")
	d.Vars["mod"] = "1"
	d.Vars["area"] = row
	d.Vars["area_select"] = areaSelectHTML(kids, "0", "", acode, acode)
	d.Get["acode"] = acode
	writePHP(a, w, "system/area.html", d)
}

func areaDel(a *app.App, w http.ResponseWriter, r *http.Request, kv map[string]string) {
	acode := or(kv["acode"], r.FormValue("acode"))
	if acode == "" || acode == "cn" {
		sortFail(w, r, "不能删除默认区域")
		return
	}
	one, err := a.DB.Ent.Area.Query().Where(area.AcodeEQ(acode)).First(bg)
	if err != nil || one.IsDefault == "1" {
		sortFail(w, r, "不能删除默认区域")
		return
	}
	_, _ = a.DB.Ent.Area.Delete().Where(area.IDEQ(one.ID)).Exec(bg)
	sortOK(w, r, "删除成功！", "/admin/Area/index")
}

func areaDelByID(a *app.App, w http.ResponseWriter, r *http.Request) {
	one, err := a.DB.Ent.Area.Get(bg, domain.Atoi(r.FormValue("id"), 0))
	if err == nil {
		if one.Acode == "cn" || one.IsDefault == "1" {
			alert(w, "不能删除默认区域")
			return
		}
		_, _ = a.DB.Ent.Area.Delete().Where(area.IDEQ(one.ID)).Exec(bg)
	}
	http.Redirect(w, r, "/admin/Area/index", http.StatusFound)
}

func menuPage(a *app.App, w http.ResponseWriter, r *http.Request, sess *auth.Session, kv map[string]string, action string) {
	switch action {
	case "add":
		if r.Method == http.MethodPost {
			menuCreate(a, w, r, sess)
			return
		}
		http.Redirect(w, r, "/admin/Menu/index", http.StatusFound)
		return
	case "del":
		menuDel(a, w, r, kv)
		return
	case "mod":
		if r.Method == http.MethodPost {
			menuUpdate(a, w, r, sess, kv)
			return
		}
		menuModForm(a, w, r, sess, kv)
		return
	}
	if r.Method == http.MethodPost {
		if r.FormValue("op") == "del" {
			menuDelByID(a, w, r)
			return
		}
		if r.FormValue("field") == "status" {
			mcode := r.FormValue("mcode")
			_, _ = a.DB.Ent.Menu.Update().Where(menu.McodeEQ(mcode)).
				SetStatus(r.FormValue("value")).SetUpdateUser(sess.Values["username"]).SetUpdateTime(domain.Now()).Save(bg)
			sortOK(w, r, "修改成功！", "/admin/Menu/index")
			return
		}
		if r.FormValue("name") != "" {
			if r.FormValue("id") != "" {
				menuUpdate(a, w, r, sess, map[string]string{"id": r.FormValue("id")})
			} else {
				menuCreate(a, w, r, sess)
			}
			return
		}
	}
	if kv["mcode"] != "" || kv["id"] != "" {
		menuModForm(a, w, r, sess, kv)
		return
	}
	d := systemBag(a, w, r, sess, "Menu")
	d.Vars["list"] = "1"
	rows, kids := menuAllRows(a)
	d.Vars["menus"] = rows
	d.Vars["menu_select"] = menuSelectHTML(kids, "0", "", "")
	d.Vars["actions"] = menuActionTypes(a)
	writePHP(a, w, "system/menu.html", d)
}

func menuAllRows(a *app.App) ([]map[string]string, map[string][]map[string]string) {
	list, _ := a.DB.Ent.Menu.Query().Order(menu.BySorting(), menu.ByID()).All(bg)
	rows := store.Rows(list)
	kids := map[string][]map[string]string{}
	for _, row := range rows {
		p := or(row["pcode"], "0")
		kids[p] = append(kids[p], row)
	}
	flat := flattenMenus(kids, "0")
	return flat, kids
}

func flattenMenus(kids map[string][]map[string]string, pcode string) []map[string]string {
	var out []map[string]string
	for _, row := range kids[pcode] {
		if row["ico"] == "" {
			row["ico"] = "fa-circle-o"
		}
		out = append(out, row)
		out = append(out, flattenMenus(kids, row["mcode"])...)
	}
	return out
}

func menuSelectHTML(kids map[string][]map[string]string, pcode, blank, selected string) string {
	var b strings.Builder
	for _, row := range kids[pcode] {
		sel := ""
		if row["mcode"] == selected {
			sel = " selected"
		}
		b.WriteString("<option value='" + esc(row["mcode"]) + "'" + sel + ">" + blank + esc(row["name"]) + "</option>")
		b.WriteString(menuSelectHTML(kids, row["mcode"], blank+"　　", selected))
	}
	return b.String()
}

func menuActionTypes(a *app.App) []map[string]string {
	rows, _ := a.DB.Maps("SELECT item, value FROM ay_type WHERE tcode='T101' ORDER BY sorting, id")
	return rows
}

func menuActionsFor(a *app.App, mcode string) string {
	rows, _ := a.DB.Maps("SELECT action FROM ay_menu_action WHERE mcode=?", mcode)
	var acts []string
	for _, r := range rows {
		acts = append(acts, r["action"])
	}
	return strings.Join(acts, ",")
}

func menuCreate(a *app.App, w http.ResponseWriter, r *http.Request, sess *auth.Session) {
	now := domain.Now()
	actor := sess.Values["username"]
	mcode := nextMcode(a)
	if err := a.DB.Ent.Menu.Create().
		SetMcode(mcode).SetPcode(or(r.FormValue("pcode"), "0")).SetName(r.FormValue("name")).SetURL(r.FormValue("url")).
		SetSorting(domain.Atoi(r.FormValue("sorting"), 255)).SetStatus(or(r.FormValue("status"), "1")).
		SetShortcut(or(r.FormValue("shortcut"), "0")).SetIco(r.FormValue("ico")).
		SetCreateUser(actor).SetUpdateUser(actor).SetCreateTime(now).SetUpdateTime(now).
		Exec(bg); err != nil {
		sortFail(w, r, err.Error())
		return
	}
	saveMenuActions(a, mcode, r.Form["actions[]"])
	http.Redirect(w, r, "/admin/Menu/index", http.StatusFound)
}

func menuUpdate(a *app.App, w http.ResponseWriter, r *http.Request, sess *auth.Session, kv map[string]string) {
	actor := sess.Values["username"]
	now := domain.Now()
	mcode := kv["mcode"]
	if mcode == "" {
		if id := or(kv["id"], r.FormValue("id")); id != "" {
			if one, err := a.DB.Ent.Menu.Get(bg, domain.Atoi(id, 0)); err == nil {
				mcode = one.Mcode
			}
		}
	}
	_, err := a.DB.Ent.Menu.Update().Where(menu.McodeEQ(mcode)).
		SetName(r.FormValue("name")).SetURL(r.FormValue("url")).SetPcode(or(r.FormValue("pcode"), "0")).
		SetSorting(domain.Atoi(r.FormValue("sorting"), 255)).SetStatus(or(r.FormValue("status"), "1")).
		SetShortcut(or(r.FormValue("shortcut"), "0")).SetIco(r.FormValue("ico")).
		SetUpdateUser(actor).SetUpdateTime(now).Save(bg)
	if err != nil {
		sortFail(w, r, err.Error())
		return
	}
	saveMenuActions(a, mcode, r.Form["actions[]"])
	http.Redirect(w, r, "/admin/Menu/index", http.StatusFound)
}

func saveMenuActions(a *app.App, mcode string, actions []string) {
	_ = a.DB.Exec("DELETE FROM ay_menu_action WHERE mcode=?", mcode)
	for _, act := range actions {
		if act == "" {
			continue
		}
		_ = a.DB.Insert("ay_menu_action", map[string]any{"mcode": mcode, "action": act})
	}
}

func menuModForm(a *app.App, w http.ResponseWriter, r *http.Request, sess *auth.Session, kv map[string]string) {
	mcode := kv["mcode"]
	if mcode == "" && kv["id"] != "" {
		if one, err := a.DB.Ent.Menu.Get(bg, domain.Atoi(kv["id"], 0)); err == nil {
			mcode = one.Mcode
		}
	}
	one, err := a.DB.Ent.Menu.Query().Where(menu.McodeEQ(mcode)).First(bg)
	if err != nil {
		http.Redirect(w, r, "/admin/Menu/index", http.StatusFound)
		return
	}
	row := store.Row(one)
	row["actions"] = menuActionsFor(a, mcode)
	_, kids := menuAllRows(a)
	d := systemBag(a, w, r, sess, "Menu")
	d.Vars["mod"] = "1"
	d.Vars["menu"] = row
	d.Vars["menu_select"] = menuSelectHTML(kids, "0", "", or(row["pcode"], "0"))
	d.Vars["actions"] = menuActionTypes(a)
	d.Get["mcode"] = mcode
	writePHP(a, w, "system/menu.html", d)
}

func menuDel(a *app.App, w http.ResponseWriter, r *http.Request, kv map[string]string) {
	mcode := or(kv["mcode"], r.FormValue("mcode"))
	if one, err := a.DB.Ent.Menu.Query().Where(menu.McodeEQ(mcode)).First(bg); err == nil {
		if n, _ := a.DB.Ent.Menu.Query().Where(menu.PcodeEQ(one.Mcode)).Count(bg); n > 0 {
			sortFail(w, r, "请先删除子菜单")
			return
		}
		_ = a.DB.Exec("DELETE FROM ay_menu_action WHERE mcode=?", mcode)
		_, _ = a.DB.Ent.Menu.Delete().Where(menu.IDEQ(one.ID)).Exec(bg)
	}
	sortOK(w, r, "删除成功！", "/admin/Menu/index")
}

func menuDelByID(a *app.App, w http.ResponseWriter, r *http.Request) {
	one, err := a.DB.Ent.Menu.Get(bg, domain.Atoi(r.FormValue("id"), 0))
	if err == nil {
		if n, _ := a.DB.Ent.Menu.Query().Where(menu.PcodeEQ(one.Mcode)).Count(bg); n > 0 {
			alert(w, "请先删除子菜单")
			return
		}
		_ = a.DB.Exec("DELETE FROM ay_menu_action WHERE mcode=?", one.Mcode)
		_, _ = a.DB.Ent.Menu.Delete().Where(menu.IDEQ(one.ID)).Exec(bg)
	}
	http.Redirect(w, r, "/admin/Menu/index", http.StatusFound)
}

func typePage(a *app.App, w http.ResponseWriter, r *http.Request, sess *auth.Session, kv map[string]string, action string) {
	switch action {
	case "add":
		if r.Method == http.MethodPost {
			typeSave(a, w, r, sess, "")
			return
		}
		http.Redirect(w, r, "/admin/Type/index", http.StatusFound)
		return
	case "del":
		typeDel(a, w, r, kv)
		return
	case "mod":
		if r.Method == http.MethodPost {
			typeSave(a, w, r, sess, or(kv["id"], r.FormValue("id")))
			return
		}
		typeModForm(a, w, r, sess, kv)
		return
	}
	if r.Method == http.MethodPost {
		if r.FormValue("op") == "del" {
			typeDelByID(a, w, r)
			return
		}
		if r.FormValue("name") != "" {
			typeSave(a, w, r, sess, r.FormValue("id"))
			return
		}
	}
	if kv["id"] != "" {
		typeModForm(a, w, r, sess, kv)
		return
	}
	d := systemBag(a, w, r, sess, "Type")
	d.Vars["list"] = "1"
	d.Vars["types"] = typeRows(a, kv)
	d.Vars["type_select"] = typeSelectHTML(a)
	writePHP(a, w, "system/type.html", d)
}

func typeRows(a *app.App, kv map[string]string) []map[string]string {
	q := "SELECT * FROM ay_type"
	args := []any{}
	if kw := kv["keyword"]; kw != "" {
		col := kv["field"]
		if col == "" {
			col = "tcode"
		}
		q += " WHERE " + col + " LIKE ?"
		args = append(args, "%"+kw+"%")
	}
	q += " ORDER BY tcode, sorting, id"
	rows, _ := a.DB.Maps(q, args...)
	return rows
}

func typeSelectHTML(a *app.App) string {
	rows, _ := a.DB.Maps("SELECT DISTINCT tcode, name FROM ay_type ORDER BY tcode")
	var b strings.Builder
	for _, row := range rows {
		b.WriteString("<option value='" + esc(row["tcode"]) + "'>" + esc(row["name"]) + "</option>")
	}
	return b.String()
}

func typeSave(a *app.App, w http.ResponseWriter, r *http.Request, sess *auth.Session, id string) {
	now := domain.Now()
	actor := sess.Values["username"]
	if id != "" {
		err := a.DB.Exec("UPDATE ay_type SET name=?, item=?, value=?, sorting=?, update_user=?, update_time=? WHERE id=?",
			r.FormValue("name"), r.FormValue("item"), r.FormValue("value"), domain.Atoi(r.FormValue("sorting"), 0), actor, now, domain.Atoi(id, 0))
		if err != nil {
			sortFail(w, r, err.Error())
			return
		}
	} else {
		rows, _ := a.DB.Maps("SELECT COALESCE(MAX(id),0) AS n FROM ay_type")
		next := 1
		if len(rows) > 0 {
			next = domain.Atoi(rows[0]["n"], 0) + 1
		}
		err := a.DB.Insert("ay_type", map[string]any{
			"id": next, "tcode": r.FormValue("tcode"), "name": r.FormValue("name"), "item": r.FormValue("item"),
			"value": r.FormValue("value"), "sorting": domain.Atoi(r.FormValue("sorting"), 0),
			"create_user": actor, "update_user": actor, "create_time": now, "update_time": now,
		})
		if err != nil {
			sortFail(w, r, err.Error())
			return
		}
	}
	http.Redirect(w, r, "/admin/Type/index", http.StatusFound)
}

func typeModForm(a *app.App, w http.ResponseWriter, r *http.Request, sess *auth.Session, kv map[string]string) {
	rows, err := a.DB.Maps("SELECT * FROM ay_type WHERE id=?", domain.Atoi(kv["id"], 0))
	if err != nil || len(rows) == 0 {
		http.Redirect(w, r, "/admin/Type/index", http.StatusFound)
		return
	}
	d := systemBag(a, w, r, sess, "Type")
	d.Vars["mod"] = "1"
	d.Vars["type"] = rows[0]
	d.Vars["type_select"] = typeSelectHTML(a)
	d.Get["id"] = kv["id"]
	writePHP(a, w, "system/type.html", d)
}

func typeDel(a *app.App, w http.ResponseWriter, r *http.Request, kv map[string]string) {
	id := domain.Atoi(or(kv["id"], r.FormValue("id")), 0)
	if id < 7 {
		sortFail(w, r, "系统内置类型不能删除")
		return
	}
	_ = a.DB.Exec("DELETE FROM ay_type WHERE id=?", id)
	sortOK(w, r, "删除成功！", "/admin/Type/index")
}

func typeDelByID(a *app.App, w http.ResponseWriter, r *http.Request) {
	id := domain.Atoi(r.FormValue("id"), 0)
	if id >= 7 {
		_ = a.DB.Exec("DELETE FROM ay_type WHERE id=?", id)
	}
	http.Redirect(w, r, "/admin/Type/index", http.StatusFound)
}

func syslogPage(a *app.App, w http.ResponseWriter, r *http.Request, sess *auth.Session, kv map[string]string, action string) {
	if action == "clear" || (r.Method == http.MethodPost && r.FormValue("op") == "clear") {
		_, _ = a.DB.Ent.Syslog.Delete().Where(syslog.IDGT(0)).Exec(bg)
		sortOK(w, r, "清空成功！", "/admin/Syslog/index")
		return
	}
	tab := or(kv["tab"], "sys")
	if tab != "sys" && tab != "spider" && tab != "ai" {
		tab = "sys"
	}
	d := systemBag(a, w, r, sess, "Syslog")
	d.Vars["log_tab"] = tab
	d.Vars["sys_tab_url"] = "/admin/Syslog/index"
	d.Vars["spider_tab_url"] = "/admin/Syslog/index?tab=spider"
	d.Vars["ai_tab_url"] = "/admin/Syslog/index?tab=ai"
	d.Vars["pathinfo"] = ""
	d.Vars["pagebar"] = ""
	d.Vars["syslogs"] = []map[string]string{}
	d.Vars["text_logs"] = []map[string]string{}
	d.Vars["text_years"] = []string{}
	d.Vars["text_months"] = []string{}
	d.Vars["text_days"] = []map[string]string{}
	d.Vars["text_empty_month"] = ""
	d.Vars["text_empty_day"] = ""
	d.Vars["text_show_day"] = ""
	d.Vars["text_truncated"] = ""
	d.Vars["text_shown"] = "0"
	d.Vars["text_date_invalid"] = ""
	if tab == "sys" {
		list, _ := a.DB.Ent.Syslog.Query().Order(syslog.ByID(sql.OrderDesc())).Limit(100).All(bg)
		d.Vars["syslogs"] = store.Rows(list)
	} else {
		assignTextLogs(d, a.Cfg.SiteRoot, tab, kv)
	}
	if sess != nil && sess.Values["ucode"] == "" {
		if u, err := a.DB.Ent.User.Get(bg, domain.Atoi(sess.Values["id"], 0)); err == nil {
			sess.Values["ucode"] = u.Ucode
		}
	}
	writePHP(a, w, "system/syslog.html", d)
}

func assignTextLogs(d Data, siteRoot, typ string, kv map[string]string) {
	now := time.Now()
	hasY := kv["year"] != ""
	hasM := kv["month"] != ""
	hasD := kv["day"] != ""
	year := now.Year()
	month := int(now.Month())
	day := now.Day()
	if hasY {
		year = domain.Atoi(kv["year"], year)
	}
	if hasM {
		month = domain.Atoi(kv["month"], month)
	}
	if hasD {
		day = domain.Atoi(kv["day"], day)
	} else if hasY || hasM {
		day = 0
	}
	months := make([]string, 12)
	for i := 1; i <= 12; i++ {
		months[i-1] = strconv.Itoa(i)
	}
	d.Vars["text_tab"] = typ
	d.Vars["text_year"] = strconv.Itoa(year)
	d.Vars["text_month"] = strconv.Itoa(month)
	d.Vars["text_day"] = strconv.Itoa(day)
	d.Vars["text_years"] = textLogYears(siteRoot, typ)
	d.Vars["text_months"] = months
	d.Vars["text_today_url"] = "/admin/Syslog/index?tab=" + typ

	dateInvalid := year < 1970 || year > 2100 || month < 1 || month > 12
	if dateInvalid {
		d.Vars["text_date_invalid"] = "1"
		return
	}
	dayNums := textLogDays(siteRoot, typ, year, month)
	d.Vars["text_empty_month"] = flag01(len(dayNums) == 0)
	var dayRows []map[string]string
	for _, dn := range dayNums {
		cur := flag01(day == dn)
		cls := "layui-btn layui-btn-sm layui-btn-primary"
		if cur == "1" {
			cls = "layui-btn layui-btn-sm"
		}
		dayRows = append(dayRows, map[string]string{
			"day": strconv.Itoa(dn), "label": strconv.Itoa(year) + "-" + pad2(month) + "-" + pad2(dn),
			"url":     "/admin/Syslog/index?tab=" + typ + "&year=" + strconv.Itoa(year) + "&month=" + strconv.Itoa(month) + "&day=" + strconv.Itoa(dn),
			"current": cur, "btn_class": cls,
		})
	}
	d.Vars["text_days"] = dayRows
	if day <= 0 {
		return
	}
	if day > 31 || (month == 2 && day > 29) {
		d.Vars["text_date_invalid"] = "1"
		d.Vars["text_empty_month"] = ""
		return
	}
	if len(dayNums) == 0 {
		return
	}
	d.Vars["text_show_day"] = "1"
	file := textLogDayFile(siteRoot, typ, year, month, day)
	lines, trunc, exists := textLogReadTail(file)
	if !exists || len(lines) == 0 {
		d.Vars["text_empty_day"] = "1"
		return
	}
	var logs []map[string]string
	for _, line := range lines {
		if line == "" {
			continue
		}
		logs = append(logs, textLogParse(typ, line))
	}
	if len(logs) == 0 {
		d.Vars["text_empty_day"] = "1"
		return
	}
	d.Vars["text_logs"] = logs
	d.Vars["text_truncated"] = flag01(trunc)
	d.Vars["text_shown"] = strconv.Itoa(len(logs))
}

func databasePage(a *app.App, w http.ResponseWriter, r *http.Request, sess *auth.Session, kv map[string]string, action string) {
	if r.Method == http.MethodPost {
		sub := r.FormValue("submit")
		if r.FormValue("op") == "backup" || sub == "bfsqlite" {
			if _, err := backupSQLite(a); err != nil {
				sortFail(w, r, err.Error())
				return
			}
			sortOK(w, r, "备份成功！", "/admin/Database/index")
			return
		}
		if a.DB.Driver == "mysql" {
			tables := safeTableList(append(r.Form["list[]"], r.Form["list"]...))
			switch sub {
			case "yh":
				if len(tables) == 0 {
					sortFail(w, r, "请选择数据表！")
					return
				}
				if err := a.DB.Exec("OPTIMIZE TABLE " + strings.Join(quoteIdent(tables), ",")); err != nil {
					sortFail(w, r, err.Error())
					return
				}
				sortOK(w, r, "优化成功！", "/admin/Database/index")
				return
			case "xf":
				if len(tables) == 0 {
					sortFail(w, r, "请选择数据表！")
					return
				}
				if err := a.DB.Exec("REPAIR TABLE " + strings.Join(quoteIdent(tables), ",")); err != nil {
					sortFail(w, r, err.Error())
					return
				}
				sortOK(w, r, "修复成功！", "/admin/Database/index")
				return
			case "bf":
				if len(tables) == 0 {
					sortFail(w, r, "请选择数据表！")
					return
				}
				if err := backupMySQLTables(a, tables); err != nil {
					sortFail(w, r, err.Error())
					return
				}
				sortOK(w, r, "备份表成功！", "/admin/Database/index")
				return
			case "bfdb":
				all, err := mysqlTableNames(a)
				if err != nil {
					sortFail(w, r, err.Error())
					return
				}
				if err := backupMySQLTables(a, all); err != nil {
					sortFail(w, r, err.Error())
					return
				}
				sortOK(w, r, "备份数据库成功！", "/admin/Database/index")
				return
			}
		}
	}
	d := systemBag(a, w, r, sess, "Database")
	db := "sqlite"
	tables := []map[string]string{}
	sumSize, sumFree := 0.0, 0.0
	if a.DB.Driver == "mysql" {
		db = "mysql"
		tables, sumSize, sumFree = mysqlTableStatus(a)
	}
	d.Vars["db"] = db
	d.Vars["tables"] = tables
	d.Vars["db_sum_size"] = trimFloat(sumSize)
	d.Vars["db_sum_free"] = trimFloat(sumFree)
	d.Vars["db_table_count"] = strconv.Itoa(len(tables))
	writePHP(a, w, "system/database.html", d)
}

func ucenterPage(a *app.App, w http.ResponseWriter, r *http.Request, sess *auth.Session) {
	if r.Method == http.MethodPost {
		one, err := a.DB.Ent.User.Get(bg, domain.Atoi(sess.Values["id"], 0))
		if err != nil || (r.FormValue("cpassword") != "" && one.Password != auth.Encrypt(r.FormValue("cpassword"))) {
			sortFail(w, r, "当前密码不正确")
			return
		}
		upd := a.DB.Ent.User.Update().Where(user.IDEQ(one.ID)).
			SetUsername(r.FormValue("username")).SetRealname(r.FormValue("realname")).SetUpdateTime(domain.Now())
		if p := r.FormValue("password"); p != "" {
			if p != r.FormValue("rpassword") {
				sortFail(w, r, "两次密码不一致")
				return
			}
			if a.DB.Conf("password_strong") != "0" && !auth.Strong(p) {
				sortFail(w, r, auth.StrongMsg())
				return
			}
			upd.SetPassword(auth.Encrypt(p))
			sess.Values["pwsecurity"] = "1"
			_ = a.Admin.Save(sess)
		}
		if _, err := upd.Save(bg); err != nil {
			sortFail(w, r, err.Error())
			return
		}
		sess.Values["username"] = r.FormValue("username")
		sess.Values["realname"] = r.FormValue("realname")
		_ = a.Admin.Save(sess)
		sortOK(w, r, "修改成功！", "/admin/Index/home")
		return
	}
	d := systemBag(a, w, r, sess, "Index")
	d.Vars["init"] = ""
	d.Vars["init_env"] = []map[string]string{}
	d.Vars["init_mysql"] = ""
	d.Vars["init_mysql_note"] = ""
	writePHP(a, w, "system/ucenter.html", d)
}

func serverPage(a *app.App, w http.ResponseWriter, r *http.Request, sess *auth.Session) {
	d := systemBag(a, w, r, sess, "Site")
	host := r.Host
	srv := map[string]string{
		"php_os": runtime.GOOS, "php_version": runtime.Version(), "php_version_low": "", "php_min_version": "8.0",
		"server_software": "gbootcms", "server_name": host, "server_port": "", "server_addr": "127.0.0.1",
		"db_driver": a.DB.Driver, "upload_max_filesize": "20M", "post_max_size": "20M",
		"http_host": host, "memory_limit": "-", "gd": "-",
		"image_backend_config": "gd", "image_backend_effective": "gd",
		"imagick": "否", "imagick_version": "-", "imagick_usable": "否", "imagick_unusable_reason": "-",
		"imagick_policy_status": "-", "imagick_deep_probe": "-",
		"image_cap_avif": "-", "image_cap_heic": "-", "image_cap_animated_gif": "-", "image_cap_webp": "-",
		"curl": "-", "pthreads": "-", "xcache": "-", "apc": "-", "eaccelerator": "-", "wincache": "-",
		"zendopcache": "-", "memcache": "-", "memcached": "-", "extensions": "-",
		"max_file_uploads": "-",
	}
	d.Vars["server"] = srv
	writePHP(a, w, "system/server.html", d)
}

func imagickProbe(a *app.App, w http.ResponseWriter, r *http.Request) {
	front.JSON(w, 1, "检测完成")
}

func deleCachePage(a *app.App, w http.ResponseWriter, r *http.Request, sess *auth.Session, acode string) {
	if r.URL.Query().Get("type") != "" {
		front.ClearPages(a)
		front.JSON(w, 1, "缓存更新成功！")
		return
	}
	d := systemBag(a, w, r, sess, "DeleCache")
	rows := mustSorts(a, acode)
	kids := map[string][]map[string]string{}
	for _, row := range rows {
		kids[row["pcode"]] = append(kids[row["pcode"]], row)
	}
	d.Vars["sort_select"] = sortSelectHTML(kids, "0", "", "", "")
	d.Vars["sort_select2"] = sortSelectHTML(kids, "0", "", "", "")
	writePHP(a, w, "content/delecache.html", d)
}

func clearSessionPage(a *app.App, w http.ResponseWriter, r *http.Request, sess *auth.Session) {
	if sess != nil {
		entries, _ := os.ReadDir(a.Admin.Dir)
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
				continue
			}
			if strings.TrimSuffix(e.Name(), ".json") == sess.ID {
				continue
			}
			_ = os.Remove(filepath.Join(a.Admin.Dir, e.Name()))
		}
	}
	front.JSON(w, 1, "")
}
