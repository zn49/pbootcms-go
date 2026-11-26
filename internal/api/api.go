package api

import (
	"context"
	"crypto/md5"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/zn49/pbootcms-go/internal/app"
	"github.com/zn49/pbootcms-go/internal/auth"
	"github.com/zn49/pbootcms-go/internal/domain"
	"github.com/zn49/pbootcms-go/internal/ent/content"
	"github.com/zn49/pbootcms-go/internal/ent/form"
	"github.com/zn49/pbootcms-go/internal/ent/formfield"
	"github.com/zn49/pbootcms-go/internal/ent/site"
	"github.com/zn49/pbootcms-go/internal/front"
	"github.com/zn49/pbootcms-go/internal/store"
	"github.com/zn49/pbootcms-go/internal/urlx"
)

var htmlAssetRe = regexp.MustCompile(`(?i)((?:src|href)\s*=\s*)(['"])([^'"]*static/upload/[^'"]*)(['"])`)

// Serve is the open JSON API.
func Serve(a *app.App, w http.ResponseWriter, r *http.Request) {
	if a.DB.Conf("api_open") != "1" {
		write(w, 0, "系统尚未开启API功能，请到后台配置")
		return
	}
	_ = r.ParseForm()
	// always require signature when API is open (api_auth=0 no longer disables auth)
	if msg := checkSig(a, r); msg != "" {
		write(w, 0, msg)
		return
	}
	acode := qv(r, "acode")
	if acode == "" {
		acode = domain.DefaultAcode(a.DB)
	}
	path := strings.ToLower(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api"), "/"))
	switch path {
	case "site", "cms/site":
		writeSite(a, w, r, acode)
	case "company", "cms/company":
		writeCompany(a, w, r, acode)
	case "label", "cms/label":
		all, _ := domain.Labels(a.DB)
		if name := qv(r, "name"); name != "" {
			write(w, 1, all[name])
			return
		}
		write(w, 1, all)
	case "nav", "cms/nav":
		writeNav(a, w, acode, qv(r, "scode"))
	case "position", "cms/position":
		writePosition(a, w, acode, qv(r, "scode"))
	case "sort", "cms/sort":
		writeSort(a, w, acode, qv(r, "scode"))
	case "pics", "cms/pics":
		writePics(a, w, acode, qv(r, "id"))
	case "slide", "cms/slide":
		writeSlide(a, w, r, acode)
	case "link", "cms/link":
		writeLink(a, w, r, acode)
	case "list", "cms/list":
		writeList(a, w, r, acode, "", "")
	case "search", "cms/search":
		if r.Method != http.MethodPost {
			write(w, 0, "请使用POST提交！")
			return
		}
		writeList(a, w, r, acode, qv(r, "keyword"), qv(r, "field"))
	case "content", "cms/content":
		row, _ := domain.ContentOne(a.DB, acode, qv(r, "id"))
		if row != nil {
			row = enrichContent(a, r, row, apiOrigin(r))
		}
		write(w, 1, row)
	case "about":
		rows, _, _ := domain.Contents(a.DB, acode, []string{qv(r, "scode")}, "date", 1, 0, nil, "", "")
		if len(rows) == 0 {
			write(w, 1, nil)
			return
		}
		write(w, 1, enrichContent(a, r, rows[0], apiOrigin(r)))
	case "msg", "cms/msg":
		writeMsg(a, w, r, acode)
	case "addmsg", "cms/addmsg":
		addMsg(a, w, r, acode)
	case "form", "cms/form":
		writeForm(a, w, r)
	case "addform", "cms/addform":
		addForm(a, w, r)
	case "do/likes":
		addStat(a, w, r, qv(r, "id"), "likes")
	case "do/oppose":
		addStat(a, w, r, qv(r, "id"), "oppose")
	case "do/islikes":
		if _, err := r.Cookie("likes_" + qv(r, "id")); err == nil {
			write(w, 1, "yes")
		} else {
			write(w, 0, "no")
		}
	case "do/isoppose":
		if _, err := r.Cookie("oppose_" + qv(r, "id")); err == nil {
			write(w, 1, "yes")
		} else {
			write(w, 0, "no")
		}
	default:
		write(w, 0, "您调用的接口不存在，请核对后重试！")
	}
}

func writeSite(a *app.App, w http.ResponseWriter, r *http.Request, acode string) {
	row, err := a.DB.Ent.Site.Query().Where(site.AcodeEQ(acode)).First(r.Context())
	if err != nil {
		write(w, 1, nil)
		return
	}
	m := store.Row(row)
	m["logo"] = assetURL(a, m["logo"])
	if name := qv(r, "name"); name != "" {
		write(w, 1, m[name])
		return
	}
	write(w, 1, m)
}

func writeCompany(a *app.App, w http.ResponseWriter, r *http.Request, acode string) {
	row, _ := domain.Company(a.DB, acode)
	if row != nil {
		row = mapsClone(row)
		row["weixin"] = assetURL(a, row["weixin"])
	}
	if name := qv(r, "name"); name != "" && row != nil {
		write(w, 1, row[name])
		return
	}
	write(w, 1, row)
}

func writeNav(a *app.App, w http.ResponseWriter, acode, scode string) {
	rows, _ := domain.Sorts(a.DB, acode)
	if scode != "" {
		var kids []map[string]string
		for _, s := range rows {
			if s["pcode"] == scode {
				kids = append(kids, sortAssets(a, s))
			}
		}
		write(w, 1, kids)
		return
	}
	write(w, 1, nestSorts(a, rows, "0"))
}

func nestSorts(a *app.App, rows []map[string]string, parent string) []any {
	var out []any
	for _, r := range rows {
		p := r["pcode"]
		if p == "" {
			p = "0"
		}
		if p != parent {
			continue
		}
		r = sortAssets(a, r)
		node := map[string]any{}
		for k, v := range r {
			node[k] = v
		}
		node["son"] = nestSorts(a, rows, r["scode"])
		out = append(out, node)
	}
	return out
}

func writePosition(a *app.App, w http.ResponseWriter, acode, scode string) {
	if scode == "" {
		write(w, 0, "必须传递当前分类scode参数")
		return
	}
	rows, _ := domain.Sorts(a.DB, acode)
	write(w, 1, domain.Chain(rows, scode))
}

func writeSort(a *app.App, w http.ResponseWriter, acode, scode string) {
	if scode == "" {
		write(w, 0, "必须传递分类scode参数")
		return
	}
	rows, _ := domain.Sorts(a.DB, acode)
	for _, s := range rows {
		if s["scode"] == scode {
			write(w, 1, sortAssets(a, s))
			return
		}
	}
	write(w, 1, nil)
}

func writePics(a *app.App, w http.ResponseWriter, acode, id string) {
	if id == "" {
		write(w, 0, "必须传递内容id参数")
		return
	}
	row, _ := domain.ContentOne(a.DB, acode, id)
	if row == nil {
		write(w, 1, []string{})
		return
	}
	var pics []string
	for _, p := range strings.Split(row["pics"], ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			pics = append(pics, assetURL(a, p))
		}
	}
	if pics == nil {
		pics = []string{}
	}
	write(w, 1, pics)
}

