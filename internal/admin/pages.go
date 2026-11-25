package admin

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"html"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"gbootcms/internal/app"
	"gbootcms/internal/auth"
	"gbootcms/internal/config"
	"gbootcms/internal/domain"
	"gbootcms/internal/fsperm"
	"gbootcms/internal/ent/area"
	"gbootcms/internal/ent/company"
	"gbootcms/internal/ent/content"
	"gbootcms/internal/ent/contentsort"
	"gbootcms/internal/ent/menu"
	"gbootcms/internal/ent/rolelevel"
	"gbootcms/internal/ent/site"
	entuser "gbootcms/internal/ent/user"
	"gbootcms/internal/ent/userrole"
	"gbootcms/internal/front"
	"gbootcms/internal/store"
)

var uploadExt = map[string]bool{
	// no svg: stored XSS when served as image/svg+xml
	"jpg": true, "jpeg": true, "png": true, "gif": true, "webp": true,
	"xls": true, "xlsx": true, "doc": true, "docx": true, "ppt": true, "pptx": true,
	"rar": true, "zip": true, "pdf": true, "txt": true, "mp4": true, "mp3": true,
}

// Serve is the admin site.
func Serve(a *app.App, w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/admin")
	path = strings.Trim(path, "/")
	if path == "" {
		path = "Index/index"
	}
	parts := strings.Split(path, "/")
	ctrl, action := parts[0], "index"
	if len(parts) > 1 {
		action = parts[1]
	}
	// mylayui posts to /admin/index/upload (lowercase)
	if strings.EqualFold(ctrl, "Index") {
		ctrl = "Index"
	}
	if strings.EqualFold(action, "upload") {
		action = "upload"
	}
	kv := map[string]string{}
	for i := 2; i+1 < len(parts); i += 2 {
		kv[parts[i]] = parts[i+1]
	}
	for k, v := range r.URL.Query() {
		if len(v) > 0 && kv[k] == "" {
			kv[k] = v[0]
		}
	}
	if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
		_ = r.ParseMultipartForm(32 << 20)
	} else {
		_ = r.ParseForm()
	}
	pub := ctrl == "Index" && (action == "index" || action == "login")
	sess := adminSess(a, r)
	if !pub {
		if sess == nil || sess.Values["id"] == "" {
			http.Redirect(w, r, "/admin/Index/index", http.StatusFound)
			return
		}
		if !allowed(sess, "/admin/"+ctrl+"/"+action) && !allowed(sess, "/admin/"+ctrl) {
			http.Error(w, "权限不足", http.StatusForbidden)
			return
		}
	}
	acode := "cn"
	if sess != nil && sess.Values["acode"] != "" {
		acode = sess.Values["acode"]
	}
	if r.Method == http.MethodPost && action != "login" {
		fs := frontSess(a, r)
		fc := r.FormValue("formcheck")
		if fc == "" {
			fc = r.Header.Get("X-Formcheck")
		}
		if fs == nil || fc == "" || fc != fs.Values["formcheck"] {
			if action == "upload" {
				front.JSON(w, 0, "表单提交校验失败,请刷新后重试！")
				return
			}
			alert(w, "表单提交校验失败,请刷新后重试！")
			return
		}
	}
	switch ctrl + "/" + action {
	case "Index/index":
		loginPage(a, w, r, "")
	case "Index/login":
		doLogin(a, w, r)
	case "Index/loginOut":
		if sess != nil {
			_ = os.Remove(filepath.Join(a.Admin.Dir, sess.ID+".json"))
		}
		http.Redirect(w, r, "/admin/Index/index", http.StatusFound)
	case "Index/home":
		if kv["clear"] == "1" {
			front.ClearPages(a)
		}
		switch r.URL.Query().Get("action") {
		case "moddb":
			if r.Method != http.MethodPost {
				http.Error(w, "method", http.StatusMethodNotAllowed)
				return
			}
			if a.DB != nil && a.DB.Driver == "sqlite" {
				abs, err := modSQLiteDB(a)
				if err != nil {
					alert(w, "修改失败！")
					return
				}
				cfgPath := a.CfgPath
				if cfgPath == "" {
					cfgPath = "config.json"
				}
				_ = config.Patch(cfgPath, map[string]any{"dsn": relDSNForConfig(a, abs)})
				alert(w, "修改成功！")
				return
			}
			alert(w, "仅支持 sqlite")
			return
		case "resetfp":
			if r.Method != http.MethodPost {
				http.Error(w, "method", http.StatusMethodNotAllowed)
				return
			}
			resetFingerprint(a, w, r)
			return
		}
		home(a, w, r, sess, acode)
	case "Index/upload":
		upload(a, w, r)
	case "Index/area":
		if n, err := a.DB.Ent.Area.Query().Where(area.AcodeEQ(r.FormValue("acode"))).Count(bg); err == nil && n > 0 && sess != nil {
			sess.Values["acode"] = r.FormValue("acode")
			_ = a.Admin.Save(sess)
		}
		http.Redirect(w, r, "/admin/Index/home", http.StatusFound)
	case "Index/ucenter":
		ucenterPage(a, w, r, sess)
	case "Index/clearSession":
		clearSessionPage(a, w, r, sess)
	case "Index/clearCache", "Index/clearOnlySysCache":
		front.ClearPages(a)
		front.JSON(w, 1, "清理缓存成功！")
	case "Site/index", "Site/mod":
		sitePage(a, w, r, sess, acode)
	case "Company/index", "Company/mod":
		companyPage(a, w, r, sess, acode)
	case "ContentSort/index", "ContentSort/add", "ContentSort/mod", "ContentSort/del":
		sortPage(a, w, r, sess, acode, kv, action)
	case "Content/index", "Content/add", "Content/mod", "Content/del":
		contentPage(a, w, r, sess, acode, kv, false, action)
	case "Single/index", "Single/add", "Single/mod", "Single/del":
		contentPage(a, w, r, sess, acode, kv, true, action)
	case "Model/index", "Model/add", "Model/mod", "Model/del":
		modelPage(a, w, r, sess, kv, action)
	case "ExtField/index", "ExtField/add", "ExtField/mod", "ExtField/del":
		extPage(a, w, r, sess, kv, action)
	case "Slide/index", "Slide/add", "Slide/mod", "Slide/del":
		slidePage(a, w, r, sess, acode, kv, action)
	case "Link/index", "Link/add", "Link/mod", "Link/del":
		linkPage(a, w, r, sess, acode, kv, action)
	case "Message/index", "Message/mod", "Message/del":
		messagePage(a, w, r, sess, acode, kv, action)
	case "Form/index", "Form/add", "Form/mod", "Form/del", "Form/clear":
		formPage(a, w, r, sess, kv, action)
	case "Tags/index", "Tags/add", "Tags/mod", "Tags/del":
		tagsPage(a, w, r, sess, acode, kv, action)
	case "Label/index", "Label/add", "Label/mod", "Label/del":
		labelPage(a, w, r, sess, action, kv)
	case "Member/index", "Member/add", "Member/mod", "Member/del":
		memberPage(a, w, r, sess, kv, action)
	case "MemberComment/index", "MemberComment/mod", "MemberComment/del":
		commentPage(a, w, r, sess, kv, action)
	case "MemberGroup/index", "MemberGroup/add", "MemberGroup/mod", "MemberGroup/del":
		memberGroupPage(a, w, r, sess, kv, action)
	case "MemberField/index", "MemberField/add", "MemberField/mod", "MemberField/del":
		memberFieldPage(a, w, r, sess, kv, action)
	case "Role/index", "Role/add", "Role/mod", "Role/del":
		rolePage(a, w, r, sess, kv, action)
	case "User/index", "User/add", "User/mod", "User/del":
		userPage(a, w, r, sess, kv, action)
	case "Area/index", "Area/add", "Area/mod", "Area/del":
		areaPage(a, w, r, sess, kv, action)
	case "Config/index":
		configPage(a, w, r, sess)
	case "Menu/index", "Menu/add", "Menu/mod", "Menu/del":
		menuPage(a, w, r, sess, kv, action)
	case "Syslog/index", "Syslog/clear":
		syslogPage(a, w, r, sess, kv, action)
	case "Type/index", "Type/add", "Type/mod", "Type/del":
		typePage(a, w, r, sess, kv, action)
	case "Database/index", "Database/mod":
		databasePage(a, w, r, sess, kv, action)
	case "Site/server":
		serverPage(a, w, r, sess)
	case "Site/imagickProbe":
		imagickProbe(a, w, r)
	case "DeleCache/index":
		deleCachePage(a, w, r, sess, acode)
	case "ImageExt/index":
		imageExtPage(a, w, r, sess)
	case "ImageExt/do_ext":
		imageExtDo(a, w, r)
	case "ImageExt/checkDataFile":
		imageExtCheckDataFile(a, w, r)
	case "Ai/ping", "Ai/generate", "Ai/rewrite", "Ai/tdk", "Ai/alt":
		aiCall(a, w, r, action)
	default:
		http.NotFound(w, r)
	}
}

