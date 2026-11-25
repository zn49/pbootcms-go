package front

import (
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gbootcms/internal/app"
	"gbootcms/internal/auth"
	"gbootcms/internal/cache"
	"gbootcms/internal/domain"
	"gbootcms/internal/ent/content"
	"gbootcms/internal/store"
	"gbootcms/internal/tpl/pboot"
	"gbootcms/internal/urlx"
)

type hit struct {
	Kind    string
	Sort    map[string]string
	Content map[string]string
	Extra   string
	Page    int
	Base    string
}

// Serve handles the public site.
func Serve(a *app.App, w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/code" {
		Captcha(a, w, r)
		return
	}
	if r.URL.Path == "/qrcode" {
		QR(w, r)
		return
	}
	if a.DB.Conf("close_site") == "1" {
		note := a.DB.Conf("close_site_note")
		if note == "" {
			note = "本站维护中，请稍后再访问，带来不便，敬请谅解！"
		}
		http.Error(w, note, http.StatusServiceUnavailable)
		return
	}
	if redirectPolicy(a, w, r) {
		return
	}
	noteSpider(a, r)
	acode := Acode(a, r)
	rules := Rules(a.DB)
	path, page := requestPath(r, rules)
	if q := r.URL.Query().Get("page"); q != "" && page == 1 {
		page = domain.Atoi(q, 1)
	}
	if r.Method == http.MethodPost {
		post(a, w, r, acode, path)
		return
	}
	h := resolve(a.DB, acode, path, page, rules)
	if h.Kind == "area" {
		lg := QueryVal(r, "lg")
		if lg != "" {
			areas, _ := domain.Areas(a.DB)
			for _, ar := range areas {
				if ar["acode"] == lg {
					auth.SetCookie(w, r, &http.Cookie{Name: "lg", Value: lg, MaxAge: 86400 * 30})
					break
				}
			}
		}
		http.Redirect(w, r, a.Cfg.SiteDir+"/", http.StatusFound)
		return
	}
	if h.Kind == "home" && QueryVal(r, "keyword") != "" {
		h.Kind = "search"
		h.Base = "search"
	}
	if h.Kind == "member" || h.Kind == "comment" {
		if member(a, w, r, acode, h) {
			return
		}
	}
	if h.Kind == "likes" || h.Kind == "oppose" {
		vote(a, w, r, h.Kind)
		return
	}
	if h.Kind == "islikes" {
		isVote(w, r, "likes")
		return
	}
	if h.Kind == "isoppose" {
		isVote(w, r, "oppose")
		return
	}
	if h.Kind == "visits" {
		doVisits(a, w, r, h)
		return
	}
	if h.Kind == "llms" && a.DB.Conf("llms_open") != "1" {
		http.NotFound(w, r)
		return
	}
	if h.Kind == "sitemap" || h.Kind == "sitemap_txt" || h.Kind == "llms" {
		sitemap(a, w, r, acode, h.Kind)
		return
	}
	html, status := render(a, w, r, acode, rules, h)
	if status == 0 {
		status = 200
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(html))
}

func Rules(db *store.DB) urlx.Rules {
	suf := db.Conf("url_rule_suffix")
	if suf == "" {
		suf = ".html"
	}
	brk := db.Conf("url_break_char")
	if brk == "" {
		brk = "_"
	}
	return urlx.Rules{
		Type:        domain.Atoi(db.Conf("url_rule_type"), 3),
		Suffix:      suf,
		Break:       brk,
		ContentPath: db.Conf("url_rule_content_path") == "1",
	}
}

func Acode(a *app.App, r *http.Request) string {
	host := r.Host
	if i := strings.IndexByte(host, ':'); i >= 0 {
		host = host[:i]
	}
	areas, _ := domain.Areas(a.DB)
	for _, ar := range areas {
		if ar["domain"] != "" && strings.EqualFold(ar["domain"], host) {
			return ar["acode"]
		}
	}
	if c, err := r.Cookie("lg"); err == nil && c.Value != "" {
		for _, ar := range areas {
			if ar["acode"] == c.Value {
				return c.Value
			}
		}
	}
	return domain.DefaultAcode(a.DB)
}

