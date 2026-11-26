package admin

import "testing"

func TestPathKVAmpersand(t *testing.T) {
	parts := []string{"Single", "index", "mcode", "1&scode=21"}
	kv := pathKV(parts)
	if kv["mcode"] != "1" || kv["scode"] != "21" {
		t.Fatalf("got %#v", kv)
	}
}
