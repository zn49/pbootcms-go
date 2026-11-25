package admin

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"gbootcms/internal/app"
	"gbootcms/internal/auth"
	"gbootcms/internal/domain"
	"gbootcms/internal/front"
)

var indexnowKeyRe = regexp.MustCompile(`^[A-Za-z0-9\-]{8,128}$`)

func contentPushBatch(a *app.App, w http.ResponseWriter, r *http.Request, sess *auth.Session, kind string) {
	_ = r.ParseForm()
	ids := r.Form["list[]"]
	if len(ids) == 0 {
		ids = r.Form["list"]
	}
	if len(ids) == 0 {
		alert(w, "请选择要推送的内容！")
		return
	}
	origin := httpOrigin(r)
	host := requestHost(r)
	var urls []string
	for _, id := range ids {
		u := r.FormValue("urls[" + id + "]")
		if u == "" {
			u = contentFrontPath(a, domain.Atoi(id, 0))
		}
		if u == "" {
			continue
		}
		urls = append(urls, buildAbsURL(origin, u))
	}
	pushURLs(a, w, r, sess, kind, urls, origin, host)
}

func contentPushOne(a *app.App, w http.ResponseWriter, r *http.Request, sess *auth.Session, kind, id string) {
	if r.Method != http.MethodPost {
		alert(w, "请使用正确方式提交！")
		return
	}
	path := contentFrontPath(a, domain.Atoi(id, 0))
	if path == "" {
		alert(w, "链接类型不允许推送！")
		return
	}
	origin := httpOrigin(r)
	pushURLs(a, w, r, sess, kind, []string{buildAbsURL(origin, path)}, origin, requestHost(r))
}

func pushURLs(a *app.App, w http.ResponseWriter, r *http.Request, sess *auth.Session, kind string, urls []string, origin, host string) {
	urls = filterSameHost(urls, host)
	if len(urls) == 0 {
		alert(w, "所选内容中没有可推送的地址，链接类型内容不允许推送！")
		return
	}
	user := ""
	if sess != nil {
		user = sess.Values["username"]
	}
	switch kind {
	case "baiduzz":
		token := a.DB.Conf("baidu_zz_token")
		if token == "" {
			alert(w, "请先到系统配置中填写百度普通收录推送token值！")
			return
		}
		api := "http://data.zz.baidu.com/urls?site=" + url.QueryEscape(origin) + "&token=" + url.QueryEscape(token)
		for _, u := range urls {
			sysLog(a, user, r, "百度普通收录推送："+truncateEvent(u))
		}
		res, err := postBaidu(api, urls)
		if err != nil {
			alert(w, "百度普通收录推送发生错误："+err.Error())
			return
		}
		if _, ok := res["error"]; ok {
			alert(w, "百度普通收录推送发生错误："+stringify(res["message"], stringify(res["error"], "")))
			return
		}
		if _, ok := res["success"]; ok {
			alert(w, "成功推送"+stringify(res["success"], "")+"条，今天剩余可推送"+stringify(res["remain"], "0")+"条数!")
			return
		}
		alert(w, "发生未知错误！")
	case "baiduks":
		token := a.DB.Conf("baidu_ks_token")
		if token == "" {
			alert(w, "请先到系统配置中填写百度快速收录推送token值！")
			return
		}
		api := "http://data.zz.baidu.com/urls?site=" + url.QueryEscape(origin) + "&token=" + url.QueryEscape(token) + "&type=daily"
		for _, u := range urls {
			sysLog(a, user, r, "百度快速收录推送："+truncateEvent(u))
		}
		res, err := postBaidu(api, urls)
		if err != nil {
			alert(w, "百度快速收录推送发生错误："+err.Error())
			return
		}
		if _, ok := res["error"]; ok {
			alert(w, "百度快速收录推送发生错误："+stringify(res["message"], stringify(res["error"], "")))
			return
		}
		if _, ok := res["success_daily"]; ok {
			alert(w, "成功推送"+stringify(res["success_daily"], "")+"条，今天剩余可推送"+stringify(res["remain_daily"], "0")+"条数!")
			return
		}
		alert(w, "发生未知错误！")
	case "indexnow":
		key := effectiveIndexnowKey(a)
		if key == "" {
			alert(w, "请先到系统配置中填写IndexNow推送密钥！")
			return
		}
		if len(urls) > 10000 {
			urls = urls[:10000]
		}
		keyLoc := resolveIndexnowKeyLocation(a, key, urls, origin)
		if !indexnowKeyFileReady(a, key, keyLoc) {
			alert(w, "密钥文件不存在或内容与推送密钥不一致，请先到系统配置保存IndexNow密钥，并检查网站根目录（或自定义密钥文件地址）写入权限！")
			return
		}
		rs := postIndexnow(key, keyLoc, urls, host)
		msg := indexnowMsg(rs, len(urls))
		for _, u := range urls {
			sysLog(a, user, r, "IndexNow推送："+truncateEvent(u))
		}
		if rs.Errno == 0 && (rs.Code == 200 || rs.Code == 202) {
			sysLog(a, user, r, "IndexNow推送成功：共"+strconv.Itoa(len(urls))+"条，"+truncateEvent(msg))
		} else {
			sysLog(a, user, r, "IndexNow推送失败：共"+strconv.Itoa(len(urls))+"条，"+truncateEvent(msg))
		}
		alert(w, msg)
	default:
		alert(w, "未知推送类型")
	}
}