func requestPath(r *http.Request, rules urlx.Rules) (string, int) {
	p := strings.Trim(r.URL.Path, "/")
	p = strings.TrimPrefix(p, "index.php/")
	p = strings.TrimPrefix(p, "index.php")
	p = strings.Trim(p, "/")
	if p != "" {
		return peelPage(p, rules)
	}
	raw, _ := url.QueryUnescape(r.URL.RawQuery)
	if raw == "" {
		return "", 1
	}
	pathPart := raw
	if i := strings.IndexAny(raw, "&?"); i >= 0 {
		pathPart = raw[:i]
	}
	pathPart = strings.Trim(pathPart, "/")
	if pathPart == "" || strings.Contains(pathPart, "=") {
		return "", 1
	}
	return peelPage(pathPart, rules)
}

func peelPage(p string, rules urlx.Rules) (string, int) {
	base, htmlSuf := urlx.StripSuffix(rules, p)
	base = strings.Trim(base, "/")
	if htmlSuf {
		return base, 1
	}
	br := rules.Break
	if br == "" {
		br = "_"
	}
	i := strings.LastIndex(base, br)
	if i > 0 {
		if n, err := strconv.Atoi(base[i+len(br):]); err == nil && n > 0 {
			return base[:i], n
		}
	}
	return base, 1
}

func resolve(db *store.DB, acode, path string, page int, rules urlx.Rules) hit {
	sorts, _ := domain.Sorts(db, acode)
	byFile := map[string]map[string]string{}
	byCode := map[string]map[string]string{}
	for _, s := range sorts {
		if s["filename"] != "" {
			byFile[s["filename"]] = s
		}
		byCode[s["scode"]] = s
	}
	h := hit{Kind: "home", Page: page}
	if path == "" {
		return h
	}
	parts := strings.Split(path, "/")
	switch strings.ToLower(parts[0]) {
	case "search", "keyword":
		h.Kind = "search"
		h.Base = "search"
		return h
	case "message":
		h.Kind = "message"
		h.Base = "message"
		if s, ok := byFile["gbook"]; ok {
			h.Sort = s
			h.Base = s["filename"]
		}
		return h
	case "form":
		h.Kind = "form"
		if len(parts) > 1 {
			h.Extra = parts[1]
		}
		return h
	case "sitemap", "sitemap_xml", "sitemap.xml":
		h.Kind = "sitemap"
		return h
	case "sitemap_txt", "sitemap.txt":
		h.Kind = "sitemap_txt"
		return h
	case "llms", "llms_txt", "llms.txt":
		h.Kind = "llms"
		return h
	case "tag":
		h.Kind = "tag"
		h.Base = path
		if len(parts) > 1 {
			h.Extra = parts[1]
		}
		return h
	case "member":
		h.Kind = "member"
		if len(parts) > 1 {
			h.Extra = parts[1]
		} else {
			h.Extra = "login"
		}
		h.Base = strings.ToLower(h.Extra)
		return h
	case "comment":
		h.Kind = "comment"
		if len(parts) > 1 {
			h.Extra = parts[1]
		}
		return h
	case "do":
		if len(parts) > 1 && parts[1] == "area" {
			h.Kind = "area"
			h.Extra = ""
			return h
		}
		if len(parts) > 1 {
			switch strings.ToLower(parts[1]) {
			case "likes", "oppose", "islikes", "isoppose", "visits":
				h.Kind = strings.ToLower(parts[1])
				if h.Kind == "visits" && len(parts) >= 4 && strings.EqualFold(parts[2], "id") {
					h.Extra = parts[3]
				}
				return h
			}
		}
	case "spider":
		h.Kind = "home"
		return h
	}
	if s, ok := byFile[path]; ok {
		return listHit(s, page)
	}
	if rules.ContentPath {
		if c, _ := domain.ContentOne(db, acode, path); c != nil {
			return contentHit(byCode[c["scode"]], c, page)
		}
	}
	rest := path
	for strings.Contains(rest, "/") {
		dir, base := splitDir(rest)
		if s, ok := byFile[dir]; ok {
			if c, _ := domain.ContentOne(db, acode, base); c != nil && c["scode"] == s["scode"] {
				return contentHit(s, c, page)
			}
		}
		if i := strings.LastIndex(dir, rules.Break); i > 0 {
			name, sid := dir[:i], dir[i+len(rules.Break):]
			if s := byCode[sid]; s != nil {
				if c, _ := domain.ContentOne(db, acode, base); c != nil && c["scode"] == sid {
					_ = name
					return contentHit(s, c, page)
				}
			}
		}
		rest = dir
		if !strings.Contains(rest, "/") {
			break
		}
	}
	h.Kind = "404"
	return h
}

