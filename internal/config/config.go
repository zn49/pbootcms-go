package config

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gbootcms/internal/fsperm"
)

// File is process config. Site settings stay in ay_config.
type File struct {
	Addr      string `json:"addr"`
	Driver    string `json:"driver"`
	DSN       string `json:"dsn"`
	SiteRoot  string `json:"site_root"`
	SiteDir   string `json:"site_dir"`   // subdir mount, like PHP SITE_DIR ("/cms"); empty = root
	AdminPath  string `json:"admin_path"`  // public admin prefix, e.g. "/x7k2m9"; "auto" = generate once
	CookieSalt string `json:"cookie_salt"` // randomizes session cookie names
}

// Default returns in-memory defaults used when config.json is missing (install mode).
func Default() File {
	wd, _ := os.Getwd()
	root := filepath.Join(wd, "site")
	return File{
		Addr:     ":8080",
		Driver:   "sqlite",
		DSN:      filepath.Join(root, "data", "gbootcms.db"),
		SiteRoot: root,
	}
}

func Load(path string) (File, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return File{}, err
	}
	b = bytes.TrimPrefix(b, []byte{0xEF, 0xBB, 0xBF}) // PowerShell UTF-8 BOM
	var f File
	if err := json.Unmarshal(b, &f); err != nil {
		return File{}, err
	}
	if f.Addr == "" {
		f.Addr = ":8080"
	}
	ApplyEnv(&f)
	f.Driver = NormalizeDriver(f.Driver)
	if f.Driver == "" {
		f.Driver = "sqlite"
	}
	if !ValidDriver(f.Driver) {
		return File{}, fmt.Errorf("不支持的数据库驱动 %q，仅支持 sqlite / mysql / pgsql", f.Driver)
	}
	if f.SiteRoot == "" {
		f.SiteRoot = "."
	}
	if !filepath.IsAbs(f.SiteRoot) {
		wd, _ := os.Getwd()
		f.SiteRoot = filepath.Join(wd, f.SiteRoot)
	}
	f.SiteDir = NormalizeSiteDir(f.SiteDir)
	rawAdmin := strings.TrimSpace(f.AdminPath)
	if strings.EqualFold(rawAdmin, "auto") || strings.EqualFold(rawAdmin, "random") {
		f.AdminPath = RandomAdminPath()
		var raw map[string]any
		if json.Unmarshal(b, &raw) == nil {
			raw["admin_path"] = f.AdminPath
			if nb, err := json.MarshalIndent(raw, "", "  "); err == nil {
				_ = fsperm.WriteFile(path, append(nb, '\n'), fsperm.FilePrivate)
			}
		}
	} else {
		f.AdminPath = NormalizeAdminPath(f.AdminPath)
	}
	if strings.TrimSpace(f.CookieSalt) == "" {
		f.CookieSalt = RandomSalt()
		var raw map[string]any
		if json.Unmarshal(b, &raw) == nil {
			raw["cookie_salt"] = f.CookieSalt
			raw["admin_path"] = f.AdminPath
			if nb, err := json.MarshalIndent(raw, "", "  "); err == nil {
				_ = fsperm.WriteFile(path, append(nb, '\n'), fsperm.FilePrivate)
			}
		}
	}
	if f.DSN == "" {
		if f.Driver != "sqlite" {
			return File{}, fmt.Errorf("%s 需要配置 dsn", f.Driver)
		}
		f.DSN = filepath.Join(f.SiteRoot, "data", "gbootcms.db")
	} else if f.Driver == "sqlite" && !filepath.IsAbs(f.DSN) && !hasScheme(f.DSN) {
		wd, _ := os.Getwd()
		f.DSN = filepath.Join(wd, f.DSN)
	}
	fsperm.HardenFile(path, fsperm.FilePrivate)
	return f, nil
}

// ApplyEnv overlays GBOOT_* env vars (container / systemd).
func ApplyEnv(f *File) {
	if v := strings.TrimSpace(os.Getenv("GBOOT_ADDR")); v != "" {
		f.Addr = v
	}
	if v := strings.TrimSpace(os.Getenv("GBOOT_DRIVER")); v != "" {
		f.Driver = NormalizeDriver(v)
	}
	if v := strings.TrimSpace(os.Getenv("GBOOT_DSN")); v != "" {
		f.DSN = v
	}
	if v := strings.TrimSpace(os.Getenv("GBOOT_SITE_ROOT")); v != "" {
		f.SiteRoot = v
		if !filepath.IsAbs(f.SiteRoot) {
			wd, _ := os.Getwd()
			f.SiteRoot = filepath.Join(wd, f.SiteRoot)
		}
	}
	if v := strings.TrimSpace(os.Getenv("GBOOT_SITE_DIR")); v != "" {
		f.SiteDir = NormalizeSiteDir(v)
	}
	if v := strings.TrimSpace(os.Getenv("GBOOT_ADMIN_PATH")); v != "" {
		f.AdminPath = NormalizeAdminPath(v)
	}
	if v := strings.TrimSpace(os.Getenv("GBOOT_COOKIE_SALT")); v != "" {
		f.CookieSalt = v
	}
}

// NormalizeAdminPath returns "/admin" or "/custom" (no trailing slash).
func NormalizeAdminPath(d string) string {
	d = strings.TrimSpace(strings.ReplaceAll(d, "\\", "/"))
	if d == "" || d == "/" {
		return "/admin"
	}
	if !strings.HasPrefix(d, "/") {
		d = "/" + d
	}
	d = strings.TrimRight(d, "/")
	// block path tricks / reserved
	low := strings.ToLower(d)
	if low == "/api" || low == "/static" || low == "/core" || low == "/data" || low == "/apps" || strings.Contains(d, "..") {
		return "/admin"
	}
	return d
}

// RandomAdminPath returns "/"+8 hex chars.
func RandomAdminPath() string {
	var b [4]byte
	_, _ = rand.Read(b[:])
	return "/" + hex.EncodeToString(b[:])
}

// NormalizeDriver maps aliases to sqlite|mysql|pgsql.
func NormalizeDriver(d string) string {
	switch strings.ToLower(strings.TrimSpace(d)) {
	case "", "sqlite", "sqlite3":
		if d == "" {
			return ""
		}
		return "sqlite"
	case "mysql", "mariadb":
		return "mysql"
	case "pgsql", "postgres", "postgresql", "pg":
		return "pgsql"
	default:
		return strings.ToLower(strings.TrimSpace(d))
	}
}

func ValidDriver(d string) bool {
	switch d {
	case "sqlite", "mysql", "pgsql":
		return true
	default:
		return false
	}
}

// NormalizeSiteDir returns "" or "/subdir" (no trailing slash).
func NormalizeSiteDir(d string) string {
	d = strings.TrimSpace(strings.ReplaceAll(d, "\\", "/"))
	if d == "" || d == "/" {
		return ""
	}
	if !strings.HasPrefix(d, "/") {
		d = "/" + d
	}
	return strings.TrimRight(d, "/")
}

func hasScheme(dsn string) bool {
	return len(dsn) > 5 && (dsn[:5] == "file:" || dsn[:5] == "tcp(" || contains(dsn, "@tcp(") || strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://"))
}

func contains(s, sub string) bool {
	return strings.Contains(s, sub)
}
