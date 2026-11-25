package pboot

import (
	"crypto/md5"
	"encoding/hex"
	"html"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"gbootcms/internal/domain"
	"gbootcms/internal/store"
	"gbootcms/internal/textmod"
	"gbootcms/internal/urlx"
)

// Page is the pager the list/search/message/comment tags fill.
type Page struct {
	Rows, Page, Pages              int
	Index, Pre, Next, Last, Numbar string
}

// Ctx is one render.
type Ctx struct {
	DB         *store.DB
	Acode      string
	Rules      urlx.Rules
	Site       map[string]string
	Company    map[string]string
	Sort       map[string]string
	Content    map[string]string
	Sorts      []map[string]string
	Labels     map[string]string
	Query      map[string]string
	Page       Page
	PageBase   string
	Member     map[string]string
	Keyword    string
	SchemeHost string
	RequestURI string
	Path       string
	Kind       string
	TplRoot    string
	SitePath   string // PHP SITE_DIR, e.g. "/cms" or ""
	CommentID  string
	ClientIP   string
	Depth      int
	apiStamp   string
}

var (
	pairNames = map[string]bool{
		"list": true, "nav": true, "sort": true, "slide": true, "content": true,
		"link": true, "message": true, "search": true, "select": true, "tags": true,
		"comment": true, "commentsub": true, "mycomment": true, "loop": true, "pics": true,
	}
	fieldRe   = regexp.MustCompile(`\{(content|sort|page):([a-zA-Z0-9_]+)([^}]*)\}`)
	plainRe   = regexp.MustCompile(`\{pboot:([a-zA-Z]+)(\s+[^}{]*)?\}`)
	labelRe   = regexp.MustCompile(`\{label:([a-zA-Z0-9_]+)\}`)
	userRe    = regexp.MustCompile(`\{user:([a-zA-Z0-9_]+)([^}]*)\}`)
	incRe     = regexp.MustCompile(`(?i)\{include\s+file\s*=\s*['"]?([^'"}\s]+)['"]?\s*\}`)
	preRe     = regexp.MustCompile(`(?s)\{pboot:pre\}(.*?)\{/pboot:pre\}`)
	fileCache sync.Map // reserved for template mtime cache
)

// RenderFile reads a theme file, expands includes, then renders tags.
func RenderFile(path string, ctx *Ctx) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	s := strings.TrimPrefix(string(b), "\uFEFF")
	root := ctx.TplRoot
	if root == "" {
		root = filepath.Dir(path)
	}
	s = expandIncludes(s, root, 0)
	var pres []string
	s = preRe.ReplaceAllStringFunc(s, func(m string) string {
		sub := preRe.FindStringSubmatch(m)
		pres = append(pres, sub[1])
		return "\x00PRE" + strconv.Itoa(len(pres)-1) + "\x00"
	})
	s = Render(s, ctx)
	for i, p := range pres {
		s = strings.ReplaceAll(s, "\x00PRE"+strconv.Itoa(i)+"\x00", p)
	}
	return s, nil
}

func underRoot(root, name string) (string, bool) {
	name = strings.ReplaceAll(name, "\\", "/")
	if name == "" || strings.Contains(name, "..") {
		return "", false
	}
	root = filepath.Clean(root)
	p := filepath.Clean(filepath.Join(root, filepath.FromSlash(name)))
	rel, err := filepath.Rel(root, p)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", false
	}
	return p, true
}

func expandIncludes(s, root string, n int) string {
	if n > 50 {
		return s
	}
	found := false
	s = incRe.ReplaceAllStringFunc(s, func(m string) string {
		found = true
		name := incRe.FindStringSubmatch(m)[1]
		p, ok := underRoot(root, name)
		if !ok {
			return ""
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return ""
		}
		return strings.TrimPrefix(string(b), "\uFEFF")
	})
	if found {
		return expandIncludes(s, root, n+1)
	}
	return s
}

// Render expands pair tags, then tokens, then if.
// A second pass picks up tags that were stored inside content fields.
func Render(s string, ctx *Ctx) string {
	if ctx.Depth > 40 {
		return s
	}
	ctx.Depth++
	defer func() { ctx.Depth-- }()
	prev := ""
	for pass := 0; pass < 4 && s != prev; pass++ {
		prev = s
		for i := 0; i < 200; i++ {
			name, attrs, inner, a, b, ok := findPair(s)
			if !ok {
				break
			}
			repl := expand(name, attrs, inner, ctx)
			s = s[:a] + repl + s[b:]
		}
		s = replaceTokens(s, ctx)
		s = evalIf(s)
	}
	return s
}