func adminSess(a *app.App, r *http.Request) *auth.Session {
	c, err := r.Cookie(CookieName())
	if err != nil {
		return nil
	}
	return a.Admin.Load(c.Value)
}

func frontSess(a *app.App, r *http.Request) *auth.Session {
	c, err := r.Cookie(FrontCookieName())
	if err != nil {
		return nil
	}
	return a.Front.Load(c.Value)
}

func allowed(s *auth.Session, path string) bool {
	if s.Values["id"] == "1" {
		return true
	}
	switch path {
	case "/admin/Index/home", "/admin/Index/loginOut", "/admin/Index/upload", "/admin/Index/area", "/admin/Index/ucenter",
		"/admin/Index/clearSession", "/admin/Index/clearCache", "/admin/Index/clearOnlySysCache",
		"/admin/DeleCache", "/admin/DeleCache/index":
		return true
	}
	for _, l := range strings.Split(s.Values["levels"], "\n") {
		l = strings.TrimSpace(l)
		if l == "" {
			continue
		}
		if l == path || strings.HasPrefix(path, strings.TrimRight(l, "/")+"/") {
			return true
		}
	}
	return false
}

func bag(a *app.App, w http.ResponseWriter, r *http.Request, sess *auth.Session) Data {
	fs := frontSess(a, r)
	if fs == nil {
		fs, _ = a.Front.New()
		if fs != nil {
			auth.SessionCookie(w, r, FrontCookieName(), fs.ID)
		}
	}
	if fs.Values["formcheck"] == "" {
		fs.Values["formcheck"] = randHex()
		_ = a.Front.Save(fs)
	}
	menus := []map[string]string{}
	list, _ := a.DB.Ent.Menu.Query().Where(menu.StatusEQ("1")).Order(menu.BySorting(), menu.ByID()).All(bg)
	rows := store.Rows(list)
	for _, row := range rows {
		if menuGroupURL(row["url"]) {
			continue
		}
		if sess != nil && !allowed(sess, row["url"]) && sess.Values["id"] != "1" {
			continue
		}
		menus = append(menus, row)
	}
	session := map[string]string{}
	if sess != nil {
		session = sess.Values
		if session["realname"] == "" {
			session["realname"] = session["username"]
		}
	}
	ctrl := ""
	if parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/"); len(parts) >= 2 {
		ctrl = parts[1]
	}
	get := map[string]string{}
	for k, vs := range r.URL.Query() {
		if len(vs) > 0 {
			get[k] = vs[0]
		}
	}
	return Data{
		Vars: map[string]any{
			"menus": menus, "menu_html": menuTree(rows, sess, r.URL.Path), "area_html": areaForm(a, sess),
			"formcheck": fs.Values["formcheck"], "title": "管理中心",
			"controller": ctrl, "url": r.URL.Path, "admin_path": Path(),
			"on1": "layui-this", "show1": "layui-show", "on2": "", "show2": "", "tab2": "新增", "haspane": "0",
		},
		Session: session,
		Get:     get,
		Funs: map[string]func([]string) string{
			"check_level": func(args []string) string {
				if sess == nil {
					return ""
				}
				if sess.Values["id"] == "1" {
					return "1"
				}
				return ""
			},
		},
	}
}

