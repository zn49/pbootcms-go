package admin

import "testing"

func TestSafeRemoteURL(t *testing.T) {
	if err := safeRemoteURL("http://127.0.0.1/a"); err == nil {
		t.Fatal("loopback")
	}
	if err := safeRemoteURL("http://192.168.1.1/"); err == nil {
		t.Fatal("private")
	}
	if err := safeRemoteURL("http://169.254.169.254/"); err == nil {
		t.Fatal("metadata")
	}
	if err := safeRemoteURL("file:///etc/passwd"); err == nil {
		t.Fatal("file")
	}
	if err := safeRemoteURL("https://example.com/a.png"); err != nil {
		t.Fatal(err)
	}
}