func listHit(s map[string]string, page int) hit {
	base := s["filename"]
	if base == "" {
		base = "list" + "_" + s["scode"]
	}
	return hit{Kind: "list", Sort: s, Page: page, Base: base}
}

func contentHit(s, c map[string]string, page int) hit {
	if c["modeltype"] == "1" || (s != nil && s["listtpl"] == "" && s["contenttpl"] != "") {
		return hit{Kind: "content", Sort: s, Content: c, Page: page}
	}
	return hit{Kind: "content", Sort: s, Content: c, Page: page}
}

func splitDir(p string) (string, string) {
	i := strings.LastIndex(p, "/")
	if i < 0 {
		return "", p
	}
	return p[:i], p[i+1:]
}

func render(a *app.App, w http.ResponseWriter, r *http.Request, acode string, rules urlx.Rules, h hit) (string, int) {
	if h.Kind == "404" {
		return notFoundPage(a, "您访问的页面不存在，请核对后再试！"), 404
	}
	if h.Kind == "form" {
		// PHP FormController is POST-only
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		return `<script>alert("提交失败，请使用POST方式提交！");history.back()</script>`, 405
	}
	site, _ := domain.Site(a.DB, acode)
	if site == nil {
		site = map[string]string{"title": "gbootcms", "theme": "default"}
	}
	theme := site["theme"]
	if theme == "" {
		theme = "default"
	}
	wap := isWap(a, r)
	dir := tplDir(a, theme, wap)
	member := memberOf(a, r)
	key := r.URL.RequestURI() + "|" + acode + "|" + boolStr(wap) + "|" + member["id"]
	ttl := time.Duration(domain.Atoi(a.DB.Conf("tpl_html_cache_time"), 0)) * time.Second
	cacheOn := a.DB.Conf("tpl_html_cache") != "0" && member["id"] == "" && r.URL.Query().Get("keyword") == "" && h.Kind != "search"
	contentID := ""
	if h.Kind == "content" && h.Content != nil {
		contentID = h.Content["id"]
	}
	if cacheOn {
		if s, ok := a.Pages.Get(key, ttl); ok {
			return s, 200
		}
	}
	sorts, _ := domain.Sorts(a.DB, acode)
	company, _ := domain.Company(a.DB, acode)
	labels, _ := domain.Labels(a.DB)
	ctx := &pboot.Ctx{
		DB: a.DB, Acode: acode, Rules: rules,
		Site: site, Company: company, Sorts: sorts, Labels: labels,
		Sort: h.Sort, Content: h.Content, Kind: h.Kind,
		PageBase: h.Base, Member: member, TplRoot: dir,
		SchemeHost: scheme(r) + "://" + r.Host,
		RequestURI: r.URL.RequestURI(),
		Path:       r.URL.RequestURI(),
		SitePath:   a.Cfg.SiteDir,
		Query:      flatQuery(r),
		Keyword:    QueryVal(r, "keyword"),
		ClientIP:   remoteIP(r),
	}
	if h.Kind == "tag" {
		ctx.Keyword = h.Extra
		ctx.Kind = "tag"
	}
	if ctx.Query == nil {
		ctx.Query = map[string]string{}
	}
	if h.Page > 1 && ctx.Query["page"] == "" {
		ctx.Query["page"] = strconv.Itoa(h.Page)
	}
	pboot.Prepare(ctx)
	if contentID != "" && !cacheOn {
		_, _ = a.DB.Ent.Content.Update().Where(content.IDEQ(domain.Atoi(contentID, 0))).AddVisits(1).Save(bg)
	}
	file := templateFile(dir, h, site)
	body, err := pboot.RenderFile(file, ctx)
	if err != nil {
		return notFoundPage(a, "模板不存在："+filepath.Base(file)), 404
	}
	if strings.Contains(body, "{pboot:mustlogin}") && member["id"] == "" {
		http.Redirect(w, r, urlx.Home(rules, "member/login", 0), http.StatusFound)
		return "", 302
	}
	body = strings.ReplaceAll(body, "{pboot:mustlogin}", "")
	if cacheOn && contentID != "" {
		body = injectVisitsScript(body, rules, contentID)
	}
	if cacheOn {
		a.Pages.Put(key, body)
	}
	return body, 200
}

