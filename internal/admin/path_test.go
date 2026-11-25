package admin

import "testing"

func TestMapPublicURL(t *testing.T) {
	SetAdminPath("/x7k2m9")
	SetCookieSalt("testsalt")
	defer SetAdminPath("/admin")
	defer SetCookieSalt("default")
	in := `href="/admin/Index/home" data-preurl="/admin" src="/apps/admin/view/a.css"`
	out := mapPublicURL(in)
	if !contains(out, `href="/x7k2m9/Index/home"`) || !contains(out, `data-preurl="/x7k2m9"`) {
		t.Fatalf("rewrite failed: %s", out)
	}
	if !contains(out, `/apps/admin/view`) {
		t.Fatal("static theme path broken")
	}
	if mapPublicLoc("/admin/Index/home") != "/x7k2m9/Index/home" {
		t.Fatal("loc")
	}
	if CookieName() == "gboot_admin" || FrontCookieName() == "gboot_front" {
		t.Fatal("cookie still branded")
	}
	if CookieName() == FrontCookieName() {
		t.Fatal("admin/front cookie collision")
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || stringIndex(s, sub) >= 0)
}

func stringIndex(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