// menuGroupURL is true for placeholder group nodes like /admin/M1001/index,
// not real controllers such as /admin/Member/index.
func menuGroupURL(href string) bool {
	href = strings.ToLower(href)
	const p = "/admin/m"
	i := strings.Index(href, p)
	if i < 0 {
		return false
	}
	rest := href[i+len(p):]
	return len(rest) > 0 && rest[0] >= '0' && rest[0] <= '9'
}

func menuHit(path, href string) bool {
	path = strings.ToLower(strings.TrimRight(strings.Split(path, "?")[0], "/"))
	href = strings.ToLower(strings.TrimRight(strings.Split(href, "?")[0], "/"))
	if href == "" || menuGroupURL(href) {
		return false
	}
	ps := strings.Split(strings.Trim(path, "/"), "/")
	hs := strings.Split(strings.Trim(href, "/"), "/")
	if len(ps) < 2 || len(hs) < 2 || ps[0] != "admin" || hs[0] != "admin" {
		return false
	}
	return ps[1] == hs[1] && ps[1] != "index"
}

func menuTree(rows []map[string]string, sess *auth.Session, path string) string {
	kids := map[string][]map[string]string{}
	var tops []map[string]string
	for _, row := range rows {
		if row["pcode"] == "" || row["pcode"] == "0" {
			tops = append(tops, row)
			continue
		}
		kids[row["pcode"]] = append(kids[row["pcode"]], row)
	}
	var b strings.Builder
	for _, top := range tops {
		var visible []map[string]string
		for _, c := range kids[top["mcode"]] {
			if menuGroupURL(c["url"]) {
				continue
			}
			if sess != nil && sess.Values["id"] != "1" && !allowed(sess, c["url"]) {
				continue
			}
			visible = append(visible, c)
		}
		if len(visible) == 0 && menuGroupURL(top["url"]) {
			continue
		}
		open := ""
		for _, c := range visible {
			if menuHit(path, c["url"]) {
				open = " layui-nav-itemed"
			}
		}
		b.WriteString(`<li class="layui-nav-item` + open + `">`)
		if len(visible) == 0 {
			b.WriteString(`<a href="` + html.EscapeString(top["url"]) + `"><i class="fa ` + html.EscapeString(top["ico"]) + `"></i> ` + html.EscapeString(top["name"]) + `</a>`)
		} else {
			b.WriteString(`<a href="javascript:;"><i class="fa ` + html.EscapeString(top["ico"]) + `"></i> ` + html.EscapeString(top["name"]) + `</a><dl class="layui-nav-child">`)
			for _, c := range visible {
				dd := ""
				if menuHit(path, c["url"]) {
					dd = ` class="layui-this"`
				}
				b.WriteString(`<dd` + dd + `><a href="` + html.EscapeString(c["url"]) + `"><i class="fa ` + html.EscapeString(c["ico"]) + `"></i> ` + html.EscapeString(c["name"]) + `</a></dd>`)
			}
			b.WriteString(`</dl>`)
		}
		b.WriteString(`</li>`)
	}
	return b.String()
}

