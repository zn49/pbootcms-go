package auth

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"gbootcms/internal/fsperm"
)

// Session is a file-backed cookie session.
type Session struct {
	ID     string            `json:"id"`
	Values map[string]string `json:"values"`
	Expire time.Time         `json:"expire"`
}

type Store struct {
	Dir string
	mu  sync.Mutex
}

func NewStore(dir string) *Store {
	_ = fsperm.MkdirAll(dir, fsperm.DirPrivate)
	return &Store{Dir: dir}
}

func (s *Store) path(id string) string { return filepath.Join(s.Dir, id+".json") }

func (s *Store) New() (*Session, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return nil, err
	}
	sess := &Session{
		ID:     hex.EncodeToString(buf),
		Values: map[string]string{},
		Expire: time.Now().Add(12 * time.Hour),
	}
	return sess, s.Save(sess)
}

func validSessionID(id string) bool {
	if id == "" || len(id) > 64 {
		return false
	}
	for _, c := range id {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') && (c < 'A' || c > 'F') {
			return false
		}
	}
	return true
}

func (s *Store) Load(id string) *Session {
	if !validSessionID(id) {
		return nil
	}
	b, err := os.ReadFile(s.path(id))
	if err != nil {
		return nil
	}
	var sess Session
	if json.Unmarshal(b, &sess) != nil || time.Now().After(sess.Expire) {
		_ = os.Remove(s.path(id))
		return nil
	}
	if sess.Values == nil {
		sess.Values = map[string]string{}
	}
	return &sess
}

func (s *Store) Save(sess *Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess.Expire = time.Now().Add(12 * time.Hour)
	b, err := json.Marshal(sess)
	if err != nil {
		return err
	}
	return fsperm.WriteFile(s.path(sess.ID), b, fsperm.FilePrivate)
}

func (s *Store) Get(sess *Session, k string) string {
	if sess == nil {
		return ""
	}
	return sess.Values[k]
}