func findPair(s string) (name, attrs, inner string, start, end int, ok bool) {
	i := 0
	for i < len(s) {
		j := strings.Index(s[i:], "{pboot:")
		if j < 0 {
			return
		}
		j += i
		rest := s[j+7:]
		nm, nlen := readName(rest)
		if !pairNames[nm] {
			i = j + 7
			continue
		}
		br := tagEnd(rest[nlen:])
		if br < 0 {
			return
		}
		attrs = rest[nlen : nlen+br]
		bodyStart := j + 7 + nlen + br + 1
		closeAt := findClose(s, bodyStart, nm)
		if closeAt < 0 {
			i = bodyStart
			continue
		}
		endTag := "{/pboot:" + nm + "}"
		return nm, attrs, s[bodyStart:closeAt], j, closeAt + len(endTag), true
	}
	return
}

func readName(s string) (string, int) {
	i := 0
	for i < len(s) && ((s[i] >= 'a' && s[i] <= 'z') || (s[i] >= 'A' && s[i] <= 'Z')) {
		i++
	}
	return s[:i], i
}

// tagEnd is the closing brace of a tag, skipping braces that belong to nested tags.
func tagEnd(s string) int {
	depth := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '{':
			depth++
		case '}':
			if depth == 0 {
				return i
			}
			depth--
		}
	}
	return -1
}

func findClose(s string, from int, name string) int {
	open := "{pboot:" + name
	clos := "{/pboot:" + name + "}"
	depth := 1
	i := from
	for i < len(s) {
		io := strings.Index(s[i:], open)
		ic := strings.Index(s[i:], clos)
		if ic < 0 {
			return -1
		}
		absC := i + ic
		if io >= 0 && i+io < absC {
			absO := i + io
			after := absO + len(open)
			if after < len(s) && isName(s[after]) {
				i = after
				continue
			}
			depth++
			i = after
			continue
		}
		depth--
		if depth == 0 {
			return absC
		}
		i = absC + len(clos)
	}
	return -1
}

