package front

import (
	"context"
	"html"
	"io"
	"net/http"
	"net/smtp"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	entg "github.com/zn49/pbootcms-go/internal/ent"
	"github.com/zn49/pbootcms-go/internal/ent/content"
	"github.com/zn49/pbootcms-go/internal/ent/form"
	"github.com/zn49/pbootcms-go/internal/ent/formfield"

	"github.com/zn49/pbootcms-go/internal/app"
	"github.com/zn49/pbootcms-go/internal/auth"
	"github.com/zn49/pbootcms-go/internal/domain"
	"github.com/zn49/pbootcms-go/internal/store"
	"github.com/zn49/pbootcms-go/internal/urlx"
)

var bg = context.Background()

// ThrottleSubmit is false when the visitor submitted within the last 10 seconds.
func ThrottleSubmit(a *app.App, w http.ResponseWriter, r *http.Request) bool {
	s := ensureFront(a, w, r)
	if last, _ := strconv.ParseInt(s.Values["lastsub"], 10, 64); last > 0 && time.Now().Unix()-last < 10 {
		return false
	}
	return true
}

// MarkSubmit records the last successful form/message submit time.
func MarkSubmit(a *app.App, w http.ResponseWriter, r *http.Request) {
	s := ensureFront(a, w, r)
	s.Values["lastsub"] = strconv.FormatInt(time.Now().Unix(), 10)
	_ = a.Front.Save(s)
}

func post(a *app.App, w http.ResponseWriter, r *http.Request, acode, path string) {
	_ = r.ParseForm()
	switch {
	case path == "message" || strings.HasPrefix(path, "message"):
		postMessage(a, w, r, acode)
	case strings.HasPrefix(path, "form/"):
		postForm(a, w, r, acode, strings.TrimPrefix(path, "form/"))
	case strings.HasPrefix(path, "comment"):
		postComment(a, w, r, acode)
	case strings.HasPrefix(path, "member"):
		member(a, w, r, acode, hit{Kind: "member", Extra: lastPart(path)})
	default:
		JSON(w, 0, "未知提交")
	}
}

func lastPart(p string) string {
	if i := strings.LastIndex(p, "/"); i >= 0 {
		return p[i+1:]
	}
	return p
}

func postMessage(a *app.App, w http.ResponseWriter, r *http.Request, acode string) {
	if !sameOriginPOST(r) {
		JSON(w, 0, "非法来源")
		return
	}
	if a.DB.Conf("message_status") != "1" {
		JSON(w, 0, "留言已关闭")
		return
	}
	s := ensureFront(a, w, r)
	if s != nil {
		if last, _ := strconv.ParseInt(s.Values["lastsub"], 10, 64); last > 0 && time.Now().Unix()-last < 10 {
			JSON(w, 0, "您提交太频繁了，请稍后再试！")
			return
		}
	}
	if a.DB.Conf("message_rqlogin") == "1" && memberOf(a, r)["id"] == "" {
		JSONTourl(w, 0, "请先登录后再留言！", urlx.Home(Rules(a.DB), "member/login", 0))
		return
	}
	if !checkCode(a, w, r, a.DB.Conf("message_check_code") != "0") {
		return
	}
	contacts := html.EscapeString(strings.TrimSpace(r.FormValue("contacts")))
	mobile := strings.TrimSpace(r.FormValue("mobile"))
	content := html.EscapeString(strings.TrimSpace(r.FormValue("content")))
	if contacts == "" || mobile == "" || content == "" {
		JSON(w, 0, "请完整填写留言")
		return
	}
	status := "1"
	if a.DB.Conf("message_verify") != "0" {
		status = "0"
	}
	ua := r.UserAgent()
	uid := 0
	if m := memberOf(a, r); m["id"] != "" {
		uid, _ = strconv.Atoi(m["id"])
	}
	now := domain.Now()
	err := a.DB.Ent.Message.Create().
		SetAcode(acode).SetContacts(cut(contacts, 10)).SetMobile(cut(mobile, 12)).SetContent(cut(content, 500)).
		SetUserIP(ipOf(r)).SetUserOs(userOS(ua)).SetUserBs(userBS(ua)).SetRecontent("").SetStatus(status).
		SetCreateUser("guest").SetUpdateUser("guest").SetCreateTime(now).SetUpdateTime(now).SetUID(uid).
		Exec(bg)
	if err != nil {
		JSON(w, 0, "保存失败")
		return
	}
	if s != nil {
		s.Values["lastsub"] = strconv.FormatInt(time.Now().Unix(), 10)
		_ = a.Front.Save(s)
	}
	go sendMail(a, "新留言 "+contacts, content)
	ClearPages(a)
	JSON(w, 1, "提交成功")
}

