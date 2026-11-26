package admin

import (
	"encoding/json"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/zn49/pbootcms-go/internal/app"
	"github.com/zn49/pbootcms-go/internal/front"
	"github.com/zn49/pbootcms-go/internal/fsperm"
)

var ueditorComment = regexp.MustCompile(`(?s)/\*.*?\*/`)

// UEditor handles /core/extend/ueditor/php/controller.php (config + uploads).
func UEditor(a *app.App, w http.ResponseWriter, r *http.Request) {
	sess := adminSess(a, r)
	if sess == nil || sess.Values["id"] == "" {
		ueditorJSON(w, map[string]any{"state": "权限不足，请重新登录"})
		return
	}
	action := r.URL.Query().Get("action")
	cfgPath := filepath.Join(a.Cfg.SiteRoot, "core", "extend", "ueditor", "php", "config.json")
	raw, err := os.ReadFile(cfgPath)
	if err != nil {
		ueditorJSON(w, map[string]any{"state": "配置文件不存在"})
		return
	}
	clean := ueditorComment.ReplaceAll(raw, nil)
	switch action {
	case "config":
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_, _ = w.Write(clean)
		return
	case "uploadimage", "uploadscrawl", "uploadvideo", "uploadfile":
		ueditorUpload(a, w, r, action == "uploadscrawl")
		return
	case "listimage", "listfile":
		ueditorList(a, w, r, action == "listimage")
		return
	case "catchimage":
		ueditorCatch(a, w, r)
		return
	default:
		ueditorJSON(w, map[string]any{"state": "请求地址出错"})
	}
}

func ueditorUpload(a *app.App, w http.ResponseWriter, r *http.Request, base64 bool) {
	_ = r.ParseMultipartForm(20 << 20)
	field := "upfile"
	f, hdr, err := r.FormFile(field)
	if err != nil {
		f, hdr, err = r.FormFile("file")
	}
	if err != nil && !base64 {
		ueditorJSON(w, map[string]any{"state": "没有文件"})
		return
	}
	var data []byte
	var orig, ext string
	if err == nil {
		defer f.Close()
		data, err = io.ReadAll(f)
		if err != nil {
			ueditorJSON(w, map[string]any{"state": "读取失败"})
			return
		}
		orig = hdr.Filename
		ext = strings.ToLower(strings.TrimPrefix(filepath.Ext(orig), "."))
	} else {
		ueditorJSON(w, map[string]any{"state": "没有文件"})
		return
	}
	if !uploadExt[ext] {
		ueditorJSON(w, map[string]any{"state": "不允许的扩展名"})
		return
	}
	day := time.Now().Format("20060102")
	dir := filepath.Join(a.Cfg.SiteRoot, "static", "upload", "image", day)
	_ = fsperm.MkdirAll(dir, fsperm.DirUpload)
	name := strconv.FormatInt(time.Now().UnixNano(), 10) + "." + ext
	dst := filepath.Join(dir, name)
	if err := fsperm.WriteFile(dst, data, fsperm.FileUpload); err != nil {
		ueditorJSON(w, map[string]any{"state": "保存失败"})
		return
	}
	if a.DB.Conf("watermark_open") == "1" && (ext == "jpg" || ext == "jpeg" || ext == "png") {
		front.Watermark(dst, a.DB.Conf("watermark_text"))
	}
	url := "/static/upload/image/" + day + "/" + name
	ueditorJSON(w, map[string]any{
		"state":    "SUCCESS",
		"url":      url,
		"title":    name,
		"original": orig,
		"type":     "." + ext,
		"size":     len(data),
	})
}

