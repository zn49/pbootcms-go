package admin

import (
	"net/http"
	"strconv"
	"strings"

	"entgo.io/ent/dialect/sql"
	"github.com/zn49/pbootcms-go/internal/app"
	"github.com/zn49/pbootcms-go/internal/auth"
	"github.com/zn49/pbootcms-go/internal/domain"
	"github.com/zn49/pbootcms-go/internal/ent/member"
	"github.com/zn49/pbootcms-go/internal/ent/membercomment"
	"github.com/zn49/pbootcms-go/internal/store"
)

func memberPage(a *app.App, w http.ResponseWriter, r *http.Request, sess *auth.Session, kv map[string]string, action string) {
	switch action {
	case "add":
		if r.Method == http.MethodPost {
			memberAdd(a, w, r, sess)
			return
		}
		http.Redirect(w, r, "/admin/Member/index", http.StatusFound)
		return
	case "del":
		memberDel(a, w, r, kv)
		return
	case "mod":
		if r.Method == http.MethodPost {
			memberMod(a, w, r, sess, kv)
			return
		}
		memberModForm(a, w, r, sess, kv)
		return
	}
	if r.Method == http.MethodPost {
		if r.FormValue("op") == "del" {
			kv["id"] = r.FormValue("id")
			memberDel(a, w, r, kv)
			return
		}
		if r.FormValue("submit") == "verify1" || r.FormValue("submit") == "verify0" {
			memberBatchStatus(a, w, r, r.FormValue("submit") == "verify1")
			return
		}
		if r.FormValue("id") != "" {
			kv["id"] = r.FormValue("id")
			memberMod(a, w, r, sess, kv)
			return
		}
		memberAdd(a, w, r, sess)
		return
	}
	if kv["id"] != "" && kv["new"] == "" {
		// detail (more) when not explicitly editing via mod
		if r.URL.Query().Get("op") == "mod" {
			memberModForm(a, w, r, sess, kv)
			return
		}
		memberMore(a, w, r, sess, kv)
		return
	}
	d := memberBag(a, w, r, sess)
	d.Vars["list"] = "1"
	d.Vars["members"] = memberRows(a, kv)
	d.Vars["groups"] = memberGroups(a)
	d.Vars["pathinfo"] = ""
	d.Vars["pagebar"] = ""
	writePHP(a, w, "member/member.html", d)
}

func memberRows(a *app.App, kv map[string]string) []map[string]string {
	q := a.DB.Ent.Member.Query().Order(member.ByID(sql.OrderDesc())).Limit(100)
	if kw := kv["keyword"]; kw != "" {
		switch kv["field"] {
		case "useremail":
			q = q.Where(member.UseremailContains(kw))
		case "usermobile":
			q = q.Where(member.UsermobileContains(kw))
		default:
			q = q.Where(member.UsernameContains(kw))
		}
	}
	list, _ := q.All(bg)
	rows := store.Rows(list)
	gmap := map[string]string{}
	for _, g := range memberGroups(a) {
		gmap[g["id"]] = g["gname"]
	}
	for _, row := range rows {
		row["gname"] = gmap[row["gid"]]
	}
	return rows
}

func memberAdd(a *app.App, w http.ResponseWriter, r *http.Request, sess *auth.Session) {
	name := strings.TrimSpace(r.FormValue("username"))
	if name == "" || r.FormValue("password") == "" {
		sortFail(w, r, "请填写账号和密码")
		return
	}
	if a.DB.Conf("password_strong") != "0" && !auth.Strong(r.FormValue("password")) {
		sortFail(w, r, auth.StrongMsg())
		return
	}
	if n, _ := a.DB.Ent.Member.Query().Where(member.UsernameEQ(name)).Count(bg); n > 0 {
		sortFail(w, r, "账号已存在")
		return
	}
	now := domain.Now()
	next := 1
	if last, err := a.DB.Ent.Member.Query().Order(member.ByID(sql.OrderDesc())).First(bg); err == nil {
		next = last.ID + 1
	}
	err := a.DB.Ent.Member.Create().
		SetUcode("M" + strconv.Itoa(next)).SetUsername(name).SetUseremail(r.FormValue("useremail")).SetUsermobile(r.FormValue("usermobile")).
		SetNickname(or(r.FormValue("nickname"), name)).SetPassword(auth.Encrypt(r.FormValue("password"))).SetHeadpic(r.FormValue("headpic")).
		SetStatus(or(r.FormValue("status"), "1")).SetActivation("1").SetGid(or(r.FormValue("gid"), "1")).
		SetWxid("").SetQqid("").SetWbid("").SetScore(domain.Atoi(r.FormValue("score"), 0)).SetRegisterTime(now).SetLoginCount(0).
		SetLastLoginIP("").SetLastLoginTime("").SetSex("").SetBirthday("").SetQq("").
		Exec(bg)
	if err != nil {
		sortFail(w, r, err.Error())
		return
	}
	sortOK(w, r, "新增成功！", "/admin/Member/index")
}