func writeSlide(a *app.App, w http.ResponseWriter, r *http.Request, acode string) {
	gid := qv(r, "gid")
	if gid == "" {
		write(w, 0, "必须传递幻灯片分组gid参数")
		return
	}
	num := domain.Atoi(qv(r, "num"), 10)
	rows, err := domain.Slides(a.DB, acode, gid, num)
	if err != nil {
		write(w, 0, err.Error())
		return
	}
	out := make([]map[string]string, 0, len(rows))
	for _, row := range rows {
		row = mapsClone(row)
		row["pic"] = assetURL(a, row["pic"])
		out = append(out, row)
	}
	write(w, 1, out)
}

func writeLink(a *app.App, w http.ResponseWriter, r *http.Request, acode string) {
	gid := qv(r, "gid")
	if gid == "" {
		write(w, 0, "必须传递友情链接分组gid参数")
		return
	}
	num := domain.Atoi(qv(r, "num"), 20)
	rows, err := domain.Links(a.DB, acode, gid, num)
	if err != nil {
		write(w, 0, err.Error())
		return
	}
	out := make([]map[string]string, 0, len(rows))
	for _, row := range rows {
		row = mapsClone(row)
		row["logo"] = assetURL(a, row["logo"])
		out = append(out, row)
	}
	write(w, 1, out)
}

