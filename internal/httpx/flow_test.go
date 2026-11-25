package httpx

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"context"

	"gbootcms/internal/app"
	"gbootcms/internal/auth"
	"gbootcms/internal/config"
	"gbootcms/internal/domain"
	"gbootcms/internal/ent/area"
	"gbootcms/internal/ent/content"
	"gbootcms/internal/ent/contentsort"
	"gbootcms/internal/ent/form"
	"gbootcms/internal/ent/member"
	"gbootcms/internal/ent/membercomment"
	"gbootcms/internal/ent/menu"
	"gbootcms/internal/ent/model"
	"gbootcms/internal/ent/role"
	"gbootcms/internal/ent/slide"
	entuser "gbootcms/internal/ent/user"
	"gbootcms/internal/front"
	"gbootcms/internal/store"
)

func TestSiteFlow(t *testing.T) {
	root := filepath.Join("..", "..", "site")
	src := filepath.Join(root, "data", "gbootcms.db")
	dir := t.TempDir()
	dst := filepath.Join(dir, "gbootcms.db")
	copyFile(t, src, dst)
	a := app.New(config.File{Driver: "sqlite", DSN: dst, SiteRoot: root})
	if err := a.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	defer a.ClearBootstrap()
	a.Admin = auth.NewStore(filepath.Join(dir, "sess", "admin"))
	a.Front = auth.NewStore(filepath.Join(dir, "sess", "front"))
	a.Pages.Dir = filepath.Join(dir, "cache")
	srv := httptest.NewServer(Handler(a))
	defer srv.Close()

	pub := client(t)
	_ = a.DB.SetConf("password_strong", "0")
	seedFrontFixtures(t, a)
	home := get(t, pub, srv.URL+"/")
	if bad := leftover(home); bad != "" {
		t.Fatalf("home still has tags: %s", bad)
	}
	if !strings.Contains(home, "GbootCMS") {
		t.Fatal("home missing site title")
	}
	now := time.Now()
	logPath := filepath.Join(root, "data", "log", "spider", now.Format("2006"), now.Format("200601"), now.Format("20060102")+".log")
	prev, readErr := os.ReadFile(logPath)
	spiderReq, _ := http.NewRequest(http.MethodGet, srv.URL+"/", nil)
	spiderReq.Header.Set("User-Agent", "Mozilla/5.0 (compatible; Baiduspider/2.0)")
	spiderRes, err := pub.Do(spiderReq)
	if err != nil {
		t.Fatal(err)
	}
	spiderRes.Body.Close()
	logged, _ := os.ReadFile(logPath)
	if !strings.Contains(string(logged), "Baidu\t/") {
		t.Fatalf("spider log: %s", snippet(string(logged)))
	}
	if readErr == nil {
		_ = os.WriteFile(logPath, prev, 0o644)
	} else {
		_ = os.Remove(logPath)
		dir := filepath.Dir(logPath)
		for i := 0; i < 3; i++ {
			_ = os.Remove(dir)
			dir = filepath.Dir(dir)
		}
	}

	sortList, _ := a.DB.Ent.ContentSort.Query().Where(contentsort.AcodeEQ("cn"), contentsort.FilenameNEQ(""), contentsort.StatusEQ("1")).All(context.Background())
	sorts := store.Rows(sortList)
	if len(sorts) == 0 {
		t.Fatal("no sorts")
	}
	for _, s := range sorts {
		body := get(t, pub, srv.URL+front.Rules(a.DB).Sort(s["filename"], "", ""))
		if bad := leftover(body); bad != "" {
			t.Fatalf("sort %s leftover tags: %s", s["filename"], bad)
		}
	}
	one, err := a.DB.Ent.Content.Query().Where(content.StatusEQ("1"), content.AcodeEQ("cn")).Order(content.ByID()).First(context.Background())
	if err != nil || one == nil {
		t.Fatal("no content")
	}
	row := store.Row(one)
	if sort, err := a.DB.Ent.ContentSort.Query().Where(contentsort.ScodeEQ(one.Scode)).First(context.Background()); err == nil {
		row["sortfile"] = sort.Filename
	}
	link := front.Rules(a.DB).Content(row["sortfile"], "", row["scode"], row["id"], row["filename"])
	detail := get(t, pub, srv.URL+link)
	if !strings.Contains(detail, row["title"]) {
		t.Fatalf("detail missing title %s: %s", row["title"], snippet(detail))
	}
	if bad := leftover(detail); bad != "" {
		t.Fatalf("detail leftover tags: %s", bad)
	}

	nf := do(t, pub, http.MethodGet, srv.URL+"/?no-such-page-xyz/", "", nil)
	if nf.StatusCode != 404 || !strings.Contains(nf.Body, "不存在") {
		t.Fatalf("404 page: %d %s", nf.StatusCode, snippet(nf.Body))
	}
	if off := do(t, pub, http.MethodGet, srv.URL+"/llms.txt", "", nil); off.StatusCode != 404 {
		t.Fatalf("llms closed want 404 got %d", off.StatusCode)
	}
	_ = a.DB.SetConf("llms_open", "1")
	for _, p := range []string{"/sitemap.xml", "/sitemap.txt", "/llms.txt"} {
		res := do(t, pub, http.MethodGet, srv.URL+p, "", nil)
		if res.StatusCode != 200 || res.Body == "" {
			t.Fatalf("%s -> %d %s", p, res.StatusCode, res.Body)
		}
	}
	sm := get(t, pub, srv.URL+"/sitemap.xml")
	if !strings.Contains(sm, "<priority>") || !strings.Contains(sm, "<lastmod>") {
		t.Fatalf("sitemap fidelity: %s", snippet(sm))
	}
	_ = a.DB.SetConf("to_https", "1")
	httpsRes := do(t, pub, http.MethodGet, srv.URL+"/", "", nil)
	if httpsRes.StatusCode != 301 || !strings.HasPrefix(httpsRes.Header.Get("Location"), "https://") {
		t.Fatalf("to_https: %d %s", httpsRes.StatusCode, httpsRes.Header.Get("Location"))
	}
	_ = a.DB.SetConf("to_https", "0")
	llmsBody := get(t, pub, srv.URL+"/llms.txt")
	if !strings.HasPrefix(llmsBody, "# ") || !strings.Contains(llmsBody, "- [") {
		t.Fatalf("llms markdown: %s", snippet(llmsBody))
	}
	q := url.QueryEscape(row["title"])
	search := get(t, pub, srv.URL+"/?search/?keyword="+q)
	if !strings.Contains(search, row["title"]) {
		t.Fatalf("search miss: %s", snippet(search))
	}
	msg := get(t, pub, srv.URL+"/?message/")
	if strings.Contains(msg, "{pboot:") {
		t.Fatalf("message leftover: %s", snippet(msg))
	}

	_ = a.DB.SetConf("message_status", "1")
	_ = a.DB.SetConf("form_status", "1")
	_ = a.DB.SetConf("register_status", "1")
	_ = a.DB.SetConf("comment_status", "1")
	code := captcha(t, pub, a, srv.URL+"/code")
	postForm(t, pub, srv.URL+"/?message/", url.Values{
		"contacts": {"测试"}, "mobile": {"13800000000"}, "content": {"你好"}, "checkcode": {code},
	}, 200, `"code":1`)
	clearLastSub(t, a, pub, srv.URL)

	if getForm := do(t, pub, http.MethodGet, srv.URL+"/?form/2/", "", nil); getForm.StatusCode != 405 {
		t.Fatalf("form GET want 405 got %d", getForm.StatusCode)
	}
	clearLastSub(t, a, pub, srv.URL)
	postForm(t, pub, srv.URL+"/?form/2/", url.Values{"tel": {"13900000000"}}, 200, `"code":1`)
	clearLastSub(t, a, pub, srv.URL)

	code = captcha(t, pub, a, srv.URL+"/code")
	postForm(t, pub, srv.URL+"/?member/register/", url.Values{
		"username": {"uflow"}, "password": {"secret"}, "checkcode": {code},
	}, 200, `"code":1`)
	clearLastSub(t, a, pub, srv.URL)
	taken := postForm(t, pub, srv.URL+"/?member/isRegister/", url.Values{"username": {"uflow"}}, 200, "")
	if !strings.Contains(taken.Body, `"code":1`) {
		t.Fatalf("isRegister taken: %s", taken.Body)
	}
	free := postForm(t, pub, srv.URL+"/?member/isRegister/", url.Values{"username": {"uflow_free"}}, 200, "")
	if !strings.Contains(free.Body, `"code":0`) {
		t.Fatalf("isRegister free: %s", free.Body)
	}
	regPage := get(t, client(t), srv.URL+"/?member/register/")
	if strings.Contains(regPage, "{pboot:isregister}") || !strings.Contains(regPage, "member/isRegister") {
		t.Fatalf("isregister tag: %s", snippet(regPage))
	}
	beforeOne, _ := a.DB.Ent.Content.Get(context.Background(), domain.Atoi(row["id"], 0))
	vjs := do(t, pub, http.MethodGet, srv.URL+"/?do/visits/id/"+row["id"], "", nil)
	if vjs.Body != "var ok;" {
		t.Fatalf("visits: %q", vjs.Body)
	}
	afterOne, _ := a.DB.Ent.Content.Get(context.Background(), domain.Atoi(row["id"], 0))
	if afterOne.Visits <= beforeOne.Visits {
		t.Fatalf("visits not bumped: before=%d after=%d", beforeOne.Visits, afterOne.Visits)
	}
	uc := do(t, pub, http.MethodGet, srv.URL+"/?member/ucenter/", "", nil)
	if uc.StatusCode != 200 || !strings.Contains(uc.Body, "uflow") {
		t.Fatalf("ucenter %d %s", uc.StatusCode, snippet(uc.Body))
	}
	code = captcha(t, pub, a, srv.URL+"/code")
	postForm(t, pub, srv.URL+"/?comment/add/", url.Values{
		"contentid": {row["id"]}, "comment": {"一条评论"}, "checkcode": {code},
	}, 200, `"code":1`)
	mine := get(t, pub, srv.URL+"/?comment/my/")
	if !strings.Contains(mine, "一条评论") || strings.Contains(mine, "{pboot:") {
		t.Fatalf("my comment: %s", snippet(mine))
	}
	cmt, err := a.DB.Ent.MemberComment.Query().Where(membercomment.CommentEQ("一条评论")).First(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	guest := client(t)
	if denied := do(t, guest, http.MethodGet, srv.URL+"/?comment/del/?id="+strconv.Itoa(cmt.ID), "", nil); strings.Contains(denied.Body, `"code":1`) {
		t.Fatal("guest deleted comment")
	}
	if gone := do(t, pub, http.MethodGet, srv.URL+"/?comment/del/?id="+strconv.Itoa(cmt.ID), "", nil); !strings.Contains(gone.Body, `"code":1`) {
		t.Fatalf("del my comment: %s", gone.Body)
	}
	if n, _ := a.DB.Ent.MemberComment.Query().Where(membercomment.IDEQ(cmt.ID)).Count(context.Background()); n != 0 {
		t.Fatal("comment still there")
	}

	var memPNG bytes.Buffer
	mw := multipart.NewWriter(&memPNG)
	mp, _ := mw.CreateFormFile("upload", "dot.png")
	_, _ = mp.Write(tinyPNG)
	_ = mw.Close()
	memUp := do(t, pub, http.MethodPost, srv.URL+"/?member/upload/", memPNG.String(), map[string]string{"Content-Type": mw.FormDataContentType()})
	if !strings.Contains(memUp.Body, `"code":1`) || !strings.Contains(memUp.Body, "/static/upload/") {
		t.Fatalf("member upload: %s", memUp.Body)
	}
	guestUp := do(t, guest, http.MethodPost, srv.URL+"/?member/upload/", memPNG.String(), map[string]string{"Content-Type": mw.FormDataContentType()})
	if !strings.Contains(guestUp.Body, `"code":0`) {
		t.Fatalf("guest upload: %s", guestUp.Body)
	}

	_ = a.DB.SetConf("close_site", "1")
	closed := do(t, pub, http.MethodGet, srv.URL+"/", "", nil)
	if closed.StatusCode != 503 {
		t.Fatalf("close_site want 503 got %d", closed.StatusCode)
	}
	_ = a.DB.SetConf("close_site", "0")

	_ = a.DB.Exec("UPDATE ay_member SET qq=? WHERE username=?", "12345", "uflow")
	uc2 := get(t, pub, srv.URL+"/?member/ucenter/")
	if !strings.Contains(uc2, "12345") {
		t.Fatalf("diy user field: %s", snippet(uc2))
	}

	_ = a.DB.SetConf("api_open", "1")
	_ = a.DB.SetConf("api_appid", "demo")
	_ = a.DB.SetConf("api_secret", "secret")
	_ = a.DB.SetConf("api_auth", "1")
	homeAPI := get(t, pub, srv.URL+"/")
	if strings.Contains(homeAPI, "{pboot:appid}") || strings.Contains(homeAPI, "{pboot:signature}") {
		t.Fatalf("api tags leftover: %s", leftover(homeAPI))
	}
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	sum := md5.Sum([]byte("demo" + "secret" + ts))
	inner := hex.EncodeToString(sum[:])
	sum = md5.Sum([]byte(inner))
	sig := hex.EncodeToString(sum[:])
	bad := get(t, pub, srv.URL+"/api/list?num=1")
	if !strings.Contains(bad, `"code":0`) {
		t.Fatalf("api should reject: %s", bad)
	}
	okBody := get(t, pub, srv.URL+"/api/cms/list?num=1&appid=demo&timestamp="+ts+"&signature="+sig)
	if !strings.Contains(okBody, `"code":1`) {
		t.Fatalf("api list: %s", okBody)
	}
	apiQ := func(path string) string {
		ts := strconv.FormatInt(time.Now().Unix(), 10)
		sum := md5.Sum([]byte("demo" + "secret" + ts))
		inner := hex.EncodeToString(sum[:])
		sum = md5.Sum([]byte(inner))
		sig := hex.EncodeToString(sum[:])
		sep := "?"
		if strings.Contains(path, "?") {
			sep = "&"
		}
		return srv.URL + path + sep + "appid=demo&timestamp=" + ts + "&signature=" + sig
	}
	site := get(t, pub, apiQ("/api/site"))
	if !strings.Contains(site, `"code":1`) || !strings.Contains(site, "GbootCMS") {
		t.Fatalf("api site: %s", site)
	}
	if name := get(t, pub, apiQ("/api/cms/site?name=title")); !strings.Contains(name, "GbootCMS") {
		t.Fatalf("api site name: %s", name)
	}
	if pos := get(t, pub, apiQ("/api/cms/position")); !strings.Contains(pos, "scode") {
		t.Fatalf("position: %s", pos)
	}
	if pos := get(t, pub, apiQ("/api/cms/position?scode="+row["scode"])); !strings.Contains(pos, row["scode"]) {
		t.Fatalf("position scode: %s", pos)
	}
	if s := get(t, pub, apiQ("/api/cms/sort?scode="+row["scode"])); !strings.Contains(s, `"code":1`) {
		t.Fatalf("sort: %s", s)
	}
	if nav := get(t, pub, apiQ("/api/cms/nav")); !strings.Contains(nav, `"son"`) {
		t.Fatalf("nav: %s", nav)
	}
	if sl := get(t, pub, apiQ("/api/cms/slide?gid=1")); !strings.Contains(sl, `"code":1`) {
		t.Fatalf("slide: %s", sl)
	}
	if lk := get(t, pub, apiQ("/api/cms/link?gid=1")); !strings.Contains(lk, `"code":1`) {
		t.Fatalf("link: %s", lk)
	}
	if pics := get(t, pub, apiQ("/api/cms/pics?id="+row["id"])); !strings.Contains(pics, `"code":1`) {
		t.Fatalf("pics: %s", pics)
	}
	found := postForm(t, pub, apiQ("/api/cms/search"), url.Values{"keyword": {row["title"]}, "num": {"5"}}, 200, `"code":1`)
	if !strings.Contains(found.Body, row["title"]) {
		t.Fatalf("search: %s", snippet(found.Body))
	}
	if msg := get(t, pub, apiQ("/api/cms/msg?num=1")); !strings.Contains(msg, `"code":1`) {
		t.Fatalf("msg: %s", msg)
	}
	voteID := domain.Atoi(row["id"], 0)
	before, _ := a.DB.Ent.Content.Get(context.Background(), voteID)
	likes0 := 0
	if before != nil {
		likes0 = before.Likes
	}
	if vote := get(t, pub, apiQ("/api/do/likes?id="+row["id"])); !strings.Contains(vote, `"code":1`) {
		t.Fatalf("likes: %s", vote)
	}
	if again := get(t, pub, apiQ("/api/do/likes?id="+row["id"])); !strings.Contains(again, `"code":1`) {
		t.Fatalf("likes again: %s", again)
	}
	after, _ := a.DB.Ent.Content.Get(context.Background(), voteID)
	if after == nil || after.Likes != likes0+1 {
		t.Fatalf("likes cookie dedup: before=%d after=%v", likes0, after)
	}
	if is := get(t, pub, apiQ("/api/do/islikes?id="+row["id"])); !strings.Contains(is, `"code":1`) {
		t.Fatalf("islikes: %s", is)
	}

	if st, err := os.Stat(filepath.Join(root, "template", "default", "wap")); err == nil && st.IsDir() {
		_ = a.DB.SetConf("open_wap", "1")
		req, _ := http.NewRequest(http.MethodGet, srv.URL+"/", nil)
		req.Header.Set("User-Agent", "Mozilla/5.0 (iPhone)")
		res, err := pub.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		b, _ := io.ReadAll(res.Body)
		res.Body.Close()
		if res.StatusCode != 200 || strings.Contains(string(b), "{pboot:") {
			t.Fatalf("wap %d %s", res.StatusCode, snippet(string(b)))
		}
	}

	adm := client(t)
	login(t, adm, a, srv.URL, "admin", "123456")
	for _, p := range []string{
		"/admin/Index/home", "/admin/Site/index", "/admin/Company/index",
		"/admin/ContentSort/index", "/admin/Content/index", "/admin/Single/index",
		"/admin/Model/index", "/admin/ExtField/index", "/admin/Slide/index",
		"/admin/Link/index", "/admin/Message/index", "/admin/Form/index",
		"/admin/Tags/index", "/admin/Label/index", "/admin/Member/index", "/admin/MemberComment/index",
		"/admin/Role/index", "/admin/User/index", "/admin/Area/index", "/admin/Config/index",
		"/admin/Menu/index", "/admin/Syslog/index", "/admin/Type/index", "/admin/Database/index",
	} {
		body := get(t, adm, srv.URL+p)
		if strings.Contains(body, "{php}") || strings.TrimSpace(body) == "权限不足" {
			t.Fatalf("%s bad: %s", p, snippet(body))
		}
	}
	fc := formcheck(t, get(t, adm, srv.URL+"/admin/Content/index?new=1"))
	res := postForm(t, adm, srv.URL+"/admin/Content/index", url.Values{
		"formcheck": {fc}, "title": {"集成测试标题"}, "scode": {row["scode"]},
		"filename": {"itest"}, "content": {"正文"}, "date": {"2026-11-17 10:00:00"},
	}, 302, "")
	if res.StatusCode != 302 {
		t.Fatalf("save content %d %s", res.StatusCode, res.Body)
	}
	savedOne, err := a.DB.Ent.Content.Query().Where(content.TitleEQ("集成测试标题")).First(context.Background())
	if err != nil || savedOne == nil {
		t.Fatal("content was not saved")
	}
	saved := store.Row(savedOne)
	again := get(t, pub, srv.URL+front.Rules(a.DB).Content(row["sortfile"], "", row["scode"], saved["id"], "itest"))
	if !strings.Contains(again, "集成测试标题") {
		t.Fatalf("new detail: %s", snippet(again))
	}

	fc = formcheck(t, get(t, adm, srv.URL+"/admin/ContentSort/index"))
	res = postForm(t, adm, srv.URL+"/admin/ContentSort/add", url.Values{
		"formcheck": {fc}, "name": {"测试栏目"}, "filename": {"flowsort"}, "mcode": {"2"}, "pcode": {"0"}, "type": {"2"}, "status": {"1"},
	}, 302, "")
	if res.StatusCode != 302 {
		t.Fatalf("save sort %d %s", res.StatusCode, res.Body)
	}
	if n, _ := a.DB.Ent.ContentSort.Query().Where(contentsort.FilenameEQ("flowsort")).Count(context.Background()); n == 0 {
		t.Fatal("sort missing")
	}

	fc = formcheck(t, get(t, adm, srv.URL+"/admin/Slide/index"))
	res = postForm(t, adm, srv.URL+"/admin/Slide/index", url.Values{
		"formcheck": {fc}, "gid": {"1"}, "pic": {"/a.png"}, "link": {"http://example.com"},
		"title": {"幻灯"}, "subtitle": {"副"}, "sorting": {"1"},
	}, 302, "")
	if res.StatusCode != 302 {
		t.Fatalf("slide %d %s", res.StatusCode, res.Body)
	}

	fc = formcheck(t, get(t, adm, srv.URL+"/admin/ExtField/index"))
	res = postForm(t, adm, srv.URL+"/admin/ExtField/index", url.Values{
		"formcheck": {fc}, "mcode": {"2"}, "name": {"ext_flow"}, "description": {"测试"},
	}, 302, "")
	if res.StatusCode != 302 {
		t.Fatalf("ext %d %s", res.StatusCode, res.Body)
	}
	editPage := get(t, adm, srv.URL+"/admin/Content/mod/id/"+saved["id"])
	if !strings.Contains(editPage, `name="pics"`) {
		t.Fatal("content form has no pics")
	}
	fc = formcheck(t, editPage)
	res = postForm(t, adm, srv.URL+"/admin/Content/mod/id/"+saved["id"], url.Values{
		"formcheck": {fc}, "id": {saved["id"]}, "title": {"集成测试标题"}, "scode": {row["scode"]},
		"filename": {"itest"}, "content": {"正文"}, "date": {"2026-11-17 10:00:00"},
		"pics": {"/a.jpg,/b.jpg"}, "picstitle": {"甲,乙"}, "istop": {"1"}, "ext_flow": {"红色"},
	}, 302, "")
	cid, _ := strconv.Atoi(saved["id"])
	edited, err := a.DB.Ent.Content.Get(context.Background(), cid)
	if err != nil || edited.Pics != "/a.jpg,/b.jpg" || edited.Istop != "1" {
		t.Fatalf("content pics/istop: %+v %v", edited, err)
	}
	extRows, err := a.DB.ExtByContent([]int{edited.ID})
	if err != nil || extRows[saved["id"]]["ext_flow"] != "红色" {
		t.Fatalf("ext_flow: %v %v", extRows, err)
	}

	sortOne, err := a.DB.Ent.ContentSort.Query().Where(contentsort.FilenameEQ("flowsort")).First(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	fc = formcheck(t, get(t, adm, srv.URL+"/admin/ContentSort/mod/scode/"+sortOne.Scode))
	res = postForm(t, adm, srv.URL+"/admin/ContentSort/mod/scode/"+sortOne.Scode, url.Values{
		"formcheck": {fc}, "name": {"测试栏目"}, "filename": {"flowsort"},
		"mcode": {"2"}, "pcode": {"0"}, "type": {"2"}, "status": {"1"}, "outlink": {"http://example.com/s"}, "subname": {"副栏"},
	}, 302, "")
	sortOne, _ = a.DB.Ent.ContentSort.Get(context.Background(), sortOne.ID)
	if sortOne.Outlink != "http://example.com/s" || sortOne.Subname != "副栏" {
		t.Fatalf("sort update: %+v", sortOne)
	}

	slideOne, err := a.DB.Ent.Slide.Query().Where(slide.TitleEQ("幻灯")).First(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	fc = formcheck(t, get(t, adm, srv.URL+"/admin/Slide/index?id="+strconv.Itoa(slideOne.ID)))
	res = postForm(t, adm, srv.URL+"/admin/Slide/index", url.Values{
		"formcheck": {fc}, "id": {strconv.Itoa(slideOne.ID)}, "gid": {"1"}, "pic": {"/a.png"},
		"link": {"http://example.com"}, "title": {"幻灯改"}, "subtitle": {"副"}, "sorting": {"1"},
	}, 302, "")
	slideOne, _ = a.DB.Ent.Slide.Get(context.Background(), slideOne.ID)
	if slideOne.Title != "幻灯改" {
		t.Fatalf("slide update: %+v", slideOne)
	}

	fc = formcheck(t, get(t, adm, srv.URL+"/admin/Model/index"))
	res = postForm(t, adm, srv.URL+"/admin/Model/index", url.Values{
		"formcheck": {fc}, "name": {"测试模型"}, "type": {"2"}, "urlname": {"list"},
	}, 302, "")
	modOne, err := a.DB.Ent.Model.Query().Where(model.NameEQ("测试模型")).First(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	fc = formcheck(t, get(t, adm, srv.URL+"/admin/Model/index?id="+strconv.Itoa(modOne.ID)))
	res = postForm(t, adm, srv.URL+"/admin/Model/index", url.Values{
		"formcheck": {fc}, "id": {strconv.Itoa(modOne.ID)}, "name": {"测试模型改"}, "type": {"2"}, "urlname": {"list"},
	}, 302, "")
	modOne, _ = a.DB.Ent.Model.Get(context.Background(), modOne.ID)
	if modOne.Name != "测试模型改" {
		t.Fatalf("model update: %+v", modOne)
	}
	fc = formcheck(t, get(t, adm, srv.URL+"/admin/Model/index"))
	postForm(t, adm, srv.URL+"/admin/Model/index", url.Values{
		"formcheck": {fc}, "op": {"del"}, "id": {strconv.Itoa(modOne.ID)},
	}, 302, "")
	if n, _ := a.DB.Ent.Model.Query().Where(model.IDEQ(modOne.ID)).Count(context.Background()); n != 0 {
		t.Fatal("model still there")
	}

	fc = formcheck(t, get(t, adm, srv.URL+"/admin/Form/index"))
	res = postForm(t, adm, srv.URL+"/admin/Form/index", url.Values{
		"formcheck": {fc}, "form_name": {"测试表单"}, "table": {"flowf"},
	}, 302, "")
	formOne, err := a.DB.Ent.Form.Query().Where(form.FormNameEQ("测试表单")).First(context.Background())
	if err != nil || formOne.TableName != "ay_diy_flowf" {
		t.Fatalf("form create: %+v %v", formOne, err)
	}
	fc = formcheck(t, get(t, adm, srv.URL+"/admin/Form/index"))
	postForm(t, adm, srv.URL+"/admin/Form/mod/id/"+strconv.Itoa(formOne.ID)+"/action/addmenu", url.Values{
		"formcheck": {fc},
	}, 302, "")
	formMenu, err := a.DB.Ent.Menu.Query().Where(menu.URLContains("Form/index/fcode/" + formOne.Fcode + "/action/showdata")).First(context.Background())
	if err != nil || formMenu.Pcode != "M157" || formMenu.Name != "测试表单" {
		t.Fatalf("addmenu: %+v %v", formMenu, err)
	}
	_ = a.DB.Ent.Menu.DeleteOneID(formMenu.ID).Exec(context.Background())
	fc = formcheck(t, get(t, adm, srv.URL+"/admin/Form/index?id="+strconv.Itoa(formOne.ID)))
	res = postForm(t, adm, srv.URL+"/admin/Form/index", url.Values{
		"formcheck": {fc}, "id": {strconv.Itoa(formOne.ID)}, "form_name": {"测试表单改"},
	}, 302, "")
	formOne, _ = a.DB.Ent.Form.Get(context.Background(), formOne.ID)
	if formOne.FormName != "测试表单改" {
		t.Fatalf("form update: %+v", formOne)
	}
	fc = formcheck(t, get(t, adm, srv.URL+"/admin/Form/index"))
	postForm(t, adm, srv.URL+"/admin/Form/index", url.Values{
		"formcheck": {fc}, "op": {"del"}, "id": {strconv.Itoa(formOne.ID)},
	}, 302, "")
	if n, _ := a.DB.Ent.Form.Query().Where(form.IDEQ(formOne.ID)).Count(context.Background()); n != 0 {
		t.Fatal("form still there")
	}
	if _, err := a.DB.Maps("SELECT 1 FROM ay_diy_flowf"); err == nil {
		t.Fatal("diy table still there")
	}

	clearRes := get(t, adm, srv.URL+"/admin/Index/clearCache")
	if !strings.Contains(clearRes, `"code":1`) || !strings.Contains(clearRes, "清理缓存成功") {
		t.Fatalf("clearCache: %s", clearRes)
	}
	sysRes := get(t, adm, srv.URL+"/admin/Index/clearOnlySysCache")
	if !strings.Contains(sysRes, `"code":1`) {
		t.Fatalf("clearOnlySysCache: %s", sysRes)
	}

	fc = formcheck(t, get(t, adm, srv.URL+"/admin/Member/index?new=1"))
	res = postForm(t, adm, srv.URL+"/admin/Member/index", url.Values{
		"formcheck": {fc}, "username": {"admflow"}, "password": {"secret"}, "nickname": {"新会员"},
		"status": {"1"}, "gid": {"1"},
	}, 302, "")
	memOne, err := a.DB.Ent.Member.Query().Where(member.UsernameEQ("admflow")).First(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	fc = formcheck(t, get(t, adm, srv.URL+"/admin/Member/mod/id/"+strconv.Itoa(memOne.ID)))
	res = postForm(t, adm, srv.URL+"/admin/Member/mod/id/"+strconv.Itoa(memOne.ID), url.Values{
		"formcheck": {fc}, "id": {strconv.Itoa(memOne.ID)}, "username": {"admflow"}, "nickname": {"新会员改"},
		"useremail": {"a@example.com"}, "usermobile": {"13900000000"}, "status": {"1"}, "gid": {"1"},
	}, 302, "")
	memOne, _ = a.DB.Ent.Member.Get(context.Background(), memOne.ID)
	if memOne.Nickname != "新会员改" || memOne.Useremail != "a@example.com" {
		t.Fatalf("member update: %+v", memOne)
	}
	fc = formcheck(t, get(t, adm, srv.URL+"/admin/Member/index"))
	postForm(t, adm, srv.URL+"/admin/Member/index", url.Values{
		"formcheck": {fc}, "op": {"del"}, "id": {strconv.Itoa(memOne.ID)},
	}, 302, "")
	if n, _ := a.DB.Ent.Member.Query().Where(member.IDEQ(memOne.ID)).Count(context.Background()); n != 0 {
		t.Fatal("member still there")
	}

	var png bytes.Buffer
	w := multipart.NewWriter(&png)
	fc = formcheck(t, get(t, adm, srv.URL+"/admin/Index/home"))
	_ = w.WriteField("formcheck", fc)
	part, _ := w.CreateFormFile("upload", "dot.png")
	_, _ = part.Write(tinyPNG)
	_ = w.Close()
	up := do(t, adm, http.MethodPost, srv.URL+"/admin/index/upload", png.String(), map[string]string{"Content-Type": w.FormDataContentType()})
	if !strings.Contains(up.Body, `"code":1`) {
		t.Fatalf("upload %s", up.Body)
	}
	var js map[string]any
	_ = json.Unmarshal([]byte(up.Body), &js)
	if arr, ok := js["data"].([]any); !ok || len(arr) == 0 {
		t.Fatalf("upload data want array got %s", up.Body)
	} else if rel, _ := arr[0].(string); rel != "" {
		_ = os.Remove(filepath.Join(root, filepath.FromSlash(strings.TrimPrefix(rel, "/"))))
	}
	ue := do(t, adm, http.MethodGet, srv.URL+"/core/extend/ueditor/php/controller.php?action=config", "", nil)
	if !strings.Contains(ue.Body, "imageActionName") {
		t.Fatalf("ueditor config: %s", snippet(ue.Body))
	}
	ueList := do(t, adm, http.MethodGet, srv.URL+"/core/extend/ueditor/php/controller.php?action=listimage&start=0&size=5", "", nil)
	if !strings.Contains(ueList.Body, `"state"`) {
		t.Fatalf("ueditor list: %s", snippet(ueList.Body))
	}
	badUp := do(t, adm, http.MethodPost, srv.URL+"/admin/Index/upload", "", nil)
	_ = badUp

	fc = formcheck(t, get(t, adm, srv.URL+"/admin/User/index"))
	res = postForm(t, adm, srv.URL+"/admin/User/index", url.Values{
		"formcheck": {fc}, "username": {"editor"}, "realname": {"编辑"}, "password": {"secret"}, "rcode": {""},
	}, 302, "")
	if res.StatusCode != 302 {
		t.Fatalf("user %d %s", res.StatusCode, res.Body)
	}
	ed := client(t)
	login(t, ed, a, srv.URL, "editor", "secret")
	denied := do(t, ed, http.MethodGet, srv.URL+"/admin/Content/index", "", nil)
	if denied.StatusCode != 403 {
		t.Fatalf("rbac want 403 got %d %s", denied.StatusCode, snippet(denied.Body))
	}
	homeAdm := get(t, ed, srv.URL+"/admin/Index/home")
	if !strings.Contains(homeAdm, "管理") && !strings.Contains(homeAdm, "editor") {
		t.Fatalf("editor home: %s", snippet(homeAdm))
	}

	fc = formcheck(t, get(t, adm, srv.URL+"/admin/Area/index"))
	res = postForm(t, adm, srv.URL+"/admin/Area/index", url.Values{
		"formcheck": {fc}, "acode": {"en"}, "name": {"英文"}, "domain": {""},
	}, 302, "")
	if res.StatusCode != 302 {
		t.Fatalf("area %d %s", res.StatusCode, res.Body)
	}
	cfg := get(t, adm, srv.URL+"/admin/Config/index")
	if !strings.Contains(cfg, "基本配置") || !strings.Contains(cfg, "邮件通知") {
		t.Fatalf("config groups: %s", snippet(cfg))
	}
	fc = formcheck(t, get(t, adm, srv.URL+"/admin/Menu/index"))
	res = postForm(t, adm, srv.URL+"/admin/Menu/index", url.Values{
		"formcheck": {fc}, "name": {"测试菜单"}, "url": {"/admin/Content/index"}, "pcode": {"0"}, "status": {"1"}, "sorting": {"255"},
	}, 302, "")
	menuOne, err := a.DB.Ent.Menu.Query().Where(menu.NameEQ("测试菜单")).First(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	fc = formcheck(t, get(t, adm, srv.URL+"/admin/Menu/index?id="+strconv.Itoa(menuOne.ID)))
	res = postForm(t, adm, srv.URL+"/admin/Menu/index", url.Values{
		"formcheck": {fc}, "id": {strconv.Itoa(menuOne.ID)}, "name": {"测试菜单改"}, "url": {"/admin/Content/index"},
		"pcode": {"0"}, "status": {"1"}, "sorting": {"255"},
	}, 302, "")
	menuOne, _ = a.DB.Ent.Menu.Get(context.Background(), menuOne.ID)
	if menuOne.Name != "测试菜单改" {
		t.Fatalf("menu update: %+v", menuOne)
	}
	fc = formcheck(t, get(t, adm, srv.URL+"/admin/Menu/index"))
	postForm(t, adm, srv.URL+"/admin/Menu/index", url.Values{
		"formcheck": {fc}, "op": {"del"}, "id": {strconv.Itoa(menuOne.ID)},
	}, 302, "")
	if n, _ := a.DB.Ent.Menu.Query().Where(menu.IDEQ(menuOne.ID)).Count(context.Background()); n != 0 {
		t.Fatal("menu still there")
	}

	fc = formcheck(t, get(t, adm, srv.URL+"/admin/Role/index"))
	res = postForm(t, adm, srv.URL+"/admin/Role/index", url.Values{
		"formcheck": {fc}, "name": {"测试角色"}, "description": {"说明"},
	}, 302, "")
	roleOne, err := a.DB.Ent.Role.Query().Where(role.NameEQ("测试角色")).First(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	fc = formcheck(t, get(t, adm, srv.URL+"/admin/Role/index?id="+strconv.Itoa(roleOne.ID)))
	res = postForm(t, adm, srv.URL+"/admin/Role/index", url.Values{
		"formcheck": {fc}, "id": {strconv.Itoa(roleOne.ID)}, "name": {"测试角色改"}, "description": {"说明"},
	}, 302, "")
	roleOne, _ = a.DB.Ent.Role.Get(context.Background(), roleOne.ID)
	if roleOne.Name != "测试角色改" {
		t.Fatalf("role update: %+v", roleOne)
	}
	fc = formcheck(t, get(t, adm, srv.URL+"/admin/Role/index"))
	postForm(t, adm, srv.URL+"/admin/Role/index", url.Values{
		"formcheck": {fc}, "op": {"del"}, "id": {strconv.Itoa(roleOne.ID)},
	}, 302, "")
	if n, _ := a.DB.Ent.Role.Query().Where(role.IDEQ(roleOne.ID)).Count(context.Background()); n != 0 {
		t.Fatal("role still there")
	}

	fc = formcheck(t, get(t, adm, srv.URL+"/admin/User/index"))
	res = postForm(t, adm, srv.URL+"/admin/User/index", url.Values{
		"formcheck": {fc}, "username": {"ops"}, "realname": {"运维"}, "password": {"secret"}, "status": {"1"}, "rcode": {"R102"},
	}, 302, "")
	userOne, err := a.DB.Ent.User.Query().Where(entuser.UsernameEQ("ops")).First(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	fc = formcheck(t, get(t, adm, srv.URL+"/admin/User/index?id="+strconv.Itoa(userOne.ID)))
	res = postForm(t, adm, srv.URL+"/admin/User/index", url.Values{
		"formcheck": {fc}, "id": {strconv.Itoa(userOne.ID)}, "username": {"ops"}, "realname": {"运维改"},
		"status": {"1"}, "rcode": {"R102"},
	}, 302, "")
	userOne, _ = a.DB.Ent.User.Get(context.Background(), userOne.ID)
	if userOne.Realname != "运维改" {
		t.Fatalf("user update: %+v", userOne)
	}
	fc = formcheck(t, get(t, adm, srv.URL+"/admin/User/index"))
	postForm(t, adm, srv.URL+"/admin/User/index", url.Values{
		"formcheck": {fc}, "op": {"del"}, "id": {strconv.Itoa(userOne.ID)},
	}, 302, "")
	if n, _ := a.DB.Ent.User.Query().Where(entuser.IDEQ(userOne.ID)).Count(context.Background()); n != 0 {
		t.Fatal("user still there")
	}

	areaOne, err := a.DB.Ent.Area.Query().Where(area.AcodeEQ("en")).First(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	fc = formcheck(t, get(t, adm, srv.URL+"/admin/Area/index?id="+strconv.Itoa(areaOne.ID)))
	res = postForm(t, adm, srv.URL+"/admin/Area/index", url.Values{
		"formcheck": {fc}, "id": {strconv.Itoa(areaOne.ID)}, "name": {"英文站"}, "pcode": {"0"}, "domain": {""},
	}, 302, "")
	areaOne, _ = a.DB.Ent.Area.Get(context.Background(), areaOne.ID)
	if areaOne.Name != "英文站" {
		t.Fatalf("area update: %+v", areaOne)
	}

	fc = formcheck(t, get(t, adm, srv.URL+"/admin/Type/index"))
	res = postForm(t, adm, srv.URL+"/admin/Type/index", url.Values{
		"formcheck": {fc}, "tcode": {"T901"}, "name": {"测试类型"}, "item": {"查看"}, "value": {"see"}, "sorting": {"1"},
	}, 302, "")
	types, err := a.DB.Maps("SELECT id, item FROM ay_type WHERE tcode=?", "T901")
	if err != nil || len(types) != 1 {
		t.Fatalf("type create: %v %v", types, err)
	}
	tid := types[0]["id"]
	fc = formcheck(t, get(t, adm, srv.URL+"/admin/Type/index?id="+tid))
	res = postForm(t, adm, srv.URL+"/admin/Type/index", url.Values{
		"formcheck": {fc}, "id": {tid}, "name": {"测试类型"}, "item": {"审核"}, "value": {"see"}, "sorting": {"1"},
	}, 302, "")
	types, _ = a.DB.Maps("SELECT item FROM ay_type WHERE id=?", tid)
	if len(types) != 1 || types[0]["item"] != "审核" {
		t.Fatalf("type update: %v", types)
	}
	fc = formcheck(t, get(t, adm, srv.URL+"/admin/Type/index"))
	postForm(t, adm, srv.URL+"/admin/Type/index", url.Values{
		"formcheck": {fc}, "op": {"del"}, "id": {tid},
	}, 302, "")

	logPage := get(t, adm, srv.URL+"/admin/Syslog/index")
	if !strings.Contains(logPage, "登录成功") {
		t.Fatalf("syslog: %s", snippet(logPage))
	}
	spiderTab := get(t, adm, srv.URL+"/admin/Syslog/index?tab=spider")
	if !strings.Contains(spiderTab, "蜘蛛日志") && !strings.Contains(spiderTab, "text_tab") {
		if !strings.Contains(spiderTab, "layui-this") {
			t.Fatalf("syslog spider tab: %s", snippet(spiderTab))
		}
	}
	fc = formcheck(t, logPage)
	postForm(t, adm, srv.URL+"/admin/Syslog/index", url.Values{"formcheck": {fc}, "op": {"clear"}}, 302, "")
	if again := get(t, adm, srv.URL+"/admin/Syslog/index"); strings.Contains(again, "登录成功") {
		t.Fatal("syslog was not cleared")
	}

	unused := filepath.Join(root, "static", "upload", "flow-unused-9f3a.txt")
	if err := os.MkdirAll(filepath.Dir(unused), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(unused, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Remove(unused)
		_ = os.Remove(filepath.Join(root, "static", "backup", "ImageExt", "flow-unused-9f3a.txt"))
		_ = os.Remove(filepath.Join(root, "static", "backup", "ImageExt"))
	})
	imgPage := get(t, adm, srv.URL+"/admin/ImageExt/index")
	fc = formcheck(t, imgPage)
	chk := get(t, adm, srv.URL+"/admin/ImageExt/checkDataFile?page=1")
	if !strings.Contains(chk, "flow-unused-9f3a.txt") {
		t.Fatalf("image ext checkDataFile: %s", snippet(chk))
	}
	postForm(t, adm, srv.URL+"/admin/ImageExt/do_ext", url.Values{
		"formcheck": {fc}, "type": {"0"}, "real_path": {"/static/upload/flow-unused-9f3a.txt"},
	}, 200, `"code":1`)
	if _, err := os.Stat(filepath.Join(root, "static", "backup", "ImageExt", "flow-unused-9f3a.txt")); err != nil {
		t.Fatal("unused upload was not moved")
	}
	ai := postForm(t, adm, srv.URL+"/admin/Ai/generate", url.Values{
		"formcheck": {fc}, "title": {"测试标题"},
	}, 200, `"code":0`)
	if !strings.Contains(ai.Body, "未启用") && !strings.Contains(ai.Body, "未配置") {
		t.Fatalf("ai: %s", ai.Body)
	}

	fc = formcheck(t, get(t, adm, srv.URL+"/admin/Database/index"))
	postForm(t, adm, srv.URL+"/admin/Database/index", url.Values{"formcheck": {fc}, "op": {"backup"}}, 302, "")
	backups, _ := os.ReadDir(filepath.Join(filepath.Dir(dst), "backup"))
	if len(backups) == 0 {
		t.Fatal("sqlite backup missing")
	}

	pub.Jar.SetCookies(mustURL(srv.URL), []*http.Cookie{{Name: "lg", Value: "en", Path: "/"}})
	en := get(t, pub, srv.URL+"/")
	if strings.Contains(en, "{pboot:") {
		t.Fatalf("en home: %s", snippet(en))
	}
}

var formcheckRe = regexp.MustCompile(`name="formcheck" value="([^"]+)"`)

func clearLastSub(t *testing.T, a *app.App, c *http.Client, base string) {
	t.Helper()
	for _, ck := range c.Jar.Cookies(mustURL(base)) {
		if ck.Name != auth.FrontCookie() {
			continue
		}
		s := a.Front.Load(ck.Value)
		if s == nil {
			return
		}
		delete(s.Values, "lastsub")
		_ = a.Front.Save(s)
		return
	}
}

func formcheck(t *testing.T, html string) string {
	t.Helper()
	m := formcheckRe.FindStringSubmatch(html)
	if m == nil {
		t.Fatalf("no formcheck: %s", snippet(html))
	}
	return m[1]
}

func login(t *testing.T, c *http.Client, a *app.App, base, user, pass string) {
	t.Helper()
	page := get(t, c, base+"/admin/Index/index")
	if !strings.Contains(page, `action="/admin/Index/login"`) {
		t.Fatalf("login action: %s", snippet(page))
	}
	if a.DB.Conf("admin_check_code") != "0" && !strings.Contains(page, `name="checkcode"`) {
		t.Fatal("captcha field missing")
	}
	fc := formcheck(t, page)
	vals := url.Values{"formcheck": {fc}, "username": {user}, "password": {pass}}
	if a.DB.Conf("admin_check_code") != "0" {
		vals.Set("checkcode", captcha(t, c, a, base+"/code"))
	}
	res := postForm(t, c, base+"/admin/Index/login", vals, 302, "")
	if res.StatusCode != 302 {
		t.Fatalf("login %s: %d %s", user, res.StatusCode, res.Body)
	}
}

func captcha(t *testing.T, c *http.Client, a *app.App, raw string) string {
	t.Helper()
	_ = get(t, c, raw)
	u, _ := url.Parse(raw)
	var id string
	for _, ck := range c.Jar.Cookies(u) {
		if ck.Name == auth.FrontCookie() {
			id = ck.Value
		}
	}
	s := a.Front.Load(id)
	if s == nil || s.Values["code"] == "" {
		t.Fatal("captcha missing")
	}
	return s.Values["code"]
}

type resp struct {
	StatusCode int
	Body       string
	Header     http.Header
}

func get(t *testing.T, c *http.Client, raw string) string {
	t.Helper()
	res := do(t, c, http.MethodGet, raw, "", nil)
	if res.StatusCode != 200 {
		t.Fatalf("GET %s -> %d %s", raw, res.StatusCode, snippet(res.Body))
	}
	return res.Body
}

func postForm(t *testing.T, c *http.Client, raw string, v url.Values, want int, contains string) resp {
	t.Helper()
	res := do(t, c, http.MethodPost, raw, v.Encode(), map[string]string{"Content-Type": "application/x-www-form-urlencoded"})
	if want != 0 && res.StatusCode != want {
		t.Fatalf("POST %s -> %d want %d body %s", raw, res.StatusCode, want, snippet(res.Body))
	}
	if contains != "" && !strings.Contains(res.Body, contains) {
		t.Fatalf("POST %s body %s", raw, snippet(res.Body))
	}
	return res
}

func do(t *testing.T, c *http.Client, method, raw, body string, hdr map[string]string) resp {
	t.Helper()
	req, err := http.NewRequest(method, raw, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	res, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	return resp{StatusCode: res.StatusCode, Body: string(b), Header: res.Header.Clone()}
}

func client(t *testing.T) *http.Client {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	return &http.Client{Jar: jar, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}}
}

func seedFrontFixtures(t *testing.T, a *app.App) {
	t.Helper()
	bg := context.Background()
	if n, _ := a.DB.Ent.Content.Query().Count(bg); n > 0 {
		return
	}
	now := domain.Now()
	if n, _ := a.DB.Ent.ContentSort.Query().Count(bg); n == 0 {
		_, err := a.DB.Ent.ContentSort.Create().
			SetAcode("cn").SetMcode("2").SetPcode("0").SetScode("1").
			SetName("新闻中心").SetFilename("news").SetStatus("1").
			SetListtpl("newslist.html").SetContenttpl("news.html").
			SetOutlink("").SetSubname("").SetDef1("").SetDef2("").SetDef3("").
			SetIco("").SetPic("").SetTitle("").SetKeywords("").SetDescription("").
			SetSorting(255).SetGtype("4").SetGid("").SetGnote("").
			SetCreateUser("admin").SetUpdateUser("admin").
			SetCreateTime(now).SetUpdateTime(now).
			Save(bg)
		if err != nil {
			t.Fatal(err)
		}
	}
	_, err := a.DB.Ent.Content.Create().
		SetAcode("cn").SetScode("1").SetSubscode("").
		SetTitle("FixtureArticle").SetTitlecolor("#333").SetSubtitle("").
		SetFilename("").SetAuthor("admin").SetSource("").SetOutlink("").
		SetDate(now).SetIco("").SetPics("").SetPicstitle("").
		SetContent("<p>fixture</p>").SetTags("").SetEnclosure("").SetKeywords("").SetDescription("").
		SetSorting(255).SetStatus("1").SetIstop("0").SetIsrecommend("0").SetIsheadline("0").
		SetGid("").SetGtype("4").SetGnote("").
		SetVisits(0).SetLikes(0).SetOppose(0).
		SetCreateUser("admin").SetUpdateUser("admin").
		SetCreateTime(now).SetUpdateTime(now).
		Save(bg)
	if err != nil {
		t.Fatal(err)
	}
	front.ClearPages(a)
}

func copyFile(t *testing.T, src, dst string) {
	t.Helper()
	in, err := os.Open(src)
	if err != nil {
		t.Fatal(err)
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		t.Fatal(err)
	}
}

func mustURL(s string) *url.URL {
	u, _ := url.Parse(s)
	return u
}

var twonavTag = regexp.MustCompile(`\{/?pboot:2nav[^}]*\}`)

func leftover(s string) string {
	// shortcut: {pboot:2nav} unsupported until nested-nav parity
	s = twonavTag.ReplaceAllString(s, "")
	for _, n := range []string{"{pboot:", "{content:", "[list:", "[slide:", "[nav:", "[select:", "[comment:"} {
		if i := strings.Index(s, n); i >= 0 {
			a := i - 30
			if a < 0 {
				a = 0
			}
			b := i + 90
			if b > len(s) {
				b = len(s)
			}
			return strings.Join(strings.Fields(s[a:b]), " ")
		}
	}
	return ""
}

func snippet(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	if len(s) > 240 {
		return s[:240]
	}
	return s
}

// 1x1 png
var tinyPNG = []byte{
	0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
	0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01, 0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53,
	0xde, 0x00, 0x00, 0x00, 0x0c, 0x49, 0x44, 0x41, 0x54, 0x08, 0xd7, 0x63, 0xf8, 0xcf, 0xc0, 0x00,
	0x00, 0x00, 0x03, 0x00, 0x01, 0x00, 0x05, 0xfe, 0xd4, 0xef, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45,
	0x4e, 0x44, 0xae, 0x42, 0x60, 0x82,
}