func postForm(a *app.App, w http.ResponseWriter, r *http.Request, acode, fcode string) {
	if a.DB.Conf("form_status") != "1" {
		JSON(w, 0, "表单已关闭")
		return
	}
	if !ThrottleSubmit(a, w, r) {
		JSON(w, 0, "您提交太频繁了，请稍后再试！")
		return
	}
	fcode = strings.Trim(fcode, "/")
	if fcode == "1" {
		JSON(w, 0, "表单提交地址有误，留言提交请使用留言专用地址!")
		return
	}
	one, err := a.DB.Ent.Form.Query().Where(form.FcodeEQ(fcode)).First(bg)
	if entg.IsNotFound(err) || one == nil {
		JSON(w, 0, "表单不存在")
		return
	}
	if err != nil {
		JSON(w, 0, "表单不存在")
		return
	}
	fm := store.Row(one)
	need := a.DB.Conf("form_check_code") != "0"
	if need && !checkCode(a, w, r, true) {
		return
	}
	table := fm["table_name"]
	if !store.Ident(table) {
		JSON(w, 0, "表名非法")
		return
	}
	fieldRows, _ := a.DB.Ent.FormField.Query().Where(formfield.FcodeEQ(fcode)).Order(formfield.BySorting(), formfield.ByID()).All(bg)
	fields := store.Rows(fieldRows)
	row := map[string]any{"create_time": domain.Now()}
	for _, f := range fields {
		name := f["name"]
		if !store.Ident(name) {
			continue
		}
		v := strings.TrimSpace(r.FormValue(name))
		if vs := r.Form[name]; len(vs) > 1 {
			v = strings.Join(vs, ",")
		}
		v = stripPbootIf(v)
		if f["required"] == "1" && v == "" {
			JSON(w, 0, f["description"]+"不能为空")
			return
		}
		row[name] = cut(v, domain.Atoi(f["length"], 200))
	}
	if len(row) <= 1 {
		JSON(w, 0, "没有可写字段")
		return
	}
	if err := a.DB.Insert(table, row); err != nil {
		JSON(w, 0, "保存失败")
		return
	}
	MarkSubmit(a, w, r)
	if a.DB.Conf("form_send_mail") == "1" {
		go sendMail(a, "表单 "+fm["form_name"], r.PostForm.Encode())
	}
	JSON(w, 1, "提交成功")
}

func stripPbootIf(s string) string {
	return pbootIfRe.ReplaceAllString(s, "")
}

var pbootIfRe = regexp.MustCompile(`(?i)pboot:if`)

func postComment(a *app.App, w http.ResponseWriter, r *http.Request, acode string) {
	if !sameOriginPOST(r) {
		JSON(w, 0, "非法来源")
		return
	}
	if a.DB.Conf("comment_status") != "1" {
		JSON(w, 0, "评论已关闭")
		return
	}
	s := ensureFront(a, w, r)
	if s != nil {
		if last, _ := strconv.ParseInt(s.Values["lastsub"], 10, 64); last > 0 && time.Now().Unix()-last < 10 {
			JSON(w, 0, "您提交太频繁了，请稍后再试！")
			return
		}
	}
	m := memberOf(a, r)
	if m["id"] == "" && a.DB.Conf("comment_anonymous") != "1" {
		JSONTourl(w, 0, "请先注册登录后再评论！", urlx.Home(Rules(a.DB), "member/login", 0))
		return
	}
	if !checkCode(a, w, r, a.DB.Conf("comment_check_code") != "0") {
		return
	}
	id := r.FormValue("contentid")
	if id == "" {
		id = QueryVal(r, "contentid")
	}
	text := html.EscapeString(strings.TrimSpace(r.FormValue("comment")))
	if id == "" || text == "" {
		JSON(w, 0, "评论内容不能为空")
		return
	}
	uid := 0
	if m["id"] != "" {
		uid, _ = strconv.Atoi(m["id"])
	}
	pid := domain.Atoi(r.FormValue("pid"), 0)
	if pid == 0 {
		pid = domain.Atoi(QueryVal(r, "pid"), 0)
	}
	puid := domain.Atoi(r.FormValue("puid"), 0)
	status := "0"
	if a.DB.Conf("comment_verify") == "0" {
		status = "1"
	}
	ua := r.UserAgent()
	now := domain.Now()
	err := a.DB.Ent.MemberComment.Create().
		SetPid(pid).SetContentid(domain.Atoi(id, 0)).SetComment(cut(text, 1000)).SetUID(uid).SetPuid(puid).SetLikes(0).SetOppose(0).SetStatus(status).
		SetUserIP(ipOf(r)).SetUserOs(userOS(ua)).SetUserBs(userBS(ua)).SetCreateTime(now).SetUpdateUser("").SetUpdateTime(now).
		Exec(bg)
	if err != nil {
		JSON(w, 0, "保存失败")
		return
	}
	if s != nil {
		s.Values["lastsub"] = strconv.FormatInt(time.Now().Unix(), 10)
		_ = a.Front.Save(s)
	}
	if a.DB.Conf("comment_send_mail") == "1" && a.DB.Conf("message_send_to") != "" {
		go sendMail(a, "您有新的文章评论信息", "评论内容："+text)
	}
	if status == "1" {
		JSON(w, 1, "评论提交成功！")
		return
	}
	JSON(w, 1, "评论提交成功,请等待管理员审核！")
}