func contentFrontPath(a *app.App, id int) string {
	if id <= 0 {
		return ""
	}
	row, _ := domain.ContentAny(a.DB, strconv.Itoa(id))
	if row == nil || row["outlink"] != "" {
		return ""
	}
	rules := front.Rules(a.DB)
	if row["modeltype"] == "1" {
		return rules.Sort(row["sortfilename"], row["urlname"], row["scode"])
	}
	return rules.Content(row["sortfilename"], row["urlname"], row["scode"], row["id"], row["filename"])
}

func httpOrigin(r *http.Request) string {
	sch := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		sch = "https"
	}
	return sch + "://" + r.Host
}

func requestHost(r *http.Request) string {
	h := r.Host
	if i := strings.IndexByte(h, ':'); i >= 0 {
		h = h[:i]
	}
	return strings.ToLower(h)
}

func buildAbsURL(origin, path string) string {
	low := strings.ToLower(path)
	if strings.HasPrefix(low, "http://") || strings.HasPrefix(low, "https://") {
		return path
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return strings.TrimRight(origin, "/") + path
}

func filterSameHost(urls []string, host string) []string {
	seen := map[string]bool{}
	var out []string
	for _, u := range urls {
		u = strings.TrimSpace(u)
		if u == "" || seen[u] {
			continue
		}
		p, err := url.Parse(u)
		if err != nil || p.Hostname() == "" {
			continue
		}
		if host != "" && strings.ToLower(p.Hostname()) != host {
			continue
		}
		seen[u] = true
		out = append(out, u)
	}
	return out
}

func postBaidu(api string, urls []string) (map[string]any, error) {
	client := &http.Client{Timeout: 90 * time.Second}
	req, err := http.NewRequest(http.MethodPost, api, strings.NewReader(strings.Join(urls, "\n")))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "text/plain")
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	var out map[string]any
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	return out, nil
}

type indexnowResult struct {
	Code  int
	Errno int
	Error string
}