func memberMod(a *app.App, w http.ResponseWriter, r *http.Request, sess *auth.Session, kv map[string]string) {
	id := domain.Atoi(or(kv["id"], r.FormValue("id")), 0)
	if f := r.FormValue("field"); f == "status" {
		_, _ = a.DB.Ent.Member.Update().Where(member.IDEQ(id)).SetStatus(r.FormValue("value")).Save(bg)
		sortOK(w, r, "修改成功！", "/admin/Member/index")
		return
	}
	if list := r.Form["list[]"]; len(list) > 0 && r.FormValue("submit") != "" {
		memberBatchStatus(a, w, r, r.FormValue("submit") == "verify1")
		return
	}
	name := strings.TrimSpace(r.FormValue("username"))
	if name != "" {
		if n, _ := a.DB.Ent.Member.Query().Where(member.UsernameEQ(name), member.IDNEQ(id)).Count(bg); n > 0 {
			sortFail(w, r, "账号已存在")
			return
		}
	}
	upd := a.DB.Ent.Member.Update().Where(member.IDEQ(id)).
		SetNickname(r.FormValue("nickname")).SetUseremail(r.FormValue("useremail")).SetUsermobile(r.FormValue("usermobile")).
		SetHeadpic(r.FormValue("headpic")).SetStatus(or(r.FormValue("status"), "1")).SetGid(or(r.FormValue("gid"), "1")).
		SetScore(domain.Atoi(r.FormValue("score"), 0))
	if name != "" {
		upd.SetUsername(name)
	}
	if p := r.FormValue("password"); p != "" {
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
	sortOK(w, r, "修改成功！", "/admin/Member/index")
}

func memberBatchStatus(a *app.App, w http.ResponseWriter, r *http.Request, on bool) {
	st := "0"
	if on {
		st = "1"
	}
	ids := r.Form["list[]"]
	if len(ids) == 0 {
		ids = r.Form["list"]
	}
	for _, id := range ids {
		_, _ = a.DB.Ent.Member.Update().Where(member.IDEQ(domain.Atoi(id, 0))).SetStatus(st).Save(bg)
	}
	sortOK(w, r, "修改成功！", "/admin/Member/index")
}

func memberDel(a *app.App, w http.ResponseWriter, r *http.Request, kv map[string]string) {
	if r.Method != http.MethodPost {
		sortFail(w, r, "请使用正确方式提交！")
		return
	}
	ids := r.Form["list[]"]
	if len(ids) == 0 {
		ids = r.Form["list"]
	}
	if id := or(kv["id"], r.FormValue("id")); id != "" && len(ids) == 0 {
		ids = []string{id}
	}
	for _, id := range ids {
		_, _ = a.DB.Ent.Member.Delete().Where(member.IDEQ(domain.Atoi(id, 0))).Exec(bg)
	}
	sortOK(w, r, "删除成功！", "/admin/Member/index")
}

func memberModForm(a *app.App, w http.ResponseWriter, r *http.Request, sess *auth.Session, kv map[string]string) {
	id := domain.Atoi(kv["id"], 0)
	one, err := a.DB.Ent.Member.Get(bg, id)
	if err != nil {
		http.Redirect(w, r, "/admin/Member/index", http.StatusFound)
		return
	}
	d := memberBag(a, w, r, sess)
	d.Vars["mod"] = "1"
	d.Vars["member"] = store.Row(one)
	d.Vars["groups"] = memberGroups(a)
	d.Get["id"] = strconv.Itoa(id)
	writePHP(a, w, "member/member.html", d)
}

func memberMore(a *app.App, w http.ResponseWriter, r *http.Request, sess *auth.Session, kv map[string]string) {
	id := domain.Atoi(kv["id"], 0)
	one, err := a.DB.Ent.Member.Get(bg, id)
	if err != nil {
		http.Redirect(w, r, "/admin/Member/index", http.StatusFound)
		return
	}
	row := store.Row(one)
	for _, g := range memberGroups(a) {
		if g["id"] == row["gid"] {
			row["gname"] = g["gname"]
			break
		}
	}
	d := memberBag(a, w, r, sess)
	d.Vars["more"] = "1"
	d.Vars["member"] = row
	d.Get["id"] = strconv.Itoa(id)
	writePHP(a, w, "member/member.html", d)
}

func memberBag(a *app.App, w http.ResponseWriter, r *http.Request, sess *auth.Session) Data {
	d := bag(a, w, r, sess)
	d.Vars["backurl"] = ""
	d.Vars["pathinfo"] = ""
	d.Vars["pagebar"] = ""
	fc, _ := d.Vars["formcheck"].(string)
	attachMemberFuns(d, "Member", fc)
	return d
}

func attachMemberFuns(d Data, ctrl, fc string) {
	d.Funs["admin_stay_input"] = func(args []string) string { return `<input type="hidden" name="stay" value="">` }
	d.Funs["get_btn_back"] = func(args []string) string {
		return `<a href="/admin/` + ctrl + `/index" class="layui-btn layui-btn-primary">返 回</a>`
	}
	d.Funs["get_btn_mod"] = func(args []string) string {
		id, field, label := argAt(args, 0), or(argAt(args, 1), "id"), or(argAt(args, 2), "修改")
		return `<a href="/admin/` + ctrl + `/mod/` + field + `/` + id + `" class="layui-btn layui-btn-xs">` + label + `</a>`
	}
	d.Funs["get_btn_more"] = func(args []string) string {
		return `<a href="/admin/` + ctrl + `/index/id/` + argAt(args, 0) + `" class="layui-btn layui-btn-xs layui-btn-primary">详情</a>`
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
	d.Funs["check_level"] = func(args []string) string { return "1" }
	d.Funs["long2ip_safe"] = func(args []string) string {
		if len(args) == 0 {
			return ""
		}
		return esc(args[0])
	}
}

func memberGroups(a *app.App) []map[string]string {
	rows, err := a.DB.Maps("SELECT id,gcode,gname,description,status,lscore,uscore,create_user,update_user,create_time,update_time FROM ay_member_group ORDER BY gcode,id")
	if err != nil {
		return nil
	}
	return rows
}

func commentPage(a *app.App, w http.ResponseWriter, r *http.Request, sess *auth.Session, kv map[string]string, action string) {
	switch action {
	case "del":
		if r.Method != http.MethodPost {
			sortFail(w, r, "请使用正确方式提交！")
			return
		}
		ids := r.Form["list[]"]
		if len(ids) == 0 {
			ids = r.Form["list"]
		}
		if id := or(kv["id"], r.FormValue("id")); id != "" && len(ids) == 0 {
			ids = []string{id}
		}
		for _, id := range ids {
			_, _ = a.DB.Ent.MemberComment.Delete().Where(membercomment.IDEQ(domain.Atoi(id, 0))).Exec(bg)
		}
		sortOK(w, r, "删除成功！", "/admin/MemberComment/index")
		return
	case "mod":
		if r.Method == http.MethodPost {
			id := domain.Atoi(or(kv["id"], r.FormValue("id")), 0)
			if f := r.FormValue("field"); f == "status" {
				_, _ = a.DB.Ent.MemberComment.Update().Where(membercomment.IDEQ(id)).SetStatus(r.FormValue("value")).Save(bg)
				sortOK(w, r, "修改成功！", "/admin/MemberComment/index")
				return
			}
			if r.FormValue("submit") == "verify1" || r.FormValue("submit") == "verify0" {
				st := "0"
				if r.FormValue("submit") == "verify1" {
					st = "1"
				}
				for _, id := range append(r.Form["list[]"], r.Form["list"]...) {
					_, _ = a.DB.Ent.MemberComment.Update().Where(membercomment.IDEQ(domain.Atoi(id, 0))).SetStatus(st).Save(bg)
				}
				sortOK(w, r, "修改成功！", "/admin/MemberComment/index")
				return
			}
			_, _ = a.DB.Ent.MemberComment.Update().Where(membercomment.IDEQ(id)).SetStatus(or(r.FormValue("status"), "1")).Save(bg)
			sortOK(w, r, "修改成功！", "/admin/MemberComment/index")
			return
		}
		http.Redirect(w, r, "/admin/MemberComment/index", http.StatusFound)
		return
	}
	if r.Method == http.MethodPost {
		if r.FormValue("op") == "del" {
			_, _ = a.DB.Ent.MemberComment.Delete().Where(membercomment.IDEQ(domain.Atoi(r.FormValue("id"), 0))).Exec(bg)
		} else {
			_, _ = a.DB.Ent.MemberComment.Update().Where(membercomment.IDEQ(domain.Atoi(r.FormValue("id"), 0))).SetStatus(r.FormValue("status")).Save(bg)
		}
		http.Redirect(w, r, "/admin/MemberComment/index", http.StatusFound)
		return
	}
	if kv["id"] != "" {
		commentMore(a, w, r, sess, kv)
		return
	}
	d := commentBag(a, w, r, sess)
	d.Vars["list"] = "1"
	d.Vars["comments"] = commentRows(a, kv)
	d.Vars["pathinfo"] = ""
	d.Vars["pagebar"] = ""
	writePHP(a, w, "member/comment.html", d)
}

func commentRows(a *app.App, kv map[string]string) []map[string]string {
	list, _ := a.DB.Ent.MemberComment.Query().Order(membercomment.ByID(sql.OrderDesc())).Limit(100).All(bg)
	rows := store.Rows(list)
	uids := map[int]bool{}
	cids := map[int]bool{}
	for _, c := range list {
		uids[c.UID] = true
		cids[c.Contentid] = true
	}
	umap := map[string]map[string]string{}
	for uid := range uids {
		if m, err := a.DB.Ent.Member.Get(bg, uid); err == nil {
			umap[strconv.Itoa(uid)] = store.Row(m)
		}
	}
	tmap := map[string]string{}
	for cid := range cids {
		if c, err := a.DB.Ent.Content.Get(bg, cid); err == nil {
			tmap[strconv.Itoa(cid)] = c.Title
		}
	}
	kw := kv["keyword"]
	var out []map[string]string
	for _, row := range rows {
		if u := umap[row["uid"]]; u != nil {
			row["username"] = u["username"]
			row["nickname"] = u["nickname"]
		}
		row["title"] = tmap[row["contentid"]]
		if kw != "" {
			switch kv["field"] {
			case "a.comment":
				if !strings.Contains(row["comment"], kw) {
					continue
				}
			case "c.username":
				if !strings.Contains(row["username"], kw) {
					continue
				}
			case "c.nickname":
				if !strings.Contains(row["nickname"], kw) {
					continue
				}
			default:
				if !strings.Contains(row["title"], kw) {
					continue
				}
			}
		}
		out = append(out, row)
	}
	return out
}

func commentMore(a *app.App, w http.ResponseWriter, r *http.Request, sess *auth.Session, kv map[string]string) {
	id := domain.Atoi(kv["id"], 0)
	one, err := a.DB.Ent.MemberComment.Get(bg, id)
	if err != nil {
		http.Redirect(w, r, "/admin/MemberComment/index", http.StatusFound)
		return
	}
	row := store.Row(one)
	if m, err := a.DB.Ent.Member.Get(bg, one.UID); err == nil {
		row["username"] = m.Username
		row["nickname"] = m.Nickname
	}
	if c, err := a.DB.Ent.Content.Get(bg, one.Contentid); err == nil {
		row["title"] = c.Title
	}
	if one.Puid > 0 {
		if m, err := a.DB.Ent.Member.Get(bg, one.Puid); err == nil {
			row["pusername"] = m.Username
		}
	}
	d := commentBag(a, w, r, sess)
	d.Vars["more"] = "1"
	d.Vars["comment"] = row
	d.Get["id"] = strconv.Itoa(id)
	writePHP(a, w, "member/comment.html", d)
}

func commentBag(a *app.App, w http.ResponseWriter, r *http.Request, sess *auth.Session) Data {
	d := bag(a, w, r, sess)
	d.Vars["backurl"] = ""
	d.Vars["pathinfo"] = ""
	d.Vars["pagebar"] = ""
	fc, _ := d.Vars["formcheck"].(string)
	attachMemberFuns(d, "MemberComment", fc)
	return d
}

func memberGroupPage(a *app.App, w http.ResponseWriter, r *http.Request, sess *auth.Session, kv map[string]string, action string) {
	switch action {
	case "add":
		if r.Method == http.MethodPost {
			memberGroupSave(a, w, r, sess, 0)
			return
		}
		http.Redirect(w, r, "/admin/MemberGroup/index", http.StatusFound)
		return
	case "del":
		memberGroupDel(a, w, r, kv)
		return
	case "mod":
		if r.Method == http.MethodPost {
			memberGroupSave(a, w, r, sess, domain.Atoi(or(kv["id"], r.FormValue("id")), 0))
			return
		}
		memberGroupModForm(a, w, r, sess, kv)
		return
	}
	d := memberGroupBag(a, w, r, sess)
	d.Vars["list"] = "1"
	d.Vars["groups"] = memberGroups(a)
	d.Vars["pagebar"] = ""
	writePHP(a, w, "member/group.html", d)
}

func memberGroupSave(a *app.App, w http.ResponseWriter, r *http.Request, sess *auth.Session, id int) {
	if f := r.FormValue("field"); f == "status" && id > 0 {
		_ = a.DB.Exec("UPDATE ay_member_group SET status=?,update_user=?,update_time=? WHERE id=?",
			r.FormValue("value"), sess.Values["username"], domain.Now(), id)
		sortOK(w, r, "修改成功！", "/admin/MemberGroup/index")
		return
	}
	now := domain.Now()
	user := sess.Values["username"]
	gcode := r.FormValue("gcode")
	gname := r.FormValue("gname")
	if gcode == "" || gname == "" {
		sortFail(w, r, "编号和名称不能为空")
		return
	}
	if id == 0 {
		err := a.DB.Exec(`INSERT INTO ay_member_group(gcode,gname,description,status,lscore,uscore,create_user,update_user,create_time,update_time) VALUES(?,?,?,?,?,?,?,?,?,?)`,
			gcode, gname, r.FormValue("description"), or(r.FormValue("status"), "1"),
			domain.Atoi(r.FormValue("lscore"), 0), domain.Atoi(r.FormValue("uscore"), 0),
			user, user, now, now)
		if err != nil {
			sortFail(w, r, err.Error())
			return
		}
		sortOK(w, r, "新增成功！", "/admin/MemberGroup/index")
		return
	}
	err := a.DB.Exec(`UPDATE ay_member_group SET gcode=?,gname=?,description=?,status=?,lscore=?,uscore=?,update_user=?,update_time=? WHERE id=?`,
		gcode, gname, r.FormValue("description"), or(r.FormValue("status"), "1"),
		domain.Atoi(r.FormValue("lscore"), 0), domain.Atoi(r.FormValue("uscore"), 0),
		user, now, id)
	if err != nil {
		sortFail(w, r, err.Error())
		return
	}
	sortOK(w, r, "修改成功！", "/admin/MemberGroup/index")
}

func memberGroupDel(a *app.App, w http.ResponseWriter, r *http.Request, kv map[string]string) {
	if r.Method != http.MethodPost {
		sortFail(w, r, "请使用正确方式提交！")
		return
	}
	id := domain.Atoi(or(kv["id"], r.FormValue("id")), 0)
	_ = a.DB.Exec("DELETE FROM ay_member_group WHERE id=?", id)
	sortOK(w, r, "删除成功！", "/admin/MemberGroup/index")
}

func memberGroupModForm(a *app.App, w http.ResponseWriter, r *http.Request, sess *auth.Session, kv map[string]string) {
	id := kv["id"]
	rows, _ := a.DB.Maps("SELECT * FROM ay_member_group WHERE id=?", domain.Atoi(id, 0))
	if len(rows) == 0 {
		http.Redirect(w, r, "/admin/MemberGroup/index", http.StatusFound)
		return
	}
	d := memberGroupBag(a, w, r, sess)
	d.Vars["mod"] = "1"
	d.Vars["group"] = rows[0]
	d.Get["id"] = id
	writePHP(a, w, "member/group.html", d)
}

func memberGroupBag(a *app.App, w http.ResponseWriter, r *http.Request, sess *auth.Session) Data {
	d := bag(a, w, r, sess)
	d.Vars["backurl"] = ""
	d.Vars["pagebar"] = ""
	fc, _ := d.Vars["formcheck"].(string)
	attachMemberFuns(d, "MemberGroup", fc)
	return d
}

func memberFieldPage(a *app.App, w http.ResponseWriter, r *http.Request, sess *auth.Session, kv map[string]string, action string) {
	switch action {
	case "add":
		if r.Method == http.MethodPost {
			memberFieldSave(a, w, r, sess, 0)
			return
		}
		http.Redirect(w, r, "/admin/MemberField/index", http.StatusFound)
		return
	case "del":
		memberFieldDel(a, w, r, kv)
		return
	case "mod":
		if r.Method == http.MethodPost {
			memberFieldSave(a, w, r, sess, domain.Atoi(or(kv["id"], r.FormValue("id")), 0))
			return
		}
		memberFieldModForm(a, w, r, sess, kv)
		return
	}
	d := memberFieldBag(a, w, r, sess)
	d.Vars["list"] = "1"
	d.Vars["fields"] = memberFields(a)
	d.Vars["pagebar"] = ""
	writePHP(a, w, "member/field.html", d)
}

func memberFields(a *app.App) []map[string]string {
	rows, err := a.DB.Maps("SELECT * FROM ay_member_field ORDER BY sorting,id")
	if err != nil {
		return nil
	}
	return rows
}

func memberFieldSave(a *app.App, w http.ResponseWriter, r *http.Request, sess *auth.Session, id int) {
	if id > 0 {
		if f := r.FormValue("field"); f == "status" || f == "required" {
			q := "UPDATE ay_member_field SET status=?,update_user=?,update_time=? WHERE id=?"
			if f == "required" {
				q = "UPDATE ay_member_field SET required=?,update_user=?,update_time=? WHERE id=?"
			}
			_ = a.DB.Exec(q, r.FormValue("value"), sess.Values["username"], domain.Now(), id)
			sortOK(w, r, "修改成功！", "/admin/MemberField/index")
			return
		}
	}
	now := domain.Now()
	user := sess.Values["username"]
	name := r.FormValue("name")
	if name == "" || !store.Ident(name) {
		sortFail(w, r, "字段名非法")
		return
	}
	if id == 0 {
		err := a.DB.Exec(`INSERT INTO ay_member_field(name,length,required,description,sorting,status,create_user,update_user,create_time,update_time) VALUES(?,?,?,?,?,?,?,?,?,?)`,
			name, domain.Atoi(r.FormValue("length"), 20), or(r.FormValue("required"), "0"), r.FormValue("description"),
			domain.Atoi(r.FormValue("sorting"), 255), or(r.FormValue("status"), "1"),
			user, user, now, now)
		if err != nil {
			sortFail(w, r, err.Error())
			return
		}
		_ = a.DB.AddColumn("ay_member", name, "TEXT", "varchar(255) NOT NULL DEFAULT ''")
		sortOK(w, r, "新增成功！", "/admin/MemberField/index")
		return
	}
	err := a.DB.Exec(`UPDATE ay_member_field SET length=?,required=?,description=?,sorting=?,status=?,update_user=?,update_time=? WHERE id=?`,
		domain.Atoi(r.FormValue("length"), 20), or(r.FormValue("required"), "0"), r.FormValue("description"),
		domain.Atoi(r.FormValue("sorting"), 255), or(r.FormValue("status"), "1"),
		user, now, id)
	if err != nil {
		sortFail(w, r, err.Error())
		return
	}
	sortOK(w, r, "修改成功！", "/admin/MemberField/index")
}

func memberFieldDel(a *app.App, w http.ResponseWriter, r *http.Request, kv map[string]string) {
	if r.Method != http.MethodPost {
		sortFail(w, r, "请使用正确方式提交！")
		return
	}
	id := domain.Atoi(or(kv["id"], r.FormValue("id")), 0)
	// shortcut: keep ay_member column; drop when schema sync exists
	_ = a.DB.Exec("DELETE FROM ay_member_field WHERE id=?", id)
	sortOK(w, r, "删除成功！", "/admin/MemberField/index")
}

func memberFieldModForm(a *app.App, w http.ResponseWriter, r *http.Request, sess *auth.Session, kv map[string]string) {
	id := kv["id"]
	rows, _ := a.DB.Maps("SELECT * FROM ay_member_field WHERE id=?", domain.Atoi(id, 0))
	if len(rows) == 0 {
		http.Redirect(w, r, "/admin/MemberField/index", http.StatusFound)
		return
	}
	d := memberFieldBag(a, w, r, sess)
	d.Vars["mod"] = "1"
	d.Vars["field"] = rows[0]
	d.Get["id"] = id
	writePHP(a, w, "member/field.html", d)
}

func memberFieldBag(a *app.App, w http.ResponseWriter, r *http.Request, sess *auth.Session) Data {
	d := bag(a, w, r, sess)
	d.Vars["backurl"] = ""
	d.Vars["pagebar"] = ""
	fc, _ := d.Vars["formcheck"].(string)
	attachMemberFuns(d, "MemberField", fc)
	return d
}
