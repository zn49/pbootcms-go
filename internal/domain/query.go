package domain

import (
	"context"
	"strconv"
	"strings"
	"time"

	"entgo.io/ent/dialect/sql"
	entg "gbootcms/internal/ent"
	"gbootcms/internal/ent/area"
	"gbootcms/internal/ent/company"
	"gbootcms/internal/ent/content"
	"gbootcms/internal/ent/contentsort"
	"gbootcms/internal/ent/extfield"
	"gbootcms/internal/ent/link"
	"gbootcms/internal/ent/member"
	"gbootcms/internal/ent/membercomment"
	"gbootcms/internal/ent/message"
	"gbootcms/internal/ent/model"
	"gbootcms/internal/ent/predicate"
	"gbootcms/internal/ent/site"
	"gbootcms/internal/ent/slide"
	"gbootcms/internal/store"
)

func Now() string { return time.Now().Format("2006-01-02 15:04:05") }

var bg = context.Background()

func Sorts(db *store.DB, acode string) ([]map[string]string, error) {
	rows, err := db.Ent.ContentSort.Query().
		Where(contentsort.AcodeEQ(acode)).
		Order(contentsort.BySorting(), contentsort.ByID()).
		All(bg)
	if err != nil {
		return nil, err
	}
	return store.Rows(rows), nil
}

func SortMap(rows []map[string]string) map[string]map[string]string {
	m := make(map[string]map[string]string, len(rows))
	for _, r := range rows {
		m[r["scode"]] = r
	}
	return m
}

func Descendants(rows []map[string]string, scode string) []string {
	kids := map[string][]string{}
	for _, r := range rows {
		kids[r["pcode"]] = append(kids[r["pcode"]], r["scode"])
	}
	var out []string
	var walk func(string)
	walk = func(c string) {
		out = append(out, c)
		for _, k := range kids[c] {
			walk(k)
		}
	}
	walk(scode)
	return out
}

func TopOf(rows []map[string]string, scode string) map[string]string {
	by := SortMap(rows)
	cur := by[scode]
	seen := map[string]bool{}
	for cur != nil && cur["pcode"] != "" && cur["pcode"] != "0" && !seen[cur["scode"]] {
		seen[cur["scode"]] = true
		next := by[cur["pcode"]]
		if next == nil {
			break
		}
		cur = next
	}
	return cur
}

func Chain(rows []map[string]string, scode string) []map[string]string {
	by := SortMap(rows)
	var rev []map[string]string
	cur := by[scode]
	seen := map[string]bool{}
	for cur != nil && !seen[cur["scode"]] {
		seen[cur["scode"]] = true
		rev = append(rev, cur)
		if cur["pcode"] == "" || cur["pcode"] == "0" {
			break
		}
		cur = by[cur["pcode"]]
	}
	for i, j := 0, len(rev)-1; i < j; i, j = i+1, j-1 {
		rev[i], rev[j] = rev[j], rev[i]
	}
	return rev
}

func contentOrders(db *store.DB, key string) []content.OrderOption {
	desc := sql.OrderDesc()
	switch key {
	case "id":
		return []content.OrderOption{content.ByID(desc)}
	case "date":
		return []content.OrderOption{content.ByIstop(desc), content.ByDate(desc), content.ByID(desc)}
	case "sorting":
		return []content.OrderOption{content.BySorting(), content.ByIstop(desc), content.ByDate(desc), content.ByID(desc)}
	case "istop":
		return []content.OrderOption{content.ByIstop(desc), content.ByDate(desc), content.ByID(desc)}
	case "isrecommend":
		return []content.OrderOption{content.ByIsrecommend(desc), content.ByDate(desc), content.ByID(desc)}
	case "isheadline":
		return []content.OrderOption{content.ByIsheadline(desc), content.ByDate(desc), content.ByID(desc)}
	case "visits":
		return []content.OrderOption{content.ByVisits(desc), content.ByDate(desc), content.ByID(desc)}
	case "likes":
		return []content.OrderOption{content.ByLikes(desc), content.ByDate(desc), content.ByID(desc)}
	case "oppose":
		return []content.OrderOption{content.ByOppose(desc), content.ByDate(desc), content.ByID(desc)}
	case "random":
		return []content.OrderOption{func(s *sql.Selector) { s.OrderBy(db.Random()) }}
	default:
		return []content.OrderOption{
			content.ByIstop(desc), content.ByIsrecommend(desc), content.ByIsheadline(desc),
			content.BySorting(), content.ByDate(desc), content.ByID(desc),
		}
	}
}

