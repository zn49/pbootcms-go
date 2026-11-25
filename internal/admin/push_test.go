package admin

import "testing"

func TestCheckIndexnowKey(t *testing.T) {
	if checkIndexnowKey("short") {
		t.Fatal("short key")
	}
	if !checkIndexnowKey("abcd1234") {
		t.Fatal("valid key")
	}
	if checkIndexnowKey("bad key!!") {
		t.Fatal("bad chars")
	}
}

func TestFilterSameHost(t *testing.T) {
	got := filterSameHost([]string{
		"https://a.com/x",
		"https://b.com/y",
		"https://a.com/x",
		"not-a-url",
	}, "a.com")
	if len(got) != 1 || got[0] != "https://a.com/x" {
		t.Fatalf("%v", got)
	}
}