func isName(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

func expand(name, attrs, inner string, ctx *Ctx) string {
	attrs = replaceTokens(attrs, ctx)
	p := textmod.Attrs(attrs)
	switch name {
	case "list":
		return expandList(p, inner, ctx, false)
	case "search":
		return expandList(p, inner, ctx, true)
	case "nav":
		return expandNav(p, inner, ctx)
	case "sort":
		return expandSort(p, inner, ctx)
	case "slide":
		return expandSlide(p, inner, ctx)
	case "content":
		return expandContent(p, inner, ctx)
	case "pics":
		return expandPics(p, inner, ctx)
	case "link":
		return expandLink(p, inner, ctx)
	case "message":
		return expandMessage(p, inner, ctx)
	case "select":
		return expandSelect(p, inner, ctx)
	case "tags":
		return expandTags(p, inner, ctx)
	case "comment":
		return expandComment(p, inner, ctx)
	case "commentsub":
		return expandCommentSub(inner, ctx)
	case "mycomment":
		return expandMyComment(p, inner, ctx)
	case "loop":
		return expandLoop(p, inner, ctx)
	default:
		return ""
	}
}

func loopItems(prefix, inner string, items []map[string]string, ctx *Ctx) string {
	var b strings.Builder
	n := strconv.Itoa(len(items))
	for i, item := range items {
		item["i"] = strconv.Itoa(i + 1)
		item["n"] = n
		chunk := subst(inner, prefix, item, ctx.Keyword)
		if prefix == "comment" {
			prev := ctx.CommentID
			ctx.CommentID = item["id"]
			b.WriteString(Render(chunk, ctx))
			ctx.CommentID = prev
			continue
		}
		b.WriteString(Render(chunk, ctx))
	}
	return b.String()
}

func subst(s, prefix string, item map[string]string, keyword string) string {
	re := regexp.MustCompile(`\[` + prefix + `:([a-zA-Z0-9_]+)([^\]]*)\]`)
	return re.ReplaceAllStringFunc(s, func(m string) string {
		sub := re.FindStringSubmatch(m)
		return textmod.Apply(item[sub[1]], sub[2], keyword)
	})
}

func expandList(p map[string]string, inner string, ctx *Ctx, search bool) string {
	num := domain.Atoi(p["num"], domain.Atoi(ctx.DB.Conf("pagesize"), 15))
	if num <= 0 {
		num = 15
	}
	scode := p["scode"]
	paginate := false
	if search {
		paginate = true
		if scode == "" {
			scode = "*"
		}
	} else if scode == "" {
		scode = ctx.Sort["scode"]
		paginate = scode != ""
	}
	var scodes []string
	all := false
	if scode == "*" || scode == "" && search {
		all = true
	} else if scode != "" {
		scodes = domain.Descendants(ctx.Sorts, scode)
	} else {
		return ""
	}
	page := 1
	if paginate {
		page = domain.Atoi(ctx.Query["page"], 1)
		if page < 1 {
			page = 1
		}
	}
	offset := 0
	limit := num
	if paginate {
		offset = (page - 1) * num
	}
	if st := p["start"]; st != "" {
		if n := domain.Atoi(st, 1); n > 1 {
			offset += n - 1
		}
	}
	filters := map[string]string{}
	for k, v := range ctx.Query {
		if strings.HasPrefix(k, "ext_") {
			filters[k] = v
		}
	}
	if p["filter"] != "" {
		filters["filter"] = p["filter"]
	}
	if p["tags"] != "" {
		filters["tags"] = p["tags"]
	}
	kw, fields := "", ""
	if search {
		kw = ctx.Keyword
		fields = p["field"]
		if ctx.Kind == "tag" {
			fields = "tags"
		}
	}
	var use []string
	if !all {
		use = scodes
	}
	rows, total, err := domain.Contents(ctx.DB, ctx.Acode, use, p["order"], limit, offset, filters, kw, fields)
	if err != nil || len(rows) == 0 && !paginate {
		if paginate {
			fillPage(ctx, 0, 1, num)
		}
		return ""
	}
	if paginate {
		fillPage(ctx, total, page, num)
	}
	items := make([]map[string]string, 0, len(rows))
	for _, r := range rows {
		items = append(items, contentItem(r, ctx))
	}
	prefix := "list"
	if search {
		prefix = "search"
	}
	return loopItems(prefix, inner, items, ctx)
}

func contentItem(r map[string]string, ctx *Ctx) map[string]string {
	item := cloneMap(r)
	item["link"] = ctx.Rules.Content(r["sortfilename"], r["urlname"], r["scode"], r["id"], r["filename"])
	item["sortlink"] = ctx.Rules.Sort(r["sortfilename"], r["urlname"], r["scode"])
	if item["ico"] == "" {
		item["ico"] = "/static/images/nopic.png"
	}
	item["likeslink"] = "/do/likes?id=" + r["id"]
	item["opposelink"] = "/do/oppose?id=" + r["id"]
	return item
}

func expandNav(p map[string]string, inner string, ctx *Ctx) string {
	parent := p["parent"]
	if parent == "" {
		parent = "0"
	}
	num := domain.Atoi(p["num"], 0)
	var items []map[string]string
	for _, s := range ctx.Sorts {
		if s["status"] == "0" {
			continue
		}
		if parent == "0" {
			if s["pcode"] != "0" && s["pcode"] != "" {
				continue
			}
		} else if s["pcode"] != parent {
			continue
		}
		items = append(items, sortItem(s, ctx))
	}
	if num > 0 && len(items) > num {
		items = items[:num]
	}
	return loopItems("nav", inner, items, ctx)
}

func expandSort(p map[string]string, inner string, ctx *Ctx) string {
	sc := p["scode"]
	var row map[string]string
	for _, s := range ctx.Sorts {
		if s["scode"] == sc {
			row = s
			break
		}
	}
	if row == nil {
		return ""
	}
	return loopItems("sort", inner, []map[string]string{sortItem(row, ctx)}, ctx)
}

func sortItem(s map[string]string, ctx *Ctx) map[string]string {
	item := cloneMap(s)
	model := modelOf(ctx, s["mcode"])
	urlname := ""
	if model != nil {
		urlname = model["urlname"]
	}
	item["link"] = ctx.Rules.Sort(s["filename"], urlname, s["scode"])
	top := domain.TopOf(ctx.Sorts, s["scode"])
	if top != nil {
		item["tcode"] = top["scode"]
		tm := modelOf(ctx, top["mcode"])
		un := ""
		if tm != nil {
			un = tm["urlname"]
		}
		item["toplink"] = ctx.Rules.Sort(top["filename"], un, top["scode"])
		item["toprows"] = strconv.Itoa(domain.CountContents(ctx.DB, ctx.Acode, domain.Descendants(ctx.Sorts, top["scode"])))
	}
	item["rows"] = strconv.Itoa(domain.CountContents(ctx.DB, ctx.Acode, domain.Descendants(ctx.Sorts, s["scode"])))
	return item
}

func modelOf(ctx *Ctx, mcode string) map[string]string {
	if ctx.DB == nil {
		return nil
	}
	rows, _ := domain.Models(ctx.DB)
	for _, m := range rows {
		if m["mcode"] == mcode {
			return m
		}
	}
	return nil
}

func expandSlide(p map[string]string, inner string, ctx *Ctx) string {
	num := domain.Atoi(p["num"], 5)
	rows, _ := domain.Slides(ctx.DB, ctx.Acode, p["gid"], num)
	var items []map[string]string
	for _, r := range rows {
		item := cloneMap(r)
		item["src"] = r["pic"]
		items = append(items, item)
	}
	return loopItems("slide", inner, items, ctx)
}

func expandContent(p map[string]string, inner string, ctx *Ctx) string {
	var row map[string]string
	if p["id"] != "" {
		row, _ = domain.ContentOne(ctx.DB, ctx.Acode, p["id"])
	} else if p["scode"] != "" {
		rows, _, _ := domain.Contents(ctx.DB, ctx.Acode, []string{p["scode"]}, "date", 1, 0, nil, "", "")
		if len(rows) > 0 {
			row = rows[0]
		}
	}
	if row == nil {
		return ""
	}
	return loopItems("content", inner, []map[string]string{contentItem(row, ctx)}, ctx)
}

func expandPics(p map[string]string, inner string, ctx *Ctx) string {
	row := ctx.Content
	if id := p["id"]; id != "" && (row == nil || row["id"] != id) {
		row, _ = domain.ContentOne(ctx.DB, ctx.Acode, id)
	}
	if row == nil {
		return ""
	}
	srcs := strings.Split(row["pics"], ",")
	titles := strings.Split(row["picstitle"], ",")
	num := domain.Atoi(p["num"], 0)
	var items []map[string]string
	for i, src := range srcs {
		src = strings.TrimSpace(src)
		if src == "" {
			continue
		}
		title := ""
		if i < len(titles) {
			title = strings.TrimSpace(titles[i])
		}
		items = append(items, map[string]string{"src": src, "title": title})
		if num > 0 && len(items) >= num {
			break
		}
	}
	return loopItems("pics", inner, items, ctx)
}

func expandLink(p map[string]string, inner string, ctx *Ctx) string {
	num := domain.Atoi(p["num"], 0)
	rows, _ := domain.Links(ctx.DB, ctx.Acode, p["gid"], num)
	return loopItems("link", inner, rows, ctx)
}

func expandMessage(p map[string]string, inner string, ctx *Ctx) string {
	num := domain.Atoi(p["num"], domain.Atoi(ctx.DB.Conf("pagesize"), 15))
	page := domain.Atoi(ctx.Query["page"], 1)
	if page < 1 {
		page = 1
	}
	rows, total, _ := domain.Messages(ctx.DB, ctx.Acode, num, (page-1)*num)
	fillPage(ctx, total, page, num)
	var items []map[string]string
	for _, r := range rows {
		item := cloneMap(r)
		item["os"] = r["user_os"]
		item["bs"] = r["user_bs"]
		item["askdate"] = textmod.Apply(r["create_time"], "style=Y-m-d", "")
		item["replydate"] = textmod.Apply(r["update_time"], "style=Y-m-d", "")
		item["nickname"] = r["contacts"]
		item["headpic"] = "/template/default/images/user.png"
		items = append(items, item)
	}
	return loopItems("message", inner, items, ctx)
}

func expandSelect(p map[string]string, inner string, ctx *Ctx) string {
	field := p["field"]
	def, _ := domain.ExtDef(ctx.DB, field)
	if def == nil {
		return ""
	}
	cur := ctx.Query[field]
	var items []map[string]string
	for _, v := range strings.Split(def["value"], ",") {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		items = append(items, map[string]string{
			"value":   v,
			"current": cur,
			"link":    withQuery(ctx, field, v),
		})
	}
	return loopItems("select", inner, items, ctx)
}

func expandTags(p map[string]string, inner string, ctx *Ctx) string {
	id := p["id"]
	row, _ := domain.ContentAny(ctx.DB, id)
	if row == nil || row["tags"] == "" {
		return ""
	}
	var items []map[string]string
	for _, t := range strings.Split(row["tags"], ",") {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		items = append(items, map[string]string{
			"text": t,
			"link": urlx.Home(ctx.Rules, "tag/"+t, 0),
		})
	}
	return loopItems("tags", inner, items, ctx)
}

func expandComment(p map[string]string, inner string, ctx *Ctx) string {
	id := p["contentid"]
	if id == "" && ctx.Content != nil {
		id = ctx.Content["id"]
	}
	num := domain.Atoi(p["num"], 20)
	page := domain.Atoi(ctx.Query["page"], 1)
	if page < 1 {
		page = 1
	}
	rows, total, _ := domain.Comments(ctx.DB, id, "0", num, (page-1)*num)
	fillPage(ctx, total, page, num)
	return loopItems("comment", inner, commentItems(rows, ctx), ctx)
}

func expandMyComment(p map[string]string, inner string, ctx *Ctx) string {
	uid := 0
	if ctx.Member != nil {
		uid = domain.Atoi(ctx.Member["id"], 0)
	}
	if uid == 0 {
		return ""
	}
	num := domain.Atoi(p["num"], domain.Atoi(ctx.DB.Conf("pagesize"), 15))
	if num <= 0 {
		num = 15
	}
	page := domain.Atoi(ctx.Query["page"], 1)
	if page < 1 {
		page = 1
	}
	rows, total, _ := domain.MyComments(ctx.DB, uid, num, (page-1)*num)
	fillPage(ctx, total, page, num)
	items := make([]map[string]string, 0, len(rows))
	for _, r := range rows {
		items = append(items, map[string]string{
			"id":        r["id"],
			"comment":   html.EscapeString(r["comment"]),
			"title":     r["title"],
			"status":    r["status"],
			"date":      textmod.Apply(r["create_time"], "style=Y-m-d H:i", ""),
			"delaction": urlx.AddQuery(urlx.Home(ctx.Rules, "comment/del", 0), "id", r["id"]),
		})
	}
	return loopItems("mycomment", inner, items, ctx)
}

func expandCommentSub(inner string, ctx *Ctx) string {
	if ctx.CommentID == "" || ctx.Content == nil {
		return ""
	}
	rows, _, _ := domain.Comments(ctx.DB, ctx.Content["id"], ctx.CommentID, 50, 0)
	return loopItems("commentsub", inner, commentItems(rows, ctx), ctx)
}

func commentItems(rows []map[string]string, ctx *Ctx) []map[string]string {
	var items []map[string]string
	for _, r := range rows {
		nick := r["nickname"]
		if nick == "" {
			nick = r["username"]
		}
		if nick == "" {
			nick = "游客"
		}
		pic := r["headpic"]
		if pic == "" {
			pic = "/template/default/images/user.png"
		}
		items = append(items, map[string]string{
			"id":          r["id"],
			"comment":     html.EscapeString(r["comment"]),
			"nickname":    html.EscapeString(nick),
			"pnickname":   html.EscapeString(nick),
			"headpic":     pic,
			"date":        textmod.Apply(r["create_time"], "style=Y-m-d H:i", ""),
			"replyaction": urlx.AddQuery(urlx.Home(ctx.Rules, "comment/add", 0), "pid", r["id"]),
		})
	}
	return items
}

func expandLoop(p map[string]string, inner string, ctx *Ctx) string {
	start := domain.Atoi(p["start"], 1)
	end := domain.Atoi(p["end"], start)
	var items []map[string]string
	for i := start; i <= end && len(items) < 500; i++ {
		items = append(items, map[string]string{"i": strconv.Itoa(i), "index": strconv.Itoa(i)})
	}
	return loopItems("loop", inner, items, ctx)
}

func fillPage(ctx *Ctx, total, page, num int) {
	pages := 1
	if num > 0 && total > 0 {
		pages = (total + num - 1) / num
	}
	if page > pages {
		page = pages
	}
	if page < 1 {
		page = 1
	}
	base := ctx.PageBase
	pg := Page{Rows: total, Page: page, Pages: pages}
	pg.Index = decorate(ctx, urlx.Home(ctx.Rules, base, 0))
	pre, next := page-1, page+1
	if pre < 1 {
		pre = 1
	}
	if next > pages {
		next = pages
	}
	pg.Pre = decorate(ctx, ctx.Rules.Page(base, pre))
	pg.Next = decorate(ctx, ctx.Rules.Page(base, next))
	pg.Last = decorate(ctx, ctx.Rules.Page(base, pages))
	var bar strings.Builder
	win := domain.Atoi(ctx.DB.Conf("pagenum"), 5)
	if win <= 0 {
		win = 5
	}
	from := page - win/2
	if from < 1 {
		from = 1
	}
	to := from + win - 1
	if to > pages {
		to = pages
	}
	for i := from; i <= to; i++ {
		cls := "page-item page-link"
		if i == page {
			cls += " active"
		}
		bar.WriteString(`<a class="` + cls + `" href="` + html.EscapeString(decorate(ctx, ctx.Rules.Page(base, i))) + `">` + strconv.Itoa(i) + `</a>`)
	}
	pg.Numbar = bar.String()
	ctx.Page = pg
}

func withQuery(ctx *Ctx, key, val string) string {
	link := ctx.RequestURI
	if link == "" {
		link = urlx.Home(ctx.Rules, ctx.PageBase, 0)
	}
	// drop existing key
	u := link
	if i := strings.Index(u, "?"); i >= 0 {
		base, raw := u[:i+1], u[i+1:]
		var keep []string
		for _, part := range strings.Split(raw, "&") {
			if part == "" || strings.HasPrefix(part, key+"=") {
				continue
			}
			keep = append(keep, part)
		}
		u = base + strings.Join(keep, "&")
		u = strings.TrimRight(u, "?&")
	}
	if val == "" {
		return u
	}
	return urlx.AddQuery(u, key, val)
}

func decorate(ctx *Ctx, link string) string {
	if ctx.Keyword != "" {
		link = urlx.AddQuery(link, "keyword", ctx.Keyword)
	}
	for k, v := range ctx.Query {
		if strings.HasPrefix(k, "ext_") && v != "" {
			link = urlx.AddQuery(link, k, v)
		}
	}
	return link
}

func replaceTokens(s string, ctx *Ctx) string {
	s = fieldRe.ReplaceAllStringFunc(s, func(m string) string {
		sub := fieldRe.FindStringSubmatch(m)
		switch sub[1] {
		case "page":
			return pageField(ctx, sub[2])
		case "content":
			if ctx.Content == nil {
				return ""
			}
			return textmod.Apply(ctx.Content[sub[2]], sub[3], ctx.Keyword)
		case "sort":
			if ctx.Sort == nil {
				return ""
			}
			return textmod.Apply(ctx.Sort[sub[2]], sub[3], ctx.Keyword)
		}
		return ""
	})
	s = strings.ReplaceAll(s, "{URL}", ctx.Path)
	s = plainRe.ReplaceAllStringFunc(s, func(m string) string {
		sub := plainRe.FindStringSubmatch(m)
		if strings.Contains(sub[2], "(") || sub[1] == "if" {
			return m
		}
		if v, ok := plainSingle(sub[1], ctx); ok {
			return v
		}
		if ctx.Labels != nil {
			if v, ok := ctx.Labels[sub[1]]; ok {
				return html.UnescapeString(v)
			}
		}
		return m
	})
	s = replaceSpecial(s, ctx)
	s = labelRe.ReplaceAllStringFunc(s, func(m string) string {
		name := labelRe.FindStringSubmatch(m)[1]
		if ctx.Labels != nil {
			return html.UnescapeString(ctx.Labels[name])
		}
		return ""
	})
	s = userRe.ReplaceAllStringFunc(s, func(m string) string {
		sub := userRe.FindStringSubmatch(m)
		return userField(ctx, sub[1], sub[2])
	})
	return s
}

func userField(ctx *Ctx, name, mods string) string {
	if ctx.Member == nil {
		return ""
	}
	if name == "password" {
		return ""
	}
	v := ctx.Member[name]
	if v == "" {
		switch name {
		case "registertime":
			v = ctx.Member["register_time"]
		case "logincount":
			v = ctx.Member["login_count"]
		case "lastlogintime":
			v = ctx.Member["last_login_time"]
		case "lastloginip":
			v = ctx.Member["last_login_ip"]
		}
	}
	return textmod.Apply(v, mods, ctx.Keyword)
}

func pageField(ctx *Ctx, name string) string {
	switch name {
	case "rows":
		return strconv.Itoa(ctx.Page.Rows)
	case "page", "current":
		return strconv.Itoa(ctx.Page.Page)
	case "pages", "count":
		return strconv.Itoa(ctx.Page.Pages)
	case "index":
		return ctx.Page.Index
	case "pre":
		return ctx.Page.Pre
	case "next":
		return ctx.Page.Next
	case "last":
		return ctx.Page.Last
	case "numbar":
		return ctx.Page.Numbar
	default:
		return ""
	}
}

var specialRe = regexp.MustCompile(`\{pboot:(qrcode|position|form|selectall)(\s+[^}]*)?\}`)

func replaceSpecial(s string, ctx *Ctx) string {
	return specialRe.ReplaceAllStringFunc(s, func(m string) string {
		sub := specialRe.FindStringSubmatch(m)
		attrs := replaceTokens(sub[2], ctx)
		p := textmod.Attrs(attrs)
		switch sub[1] {
		case "qrcode":
			raw := p["string"]
			return `<img src="/qrcode?s=` + urlQuery(raw) + `" alt="qrcode">`
		case "position":
			return positionHTML(ctx, p["separator"])
		case "form":
			return urlx.Home(ctx.Rules, "form/"+p["fcode"], 0)
		case "selectall":
			return selectAll(ctx, p)
		}
		return ""
	})
}

func selectAll(ctx *Ctx, p map[string]string) string {
	field := p["field"]
	cls := p["class"]
	if ctx.Query[field] == "" && p["active"] != "" {
		cls = p["active"]
	}
	return `<a href="` + html.EscapeString(withQuery(ctx, field, "")) + `" class="` + html.EscapeString(cls) + `">` + html.EscapeString(p["text"]) + `</a>`
}

func positionHTML(ctx *Ctx, sep string) string {
	if sep == "" {
		sep = ">>"
	}
	parts := []string{`<a href="/">首页</a>`}
	if ctx.Sort != nil && ctx.Sort["scode"] != "" {
		for _, s := range domain.Chain(ctx.Sorts, ctx.Sort["scode"]) {
			item := sortItem(s, ctx)
			parts = append(parts, `<a href="`+html.EscapeString(item["link"])+`">`+html.EscapeString(s["name"])+`</a>`)
		}
	}
	if ctx.Kind == "content" && ctx.Content != nil {
		parts = append(parts, html.EscapeString(ctx.Content["title"]))
	}
	return strings.Join(parts, " "+sep+" ")
}

func plainSingle(name string, ctx *Ctx) (string, bool) {
	site, co := ctx.Site, ctx.Company
	if site == nil {
		site = map[string]string{}
	}
	if co == nil {
		co = map[string]string{}
	}
	switch name {
	case "sitetitle":
		return site["title"], true
	case "sitesubtitle":
		return site["subtitle"], true
	case "sitekeywords":
		return site["keywords"], true
	case "sitedescription":
		return site["description"], true
	case "sitelogo":
		return site["logo"], true
	case "siteicp":
		return site["icp"], true
	case "sitecopyright":
		return site["copyright"], true
	case "sitestatistical":
		return site["statistical"], true
	case "companyname":
		return co["name"], true
	case "companyaddress":
		return co["address"], true
	case "companyphone":
		return co["phone"], true
	case "companymobile":
		return co["mobile"], true
	case "companyemail":
		return co["email"], true
	case "companyqq":
		return co["qq"], true
	case "companyweixin":
		return co["weixin"], true
	case "companyblicense":
		return co["blicense"], true
	case "companycontact":
		return co["contact"], true
	case "companyfax":
		return co["fax"], true
	case "companyother":
		return co["other"], true
	case "pagetitle":
		return ctx.pageTitle(), true
	case "pagekeywords":
		return ctx.pageKeywords(), true
	case "pagedescription":
		return ctx.pageDesc(), true
	case "sitetplpath":
		theme := site["theme"]
		if theme == "" {
			theme = "default"
		}
		return "/template/" + theme, true
	case "sitepath":
		return ctx.SitePath, true
	case "httpurl":
		return strings.TrimRight(ctx.SchemeHost, "/"), true
	case "pageurl":
		return strings.TrimRight(ctx.SchemeHost, "/") + ctx.RequestURI, true
	case "checkcode":
		return "/code", true
	case "islogin":
		if ctx.Member != nil && ctx.Member["id"] != "" {
			return "1", true
		}
		return "0", true
	case "registerstatus":
		return flag(ctx, "register_status", "0"), true
	case "loginstatus":
		return flag(ctx, "login_status", "0"), true
	case "commentstatus":
		return flag(ctx, "comment_status", "0"), true
	case "commentcodestatus":
		return flag(ctx, "comment_check_code", "1"), true
	case "msgcodestatus":
		return flag(ctx, "message_check_code", "1"), true
	case "formcodestatus":
		return flag(ctx, "form_check_code", "1"), true
	case "registercodestatus":
		return flag(ctx, "register_check_code", "1"), true
	case "logincodestatus":
		return flag(ctx, "login_check_code", "1"), true
	case "appid":
		if ctx.DB != nil {
			return ctx.DB.Conf("api_appid"), true
		}
		return "", true
	case "timestamp":
		return ctx.apiTime(), true
	case "signature":
		return ctx.apiSignature(), true
	case "userip":
		return ctx.ClientIP, true
	case "ucenter":
		return urlx.Home(ctx.Rules, "member/ucenter", 0), true
	case "mycommentpage":
		return urlx.Home(ctx.Rules, "comment/my", 0), true
	case "login":
		u := urlx.Home(ctx.Rules, "member/login", 0)
		if ctx.Query != nil {
			if b := ctx.Query["backurl"]; b != "" {
				u = urlx.AddQuery(u, "backurl", b)
			}
		}
		return u, true
	case "register":
		return urlx.Home(ctx.Rules, "member/register", 0), true
	case "retrieve":
		return urlx.Home(ctx.Rules, "member/retrieve", 0), true
	case "umodify":
		return urlx.Home(ctx.Rules, "member/umodify", 0), true
	case "sendemail":
		return urlx.Home(ctx.Rules, "member/sendEmail", 0), true
	case "isregister":
		return urlx.Home(ctx.Rules, "member/isRegister", 0), true
	case "logout":
		return urlx.Home(ctx.Rules, "member/logout", 0), true
	case "upload":
		return urlx.Home(ctx.Rules, "member/upload", 0), true
	case "msgaction":
		return urlx.Home(ctx.Rules, "message", 0), true
	case "scaction":
		return urlx.Home(ctx.Rules, "search", 0), true
	case "lgpath":
		return urlx.Home(ctx.Rules, "do/area", -1), true
	case "keyword":
		return html.EscapeString(ctx.Keyword), true
	case "commentaction":
		id := ""
		if ctx.Content != nil {
			id = ctx.Content["id"]
		}
		return urlx.AddQuery(urlx.Home(ctx.Rules, "comment/add", 0), "contentid", id), true
	case "mustlogin":
		return "", true
	default:
		return "", false
	}
}

func flag(ctx *Ctx, key, def string) string {
	if ctx.DB == nil {
		return def
	}
	v := ctx.DB.Conf(key)
	if v == "" {
		return def
	}
	if v == "0" {
		return "0"
	}
	return "1"
}

func (ctx *Ctx) pageTitle() string {
	site := ""
	if ctx.Site != nil {
		site = ctx.Site["title"]
	}
	switch ctx.Kind {
	case "content":
		if ctx.Content != nil && ctx.Content["title"] != "" {
			return ctx.Content["title"] + "-" + site
		}
	case "list", "search", "message", "tag":
		if ctx.Sort != nil {
			t := ctx.Sort["title"]
			if t == "" {
				t = ctx.Sort["name"]
			}
			if t != "" {
				return t + "-" + site
			}
		}
	case "comment":
		return "我的评论-" + site
	case "member":
		titles := map[string]string{
			"login": "会员登录", "register": "会员注册", "retrieve": "找回密码",
			"ucenter": "个人中心", "umodify": "资料修改",
		}
		if t := titles[strings.ToLower(ctx.PageBase)]; t != "" {
			return t + "-" + site
		}
	}
	sub := ""
	if ctx.Site != nil {
		sub = ctx.Site["subtitle"]
	}
	if sub != "" {
		return site + "-" + sub
	}
	return site
}

func (ctx *Ctx) pageKeywords() string {
	if ctx.Kind == "content" && ctx.Content != nil && ctx.Content["keywords"] != "" {
		return ctx.Content["keywords"]
	}
	if ctx.Sort != nil && ctx.Sort["keywords"] != "" {
		return ctx.Sort["keywords"]
	}
	if ctx.Site != nil {
		return ctx.Site["keywords"]
	}
	return ""
}

func (ctx *Ctx) pageDesc() string {
	if ctx.Kind == "content" && ctx.Content != nil && ctx.Content["description"] != "" {
		return ctx.Content["description"]
	}
	if ctx.Sort != nil && ctx.Sort["description"] != "" {
		return ctx.Sort["description"]
	}
	if ctx.Site != nil {
		return ctx.Site["description"]
	}
	return ""
}

func evalIf(s string) string {
	for n := 0; n < 80; n++ {
		a, b, cond, inner, els, ok := findIf(s)
		if !ok {
			return s
		}
		body := els
		if textmod.Truth(cond) {
			body = inner
		}
		s = s[:a] + body + s[b:]
	}
	return s
}

func findIf(s string) (start, end int, cond, inner, els string, ok bool) {
	i := 0
	for i < len(s) {
		j := strings.Index(s[i:], "{pboot:")
		if j < 0 {
			return
		}
		j += i
		rest := s[j+7:]
		k := 0
		for k < len(rest) && rest[k] >= '0' && rest[k] <= '9' {
			k++
		}
		if !strings.HasPrefix(rest[k:], "if(") {
			i = j + 7
			continue
		}
		p := k + 3
		depth := 1
		cs := p
		for p < len(rest) && depth > 0 {
			switch rest[p] {
			case '(':
				depth++
			case ')':
				depth--
			}
			if depth == 0 {
				break
			}
			p++
		}
		if p >= len(rest) || rest[p] != ')' || p+1 >= len(rest) || rest[p+1] != '}' {
			i = j + 7
			continue
		}
		cond = rest[cs:p]
		ifName := rest[:k] + "if"
		bodyStart := j + 7 + p + 2
		closeTag := "{/pboot:" + ifName + "}"
		rel := strings.Index(s[bodyStart:], closeTag)
		if rel < 0 {
			i = bodyStart
			continue
		}
		chunk := s[bodyStart : bodyStart+rel]
		if strings.Contains(chunk, "{pboot:") && regexp.MustCompile(`\{pboot:\d*if\(`).MatchString(chunk) {
			i = bodyStart
			continue
		}
		inner, els = chunk, ""
		if ep := strings.Index(chunk, "{else}"); ep >= 0 {
			inner = chunk[:ep]
			els = chunk[ep+6:]
		}
		return j, bodyStart + rel + len(closeTag), cond, inner, els, true
	}
	return
}

func cloneMap(in map[string]string) map[string]string {
	out := make(map[string]string, len(in)+8)
	for k, v := range in {
		out[k] = v
	}
	return out
}

func urlQuery(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, " ", "%20"), "&", "%26")
}