func keywordCols(fields string) []string {
	allow := map[string]bool{"title": true, "content": true, "keywords": true, "tags": true}
	if fields == "" {
		return []string{"title", "content"}
	}
	var cols []string
	for _, f := range strings.Split(fields, "|") {
		f = strings.TrimSpace(f)
		if allow[f] {
			cols = append(cols, f)
		}
	}
	if len(cols) == 0 {
		return []string{"title", "content"}
	}
	return cols
}

func keywordPred(keyword, fields string) predicate.Content {
	var preds []predicate.Content
	for _, c := range keywordCols(fields) {
		switch c {
		case "title":
			preds = append(preds, content.TitleContains(keyword))
		case "content":
			preds = append(preds, content.ContentContains(keyword))
		case "keywords":
			preds = append(preds, content.KeywordsContains(keyword))
		case "tags":
			preds = append(preds, content.TagsContains(keyword))
		}
	}
	return content.Or(preds...)
}

func extPred(filters map[string]string) predicate.Content {
	return func(s *sql.Selector) {
		e := sql.Table("ay_content_ext").As("e")
		joined := false
		for k, v := range filters {
			if v == "" || !strings.HasPrefix(k, "ext_") || !store.Ident(k) {
				continue
			}
			if !joined {
				s.Join(e).On(e.C("contentid"), s.C("id"))
				joined = true
			}
			col := e.C(k)
			s.Where(sql.Or(
				sql.EQ(col, v),
				sql.Like(col, v+",%"),
				sql.Like(col, "%,"+v+",%"),
				sql.Like(col, "%,"+v),
			))
		}
	}
}

func contentQuery(db *store.DB, acode string, scodes []string, filters map[string]string, keyword, fields string) *entg.ContentQuery {
	q := db.Ent.Content.Query().Where(content.AcodeEQ(acode), content.StatusEQ("1"))
	if scodes != nil {
		q = q.Where(content.ScodeIn(scodes...))
	}
	ext := map[string]string{}
	for k, v := range filters {
		if strings.HasPrefix(k, "ext_") {
			ext[k] = v
		}
	}
	if field, vals, ok := splitFilter(filters["filter"]); ok && strings.HasPrefix(field, "ext_") && len(vals) > 0 {
		ext[field] = vals[0]
	}
	if hasExtFilter(ext) {
		q = q.Where(extPred(ext))
	}
	if p := tagPred(filters["tags"]); p != nil {
		q = q.Where(p)
	}
	if p := flagFilter(filters["filter"]); p != nil {
		q = q.Where(p)
	}
	if keyword != "" {
		q = q.Where(keywordPred(keyword, fields))
	}
	return q
}

func splitFilter(raw string) (string, []string, bool) {
	field, rest, ok := strings.Cut(raw, "|")
	field = strings.TrimSpace(field)
	if !ok || field == "" {
		return "", nil, false
	}
	var vals []string
	for _, v := range strings.Split(rest, ",") {
		v = strings.TrimSpace(v)
		if v != "" {
			vals = append(vals, v)
		}
	}
	return field, vals, len(vals) > 0
}

func tagPred(tags string) predicate.Content {
	var preds []predicate.Content
	for _, t := range strings.Split(tags, ",") {
		t = strings.TrimSpace(t)
		if t != "" {
			preds = append(preds, content.TagsContains(t))
		}
	}
	if len(preds) == 0 {
		return nil
	}
	return content.Or(preds...)
}

