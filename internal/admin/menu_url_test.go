package admin

import "testing"

func TestMenuGroupURL(t *testing.T) {
	if !menuGroupURL("/admin/M1001/index") {
		t.Fatal("M1001 should be group")
	}
	if menuGroupURL("/admin/Member/index") {
		t.Fatal("Member is not group")
	}
	if menuGroupURL("/admin/MemberComment/index") {
		t.Fatal("MemberComment is not group")
	}
	if menuGroupURL("/admin/Content/index") {
		t.Fatal("Content is not group")
	}
}