func writeList(a *app.App, w http.ResponseWriter, r *http.Request, acode, keyword, fields string) {
	num := domain.Atoi(qv(r, "num"), domain.Atoi(a.DB.Conf("pagesize"), 15))
	if num <= 0 {
		num = 15
	}
	page := domain.Atoi(qv(r, "page"), 1)
	if page < 1 {
		page = 1
	}
	sorts, _ := domain.Sorts(a.DB, acode)
	scodes := scodeSet(sorts, qv(r, "scode"))
	filters := map[string]string{}
	if t := qv(r, "tags"); t != "" {
		filters["tags"] = t
	}
	if f := qv(r, "filter"); f != "" {
		filters["filter"] = f
	}
	for k, v := range r.Form {
		if strings.HasPrefix(k, "ext_") && len(v) > 0 {
			filters[k] = v[0]
		}
	}
	for k, v := range r.URL.Query() {
		if strings.HasPrefix(k, "ext_") && len(v) > 0 && filters[k] == "" {
			filters[k] = v[0]
		}
	}
	rows, total, err := domain.Contents(a.DB, acode, scodes, qv(r, "order"), num, (page-1)*num, filters, keyword, fields)
	if err != nil {
		write(w, 0, err.Error())
		return
	}
	pages := 0
	if total > 0 {
		pages = (total + num - 1) / num
	}
	if page > 1 && page > pages {
		write(w, 0, "已经到底了！")
		return
	}
	host := apiOrigin(r)
	out := make([]map[string]string, 0, len(rows))
	for _, row := range rows {
		out = append(out, enrichContent(a, r, row, host))
	}
	write(w, 1, out)
}

func scodeSet(sorts []map[string]string, raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "*" {
		return nil
	}
	var out []string
	seen := map[string]bool{}
	for _, p := range strings.Split(raw, ",") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		for _, s := range domain.Descendants(sorts, p) {
			if !seen[s] {
				seen[s] = true
				out = append(out, s)
			}
		}
	}
	if len(out) == 0 {
		return []string{}
	}
	return out
}

func writeMsg(a *app.App, w http.ResponseWriter, r *http.Request, acode string) {
	num := domain.Atoi(qv(r, "num"), domain.Atoi(a.DB.Conf("pagesize"), 15))
	if num <= 0 {
		num = 15
	}
	page := domain.Atoi(qv(r, "page"), 1)
	if page < 1 {
		page = 1
	}
	rows, total, err := domain.Messages(a.DB, acode, num, (page-1)*num)
	if err != nil {
		write(w, 0, err.Error())
		return
	}
	pages := 0
	if total > 0 {
		pages = (total + num - 1) / num
	}
	if page > 1 && page > pages {
		write(w, 0, "已经到底了！")
		return
	}
	write(w, 1, rows)
}

func addMsg(a *app.App, w http.ResponseWriter, r *http.Request, acode string) {
	if r.Method != http.MethodPost {
		write(w, 0, "留言提交失败，请使用POST方式提交！")
		return
	}
	if a.DB.Conf("message_status") != "1" {
		write(w, 0, "系统已经关闭留言功能，请到后台开启再试！")
		return
	}
	if !front.ThrottleSubmit(a, w, r) {
		write(w, 0, "您提交太频繁了，请稍后再试！")
		return
	}
	contacts := strings.TrimSpace(r.FormValue("contacts"))
	mobile := strings.TrimSpace(r.FormValue("mobile"))
	body := strings.TrimSpace(r.FormValue("content"))
	if contacts == "" || mobile == "" || body == "" {
		write(w, 0, "请完整填写留言")
		return
	}
	status := "1"
	if a.DB.Conf("message_verify") != "0" {
		status = "0"
	}
	now := domain.Now()
	err := a.DB.Ent.Message.Create().
		SetAcode(acode).SetContacts(cut(contacts, 10)).SetMobile(cut(mobile, 12)).SetContent(cut(body, 500)).
		SetUserIP(front.ClientIP(r)).SetUserOs("unknown").SetUserBs("unknown").SetRecontent("").SetStatus(status).
		SetCreateUser("api").SetUpdateUser("api").SetCreateTime(now).SetUpdateTime(now).SetUID(0).
		Exec(r.Context())
	if err != nil {
		write(w, 0, "留言提交失败！")
		return
	}
	front.MarkSubmit(a, w, r)
	write(w, 1, "留言提交成功！")
}