func flagFilter(raw string) predicate.Content {
	field, vals, ok := splitFilter(raw)
	if !ok || strings.HasPrefix(field, "ext_") {
		return nil
	}
	var preds []predicate.Content
	for _, v := range vals {
		switch field {
		case "istop":
			preds = append(preds, content.IstopEQ(v))
		case "isrecommend":
			preds = append(preds, content.IsrecommendEQ(v))
		case "isheadline":
			preds = append(preds, content.IsheadlineEQ(v))
		case "scode":
			preds = append(preds, content.ScodeEQ(v))
		case "title":
			preds = append(preds, content.TitleContains(v))
		case "author":
			preds = append(preds, content.AuthorEQ(v))
		}
	}
	if len(preds) == 0 {
		return nil
	}
	return content.Or(preds...)
}

func hasExtFilter(filters map[string]string) bool {
	for k, v := range filters {
		if v != "" && strings.HasPrefix(k, "ext_") && store.Ident(k) {
			return true
		}
	}
	return false
}

func decorate(db *store.DB, rows []*entg.Content) ([]map[string]string, error) {
	if len(rows) == 0 {
		return nil, nil
	}
	scodes := make([]string, 0, len(rows))
	ids := make([]int, len(rows))
	seen := map[string]bool{}
	for i, r := range rows {
		ids[i] = r.ID
		if !seen[r.Scode] {
			seen[r.Scode] = true
			scodes = append(scodes, r.Scode)
		}
	}
	sorts, err := db.Ent.ContentSort.Query().Where(contentsort.ScodeIn(scodes...)).All(bg)
	if err != nil {
		return nil, err
	}
	sortBy := map[string]*entg.ContentSort{}
	var mcodes []string
	seenM := map[string]bool{}
	for _, s := range sorts {
		sortBy[s.Scode] = s
		if !seenM[s.Mcode] {
			seenM[s.Mcode] = true
			mcodes = append(mcodes, s.Mcode)
		}
	}
	modelBy := map[string]*entg.Model{}
	if len(mcodes) > 0 {
		models, err := db.Ent.Model.Query().Where(model.McodeIn(mcodes...)).All(bg)
		if err != nil {
			return nil, err
		}
		for _, m := range models {
			modelBy[m.Mcode] = m
		}
	}
	exts, err := db.ExtByContent(ids)
	if err != nil {
		return nil, err
	}
	out := make([]map[string]string, 0, len(rows))
	for _, r := range rows {
		m := store.Row(r)
		if s := sortBy[r.Scode]; s != nil {
			m["sortname"] = s.Name
			m["sortfilename"] = s.Filename
			m["sortpcode"] = s.Pcode
			m["sortpic"] = s.Pic
			m["sortico"] = s.Ico
			m["slisttpl"] = s.Listtpl
			m["scontenttpl"] = s.Contenttpl
			if md := modelBy[s.Mcode]; md != nil {
				m["urlname"] = md.Urlname
				m["modeltype"] = md.Type
				m["mlisttpl"] = md.Listtpl
				m["mcontenttpl"] = md.Contenttpl
			}
		}
		for k, v := range exts[m["id"]] {
			if k == "extid" || strings.HasPrefix(k, "ext_") {
				m[k] = v
			}
		}
		out = append(out, m)
	}
	return out, nil
}

// Contents lists published rows. scodes nil means every column in the area.
func Contents(db *store.DB, acode string, scodes []string, order string, limit, offset int, filters map[string]string, keyword, fields string) ([]map[string]string, int, error) {
	if scodes != nil && len(scodes) == 0 {
		return nil, 0, nil
	}
	q := contentQuery(db, acode, scodes, filters, keyword, fields)
	total, err := q.Count(bg)
	if err != nil {
		return nil, 0, err
	}
	q = q.Order(contentOrders(db, order)...)
	if limit > 0 {
		q = q.Limit(limit)
		if offset > 0 {
			q = q.Offset(offset)
		}
	}
	rows, err := q.All(bg)
	if err != nil {
		return nil, 0, err
	}
	out, err := decorate(db, rows)
	return out, total, err
}

func ContentOne(db *store.DB, acode, key string) (map[string]string, error) {
	preds := []predicate.Content{content.FilenameEQ(key)}
	if id, err := strconv.Atoi(key); err == nil {
		preds = append(preds, content.IDEQ(id))
	}
	row, err := db.Ent.Content.Query().
		Where(content.AcodeEQ(acode), content.StatusEQ("1"), content.Or(preds...)).
		First(bg)
	if entg.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	rows, err := decorate(db, []*entg.Content{row})
	if err != nil || len(rows) == 0 {
		return nil, err
	}
	return rows[0], nil
}

