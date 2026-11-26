package httpx

import (
	"log/slog"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/zn49/pbootcms-go/internal/admin"
	"github.com/zn49/pbootcms-go/internal/api"
	"github.com/zn49/pbootcms-go/internal/app"
	"github.com/zn49/pbootcms-go/internal/front"
	"github.com/zn49/pbootcms-go/internal/install"
)

func Listen(a *app.App) error {
	applyAdminCfg(a)
	driver, adm := "-", "-"
	if a.IsBootstrapped() && a.DB != nil {
		driver = a.DB.Driver
		adm = admin.Path()
	} else if install.Needed(a) {
		adm = "/install"
	}
	slog.Info("listen", "addr", a.Cfg.Addr, "root", a.Cfg.SiteRoot, "driver", driver, "admin", adm)
	return http.ListenAndServe(a.Cfg.Addr, Handler(a))
}

func applyAdminCfg(a *app.App) {
	admin.SetAdminPath(a.Cfg.AdminPath)
	admin.SetCookieSalt(a.Cfg.CookieSalt)
	admin.SetSiteDirConst(a.Cfg.SiteDir)
}

func Handler(a *app.App) http.Handler {
	applyAdminCfg(a)
	mux := http.NewServeMux()
	mux.HandleFunc("/install", func(w http.ResponseWriter, r *http.Request) {
		install.Serve(a, w, r)
	})
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) { api.Serve(a, w, r) })
	mux.HandleFunc("/core/extend/ueditor/php/controller.php", func(w http.ResponseWriter, r *http.Request) {
		admin.UEditor(a, w, r)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if install.Needed(a) {
			http.Redirect(w, r, "/install", http.StatusFound)
			return
		}
		if serveAdmin(a, w, r) {
			return
		}
		if serveFile(a, w, r) {
			return
		}
		front.Serve(a, w, r)
	})
	return securityHeaders(allowIP(a, mux))
}

// serveAdmin matches the current (possibly rotated) admin path dynamically.
func serveAdmin(a *app.App, w http.ResponseWriter, r *http.Request) bool {
	prefix := admin.Path()
	p := r.URL.Path
	if prefix != "/admin" && (p == "/admin" || strings.HasPrefix(p, "/admin/")) {
		http.NotFound(w, r)
		return true
	}
	if p != prefix && !strings.HasPrefix(p, prefix+"/") {
		return false
	}
	rest := strings.TrimPrefix(p, prefix)
	r2 := *r
	u := *r.URL
	if rest == "" || rest == "/" {
		u.Path = "/admin"
	} else {
		u.Path = "/admin" + rest
	}
	r2.URL = &u
	admin.Serve(a, admin.WrapPath(w), &r2)
	return true
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "SAMEORIGIN")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		h.Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		h.Del("Server")
		h.Del("X-Powered-By")
		if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
			h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		next.ServeHTTP(w, r)
	})
}

func serveFile(a *app.App, w http.ResponseWriter, r *http.Request) bool {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		return false
	}
	p := path.Clean("/" + r.URL.Path) // URL path: always slash (filepath.Clean breaks Windows)
	if !publicAsset(p) {
		return false
	}
	low := strings.ToLower(p)
	if strings.HasPrefix(low, "/static/backup") || strings.HasSuffix(low, ".sql") || strings.HasSuffix(low, ".db") || strings.HasSuffix(low, ".sqlite") || strings.HasSuffix(low, ".bak") || strings.HasSuffix(low, ".php") || strings.HasSuffix(low, ".go") || strings.HasSuffix(low, ".env") {
		return false
	}
	full := filepath.Clean(filepath.Join(a.Cfg.SiteRoot, filepath.FromSlash(p)))
	root := filepath.Clean(a.Cfg.SiteRoot)
	rel, err := filepath.Rel(root, full)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return false
	}
	st, err := os.Stat(full)
	if err != nil || st.IsDir() {
		return false
	}
	http.ServeFile(w, r, full)
	return true
}

// publicAsset allowlists files the browser may fetch under site_root.
// Admin CSS/JS live at /apps/admin/view/; UEditor at /core/extend/ueditor/;
// theme images/css under /template/ (theme .html is not public).
func publicAsset(p string) bool {
	low := strings.ToLower(p)
	switch low {
	case "/favicon.ico", "/robots.txt":
		return true
	}
	for _, pre := range []string{
		"/static/",
		"/skin/",
		"/uploads/",
		"/apps/admin/view/",
		"/core/extend/ueditor/",
		"/template/",
	} {
		if !strings.HasPrefix(low, pre) {
			continue
		}
		// do not expose front theme HTML sources
		if strings.HasPrefix(low, "/template/") && strings.HasSuffix(low, ".html") {
			return false
		}
		return true
	}
	// root verification files (baidu/google), not nested paths
	if strings.Count(low, "/") == 1 && (strings.HasSuffix(low, ".txt") || strings.HasSuffix(low, ".html")) {
		return !strings.Contains(low, "..")
	}
	return false
}

func allowIP(a *app.App, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ipAllowed(a, r) {
			next.ServeHTTP(w, r)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api/") {
			front.JSON(w, 0, "forbidden")
			return
		}
		http.Error(w, "forbidden", http.StatusForbidden)
	})
}

func ipAllowed(a *app.App, r *http.Request) bool {
	if a.DB == nil {
		return true
	}
	ip := r.RemoteAddr
	if i := strings.LastIndex(ip, ":"); i >= 0 {
		ip = ip[:i]
	}
	if allow := a.DB.Conf("ip_allow"); allow != "" {
		ok := false
		for _, item := range strings.Split(allow, ",") {
			if strings.TrimSpace(item) == ip {
				ok = true
			}
		}
		if !ok {
			return false
		}
	}
	if deny := a.DB.Conf("ip_deny"); deny != "" {
		for _, item := range strings.Split(deny, ",") {
			if strings.TrimSpace(item) == ip {
				return false
			}
		}
	}
	return true
}
