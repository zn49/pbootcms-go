package install

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"fmt"
	"html"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	mysqldriver "github.com/go-sql-driver/mysql"

	"github.com/zn49/pbootcms-go/internal/admin"
	"github.com/zn49/pbootcms-go/internal/app"
	"github.com/zn49/pbootcms-go/internal/auth"
	"github.com/zn49/pbootcms-go/internal/config"
	"github.com/zn49/pbootcms-go/internal/ent/site"
	entuser "github.com/zn49/pbootcms-go/internal/ent/user"
	"github.com/zn49/pbootcms-go/internal/fsperm"
)

const lockName = "install.lock"

var (
	mu     sync.Mutex
	userRe = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]{2,31}$`)
)

// LockPath is site_root/data/install.lock.
func LockPath(siteRoot string) string {
	return filepath.Join(siteRoot, "data", lockName)
}

// Locked reports whether installation has finished.
func Locked(siteRoot string) bool {
	_, err := os.Stat(LockPath(siteRoot))
	return err == nil
}

// WriteLock marks the site as installed.
func WriteLock(siteRoot string) error {
	if err := fsperm.MkdirAll(filepath.Join(siteRoot, "data"), fsperm.DirPrivate); err != nil {
		return err
	}
	body := []byte("installed " + time.Now().Format(time.RFC3339) + "\n")
	return fsperm.WriteFile(LockPath(siteRoot), body, fsperm.FilePrivate)
}

// Needed is true when the wizard must run.
func Needed(a *app.App) bool {
	if a == nil {
		return true
	}
	return !Locked(a.Cfg.SiteRoot)
}

// Serve handles GET/POST /install.
func Serve(a *app.App, w http.ResponseWriter, r *http.Request) {
	if Locked(a.Cfg.SiteRoot) {
		http.Error(w, "已安装。如需重装请删除 site/data/install.lock 后重启。", http.StatusForbidden)
		return
	}
	switch r.Method {
	case http.MethodGet, http.MethodHead:
		writePage(w, "", formDefaults(a))
	case http.MethodPost:
		if err := r.ParseForm(); err != nil {
			writePage(w, "表单无效", formFrom(r))
			return
		}
		in := inputFrom(r)
		if in.Mode == "import" {
			adminPath, err := CompleteImport(a, in)
			if err != nil {
				writePage(w, err.Error(), formFrom(r))
				return
			}
			writeDoneImport(w, adminPath)
			return
		}
		adminPath, plain, err := Complete(a, in)
		if err != nil {
			f := formFrom(r)
			f.Password = in.Password
			writePage(w, err.Error(), f)
			return
		}
		writeDone(w, adminPath, plain)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// Input is the install form.
type Input struct {
	Mode                                     string // install | import
	Title, Username, Password, Confirm, Addr string
	Driver                                   string
	SQLitePath                               string
	DBHost, DBPort, DBUser, DBPass, DBName   string
}

type formVals struct {
	Mode                                   string
	Title, Username, Password, Addr        string
	Driver                                 string
	SQLitePath                             string
	DBHost, DBPort, DBUser, DBPass, DBName string
}

func formDefaults(a *app.App) formVals {
	addr := ":8080"
	if a != nil && a.Cfg.Addr != "" {
		addr = a.Cfg.Addr
	}
	return formVals{
		Mode: "install", Title: "PbootCMS-Go", Username: "admin", Password: GenPassword(), Addr: addr,
		Driver: "sqlite", SQLitePath: "site/data/" + config.RandomDBFile(),
		DBHost: "127.0.0.1", DBPort: "3306", DBUser: "root", DBName: "pbootcms",
	}
}

func formFrom(r *http.Request) formVals {
	mode := strings.TrimSpace(r.FormValue("mode"))
	if mode != "import" {
		mode = "install"
	}
	f := formVals{
		Mode:       mode,
		Title:      strings.TrimSpace(r.FormValue("title")),
		Username:   strings.TrimSpace(r.FormValue("username")),
		Password:   r.FormValue("password"),
		Addr:       strings.TrimSpace(r.FormValue("addr")),
		Driver:     config.NormalizeDriver(r.FormValue("driver")),
		SQLitePath: strings.TrimSpace(r.FormValue("sqlite_path")),
		DBHost:     strings.TrimSpace(r.FormValue("db_host")),
		DBPort:     strings.TrimSpace(r.FormValue("db_port")),
		DBUser:     strings.TrimSpace(r.FormValue("db_user")),
		DBPass:     r.FormValue("db_pass"),
		DBName:     strings.TrimSpace(r.FormValue("db_name")),
	}
	if f.Driver == "" {
		f.Driver = "sqlite"
	}
	if f.Password == "" {
		f.Password = GenPassword()
	}
	return f
}

func inputFrom(r *http.Request) Input {
	f := formFrom(r)
	pwd := r.FormValue("password")
	if pwd == "" {
		pwd = f.Password
	}
	return Input{
		Mode: f.Mode, Title: f.Title, Username: strings.TrimSpace(r.FormValue("username")),
		Password: pwd, Confirm: pwd,
		Addr: f.Addr, Driver: f.Driver, SQLitePath: f.SQLitePath,
		DBHost: f.DBHost, DBPort: f.DBPort, DBUser: f.DBUser, DBPass: f.DBPass, DBName: f.DBName,
	}
}

// GenPassword returns a 16-char password that satisfies auth.Strong.
func GenPassword() string {
	const (
		upper = "ABCDEFGHJKLMNPQRSTUVWXYZ"
		lower = "abcdefghijkmnopqrstuvwxyz"
		digit = "23456789"
		sym   = "!@#$%^&*"
		all   = upper + lower + digit + sym
	)
	pick := func(set string) byte {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(set))))
		if err != nil {
			return set[0]
		}
		return set[n.Int64()]
	}
	b := make([]byte, 16)
	b[0], b[1], b[2], b[3] = pick(upper), pick(lower), pick(digit), pick(sym)
	for i := 4; i < len(b); i++ {
		b[i] = pick(all)
	}
	// shuffle
	for i := len(b) - 1; i > 0; i-- {
		jBig, err := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
		j := 0
		if err == nil {
			j = int(jBig.Int64())
		}
		b[i], b[j] = b[j], b[i]
	}
	return string(b)
}

// Complete writes config, bootstraps DB, sets admin/site, writes lock.
// The returned plain password is what was stored (generated if input empty).
func Complete(a *app.App, in Input) (adminPath, plain string, err error) {
	mu.Lock()
	defer mu.Unlock()
	if Locked(a.Cfg.SiteRoot) {
		return "", "", fmt.Errorf("已安装")
	}
	if in.Title == "" {
		return "", "", fmt.Errorf("请填写站点名称")
	}
	if !userRe.MatchString(in.Username) {
		return "", "", fmt.Errorf("管理员账号须字母开头，3–32 位字母数字下划线")
	}
	if in.Password == "" {
		in.Password = GenPassword()
		in.Confirm = in.Password
	}
	if in.Password != in.Confirm {
		return "", "", fmt.Errorf("两次密码不一致")
	}
	if !auth.Strong(in.Password) {
		return "", "", fmt.Errorf("%s", auth.StrongMsg())
	}
	plain = in.Password
	driver := config.NormalizeDriver(in.Driver)
	if !config.ValidDriver(driver) {
		return "", "", fmt.Errorf("请选择数据库类型")
	}
	addr := in.Addr
	if addr == "" {
		addr = ":8080"
	}

	wd, _ := os.Getwd()
	siteRoot := a.Cfg.SiteRoot
	if siteRoot == "" {
		siteRoot = filepath.Join(wd, "site")
	}

	dsn, err := buildDSN(wd, siteRoot, in, driver)
	if err != nil {
		return "", "", err
	}

	cfgPath := a.CfgPath
	if cfgPath == "" {
		cfgPath = "config.json"
	}
	adminPath = config.RandomAdminPath()
	salt := config.RandomSalt()
	dsnOut := dsn
	if driver == "sqlite" {
		dsnOut = relToWD(wd, dsn)
	}
	raw := map[string]any{
		"addr":        addr,
		"driver":      driver,
		"dsn":         dsnOut,
		"site_root":   relToWD(wd, siteRoot),
		"admin_path":  adminPath,
		"cookie_salt": salt,
	}
	nb, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return "", "", err
	}
	if err := fsperm.WriteFile(cfgPath, append(nb, '\n'), fsperm.FilePrivate); err != nil {
		return "", "", fmt.Errorf("写入配置失败: %w", err)
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		return "", "", err
	}
	a.Cfg = cfg
	a.CfgPath = cfgPath
	if err := a.Bootstrap(); err != nil {
		return "", "", fmt.Errorf("初始化数据库失败: %w", err)
	}
	admin.SetAdminPath(cfg.AdminPath)
	admin.SetCookieSalt(cfg.CookieSalt)

	ctx := context.Background()
	n, err := a.DB.Ent.User.Update().Where(entuser.IDEQ(1)).
		SetUsername(in.Username).
		SetRealname("超级管理员").
		SetPassword(auth.Encrypt(in.Password)).
		SetUpdateUser(in.Username).
		SetUpdateTime(time.Now().Format("2006-01-02 15:04:05")).
		Save(ctx)
	if err != nil {
		return "", "", fmt.Errorf("设置管理员失败: %w", err)
	}
	if n == 0 {
		return "", "", fmt.Errorf("库中无默认管理员，请检查初始 SQL")
	}
	if _, err := a.DB.Ent.Site.Update().Where(site.AcodeEQ("cn")).SetTitle(in.Title).Save(ctx); err != nil {
		return "", "", fmt.Errorf("设置站点名称失败: %w", err)
	}
	ApplyMinimalDefaults(a)

	if err := WriteLock(cfg.SiteRoot); err != nil {
		return "", "", fmt.Errorf("写入安装锁失败: %w", err)
	}
	return adminPath, plain, nil
}

// CompleteImport writes config pointing at an existing Pboot/ay_* database and locks install.
// Does not create admin or import seed SQL; upload/template must already be in place.
func CompleteImport(a *app.App, in Input) (adminPath string, err error) {
	mu.Lock()
	defer mu.Unlock()
	if Locked(a.Cfg.SiteRoot) {
		return "", fmt.Errorf("已安装")
	}
	driver := config.NormalizeDriver(in.Driver)
	if !config.ValidDriver(driver) {
		return "", fmt.Errorf("请选择数据库类型")
	}
	addr := in.Addr
	if addr == "" {
		addr = ":8080"
	}
	wd, _ := os.Getwd()
	siteRoot := a.Cfg.SiteRoot
	if siteRoot == "" {
		siteRoot = filepath.Join(wd, "site")
	}
	dsn, err := buildDSN(wd, siteRoot, in, driver)
	if err != nil {
		return "", err
	}
	if driver == "sqlite" {
		st, err := os.Stat(dsn)
		if err != nil || st.IsDir() || st.Size() == 0 {
			return "", fmt.Errorf("请填写已有 SQLite 库路径（文件须存在且非空）")
		}
	}
	if err := probePbootDB(driver, dsn); err != nil {
		return "", err
	}

	cfgPath := a.CfgPath
	if cfgPath == "" {
		cfgPath = "config.json"
	}
	adminPath = config.RandomAdminPath()
	salt := config.RandomSalt()
	dsnOut := dsn
	if driver == "sqlite" {
		dsnOut = relToWD(wd, dsn)
	}
	raw := map[string]any{
		"addr": addr, "driver": driver, "dsn": dsnOut,
		"site_root": relToWD(wd, siteRoot), "admin_path": adminPath, "cookie_salt": salt,
	}
	nb, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return "", err
	}
	if err := fsperm.WriteFile(cfgPath, append(nb, '\n'), fsperm.FilePrivate); err != nil {
		return "", fmt.Errorf("写入配置失败: %w", err)
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return "", err
	}
	a.Cfg = cfg
	a.CfgPath = cfgPath
	if err := a.Bootstrap(); err != nil {
		return "", fmt.Errorf("打开数据库失败: %w", err)
	}
	admin.SetAdminPath(cfg.AdminPath)
	admin.SetCookieSalt(cfg.CookieSalt)
	if err := WriteLock(cfg.SiteRoot); err != nil {
		return "", fmt.Errorf("写入安装锁失败: %w", err)
	}
	return adminPath, nil
}

func probePbootDB(driver, dsn string) error {
	sqlDrv, sqlDSN := "sqlite", "file:"+filepath.ToSlash(dsn)+"?_pragma=busy_timeout(5000)"
	q := `SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?`
	switch driver {
	case "mysql":
		sqlDrv, sqlDSN = "mysql", dsn
		q = `SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name=?`
	case "pgsql":
		sqlDrv, sqlDSN = "pgx", dsn
		q = `SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = current_schema() AND table_name=?`
	}
	db, err := sql.Open(sqlDrv, sqlDSN)
	if err != nil {
		return fmt.Errorf("无法连接数据库: %w", err)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		return fmt.Errorf("无法连接数据库: %w", err)
	}
	var n int
	if err := db.QueryRow(q, "ay_user").Scan(&n); err != nil {
		return fmt.Errorf("检查库结构失败: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("目标库不像 Pboot 站点库（缺少 ay_user），请确认路径/连接")
	}
	return nil
}

func buildDSN(wd, siteRoot string, in Input, driver string) (string, error) {
	switch driver {
	case "sqlite":
		p := strings.TrimSpace(in.SQLitePath)
		if p == "" {
			p = filepath.Join(siteRoot, "data", config.RandomDBFile())
		} else if !filepath.IsAbs(p) {
			p = filepath.Join(wd, filepath.FromSlash(p))
		}
		return p, nil
	case "mysql":
		host := or(in.DBHost, "127.0.0.1")
		port := or(in.DBPort, "3306")
		if in.DBUser == "" || in.DBName == "" {
			return "", fmt.Errorf("请填写 MySQL 用户名和数据库名（库需已创建）")
		}
		cfg := mysqldriver.NewConfig()
		cfg.User = in.DBUser
		cfg.Passwd = in.DBPass
		cfg.Net = "tcp"
		cfg.Addr = net.JoinHostPort(host, port)
		cfg.DBName = in.DBName
		cfg.ParseTime = true
		cfg.Params = map[string]string{"charset": "utf8mb4"}
		return cfg.FormatDSN(), nil
	case "pgsql":
		host := or(in.DBHost, "127.0.0.1")
		port := or(in.DBPort, "5432")
		if in.DBUser == "" || in.DBName == "" {
			return "", fmt.Errorf("请填写 PostgreSQL 用户名和数据库名（库需已创建）")
		}
		u := url.URL{
			Scheme: "postgres",
			User:   url.UserPassword(in.DBUser, in.DBPass),
			Host:   net.JoinHostPort(host, port),
			Path:   "/" + in.DBName,
		}
		q := u.Query()
		q.Set("sslmode", "disable")
		u.RawQuery = q.Encode()
		return u.String(), nil
	default:
		return "", fmt.Errorf("不支持的数据库")
	}
}

func or(a, b string) string {
	if strings.TrimSpace(a) == "" {
		return b
	}
	return a
}

// ApplyMinimalDefaults turns off optional front features (also used after install).
func ApplyMinimalDefaults(a *app.App) {
	for _, k := range []string{
		"message_status", "form_status", "message_send_mail", "form_send_mail",
		"api_open", "register_status", "login_status", "comment_status",
		"comment_anonymous", "comment_send_mail", "llms_open", "spiderlog",
		"open_wap", "watermark_open", "message_rqlogin", "gzip",
	} {
		_ = a.DB.SetConf(k, "0")
	}
	for _, k := range []string{
		"password_strong", "admin_check_code", "message_verify", "comment_verify",
		"message_check_code", "form_check_code", "comment_check_code",
		"register_check_code", "login_check_code", "api_auth",
	} {
		_ = a.DB.SetConf(k, "1")
	}
}

func relToWD(wd, abs string) string {
	rel, err := filepath.Rel(wd, abs)
	if err != nil || strings.HasPrefix(rel, "..") {
		return filepath.ToSlash(abs)
	}
	return filepath.ToSlash(rel)
}

func writePage(w http.ResponseWriter, errMsg string, f formVals) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	errHTML := ""
	if errMsg != "" {
		errHTML = `<p class="err">` + html.EscapeString(errMsg) + `</p>`
	}
	pwd := f.Password
	if pwd == "" {
		pwd = GenPassword()
	}
	mode := f.Mode
	if mode != "import" {
		mode = "install"
	}
	_, _ = fmt.Fprintf(w, pageHTML,
		errHTML,
		checked(mode, "install"), checked(mode, "import"),
		html.EscapeString(f.Title),
		html.EscapeString(f.Username),
		html.EscapeString(pwd),
		sel(f.Driver, "sqlite"), sel(f.Driver, "mysql"), sel(f.Driver, "pgsql"),
		html.EscapeString(f.SQLitePath),
		html.EscapeString(f.DBHost),
		html.EscapeString(f.DBPort),
		html.EscapeString(f.DBUser),
		html.EscapeString(f.DBPass),
		html.EscapeString(f.DBName),
		html.EscapeString(f.Addr),
	)
}

func sel(cur, want string) string {
	if cur == want {
		return " selected"
	}
	return ""
}

func checked(cur, want string) string {
	if cur == want {
		return " checked"
	}
	return ""
}

func writeDone(w http.ResponseWriter, adminPath, password string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	ap := html.EscapeString(adminPath)
	_, _ = fmt.Fprintf(w, doneHTML, ap, ap, html.EscapeString(password))
}

func writeDoneImport(w http.ResponseWriter, adminPath string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	ap := html.EscapeString(adminPath)
	_, _ = fmt.Fprintf(w, doneImportHTML, ap, ap)
}

const pageHTML = `<!doctype html>
<html lang="zh-CN">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>PbootCMS-Go 安装</title>
<style>
body{font-family:system-ui,sans-serif;max-width:28rem;margin:3rem auto;padding:0 1rem;color:#1a1a1a;background:#f6f6f4}
h1{font-size:1.4rem;font-weight:600;margin:0 0 .25rem}
p.sub{color:#666;margin:0 0 1.5rem;font-size:.9rem}
label{display:block;margin:.75rem 0 .25rem;font-size:.85rem}
input,select{width:100%%;box-sizing:border-box;padding:.5rem .6rem;border:1px solid #ccc;border-radius:4px;font-size:1rem}
button{margin-top:1.25rem;width:100%%;padding:.65rem;border:0;border-radius:4px;background:#1a1a1a;color:#fff;font-size:1rem;cursor:pointer}
.err{background:#fee;border:1px solid #e99;color:#900;padding:.6rem .75rem;border-radius:4px}
.hint{font-size:.8rem;color:#888;margin-top:.15rem}
.hid{display:none}
.mode{display:flex;gap:1rem;margin:0 0 1rem;font-size:.95rem}
.mode label{display:flex;align-items:center;gap:.4rem;margin:0;cursor:pointer}
.mode input{width:auto}
</style>
</head>
<body>
<h1>PbootCMS-Go 安装</h1>
<p class="sub">请先选择全新安装，或导入已有 Pboot 站点数据库。</p>
%s
<form method="post" action="/install" autocomplete="off">
<div class="mode">
<label><input type="radio" name="mode" value="install" id="mode-install"%s> 全新安装</label>
<label><input type="radio" name="mode" value="import" id="mode-import"%s> 导入已有库</label>
</div>
<div id="box-fresh">
<label>站点名称</label>
<input name="title" id="fld-title" value="%s">
<label>管理员账号</label>
<input name="username" id="fld-user" pattern="[a-zA-Z][a-zA-Z0-9_]{2,31}" value="%s">
<p class="hint">字母开头，3–32 位字母数字下划线</p>
<label>管理员密码</label>
<input name="password" id="admin_pass" type="text" minlength="8" value="%s" autocomplete="off">
<p class="hint">已自动生成高强度密码，请复制保存；可改。<a href="/install">换一个</a></p>
</div>
<label>数据库</label>
<select name="driver" id="driver">
<option value="sqlite"%s>SQLite（文件库，推荐本地）</option>
<option value="mysql"%s>MySQL / MariaDB</option>
<option value="pgsql"%s>PostgreSQL</option>
</select>
<div id="box-sqlite">
<label>SQLite 路径</label>
<input name="sqlite_path" value="%s">
<p class="hint" id="hint-sqlite-install">已随机文件名；相对项目目录，空库会自动导入结构</p>
<p class="hint hid" id="hint-sqlite-import">指向<strong>已有</strong> Pboot SQLite 库（非空）；upload / template / skin / uploads 请事先拷到 site 下</p>
</div>
<div id="box-remote" class="hid">
<label>主机</label>
<input name="db_host" value="%s">
<label>端口</label>
<input name="db_port" id="db_port" value="%s">
<label>用户名</label>
<input name="db_user" value="%s">
<label>密码</label>
<input name="db_pass" type="password" value="%s">
<label>数据库名</label>
<input name="db_name" value="%s">
<p class="hint" id="hint-remote-install">请先建好空库；安装时会导入表结构。MySQL 默认 3306，PostgreSQL 默认 5432</p>
<p class="hint hid" id="hint-remote-import">填写<strong>已有</strong>站点库连接；不会改库内管理员与内容</p>
</div>
<label>监听地址</label>
<input name="addr" value="%s">
<p class="hint">如 :8080；改端口后需重启进程</p>
<button type="submit" id="btn-go">开始安装</button>
</form>
<script>
(function(){
  var d=document.getElementById('driver');
  var sq=document.getElementById('box-sqlite');
  var rm=document.getElementById('box-remote');
  var port=document.getElementById('db_port');
  var fresh=document.getElementById('box-fresh');
  var btn=document.getElementById('btn-go');
  var title=document.getElementById('fld-title');
  var user=document.getElementById('fld-user');
  var pass=document.getElementById('admin_pass');
  function mode(){
    var imp=document.getElementById('mode-import').checked;
    fresh.className=imp?'hid':'';
    title.required=!imp; user.required=!imp; pass.required=!imp;
    btn.textContent=imp?'开始导入':'开始安装';
    document.getElementById('hint-sqlite-install').className=imp?'hint hid':'hint';
    document.getElementById('hint-sqlite-import').className=imp?'hint':'hint hid';
    document.getElementById('hint-remote-install').className=imp?'hint hid':'hint';
    document.getElementById('hint-remote-import').className=imp?'hint':'hint hid';
  }
  function sync(){
    var v=d.value;
    sq.className=v==='sqlite'?'':'hid';
    rm.className=v==='sqlite'?'hid':'';
    if(v==='mysql'&&(port.value===''||port.value==='5432'))port.value='3306';
    if(v==='pgsql'&&(port.value===''||port.value==='3306'))port.value='5432';
  }
  d.addEventListener('change',sync);
  document.getElementById('mode-install').addEventListener('change',mode);
  document.getElementById('mode-import').addEventListener('change',mode);
  sync();mode();
})();
</script>
</body>
</html>`

const doneHTML = `<!doctype html>
<html lang="zh-CN">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>安装完成</title>
<style>
body{font-family:system-ui,sans-serif;max-width:28rem;margin:3rem auto;padding:0 1rem;color:#1a1a1a;background:#f6f6f4}
a{color:#1a1a1a}
code{background:#eee;padding:.1rem .35rem;border-radius:3px}
</style>
</head>
<body>
<h1>安装完成</h1>
<p>请立即打开后台并妥善保存地址与密码（勿泄露）：</p>
<p>后台：<a href="%s"><code>%s</code></a></p>
<p>密码：<code>%s</code></p>
<p>配置已写入 <code>config.json</code>。监听地址若有改动，请重启进程后再访问。</p>
</body>
</html>`

const doneImportHTML = `<!doctype html>
<html lang="zh-CN">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>导入完成</title>
<style>
body{font-family:system-ui,sans-serif;max-width:28rem;margin:3rem auto;padding:0 1rem;color:#1a1a1a;background:#f6f6f4}
a{color:#1a1a1a}
code{background:#eee;padding:.1rem .35rem;border-radius:3px}
</style>
</head>
<body>
<h1>导入完成</h1>
<p>已接入现有数据库。管理员账号与密码沿用原站；请保存新的后台路径：</p>
<p>后台：<a href="%s"><code>%s</code></a></p>
<p>配置已写入 <code>config.json</code>。若改了监听地址请重启后再访问。模板、上传目录请确认已放到 <code>site/</code>。</p>
</body>
</html>`