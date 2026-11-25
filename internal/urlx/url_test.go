package urlx

import "testing"

func TestMode3(t *testing.T) {
	r := Rules{Type: 3, Suffix: ".html", Break: "_"}
	if got := r.Sort("aboutus", "about", "1"); got != "/?aboutus/" {
		t.Fatalf("sort %s", got)
	}
	if got := r.Content("company", "list", "3", "4", ""); got != "/?company/4.html" {
		t.Fatalf("content %s", got)
	}
	if got := r.Page("aboutus", 2); got != "/?aboutus_2/" {
		t.Fatalf("page %s", got)
	}
	if got := Home(r, "", 0); got != "/?" {
		t.Fatalf("home %s", got)
	}
}

func TestMode2(t *testing.T) {
	r := Rules{Type: 2}
	if got := r.Content("news", "list", "2", "9", "hello"); got != "/news/hello.html" {
		t.Fatalf("content %s", got)
	}
}