func writeForm(a *app.App, w http.ResponseWriter, r *http.Request) {
	fcode := qv(r, "fcode")
	if fcode == "" {
		write(w, 0, "必须传递表单编码fcode")
		return
	}
	one, err := a.DB.Ent.Form.Query().Where(form.FcodeEQ(fcode)).First(r.Context())
	if err != nil || !store.Ident(one.TableName) {
		write(w, 0, "传递的fcode有误")
		return
	}
	num := domain.Atoi(qv(r, "num"), domain.Atoi(a.DB.Conf("pagesize"), 15))
	if num <= 0 {
		num = 15
	}
	page := domain.Atoi(qv(r, "page"), 1)
	if page < 1 {
		page = 1
	}
	tq, err := a.DB.QuoteIdent(one.TableName)
	if err != nil {
		write(w, 0, "传递的fcode有误")
		return
	}
	rows, err := a.DB.Maps("SELECT * FROM "+tq+" ORDER BY id DESC LIMIT ? OFFSET ?", num, (page-1)*num)
	if err != nil {
		write(w, 0, err.Error())
		return
	}
	write(w, 1, rows)
}

func addForm(a *app.App, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		write(w, 0, "表单提交失败，请使用POST方式提交！")
		return
	}
	fcode := qv(r, "fcode")
	if fcode == "" {
		write(w, 0, "传递的表单编码fcode有误！")
		return
	}
	if fcode == "1" {
		write(w, 0, "表单提交地址有误，留言提交请使用留言专用地址!")
		return
	}
	if a.DB.Conf("form_status") != "1" {
		write(w, 0, "系统已经关闭表单功能，请到后台开启再试！")
		return
	}
	if !front.ThrottleSubmit(a, w, r) {
		write(w, 0, "您提交太频繁了，请稍后再试！")
		return
	}
	one, err := a.DB.Ent.Form.Query().Where(form.FcodeEQ(fcode)).First(r.Context())
	if err != nil || !store.Ident(one.TableName) {
		write(w, 0, "传递的表单编码fcode有误！")
		return
	}
	fields, err := a.DB.Ent.FormField.Query().Where(formfield.FcodeEQ(fcode)).All(r.Context())
	if err != nil || len(fields) == 0 {
		write(w, 0, "接收表单不存在任何字段，请核对后重试！")
		return
	}
	row := map[string]any{"create_time": domain.Now()}
	for _, f := range fields {
		if !store.Ident(f.Name) {
			continue
		}
		val := strings.TrimSpace(r.FormValue(f.Name))
		if vs := r.Form[f.Name]; len(vs) > 1 {
			val = strings.Join(vs, ",")
		}
		val = strings.ReplaceAll(val, "pboot:if", "")
		if f.Required == "1" && val == "" {
			write(w, 0, f.Description+"不能为空！")
			return
		}
		row[f.Name] = cut(val, f.Length)
	}
	if err := a.DB.Insert(one.TableName, row); err != nil {
		write(w, 0, "表单提交失败！")
		return
	}
	front.MarkSubmit(a, w, r)
	write(w, 1, "表单提交成功！")
}

func addStat(a *app.App, w http.ResponseWriter, r *http.Request, id, col string) {
	n := domain.Atoi(id, 0)
	if n == 0 || (col != "likes" && col != "oppose") {
		write(w, 0, "bad")
		return
	}
	ck := col + "_" + strconv.Itoa(n)
	if _, err := r.Cookie(ck); err == nil {
		write(w, 1, "ok")
		return
	}
	u := a.DB.Ent.Content.Update().Where(content.IDEQ(n))
	if col == "oppose" {
		u.AddOppose(1)
	} else {
		u.AddLikes(1)
	}
	if _, err := u.Save(context.Background()); err != nil {
		write(w, 0, err.Error())
		return
	}
	auth.SetCookie(w, r, &http.Cookie{Name: ck, Value: "1", MaxAge: 31536000})
	write(w, 1, "ok")
}