func areaForm(a *app.App, sess *auth.Session) string {
	rows, _ := domain.Areas(a.DB)
	if len(rows) <= 1 || sess == nil {
		return ""
	}
	cur := sess.Values["acode"]
	var b strings.Builder
	b.WriteString(`<form method="post" action="/admin/Index/area" class="area-select"><input type="hidden" name="formcheck" value="{$formcheck}"><div class="layui-col-xs8"><select name="acode">`)
	for _, row := range rows {
		sel := ""
		if row["acode"] == cur {
			sel = " selected"
		}
		b.WriteString(`<option value="` + html.EscapeString(row["acode"]) + `"` + sel + `>` + html.EscapeString(row["name"]) + `</option>`)
	}
	b.WriteString(`</select></div><div class="layui-col-xs4"><button type="submit" class="layui-btn layui-btn-sm">切换</button></div></form>`)
	return b.String()
}

func loginPage(a *app.App, w http.ResponseWriter, r *http.Request, errMsg string) {
	d := bag(a, w, r, nil)
	code := "0"
	if a.DB.Conf("admin_check_code") != "0" {
		code = "1"
	}
	d.Vars["checkcode"] = code
	d.Vars["error"] = errMsg
	d.Vars["title"] = "登录"
	write(w, "login.html", d)
}