func checkCode(a *app.App, w http.ResponseWriter, r *http.Request, need bool) bool {
	if !need {
		return true
	}
	s := loadFront(a, r)
	want := ""
	if s != nil {
		want = s.Values["code"]
	}
	got := strings.TrimSpace(r.FormValue("checkcode"))
	if want == "" || !strings.EqualFold(got, want) {
		JSON(w, 0, "验证码错误")
		return false
	}
	delete(s.Values, "code")
	_ = a.Front.Save(s)
	return true
}

func cut(s string, n int) string {
	r := []rune(s)
	if n > 0 && len(r) > n {
		return string(r[:n])
	}
	return s
}

func ipOf(r *http.Request) string { return ClientIP(r) }

// ClientIP is the remote host, trimmed to the 11 characters ay_* stores.
func ClientIP(r *http.Request) string {
	host := r.RemoteAddr
	if i := strings.LastIndex(host, ":"); i >= 0 {
		host = host[:i]
	}
	host = strings.Trim(host, "[]")
	if len(host) > 11 {
		host = host[:11]
	}
	return host
}

func sendMail(a *app.App, subject, body string) {
	if a.DB.Conf("message_send_mail") == "0" && a.DB.Conf("form_send_mail") != "1" {
		return
	}
	host := a.DB.Conf("smtp_server")
	user := a.DB.Conf("smtp_username")
	pass := a.DB.Conf("smtp_password")
	to := mailHeader(a.DB.Conf("message_send_to"))
	subject = mailHeader(subject)
	if host == "" || user == "" || to == "" {
		return
	}
	port := a.DB.Conf("smtp_port")
	if port == "" {
		port = "25"
	}
	addr := host + ":" + port
	msg := "To: " + to + "\r\nSubject: " + subject + "\r\n\r\n" + body
	var auth smtp.Auth
	if pass != "" {
		auth = smtp.PlainAuth("", user, pass, host)
	}
	_ = smtp.SendMail(addr, auth, user, []string{to}, []byte(msg))
}

func mailHeader(s string) string {
	return strings.Map(func(r rune) rune {
		if r == '\r' || r == '\n' {
			return -1
		}
		return r
	}, strings.TrimSpace(s))
}

func vote(a *app.App, w http.ResponseWriter, r *http.Request, field string) {
	id := domain.Atoi(QueryVal(r, "id"), 0)
	if id == 0 || (field != "likes" && field != "oppose") {
		voteDone(w, r, 0, "bad")
		return
	}
	ck := field + "_" + strconv.Itoa(id)
	if _, err := r.Cookie(ck); err == nil {
		voteDone(w, r, 1, "ok")
		return
	}
	u := a.DB.Ent.Content.Update().Where(content.IDEQ(id))
	if field == "oppose" {
		u.AddOppose(1)
	} else {
		u.AddLikes(1)
	}
	_, _ = u.Save(bg)
	auth.SetCookie(w, r, &http.Cookie{Name: ck, Value: "1", MaxAge: 31536000})
	voteDone(w, r, 1, "ok")
}

func isVote(w http.ResponseWriter, r *http.Request, field string) {
	id := QueryVal(r, "id")
	ck := field + "_" + id
	if id != "" {
		if _, err := r.Cookie(ck); err == nil {
			JSON(w, 1, "yes")
			return
		}
	}
	JSON(w, 0, "no")
}

func voteDone(w http.ResponseWriter, r *http.Request, code int, msg string) {
	if wantsJSON(r) || strings.Contains(r.URL.Path, "/api/") {
		JSON(w, code, msg)
		return
	}
	ref := safeRefererPath(r)
	http.Redirect(w, r, ref, http.StatusFound)
}