func cut(s string, n int) string {
	r := []rune(s)
	if n > 0 && len(r) > n {
		return string(r[:n])
	}
	return s
}

func qv(r *http.Request, key string) string {
	if v := r.URL.Query().Get(key); v != "" {
		return v
	}
	return r.FormValue(key)
}

func checkSig(a *app.App, r *http.Request) string {
	appid := qv(r, "appid")
	ts := qv(r, "timestamp")
	sig := qv(r, "signature")
	if appid == "" || ts == "" || sig == "" {
		return "请求失败：缺少认证参数"
	}
	if appid != a.DB.Conf("api_appid") || a.DB.Conf("api_secret") == "" {
		return "请求失败：管理后台接口认证用户配置有误"
	}
	n, err := strconv.ParseInt(ts, 10, 64)
	now := time.Now().Unix()
	if err != nil || n <= 0 || now-n > 15 || n-now > 15 {
		return "请求失败：接口时间戳验证失败！"
	}
	sum := md5.Sum([]byte(appid + a.DB.Conf("api_secret") + ts))
	inner := hex.EncodeToString(sum[:])
	sum = md5.Sum([]byte(inner))
	want := hex.EncodeToString(sum[:])
	if subtle.ConstantTimeCompare([]byte(want), []byte(sig)) != 1 {
		return "请求失败：接口签名信息错误！"
	}
	return ""
}

func write(w http.ResponseWriter, code int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]any{"code": code, "data": data})
}

func apiOrigin(r *http.Request) string {
	sch := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		sch = "https"
	}
	return sch + "://" + r.Host
}

func assetURL(a *app.App, path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return path
	}
	low := strings.ToLower(path)
	if strings.HasPrefix(low, "http://") || strings.HasPrefix(low, "https://") {
		return path
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	sd := a.Cfg.SiteDir
	if sd != "" && (path == sd || strings.HasPrefix(path, sd+"/")) {
		return path
	}
	return sd + path
}

func sortAssets(a *app.App, row map[string]string) map[string]string {
	row = mapsClone(row)
	row["ico"] = assetURL(a, row["ico"])
	row["pic"] = assetURL(a, row["pic"])
	return row
}

func enrichContent(a *app.App, r *http.Request, row map[string]string, host string) map[string]string {
	row = mapsClone(row)
	row["ico"] = assetURL(a, row["ico"])
	row["enclosure"] = assetURL(a, row["enclosure"])
	if row["pics"] != "" {
		var parts []string
		for _, p := range strings.Split(row["pics"], ",") {
			p = strings.TrimSpace(p)
			if p != "" {
				parts = append(parts, assetURL(a, p))
			}
		}
		row["pics"] = strings.Join(parts, ",")
	}
	row["content"] = absolutizeHTML(a, host, row["content"])
	rules := front.Rules(a.DB)
	if row["outlink"] != "" {
		row["apilink"] = row["outlink"]
	} else {
		row["apilink"] = "/api/content?id=" + row["id"]
	}
	row["contentlink"] = rules.Content(row["sortfilename"], row["urlname"], row["scode"], row["id"], row["filename"])
	row["likeslink"] = urlx.Home(rules, "do/likes", -1) + "?id=" + row["id"]
	row["opposelink"] = urlx.Home(rules, "do/oppose", -1) + "?id=" + row["id"]
	_ = r
	return row
}

func absolutizeHTML(a *app.App, host, html string) string {
	if html == "" {
		return html
	}
	return htmlAssetRe.ReplaceAllStringFunc(html, func(m string) string {
		sub := htmlAssetRe.FindStringSubmatch(m)
		if len(sub) < 5 {
			return m
		}
		path := assetURL(a, sub[3])
		if strings.HasPrefix(strings.ToLower(path), "http://") || strings.HasPrefix(strings.ToLower(path), "https://") {
			return sub[1] + sub[2] + path + sub[4]
		}
		return sub[1] + sub[2] + host + path + sub[4]
	})
}

func mapsClone(m map[string]string) map[string]string {
	if m == nil {
		return nil
	}
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}