func doLogin(a *app.App, w http.ResponseWriter, r *http.Request) {
	ip := loginIP(r)
	if !auth.LoginAllowed(ip, 5, 900) {
		loginPage(a, w, r, "尝试过多，请15分钟后再试")
		return
	}
	fs := frontSess(a, r)
	if fs == nil || r.FormValue("formcheck") != fs.Values["formcheck"] {
		loginPage(a, w, r, "表单提交校验失败,请刷新后重试！")
		return
	}
	burnCaptcha := func() {
		if fs != nil && fs.Values["code"] != "" {
			delete(fs.Values, "code")
			_ = a.Front.Save(fs)
		}
	}
	if a.DB.Conf("admin_check_code") != "0" {
		if !strings.EqualFold(strings.TrimSpace(r.FormValue("checkcode")), fs.Values["code"]) {
			burnCaptcha()
			auth.LoginFail(ip)
			loginPage(a, w, r, "验证码错误")
			return
		}
	}
	name := strings.TrimSpace(r.FormValue("username"))
	one, err := a.DB.Ent.User.Query().Where(entuser.UsernameEQ(name), entuser.StatusEQ("1")).First(bg)
	if err != nil || one.Password != auth.Encrypt(r.FormValue("password")) {
		burnCaptcha()
		auth.LoginFail(ip)
		loginPage(a, w, r, "账号或密码错误")
		return
	}
	burnCaptcha()
	auth.LoginOK(ip)
	row := store.Row(one)
	s, _ := a.Admin.New()
	s.Values["id"] = row["id"]
	s.Values["username"] = row["username"]
	s.Values["realname"] = row["realname"]
	s.Values["ucode"] = row["ucode"]
	s.Values["acode"] = domain.DefaultAcode(a.DB)
	s.Values["levels"] = levelsOf(a, row["ucode"])
	weak := one.Password == auth.Encrypt("123456")
	if weak {
		s.Values["pwsecurity"] = ""
	} else {
		s.Values["pwsecurity"] = "1"
	}
	_ = a.Admin.Save(s)
	auth.SessionCookie(w, r, CookieName(), s.ID)
	if weak {
		http.Redirect(w, r, "/admin/Index/ucenter", http.StatusFound)
		return
	}
	_ = a.DB.Ent.Syslog.Create().
		SetLevel("info").SetEvent("登录成功").SetUserIP(front.ClientIP(r)).SetUserOs("").SetUserBs("").
		SetCreateUser(name).SetCreateTime(domain.Now()).
		Exec(bg)
	http.Redirect(w, r, "/admin/Index/home", http.StatusFound)
}

func loginIP(r *http.Request) string {
	host := r.RemoteAddr
	if i := strings.LastIndex(host, ":"); i >= 0 {
		host = host[:i]
	}
	return strings.Trim(host, "[]")
}

func levelsOf(a *app.App, ucode string) string {
	roles, _ := a.DB.Ent.UserRole.Query().Where(userrole.UcodeEQ(ucode)).All(bg)
	var rcodes []string
	for _, r := range roles {
		rcodes = append(rcodes, r.Rcode)
	}
	if len(rcodes) == 0 {
		return ""
	}
	rows, _ := a.DB.Ent.RoleLevel.Query().Where(rolelevel.RcodeIn(rcodes...)).All(bg)
	var b strings.Builder
	for _, r := range rows {
		b.WriteString(r.Level)
		b.WriteByte('\n')
	}
	return b.String()
}