// safeRefererPath returns a same-host path from Referer, or "/".
func safeRefererPath(r *http.Request) string {
	ref := strings.TrimSpace(r.Referer())
	if ref == "" {
		return "/"
	}
	u, err := url.Parse(ref)
	if err != nil {
		return "/"
	}
	if u.Scheme != "" && u.Host != "" && !strings.EqualFold(u.Host, r.Host) {
		return "/"
	}
	p := u.EscapedPath()
	if p == "" {
		p = "/"
	}
	if !strings.HasPrefix(p, "/") || strings.HasPrefix(p, "//") {
		return "/"
	}
	if u.RawQuery != "" {
		p += "?" + u.RawQuery
	}
	return p
}

func sitemap(a *app.App, w http.ResponseWriter, r *http.Request, acode, kind string) {
	if kind == "llms" {
		writeLLMS(a, w, r, acode)
		return
	}
	sorts, _ := domain.Sorts(a.DB, acode)
	enrichSortModels(a, sorts)
	rules := Rules(a.DB)
	host := scheme(r) + "://" + r.Host
	today := time.Now().Format("2006-01-02")
	type node struct {
		loc, lastmod, priority, changefreq string
	}
	var nodes []node
	nodes = append(nodes, node{host + "/", today, "1.00", "always"})
	for _, s := range sorts {
		if s["status"] == "0" || s["outlink"] != "" {
			continue
		}
		link := host + rules.Sort(s["filename"], s["urlname"], s["scode"])
		nodes = append(nodes, node{link, today, "0.80", "daily"})
		if s["type"] == "1" {
			continue
		}
		rows, _, _ := domain.Contents(a.DB, acode, []string{s["scode"]}, "date", 5000, 0, nil, "", "")
		for _, c := range rows {
			if c["outlink"] != "" {
				continue
			}
			lm := today
			if d := c["date"]; len(d) >= 10 {
				lm = d[:10]
			}
			nodes = append(nodes, node{
				host + rules.Content(c["sortfilename"], c["urlname"], c["scode"], c["id"], c["filename"]),
				lm, "0.60", "daily",
			})
		}
	}
	if kind == "sitemap_txt" {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		var lines []string
		for _, n := range nodes {
			lines = append(lines, n.loc)
		}
		_, _ = io.WriteString(w, strings.Join(lines, "\n"))
		return
	}
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	_, _ = io.WriteString(w, `<?xml version="1.0" encoding="UTF-8"?>`+"\n"+`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`+"\n")
	for _, n := range nodes {
		_, _ = io.WriteString(w, "\n<url>\n    <loc>"+xmlEsc(n.loc)+"</loc>\n    <priority>"+n.priority+"</priority>\n    <lastmod>"+xmlEsc(n.lastmod)+"</lastmod>\n    <changefreq>"+n.changefreq+"</changefreq>\n</url>")
	}
	_, _ = io.WriteString(w, "\n</urlset>")
}