func postIndexnow(key, keyLocation string, urls []string, host string) indexnowResult {
	payload := map[string]any{"host": host, "key": key, "urlList": urls}
	if keyLocation != "" {
		payload["keyLocation"] = keyLocation
	}
	body, _ := json.Marshal(payload)
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest(http.MethodPost, "https://api.indexnow.org/indexnow", bytes.NewReader(body))
	if err != nil {
		return indexnowResult{Errno: 1, Error: err.Error()}
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	res, err := client.Do(req)
	if err != nil {
		return indexnowResult{Errno: 1, Error: err.Error()}
	}
	defer res.Body.Close()
	_, _ = io.Copy(io.Discard, res.Body)
	return indexnowResult{Code: res.StatusCode}
}

func indexnowMsg(rs indexnowResult, count int) string {
	if rs.Errno != 0 {
		err := strings.NewReplacer(`"`, "", `'`, "").Replace(rs.Error)
		return "IndexNow推送请求失败：" + err + "，请检查服务器网络及HTTPS证书配置！"
	}
	switch rs.Code {
	case 200:
		return "成功推送" + strconv.Itoa(count) + "条地址到IndexNow！"
	case 202:
		return "已接收" + strconv.Itoa(count) + "条地址，密钥待校验；首次推送出现属正常现象，请确认密钥文件可公网访问！"
	case 400:
		return "IndexNow推送失败：请求格式错误，请检查密钥及推送地址！"
	case 403:
		return "IndexNow推送失败：密钥无效，或与密钥文件中的内容不一致！"
	case 422:
		return "IndexNow推送失败：推送地址不属于本站点，或与密钥文件所在目录不匹配！"
	case 429:
		return "IndexNow推送失败：推送过于频繁，请稍后再试！"
	case 0:
		return "IndexNow推送失败：未获取到接口响应，请稍后再试！"
	default:
		return "IndexNow推送失败：接口返回状态码" + strconv.Itoa(rs.Code) + "！"
	}
}

func checkIndexnowKey(key string) bool {
	return indexnowKeyRe.MatchString(key)
}

func effectiveIndexnowKey(a *app.App) string {
	key := strings.TrimSpace(a.DB.Conf("indexnow_key"))
	if checkIndexnowKey(key) {
		return key
	}
	loc := strings.TrimSpace(a.DB.Conf("indexnow_key_location"))
	file := resolveIndexnowLocalFile(a, loc)
	if file == "" {
		return ""
	}
	b, err := os.ReadFile(file)
	if err != nil {
		return ""
	}
	key = strings.TrimSpace(string(b))
	if !checkIndexnowKey(key) || filepath.Base(file) != key+".txt" {
		return ""
	}
	return key
}

func saveIndexnowKeyFiles(a *app.App, key, loc string) error {
	if !checkIndexnowKey(key) {
		return nil
	}
	if err := os.WriteFile(filepath.Join(a.Cfg.SiteRoot, key+".txt"), []byte(key), 0o644); err != nil {
		return err
	}
	loc = strings.TrimSpace(loc)
	if loc == "" {
		return nil
	}
	base := filepath.Base(strings.ReplaceAll(loc, "\\", "/"))
	if i := strings.Index(base, "?"); i >= 0 {
		base = base[:i]
	}
	if base != key+".txt" {
		return nil
	}
	file := resolveIndexnowLocalFile(a, loc)
	if file == "" {
		return nil
	}
	return os.WriteFile(file, []byte(key), 0o644)
}

func resolveIndexnowLocalFile(a *app.App, keyLocation string) string {
	keyLocation = strings.TrimSpace(keyLocation)
	if keyLocation == "" {
		return ""
	}
	rel := ""
	low := strings.ToLower(keyLocation)
	if strings.HasPrefix(low, "http://") || strings.HasPrefix(low, "https://") {
		p, err := url.Parse(keyLocation)
		if err != nil || p.Path == "" || p.Path == "/" {
			return ""
		}
		rel = p.Path
	} else {
		rel = keyLocation
	}
	rel = strings.ReplaceAll(rel, "\\", "/")
	if sd := strings.Trim(a.Cfg.SiteDir, "/"); sd != "" {
		if rel == "/"+sd || strings.HasPrefix(rel, "/"+sd+"/") {
			rel = strings.TrimPrefix(rel, "/"+sd)
		} else if rel == sd || strings.HasPrefix(rel, sd+"/") {
			rel = strings.TrimPrefix(rel, sd)
		}
	}
	rel = strings.TrimPrefix(rel, "/")
	if rel == "" || strings.Contains(rel, "..") {
		return ""
	}
	return filepath.Join(a.Cfg.SiteRoot, filepath.FromSlash(rel))
}

func indexnowKeyLocationURL(a *app.App, key, cfg, origin string) string {
	cfg = strings.TrimSpace(cfg)
	if cfg != "" {
		low := strings.ToLower(cfg)
		if strings.HasPrefix(low, "http://") || strings.HasPrefix(low, "https://") {
			return cfg
		}
		rel := strings.TrimPrefix(strings.ReplaceAll(cfg, "\\", "/"), "/")
		return strings.TrimRight(origin, "/") + a.Cfg.SiteDir + "/" + rel
	}
	return strings.TrimRight(origin, "/") + a.Cfg.SiteDir + "/" + key + ".txt"
}

func resolveIndexnowKeyLocation(a *app.App, key string, urls []string, origin string) string {
	cfg := a.DB.Conf("indexnow_key_location")
	loc := indexnowKeyLocationURL(a, key, cfg, origin)
	dir := loc
	if i := strings.LastIndex(dir, "/"); i >= 0 {
		dir = dir[:i+1]
	}
	for _, u := range urls {
		if !strings.HasPrefix(u, dir) {
			return indexnowKeyLocationURL(a, key, "", origin)
		}
	}
	return loc
}

func indexnowKeyFileReady(a *app.App, key, keyLocation string) bool {
	if !checkIndexnowKey(key) || strings.TrimSpace(keyLocation) == "" {
		return false
	}
	file := resolveIndexnowLocalFile(a, keyLocation)
	if file == "" {
		// cannot map to local path (remote URL) — leave to IndexNow engine
		low := strings.ToLower(keyLocation)
		if strings.HasPrefix(low, "http://") || strings.HasPrefix(low, "https://") {
			return true
		}
		file = filepath.Join(a.Cfg.SiteRoot, key+".txt")
	}
	b, err := os.ReadFile(file)
	return err == nil && strings.TrimSpace(string(b)) == key
}

func sysLog(a *app.App, user string, r *http.Request, event string) {
	if user == "" {
		user = "system"
	}
	_ = a.DB.Ent.Syslog.Create().
		SetLevel("info").SetEvent(truncateEvent(event)).SetUserIP(front.ClientIP(r)).
		SetUserOs("").SetUserBs("").SetCreateUser(user).SetCreateTime(domain.Now()).
		Exec(bg)
}

func truncateEvent(s string) string {
	r := []rune(s)
	if len(r) > 200 {
		return string(r[:200])
	}
	return s
}

func stringify(v any, fallback string) string {
	switch t := v.(type) {
	case string:
		return t
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	case json.Number:
		return t.String()
	default:
		if v == nil {
			return fallback
		}
		b, err := json.Marshal(v)
		if err != nil {
			return fallback
		}
		return strings.Trim(string(b), `"`)
	}
}