func ContentAny(db *store.DB, id string) (map[string]string, error) {
	n, err := strconv.Atoi(id)
	if err != nil {
		return nil, nil
	}
	row, err := db.Ent.Content.Get(bg, n)
	if entg.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	rows, err := decorate(db, []*entg.Content{row})
	if err != nil || len(rows) == 0 {
		return nil, err
	}
	return rows[0], nil
}

func Neighbors(db *store.DB, acode, scode, id, date string) (pre, next map[string]string) {
	n, _ := strconv.Atoi(id)
	base := func() *entg.ContentQuery {
		return db.Ent.Content.Query().Where(content.AcodeEQ(acode), content.ScodeEQ(scode), content.StatusEQ("1"))
	}
	if row, err := base().Where(content.Or(
		content.DateLT(date),
		content.And(content.DateEQ(date), content.IDLT(n)),
	)).Order(content.ByDate(sql.OrderDesc()), content.ByID(sql.OrderDesc())).First(bg); err == nil {
		if rows, e := decorate(db, []*entg.Content{row}); e == nil && len(rows) > 0 {
			pre = rows[0]
		}
	}
	if row, err := base().Where(content.Or(
		content.DateGT(date),
		content.And(content.DateEQ(date), content.IDGT(n)),
	)).Order(content.ByDate(), content.ByID()).First(bg); err == nil {
		if rows, e := decorate(db, []*entg.Content{row}); e == nil && len(rows) > 0 {
			next = rows[0]
		}
	}
	return pre, next
}

func CountContents(db *store.DB, acode string, scodes []string) int {
	if len(scodes) == 0 {
		return 0
	}
	n, _ := db.Ent.Content.Query().
		Where(content.AcodeEQ(acode), content.StatusEQ("1"), content.ScodeIn(scodes...)).
		Count(bg)
	return n
}

func oneRow(err error, v any) (map[string]string, error) {
	if entg.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return store.Row(v), nil
}

func Site(db *store.DB, acode string) (map[string]string, error) {
	row, err := db.Ent.Site.Query().Where(site.AcodeEQ(acode)).First(bg)
	return oneRow(err, row)
}

func Company(db *store.DB, acode string) (map[string]string, error) {
	row, err := db.Ent.Company.Query().Where(company.AcodeEQ(acode)).First(bg)
	return oneRow(err, row)
}

func Slides(db *store.DB, acode, gid string, num int) ([]map[string]string, error) {
	if gid == "" {
		gid = "1"
	}
	q := db.Ent.Slide.Query().Where(slide.AcodeEQ(acode), slide.GidEQ(Atoi(gid, 1))).Order(slide.BySorting(), slide.ByID())
	if num > 0 {
		q = q.Limit(num)
	}
	rows, err := q.All(bg)
	return store.Rows(rows), err
}

func Links(db *store.DB, acode, gid string, num int) ([]map[string]string, error) {
	q := db.Ent.Link.Query().Where(link.AcodeEQ(acode)).Order(link.BySorting(), link.ByID())
	if gid != "" {
		q = q.Where(link.GidEQ(Atoi(gid, 0)))
	}
	if num > 0 {
		q = q.Limit(num)
	}
	rows, err := q.All(bg)
	return store.Rows(rows), err
}

func Messages(db *store.DB, acode string, limit, offset int) ([]map[string]string, int, error) {
	q := db.Ent.Message.Query().Where(message.AcodeEQ(acode), message.StatusEQ("1"))
	total, err := q.Count(bg)
	if err != nil {
		return nil, 0, err
	}
	q = q.Order(message.ByID(sql.OrderDesc()))
	if limit > 0 {
		q = q.Limit(limit)
		if offset > 0 {
			q = q.Offset(offset)
		}
	}
	rows, err := q.All(bg)
	return store.Rows(rows), total, err
}

