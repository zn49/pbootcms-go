package httpx

import "testing"

func TestPublicAsset(t *testing.T) {
	allow := []string{
		"/favicon.ico",
		"/robots.txt",
		"/static/a.png",
		"/skin/css/x.css",
		"/uploads/1.jpg",
		"/baidu_verify.txt",
		"/apps/admin/view/default/css/comm.css",
		"/apps/admin/view/default/js/comm.js",
		"/core/extend/ueditor/ueditor.config.js",
		"/template/default/images/user.png",
	}
	deny := []string{
		"/",
		"/apps/admin/controller/x",
		"/data/.app_salt",
		"/template/default/html/index.html",
	}
	for _, p := range allow {
		if !publicAsset(p) {
			t.Fatalf("want allow %s", p)
		}
	}
	for _, p := range deny {
		if publicAsset(p) {
			t.Fatalf("want deny %s", p)
		}
	}
}