func home(a *app.App, w http.ResponseWriter, r *http.Request, sess *auth.Session, acode string) {
	d := bag(a, w, r, sess)
	userInfo := map[string]string{
		"username": sess.Values["username"], "realname": sess.Values["realname"],
		"update_time": "", "last_login_ip": "", "login_count": "0",
	}
	if one, err := a.DB.Ent.User.Get(bg, domain.Atoi(sess.Values["id"], 0)); err == nil {
		row := store.Row(one)
		userInfo = row
		if one.Password == auth.Encrypt("123456") {
			d.Session["pwsecurity"] = ""
		} else {
			d.Session["pwsecurity"] = "1"
		}
	}
	d.Vars["user_info"] = userInfo
	if dbInsecure(a) {
		d.Vars["dbsecurity"] = ""
	} else {
		d.Vars["dbsecurity"] = "1"
	}
	d.Vars["sum_msg"] = "0"
	if n, err := a.DB.Ent.Message.Query().Count(bg); err == nil {
		d.Vars["sum_msg"] = strconv.Itoa(n)
	}
	models, _ := domain.Models(a.DB)
	var modelMsg []map[string]string
	for _, m := range models {
		if m["status"] != "" && m["status"] != "1" {
			continue
		}
		cnt := 0
		sorts, _ := a.DB.Ent.ContentSort.Query().Where(contentsort.AcodeEQ(acode), contentsort.McodeEQ(m["mcode"])).All(bg)
		var scodes []string
		for _, s := range sorts {
			scodes = append(scodes, s.Scode)
		}
		if len(scodes) > 0 {
			cnt, _ = a.DB.Ent.Content.Query().Where(content.AcodeEQ(acode), content.ScodeIn(scodes...)).Count(bg)
		}
		modelMsg = append(modelMsg, map[string]string{"mcode": m["mcode"], "name": m["name"], "type": m["type"], "count": strconv.Itoa(cnt)})
	}
	d.Vars["model_msg"] = modelMsg
	host, port, _ := strings.Cut(r.Host, ":")
	if host == "" {
		host = "localhost"
	}
	d.Vars["server"] = map[string]string{
		"php_os": runtime.GOOS, "php_version": runtime.Version(), "php_version_low": "", "php_min_version": "8.0",
		"server_software": "-", "server_name": host, "server_port": port,
		"server_addr": "127.0.0.1", "db_driver": a.DB.Driver, "upload_max_filesize": "20M", "post_max_size": "20M",
	}
	d.Vars["branch"] = "3.X"
	d.Vars["revise"] = "0"
	d.Vars["snuser"] = a.DB.Conf("sn_user")
	d.Vars["site"] = ""
	d.Funs["long2ip_safe"] = func(args []string) string {
		if len(args) == 0 {
			return "0.0.0.0"
		}
		return long2ipSafe(args[0])
	}
	writePHP(a, w, "system/home.html", d)
}

func long2ipSafe(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return "0.0.0.0"
	}
	if strings.Contains(v, ".") {
		return v
	}
	n, err := strconv.ParseUint(v, 10, 32)
	if err != nil {
		return v
	}
	return fmt.Sprintf("%d.%d.%d.%d", byte(n>>24), byte(n>>16), byte(n>>8), byte(n))
}

func sitePage(a *app.App, w http.ResponseWriter, r *http.Request, sess *auth.Session, acode string) {
	if r.Method == http.MethodPost {
		theme := filepath.Base(r.FormValue("theme"))
		if theme == "" || theme == "." || theme == ".." {
			theme = "default"
		}
		_, err := a.DB.Ent.Site.Update().Where(site.AcodeEQ(acode)).
			SetTitle(r.FormValue("title")).SetSubtitle(r.FormValue("subtitle")).SetDomain(r.FormValue("domain")).
			SetLogo(r.FormValue("logo")).SetKeywords(r.FormValue("keywords")).SetDescription(r.FormValue("description")).
			SetIcp(r.FormValue("icp")).SetTheme(theme).SetCopyright(r.FormValue("copyright")).SetStatistical(r.FormValue("statistical")).
			Save(bg)
		if err != nil {
			if wantsJSON(r) {
				front.JSON(w, 0, err.Error())
				return
			}
			alert(w, err.Error())
			return
		}
		front.ClearPages(a)
		if wantsJSON(r) {
			front.JSON(w, 1, "修改成功！")
			return
		}
		http.Redirect(w, r, "/admin/Site/index", http.StatusFound)
		return
	}
	row := map[string]string{}
	if one, err := a.DB.Ent.Site.Query().Where(site.AcodeEQ(acode)).First(bg); err == nil {
		row = store.Row(one)
	}
	themes := []string{"default"}
	tplDir := filepath.Join(a.Cfg.SiteRoot, "template")
	if entries, err := os.ReadDir(tplDir); err == nil {
		themes = nil
		for _, e := range entries {
			if e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
				themes = append(themes, e.Name())
			}
		}
		if len(themes) == 0 {
			themes = []string{"default"}
		}
	}
	d := bag(a, w, r, sess)
	d.Vars["sites"] = row
	d.Vars["themes"] = themes
	writePHP(a, w, "content/site.html", d)
}

