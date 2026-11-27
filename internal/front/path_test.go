package front

import (
	"net/http"
	"testing"

	"github.com/zn49/pbootcms-go/internal/urlx"
)

func TestPeelPageListScodeNotPage(t *testing.T) {
	r := urlx.Rules{Type: 3, Suffix: ".html", Break: "_"}
	path, page := peelPage("list_14", r)
	if path != "list_14" || page != 1 {
		t.Fatalf("list_14 -> %q page %d", path, page)
	}
	path, page = peelPage("list_14_2", r)
	if path != "list_14" || page != 2 {
		t.Fatalf("list_14_2 -> %q page %d", path, page)
	}
	path, page = peelPage("about_2", r)
	if path != "about" || page != 2 {
		t.Fatalf("about_2 -> %q page %d", path, page)
	}
	path, page = peelPage("lm4", r)
	if path != "lm4" || page != 1 {
		t.Fatalf("lm4 -> %q page %d", path, page)
	}
}

func TestRequestPathCompatibleList(t *testing.T) {
	r := urlx.Rules{Type: 3, Suffix: ".html", Break: "_"}
	req, _ := http.NewRequest(http.MethodGet, "http://x/?list_14/", nil)
	path, page := requestPath(req, r)
	if path != "list_14" || page != 1 {
		t.Fatalf("got %q page %d", path, page)
	}
}