func Labels(db *store.DB) (map[string]string, error) {
	rows, err := db.Ent.Label.Query().All(bg)
	if err != nil {
		return nil, err
	}
	m := map[string]string{}
	for _, r := range rows {
		m[r.Name] = r.Value
	}
	return m, nil
}

func Areas(db *store.DB) ([]map[string]string, error) {
	rows, err := db.Ent.Area.Query().Order(area.ByID()).All(bg)
	return store.Rows(rows), err
}

func DefaultAcode(db *store.DB) string {
	rows, _ := Areas(db)
	for _, r := range rows {
		if r["is_default"] == "1" {
			return r["acode"]
		}
	}
	if len(rows) > 0 {
		return rows[0]["acode"]
	}
	return "cn"
}

func ExtDef(db *store.DB, name string) (map[string]string, error) {
	row, err := db.Ent.ExtField.Query().Where(extfield.NameEQ(name)).First(bg)
	return oneRow(err, row)
}

func Models(db *store.DB) ([]map[string]string, error) {
	rows, err := db.Ent.Model.Query().Order(model.ByID()).All(bg)
	return store.Rows(rows), err
}

func Comments(db *store.DB, contentID, pid string, limit, offset int) ([]map[string]string, int, error) {
	cid, pidn := Atoi(contentID, 0), Atoi(pid, 0)
	q := db.Ent.MemberComment.Query().Where(
		membercomment.ContentidEQ(cid), membercomment.PidEQ(pidn), membercomment.StatusEQ("1"),
	)
	total, _ := q.Count(bg)
	q = q.Order(membercomment.ByID(sql.OrderDesc()))
	if limit > 0 {
		q = q.Limit(limit)
		if offset > 0 {
			q = q.Offset(offset)
		}
	}
	rows, err := q.All(bg)
	if err != nil {
		return nil, 0, err
	}
	uids := make([]int, 0, len(rows))
	seen := map[int]bool{}
	for _, r := range rows {
		if r.UID != 0 && !seen[r.UID] {
			seen[r.UID] = true
			uids = append(uids, r.UID)
		}
	}
	mem := map[int]*entg.Member{}
	if len(uids) > 0 {
		ms, err := db.Ent.Member.Query().Where(member.IDIn(uids...)).All(bg)
		if err != nil {
			return nil, 0, err
		}
		for _, m := range ms {
			mem[m.ID] = m
		}
	}
	out := make([]map[string]string, 0, len(rows))
	for _, r := range rows {
		m := store.Row(r)
		if u := mem[r.UID]; u != nil {
			m["nickname"] = u.Nickname
			m["headpic"] = u.Headpic
			m["username"] = u.Username
		}
		out = append(out, m)
	}
	return out, total, nil
}

// MyComments lists one member's comments, including ones still waiting for review.
func MyComments(db *store.DB, uid, limit, offset int) ([]map[string]string, int, error) {
	if uid == 0 {
		return nil, 0, nil
	}
	q := db.Ent.MemberComment.Query().Where(membercomment.UIDEQ(uid))
	total, err := q.Count(bg)
	if err != nil {
		return nil, 0, err
	}
	q = q.Order(membercomment.ByID(sql.OrderDesc()))
	if limit > 0 {
		q = q.Limit(limit)
		if offset > 0 {
			q = q.Offset(offset)
		}
	}
	rows, err := q.All(bg)
	if err != nil {
		return nil, 0, err
	}
	ids := make([]int, 0, len(rows))
	seen := map[int]bool{}
	for _, r := range rows {
		if r.Contentid != 0 && !seen[r.Contentid] {
			seen[r.Contentid] = true
			ids = append(ids, r.Contentid)
		}
	}
	titles := map[int]string{}
	if len(ids) > 0 {
		cs, err := db.Ent.Content.Query().Where(content.IDIn(ids...)).All(bg)
		if err != nil {
			return nil, 0, err
		}
		for _, c := range cs {
			titles[c.ID] = c.Title
		}
	}
	out := make([]map[string]string, 0, len(rows))
	for _, r := range rows {
		m := store.Row(r)
		m["title"] = titles[r.Contentid]
		out = append(out, m)
	}
	return out, total, nil
}

func Atoi(s string, def int) int {
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}
