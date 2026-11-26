package admin

import (
	"strings"
	"testing"
)

func TestApplyConstsStripsPHP(t *testing.T) {
	in := `<a href="{php}echo $link{/php}">查看</a>`
	out := applyConsts(in)
	if out != `<a href="">查看</a>` {
		t.Fatalf("got %q", out)
	}
}

func TestUeditorHolderUnescapes(t *testing.T) {
	d := Data{Vars: map[string]any{"content": map[string]string{"content": "&lt;p&gt;hi&lt;/p&gt;"}}}
	got := ueditorHolder("$content->content", d)
	if got != "<p>hi</p>" {
		t.Fatalf("got %q", got)
	}
}

func TestExpandFunsUntilDone(t *testing.T) {
	d := Data{Funs: map[string]func([]string) string{
		"n": func(args []string) string { return "x" },
	}}
	var b strings.Builder
	const n = 2500
	for i := 0; i < n; i++ {
		b.WriteString("{fun=n()}")
	}
	out := expandFuns(b.String(), d)
	if strings.Contains(out, "{fun=") {
		t.Fatalf("unexpanded fun left")
	}
	if got := strings.Count(out, "x"); got != n {
		t.Fatalf("got %d xs want %d", got, n)
	}
}