// Prepare fills sort breadcrumbs and content neighbor links once per page.
func (ctx *Ctx) apiTime() string {
	if ctx.apiStamp == "" {
		ctx.apiStamp = strconv.FormatInt(time.Now().Unix(), 10)
	}
	return ctx.apiStamp
}

func (ctx *Ctx) apiSignature() string {
	appid, secret := "", ""
	if ctx.DB != nil {
		appid = ctx.DB.Conf("api_appid")
		secret = ctx.DB.Conf("api_secret")
	}
	sum := md5.Sum([]byte(appid + secret + ctx.apiTime()))
	inner := hex.EncodeToString(sum[:])
	sum = md5.Sum([]byte(inner))
	return hex.EncodeToString(sum[:])
}

func Prepare(ctx *Ctx) {
	if ctx.Sort != nil && ctx.Sort["scode"] != "" {
		ctx.Sort = sortItem(ctx.Sort, ctx)
	}
	if ctx.Content != nil && ctx.Content["id"] != "" {
		pre, next := domain.Neighbors(ctx.DB, ctx.Acode, ctx.Content["scode"], ctx.Content["id"], ctx.Content["date"])
		item := contentItem(ctx.Content, ctx)
		item["precontent"] = neighborHTML(pre, ctx)
		item["nextcontent"] = neighborHTML(next, ctx)
		ctx.Content = item
	}
}

func neighborHTML(row map[string]string, ctx *Ctx) string {
	if row == nil {
		return "没有了"
	}
	item := contentItem(row, ctx)
	return `<a href="` + html.EscapeString(item["link"]) + `">` + html.EscapeString(row["title"]) + `</a>`
}

// keep fileCache linked so the type stays in the binary for a later mtime cache.
var _ = &fileCache