func writeLLMS(a *app.App, w http.ResponseWriter, r *http.Request, acode string) {
	site, _ := domain.Site(a.DB, acode)
	rules := Rules(a.DB)
	host := scheme(r) + "://" + r.Host
	title := llmsPlain(site["title"], 200)
	if title == "" {
		title = r.Host
	}
	var blocks []string
	blocks = append(blocks, "# "+title)
	if d := llmsPlain(site["description"], 500); d != "" {
		blocks = append(blocks, "> "+d)
	}
	if intro := llmsPlain(a.DB.Conf("llms_intro"), 200); intro != "" {
		blocks = append(blocks, intro)
	}
	sorts, _ := domain.Sorts(a.DB, acode)
	byCode := map[string]map[string]string{}
	kids := map[string][]string{}
	var roots []string
	for _, s := range sorts {
		if s["status"] == "0" {
			continue
		}
		byCode[s["scode"]] = s
	}
	for _, s := range sorts {
		if s["status"] == "0" {
			continue
		}
		p := s["pcode"]
		if p != "" && p != "0" && byCode[p] != nil {
			kids[p] = append(kids[p], s["scode"])
		} else {
			roots = append(roots, s["scode"])
		}
	}
	opt := map[string]bool{}
	for _, sc := range strings.Split(a.DB.Conf("llms_optional_scodes"), ",") {
		sc = strings.TrimSpace(sc)
		if sc != "" && byCode[sc] != nil {
			opt[sc] = true
		}
	}
	for _, sc := range roots {
		if opt[sc] {
			continue
		}
		lines := llmsSection(a, acode, host, rules, sc, byCode, kids, opt)
		if len(lines) == 0 {
			continue
		}
		blocks = append(blocks, "## "+llmsPlain(byCode[sc]["name"], 200)+"\n"+strings.Join(lines, "\n"))
	}
	var optLines []string
	for sc := range opt {
		if llmsHasOptAncestor(sc, byCode, opt) {
			continue
		}
		optLines = append(optLines, llmsSection(a, acode, host, rules, sc, byCode, kids, map[string]bool{})...)
	}
	if a.DB.Conf("llms_with_sitemap") != "0" {
		optLines = append(optLines, llmsLink("Sitemap", host+urlx.Home(rules, "sitemap.xml", -1), "站点全量URL清单"))
	}
	if len(optLines) > 0 {
		blocks = append(blocks, "## Optional\n"+strings.Join(optLines, "\n"))
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Robots-Tag", "noindex, nofollow")
	_, _ = io.WriteString(w, strings.Join(blocks, "\n\n")+"\n")
}

func llmsSection(a *app.App, acode, host string, rules urlx.Rules, scode string, byCode map[string]map[string]string, kids map[string][]string, skip map[string]bool) []string {
	s := byCode[scode]
	if s == nil {
		return nil
	}
	var lines []string
	if s["outlink"] == "" {
		note := llmsNote(a, s["description"])
		url := host + rules.Sort(s["filename"], s["urlname"], s["scode"])
		if s["type"] == "1" {
			if a.DB.Conf("llms_inc_about") != "0" {
				lines = append(lines, llmsLink(s["name"], url, note))
			}
		} else {
			if a.DB.Conf("llms_inc_list") != "0" {
				lines = append(lines, llmsLink(s["name"], url, note))
			}
			if a.DB.Conf("llms_inc_content") != "0" {
				num := domain.Atoi(a.DB.Conf("llms_sort_num"), 20)
				if num < 1 {
					num = 20
				}
				if num > 500 {
					num = 500
				}
				rows, _, _ := domain.Contents(a.DB, acode, []string{scode}, "date", num, 0, nil, "", "")
				for _, c := range rows {
					if c["outlink"] != "" {
						continue
					}
					cu := host + rules.Content(c["sortfilename"], c["urlname"], c["scode"], c["id"], c["filename"])
					lines = append(lines, llmsLink(c["title"], cu, llmsNote(a, c["description"])))
				}
			}
		}
	}
	for _, child := range kids[scode] {
		if skip[child] {
			continue
		}
		lines = append(lines, llmsSection(a, acode, host, rules, child, byCode, kids, skip)...)
	}
	return lines
}

func llmsHasOptAncestor(scode string, byCode map[string]map[string]string, opt map[string]bool) bool {
	seen := map[string]bool{}
	p := byCode[scode]["pcode"]
	for p != "" && p != "0" && byCode[p] != nil && !seen[p] {
		if opt[p] {
			return true
		}
		seen[p] = true
		p = byCode[p]["pcode"]
	}
	return false
}

func llmsNote(a *app.App, desc string) string {
	if a.DB.Conf("llms_desc_source") == "none" {
		return ""
	}
	n := domain.Atoi(a.DB.Conf("llms_desc_len"), 100)
	if n < 1 {
		n = 100
	}
	return llmsPlain(desc, n)
}

func llmsLink(name, url, note string) string {
	line := "- [" + llmsPlain(name, 200) + "](" + url + ")"
	if note != "" {
		line += ": " + note
	}
	return line
}

func llmsPlain(s string, max int) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	for strings.Contains(s, "  ") {
		s = strings.ReplaceAll(s, "  ", " ")
	}
	if max > 0 {
		r := []rune(s)
		if len(r) > max {
			s = string(r[:max])
		}
	}
	return s
}

func xmlEsc(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	return s
}

func enrichSortModels(a *app.App, sorts []map[string]string) {
	models, err := a.DB.Ent.Model.Query().All(bg)
	if err != nil {
		return
	}
	by := map[string]struct{ urlname, typ string }{}
	for _, m := range models {
		by[m.Mcode] = struct{ urlname, typ string }{m.Urlname, m.Type}
	}
	for _, s := range sorts {
		if md, ok := by[s["mcode"]]; ok {
			s["urlname"] = md.urlname
			s["type"] = md.typ
		}
	}
}

func sameOriginPOST(r *http.Request) bool {
	if o := r.Header.Get("Origin"); o != "" {
		u, err := url.Parse(o)
		return err == nil && strings.EqualFold(u.Host, r.Host)
	}
	if ref := r.Referer(); ref != "" {
		u, err := url.Parse(ref)
		return err == nil && strings.EqualFold(u.Host, r.Host)
	}
	return true
}
