package admin

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/zn49/pbootcms-go/internal/app"
	"github.com/zn49/pbootcms-go/internal/auth"
	"github.com/zn49/pbootcms-go/internal/config"
	"github.com/zn49/pbootcms-go/internal/front"
	"github.com/zn49/pbootcms-go/internal/fsperm"
)

// expireCookie clears a cookie on the client.
func expireCookie(w http.ResponseWriter, name string) {
	// clear cookie; Secure flag not required for MaxAge=-1 delete
	http.SetCookie(w, &http.Cookie{Name: name, Value: "", Path: "/", MaxAge: -1, SameSite: http.SameSiteLaxMode})
}

func clearLegacyCookies(w http.ResponseWriter, extra ...string) {
	for _, n := range LegacyCookieNames() {
		expireCookie(w, n)
	}
	for _, n := range extra {
		if n != "" {
			expireCookie(w, n)
		}
	}
}

// dbInsecure is true when sqlite DSN still contains a known default name.
func dbInsecure(a *app.App) bool {
	if a.DB == nil || a.DB.Driver != "sqlite" {
		return false
	}
	low := strings.ToLower(a.Cfg.DSN)
	return strings.Contains(low, "pbootcms") || strings.Contains(low, "pbootcms.db") || strings.Contains(low, "gbootcms.db")
}

// modSQLiteDB copies the open sqlite file to a random name, updates config, reopens DB.
func modSQLiteDB(a *app.App) (newAbs string, err error) {
	if a.DB == nil || a.DB.Driver != "sqlite" {
		return a.Cfg.DSN, nil
	}
	oldAbs := a.Cfg.DSN
	dir := filepath.Dir(oldAbs)
	newName := config.RandomDBFile()
	newAbs = filepath.Join(dir, newName)

	if err := a.ClearBootstrap(); err != nil {
		return "", err
	}
	// copy then remove old (Windows-friendly vs rename-while-locked)
	in, err := os.ReadFile(oldAbs)
	if err != nil {
		_ = a.Bootstrap()
		return "", err
	}
	if err := fsperm.WriteFile(newAbs, in, fsperm.FilePrivate); err != nil {
		_ = a.Bootstrap()
		return "", err
	}
	a.Cfg.DSN = newAbs
	if err := a.Bootstrap(); err != nil {
		// rollback
		a.Cfg.DSN = oldAbs
		_ = a.Bootstrap()
		_ = os.Remove(newAbs)
		return "", err
	}
	_ = os.Remove(oldAbs)
	return newAbs, nil
}

func relDSNForConfig(a *app.App, abs string) string {
	wd, _ := os.Getwd()
	if rel, err := filepath.Rel(wd, abs); err == nil && !strings.HasPrefix(rel, "..") {
		return filepath.ToSlash(rel)
	}
	if rel, err := filepath.Rel(a.Cfg.SiteRoot, abs); err == nil {
		// prefer site_root-relative only if config historically used that — keep cwd-relative when possible
		_ = rel
	}
	return filepath.ToSlash(abs)
}

// resetFingerprint rotates admin_path, cookie salt, and sqlite filename; applies in-process.
func resetFingerprint(a *app.App, w http.ResponseWriter, r *http.Request) {
	oldAdmin := CookieName()
	oldFront := FrontCookieName()

	newPath := config.RandomAdminPath()
	newSalt := config.RandomSalt()
	fields := map[string]any{
		"admin_path":  newPath,
		"cookie_salt": newSalt,
	}

	var newDSN string
	if a.DB != nil && a.DB.Driver == "sqlite" {
		abs, err := modSQLiteDB(a)
		if err != nil {
			front.JSON(w, 0, "数据库改名失败: "+err.Error())
			return
		}
		newDSN = relDSNForConfig(a, abs)
		fields["dsn"] = newDSN
	}

	cfgPath := a.CfgPath
	if cfgPath == "" {
		cfgPath = "config.json"
	}
	if err := config.Patch(cfgPath, fields); err != nil {
		front.JSON(w, 0, "写入配置失败: "+err.Error())
		return
	}

	a.Cfg.AdminPath = newPath
	a.Cfg.CookieSalt = newSalt
	SetAdminPath(newPath)
	SetCookieSalt(newSalt)

	clearLegacyCookies(w, oldAdmin, oldFront)
	// keep user logged in under new cookie name
	if c, err := r.Cookie(oldAdmin); err == nil && c.Value != "" {
		auth.SessionCookie(w, r, CookieName(), c.Value)
	}

	front.JSON(w, 1, map[string]string{
		"admin_path": newPath,
		"admin_url":  newPath + "/Index/home",
		"dsn":        newDSN,
		"msg":        "指纹已重置，请使用新后台地址访问",
	})
}
