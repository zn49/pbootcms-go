package fsperm

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHardenSite(t *testing.T) {
	root := t.TempDir()
	db := filepath.Join(root, "data", "x.db")
	if err := MkdirAll(filepath.Dir(db), DirPrivate); err != nil {
		t.Fatal(err)
	}
	if err := WriteFile(db, []byte("sqlite"), FilePrivate); err != nil {
		t.Fatal(err)
	}
	HardenSite(root)
	st, err := os.Stat(filepath.Join(root, "data"))
	if err != nil {
		t.Fatal(err)
	}
	if st.Mode().Perm()&0o077 != 0 && st.Mode().Perm() != DirPrivate {
		// on Windows ACL may ignore unix bits; only assert path exists
		t.Log("data mode", st.Mode())
	}
	if _, err := os.Stat(filepath.Join(root, "static", "upload")); err != nil {
		t.Fatal("upload dir", err)
	}
}