func doVisits(a *app.App, w http.ResponseWriter, r *http.Request, h hit) {
	id := domain.Atoi(h.Extra, 0)
	if id == 0 {
		id = domain.Atoi(QueryVal(r, "id"), 0)
	}
	w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
	if id == 0 {
		_, _ = w.Write([]byte("var error;"))
		return
	}
	_, _ = a.DB.Ent.Content.Update().Where(content.IDEQ(id)).AddVisits(1).Save(bg)
	_, _ = w.Write([]byte("var ok;"))
}

func injectVisitsScript(body string, rules urlx.Rules, id string) string {
	if id == "" || strings.Contains(body, "do/visits") {
		return body
	}
	src := urlx.Home(rules, "do/visits/id/"+id, -1)
	script := "<script src='" + src + "' async='async'></script>\n"
	low := strings.ToLower(body)
	if i := strings.LastIndex(low, "</body>"); i >= 0 {
		return body[:i] + script + body[i:]
	}
	return body + script
}

func templateFile(dir string, h hit, site map[string]string) string {
	name := "index.html"
	switch h.Kind {
	case "home":
		name = "index.html"
	case "list":
		name = firstNonEmpty(h.Sort["listtpl"], h.Sort["slisttpl"], "newslist.html")
	case "content":
		name = firstNonEmpty(h.Sort["contenttpl"], h.Sort["scontenttpl"], "news.html")
		if h.Content != nil && h.Content["modeltype"] == "1" && name == "news.html" {
			name = "about.html"
		}
	case "search", "tag":
		name = "search.html"
	case "message":
		name = "message.html"
	case "comment":
		if h.Extra == "my" {
			name = "member/mycomment.html"
		}
	case "member":
		switch strings.ToLower(h.Extra) {
		case "login", "register", "retrieve", "ucenter", "umodify":
			name = "member/" + strings.ToLower(h.Extra) + ".html"
		default:
			name = "member/login.html"
		}
	}
	if h.Kind == "list" && h.Sort != nil && h.Sort["listtpl"] == "" && h.Sort["contenttpl"] != "" {
		name = h.Sort["contenttpl"]
	}
	return filepath.Join(dir, name)
}

func firstNonEmpty(v ...string) string {
	for _, s := range v {
		if s != "" {
			return s
		}
	}
	return "index.html"
}

func tplDir(a *app.App, theme string, wap bool) string {
	base := filepath.Join(a.Cfg.SiteRoot, "template", theme)
	if wap {
		if st, err := os.Stat(filepath.Join(base, "wap")); err == nil && st.IsDir() {
			return filepath.Join(base, "wap")
		}
	}
	sub := a.DB.Conf("tpl_html_dir")
	if sub == "" {
		sub = "html"
	}
	return filepath.Join(base, sub)
}

func isWap(a *app.App, r *http.Request) bool {
	if a.DB.Conf("open_wap") != "1" {
		return false
	}
	host := r.Host
	if i := strings.IndexByte(host, ':'); i >= 0 {
		host = host[:i]
	}
	if wd := a.DB.Conf("wap_domain"); wd != "" && strings.EqualFold(wd, host) {
		return true
	}
	ua := r.UserAgent()
	return strings.Contains(ua, "Mobile") || strings.Contains(ua, "Android") || strings.Contains(ua, "iPhone")
}

func flatQuery(r *http.Request) map[string]string {
	out := map[string]string{}
	for k, vs := range r.URL.Query() {
		if len(vs) > 0 {
			out[k] = vs[0]
		}
	}
	raw, _ := url.QueryUnescape(r.URL.RawQuery)
	for _, part := range strings.FieldsFunc(raw, func(r rune) bool { return r == '&' || r == '?' }) {
		k, v, ok := strings.Cut(part, "=")
		if ok && out[k] == "" {
			out[k] = v
		}
	}
	return out
}

// QueryVal reads a query value from the parsed query or a mode-3 raw query.
func QueryVal(r *http.Request, key string) string {
	if v := r.URL.Query().Get(key); v != "" {
		return v
	}
	return flatQuery(r)[key]
}