func companyPage(a *app.App, w http.ResponseWriter, r *http.Request, sess *auth.Session, acode string) {
	if r.Method == http.MethodPost {
		_, err := a.DB.Ent.Company.Update().Where(company.AcodeEQ(acode)).
			SetName(r.FormValue("name")).SetAddress(r.FormValue("address")).SetPostcode(r.FormValue("postcode")).
			SetContact(r.FormValue("contact")).SetMobile(r.FormValue("mobile")).SetPhone(r.FormValue("phone")).
			SetFax(r.FormValue("fax")).SetEmail(r.FormValue("email")).SetQq(r.FormValue("qq")).
			SetWeixin(r.FormValue("weixin")).SetBlicense(r.FormValue("blicense")).SetOther(r.FormValue("other")).
			Save(bg)
		if err != nil {
			if wantsJSON(r) {
				front.JSON(w, 0, err.Error())
				return
			}
			alert(w, err.Error())
			return
		}
		front.ClearPages(a)
		if wantsJSON(r) {
			front.JSON(w, 1, "修改成功！")
			return
		}
		http.Redirect(w, r, "/admin/Company/index", http.StatusFound)
		return
	}
	row := map[string]string{}
	if one, err := a.DB.Ent.Company.Query().Where(company.AcodeEQ(acode)).First(bg); err == nil {
		row = store.Row(one)
	}
	d := bag(a, w, r, sess)
	d.Vars["companys"] = row
	writePHP(a, w, "content/company.html", d)
}

func write(w http.ResponseWriter, name string, d Data) {
	s, err := Render(name, d)
	if err != nil {
		http.Error(w, "页面渲染失败", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(s))
}

func writePHP(a *app.App, w http.ResponseWriter, rel string, d Data) {
	s, err := RenderPHP(a.Cfg.SiteRoot, rel, d)
	if err != nil {
		http.Error(w, "页面渲染失败", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(s))
}

func wantsJSON(r *http.Request) bool {
	if r.Header.Get("X-Requested-With") == "XMLHttpRequest" {
		return true
	}
	return strings.Contains(r.Header.Get("Accept"), "application/json")
}

func alert(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte("<script>alert(" + strconv.Quote(msg) + ");history.back()</script>"))
}

func randHex() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func upload(a *app.App, w http.ResponseWriter, r *http.Request) {
	_ = r.ParseMultipartForm(20 << 20)
	f, hdr, err := r.FormFile("upload")
	if err != nil {
		f, hdr, err = r.FormFile("file")
	}
	if err != nil {
		front.JSON(w, 0, "没有文件")
		return
	}
	defer f.Close()
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(hdr.Filename), "."))
	if !uploadExt[ext] {
		front.JSON(w, 0, "不允许的扩展名")
		return
	}
	dir := filepath.Join(a.Cfg.SiteRoot, "static", "upload", "image", time.Now().Format("20060102"))
	_ = fsperm.MkdirAll(dir, fsperm.DirUpload)
	name := strconv.FormatInt(time.Now().UnixNano(), 10) + "." + ext
	dst := filepath.Join(dir, name)
	out, err := fsperm.Create(dst, fsperm.FileUpload)
	if err != nil {
		front.JSON(w, 0, "保存失败")
		return
	}
	defer out.Close()
	if _, err := io.Copy(out, f); err != nil {
		front.JSON(w, 0, "保存失败")
		return
	}
	forceWM := r.FormValue("watermark") == "1" || r.URL.Query().Get("watermark") == "1"
	if segs := strings.Split(strings.Trim(r.URL.Path, "/"), "/"); len(segs) >= 2 {
		for i := 0; i+1 < len(segs); i++ {
			if strings.EqualFold(segs[i], "watermark") && segs[i+1] == "1" {
				forceWM = true
			}
		}
	}
	if (forceWM || a.DB.Conf("watermark_open") == "1") && (ext == "jpg" || ext == "jpeg" || ext == "png") {
		front.Watermark(dst, a.DB.Conf("watermark_text"))
	}
	rel := "/static/upload/image/" + time.Now().Format("20060102") + "/" + name
	// mylayui expects data[0]
	front.JSON(w, 1, []string{rel})
}
