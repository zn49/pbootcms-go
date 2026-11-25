package admin

import (
	"path/filepath"
	"testing"
)

func TestAESRoundTrip(t *testing.T) {
	dir := t.TempDir()
	_ = filepath.Join(dir, "data")
	enc := aesEncrypt(dir, "sk-test-key-123456")
	if enc == "" {
		t.Fatal("encrypt empty")
	}
	got := aesDecrypt(dir, enc)
	if got != "sk-test-key-123456" {
		t.Fatalf("got %q", got)
	}
	if maskSecret(got) != "sk-t****3456" {
		t.Fatalf("mask %q", maskSecret(got))
	}
	if !isMaskedSecret("sk-t****3456") {
		t.Fatal("masked detect")
	}
}
