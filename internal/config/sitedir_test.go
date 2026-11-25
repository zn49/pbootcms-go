package config

import "testing"

func TestNormalizeSiteDir(t *testing.T) {
	cases := map[string]string{
		"":        "",
		"/":       "",
		"cms":     "/cms",
		"/cms/":   "/cms",
		"\\cms\\": "/cms",
	}
	for in, want := range cases {
		if got := NormalizeSiteDir(in); got != want {
			t.Fatalf("%q: got %q want %q", in, got, want)
		}
	}
}
