package cache

import (
	"crypto/sha1"
	"encoding/hex"
	"os"
	"path/filepath"
	"time"

	"gbootcms/internal/fsperm"
)

type Page struct {
	Dir string
}

func (p Page) file(key string) string {
	sum := sha1.Sum([]byte(key))
	name := hex.EncodeToString(sum[:])
	return filepath.Join(p.Dir, name[:2], name+".html")
}

func (p Page) Get(key string, ttl time.Duration) (string, bool) {
	if p.Dir == "" || ttl <= 0 {
		return "", false
	}
	f := p.file(key)
	st, err := os.Stat(f)
	if err != nil || time.Since(st.ModTime()) > ttl {
		return "", false
	}
	b, err := os.ReadFile(f)
	if err != nil {
		return "", false
	}
	return string(b), true
}

func (p Page) Put(key, html string) {
	if p.Dir == "" {
		return
	}
	f := p.file(key)
	_ = fsperm.MkdirAll(filepath.Dir(f), fsperm.DirPrivate)
	_ = fsperm.WriteFile(f, []byte(html), fsperm.FileCache)
}

func Clear(dir string) {
	os.RemoveAll(dir)
	_ = fsperm.MkdirAll(dir, fsperm.DirPrivate)
}
