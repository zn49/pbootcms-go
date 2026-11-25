package auth

import (
	"sync"
	"time"
)

// LoginAttempt tracks per-key failed logins (in-memory; process-local).
type loginBucket struct {
	fails []int64
}

var loginMu sync.Mutex
var loginFails = map[string]*loginBucket{}

// LoginAllowed reports whether key may attempt login. limit fails inside windowSec.
func LoginAllowed(key string, limit int, windowSec int64) bool {
	if key == "" || limit <= 0 {
		return true
	}
	now := time.Now().Unix()
	loginMu.Lock()
	defer loginMu.Unlock()
	b := loginFails[key]
	if b == nil {
		return true
	}
	var keep []int64
	for _, t := range b.fails {
		if now-t < windowSec {
			keep = append(keep, t)
		}
	}
	b.fails = keep
	return len(keep) < limit
}

// LoginFail records a failed attempt.
func LoginFail(key string) {
	if key == "" {
		return
	}
	now := time.Now().Unix()
	loginMu.Lock()
	defer loginMu.Unlock()
	b := loginFails[key]
	if b == nil {
		b = &loginBucket{}
		loginFails[key] = b
	}
	b.fails = append(b.fails, now)
}

// LoginOK clears failures after success.
func LoginOK(key string) {
	if key == "" {
		return
	}
	loginMu.Lock()
	defer loginMu.Unlock()
	delete(loginFails, key)
}