func ueditorList(a *app.App, w http.ResponseWriter, r *http.Request, images bool) {
	root := filepath.Join(a.Cfg.SiteRoot, "static", "upload")
	start, _ := strconv.Atoi(r.URL.Query().Get("start"))
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))
	if size <= 0 {
		size = 20
	}
	if start < 0 {
		start = 0
	}
	var files []map[string]any
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(d.Name()))
		if images {
			switch ext {
			case ".png", ".jpg", ".jpeg", ".gif", ".bmp", ".webp", ".svg":
			default:
				return nil
			}
		}
		rel, err := filepath.Rel(a.Cfg.SiteRoot, path)
		if err != nil || strings.HasPrefix(rel, "..") {
			return nil
		}
		url := "/" + filepath.ToSlash(rel)
		mtime := ""
		if info, e := d.Info(); e == nil {
			mtime = info.ModTime().Format("2006-01-02 15:04:05")
		}
		files = append(files, map[string]any{"url": url, "mtime": mtime})
		return nil
	})
	// newest first
	for i, j := 0, len(files)-1; i < j; i, j = i+1, j-1 {
		files[i], files[j] = files[j], files[i]
	}
	total := len(files)
	end := start + size
	if end > total {
		end = total
	}
	list := []map[string]any{}
	if start < total {
		list = files[start:end]
	}
	state := "SUCCESS"
	if total == 0 {
		state = "no match file"
	}
	ueditorJSON(w, map[string]any{"state": state, "list": list, "start": start, "total": total})
}

func ueditorCatch(a *app.App, w http.ResponseWriter, r *http.Request) {
	if a.DB.Conf("ueditor_catch_remote") != "1" {
		ueditorJSON(w, map[string]any{"state": "远程图片抓取未开启", "list": []any{}})
		return
	}
	_ = r.ParseForm()
	sources := append(r.Form["source[]"], r.Form["source"]...)
	var list []map[string]any
	client := ssrfHTTPClient(15 * time.Second)
	for _, src := range sources {
		src = strings.TrimSpace(src)
		item := map[string]any{"state": "ERROR", "url": "", "size": 0, "title": "", "original": "", "source": src}
		u, err := url.Parse(src)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			item["state"] = "链接无效"
			list = append(list, item)
			continue
		}
		if err := safeRemoteURL(src); err != nil {
			item["state"] = err.Error()
			list = append(list, item)
			continue
		}
		resp, err := client.Get(src)
		if err != nil {
			item["state"] = "抓取失败"
			list = append(list, item)
			continue
		}
		data, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
		resp.Body.Close()
		if err != nil || resp.StatusCode >= 400 || len(data) == 0 {
			item["state"] = "抓取失败"
			list = append(list, item)
			continue
		}
		ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(u.Path), "."))
		if ext == "" {
			ct := resp.Header.Get("Content-Type")
			switch {
			case strings.Contains(ct, "png"):
				ext = "png"
			case strings.Contains(ct, "gif"):
				ext = "gif"
			case strings.Contains(ct, "webp"):
				ext = "webp"
			default:
				ext = "jpg"
			}
		}
		if !uploadExt[ext] {
			item["state"] = "不允许的扩展名"
			list = append(list, item)
			continue
		}
		day := time.Now().Format("20060102")
		dir := filepath.Join(a.Cfg.SiteRoot, "static", "upload", "image", day)
		_ = fsperm.MkdirAll(dir, fsperm.DirUpload)
		name := strconv.FormatInt(time.Now().UnixNano(), 10) + "." + ext
		dst := filepath.Join(dir, name)
		if err := fsperm.WriteFile(dst, data, fsperm.FileUpload); err != nil {
			item["state"] = "保存失败"
			list = append(list, item)
			continue
		}
		rel := "/static/upload/image/" + day + "/" + name
		item["state"] = "SUCCESS"
		item["url"] = rel
		item["size"] = len(data)
		item["title"] = name
		item["original"] = filepath.Base(u.Path)
		list = append(list, item)
	}
	state := "ERROR"
	if len(list) > 0 {
		for _, it := range list {
			if it["state"] == "SUCCESS" {
				state = "SUCCESS"
				break
			}
		}
	}
	ueditorJSON(w, map[string]any{"state": state, "list": list})
}

func ueditorJSON(w http.ResponseWriter, v map[string]any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(v)
}