func scheme(r *http.Request) string {
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		return "https"
	}
	return "http"
}

// redirectPolicy mirrors PHP HomeController HTTPS / main-domain 301s.
func redirectPolicy(a *app.App, w http.ResponseWriter, r *http.Request) bool {
	if a.DB.Conf("to_https") != "" && a.DB.Conf("to_https") != "0" && scheme(r) != "https" {
		http.Redirect(w, r, "https://"+r.Host+r.URL.RequestURI(), http.StatusMovedPermanently)
		return true
	}
	main := strings.TrimSpace(a.DB.Conf("main_domain"))
	if main != "" && a.DB.Conf("to_main_domain") != "" && a.DB.Conf("to_main_domain") != "0" {
		if !(a.DB.Conf("wap_domain") != "" && isWap(a, r)) {
			host := requestHostOnly(r)
			if !strings.EqualFold(host, main) {
				port := ""
				if i := strings.LastIndex(r.Host, ":"); i >= 0 {
					port = r.Host[i:]
				}
				http.Redirect(w, r, scheme(r)+"://"+main+port, http.StatusMovedPermanently)
				return true
			}
		}
	}
	return false
}

func requestHostOnly(r *http.Request) string {
	h := r.Host
	if i := strings.IndexByte(h, ':'); i >= 0 {
		h = h[:i]
	}
	return h
}

func boolStr(b bool) string {
	if b {
		return "1"
	}
	return "0"
}

func memberOf(a *app.App, r *http.Request) map[string]string {
	s := loadFront(a, r)
	if s == nil || s.Values["uid"] == "" {
		return map[string]string{}
	}
	uid := domain.Atoi(s.Values["uid"], 0)
	rows, err := a.DB.Maps("SELECT * FROM ay_member WHERE id=?", uid)
	if err != nil || len(rows) == 0 {
		return map[string]string{}
	}
	return enrichMember(a, rows[0])
}

func notFoundPage(a *app.App, info string) string {
	paths := []string{
		filepath.Join(a.Cfg.SiteRoot, "404.html"),
		filepath.Join(a.Cfg.SiteRoot, "core", "template", "error.html"),
	}
	for _, p := range paths {
		b, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		s := string(b)
		s = strings.ReplaceAll(s, "{info}", info)
		s = strings.ReplaceAll(s, "{js}", "")
		s = strings.ReplaceAll(s, "{url}", "/")
		s = strings.ReplaceAll(s, "{time}", "0")
		s = strings.ReplaceAll(s, "{coredir}", "/core")
		s = strings.ReplaceAll(s, "{sitedir}", "")
		s = strings.ReplaceAll(s, "{appversion}", "")
		s = strings.ReplaceAll(s, "{phpversion}", "")
		s = strings.ReplaceAll(s, "{serveros}", "")
		s = strings.ReplaceAll(s, "{serversoft}", "")
		s = strings.ReplaceAll(s, "{dbtype}", "")
		s = strings.ReplaceAll(s, "{dbdriver}", "")
		s = strings.ReplaceAll(s, "{dbversion}", "")
		return s
	}
	return "<!doctype html><html><head><meta charset=utf-8><title>404</title></head><body><h1>404</h1><p>" + htmlEscape(info) + `</p><p><a href="/">返回首页</a></p></body></html>`
}

func htmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

func remoteIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return strings.TrimSpace(strings.Split(xff, ",")[0])
	}
	host := r.RemoteAddr
	if i := strings.LastIndex(host, ":"); i >= 0 {
		host = host[:i]
	}
	return strings.Trim(host, "[]")
}

func loadFront(a *app.App, r *http.Request) *auth.Session {
	c, err := r.Cookie(auth.FrontCookie())
	if err != nil {
		return nil
	}
	return a.Front.Load(c.Value)
}

func ensureFront(a *app.App, w http.ResponseWriter, r *http.Request) *auth.Session {
	if s := loadFront(a, r); s != nil {
		return s
	}
	s, _ := a.Front.New()
	auth.SessionCookie(w, r, auth.FrontCookie(), s.ID)
	return s
}

func JSON(w http.ResponseWriter, code int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]any{"code": code, "data": data})
}

// ClearPages drops the HTML cache for the site.
func ClearPages(a *app.App) { cache.Clear(a.Pages.Dir) }
