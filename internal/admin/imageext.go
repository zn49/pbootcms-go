package admin

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"gbootcms/internal/app"
	"gbootcms/internal/auth"
	"gbootcms/internal/domain"
	"gbootcms/internal/front"
	"gbootcms/internal/fsperm"
)

func imageExtPage(a *app.App, w http.ResponseWriter, r *http.Request, sess *auth.Session) {
	d := systemBag(a, w, r, sess, "ImageExt")
	writePHP(a, w, "system/extimage.html", d)
}

func imageExtCheckDataFile(a *app.App, w http.ResponseWriter, r *http.Request) {
	page := domain.Atoi(r.URL.Query().Get("page"), 1)
	if page < 1 {
		page = 1
	}
	const pageSize = 30
	files, err := redundantFiles(a)
	if err != nil {
		front.JSON(w, 0, err.Error())
		return
	}
	start := (page - 1) * pageSize
	if start > len(files) {
		start = len(files)
	}
	end := start + pageSize
	if end > len(files) {
		end = len(files)
	}
	scheme := "http://"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https://"
	}
	host := r.Host
	var data []map[string]string
	for _, key := range files[start:end] {
		row := map[string]string{
			"real_path":   key,
			"static_path": scheme + host + key,
			"update_time": "",
		}
		rel := strings.TrimPrefix(filepath.ToSlash(key), "/static/upload/")
		fsPath := filepath.Join(a.Cfg.SiteRoot, "static", "upload", filepath.FromSlash(rel))
		if st, err := os.Stat(fsPath); err == nil {
			row["update_time"] = st.ModTime().Format("2006-01-02 15:04:05")
		}
		data = append(data, row)
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"code": 0, "msg": "", "count": len(files), "data": data,
	})
}

func imageExtDo(a *app.App, w http.ResponseWriter, r *http.Request) {
	files, err := redundantFiles(a)
	if err != nil {
		front.JSON(w, 0, err.Error())
		return
	}
	want := map[string]bool{}
	if r.FormValue("type") != "1" {
		for _, p := range r.Form["real_path"] {
			want[p] = true
		}
		if raw := r.FormValue("list"); raw != "" {
			var list []string
			if json.Unmarshal([]byte(raw), &list) == nil {
				for _, p := range list {
					want[p] = true
				}
			}
		}
	}
	moved := 0
	for _, key := range files {
		if len(want) > 0 && !want[key] {
			continue
		}
		if moveUpload(a, key) == nil {
			moved++
		}
	}
	front.JSON(w, 1, fmt.Sprintf("成功迁移 %d 个文件", moved))
}

// shortcut: loads every text column to see which uploads are referenced, switch to a path index if the library is large
func redundantFiles(a *app.App) ([]string, error) {
	root := filepath.Join(a.Cfg.SiteRoot, "static", "upload")
	blob := referencedText(a)
	var out []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || strings.HasPrefix(d.Name(), ".") {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil || strings.HasPrefix(rel, "..") {
			return nil
		}
		key := "/static/upload/" + filepath.ToSlash(rel)
		if !strings.Contains(blob, key) {
			out = append(out, key)
		}
		return nil
	})
	if os.IsNotExist(err) {
		return nil, nil
	}
	return out, err
}

func referencedText(a *app.App) string {
	var b strings.Builder
	for _, table := range []string{"ay_content", "ay_content_ext", "ay_content_sort", "ay_slide", "ay_link", "ay_company", "ay_site", "ay_label", "ay_member"} {
		rows, err := a.DB.Maps("SELECT * FROM " + table)
		if err != nil {
			continue
		}
		for _, row := range rows {
			for _, v := range row {
				b.WriteString(v)
				b.WriteByte('\n')
			}
		}
	}
	return b.String()
}

func moveUpload(a *app.App, key string) error {
	rel := strings.TrimPrefix(filepath.ToSlash(key), "/static/upload/")
	if rel == key || rel == "" || strings.Contains(rel, "..") {
		return fmt.Errorf("路径非法")
	}
	src := filepath.Join(a.Cfg.SiteRoot, "static", "upload", filepath.FromSlash(rel))
	dst := filepath.Join(a.Cfg.SiteRoot, "static", "backup", "ImageExt", filepath.FromSlash(rel))
	if err := fsperm.MkdirAll(filepath.Dir(dst), fsperm.DirPrivate); err != nil {
		return err
	}
	if err := os.Rename(src, dst); err != nil {
		return err
	}
	fsperm.HardenFile(dst, fsperm.FilePrivate)
	return nil
}
