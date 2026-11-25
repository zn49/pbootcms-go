// Package fsperm centralizes directory/file modes for container-safe deploys.
package fsperm

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const (
	DirPrivate  os.FileMode = 0o750
	DirUpload   os.FileMode = 0o750
	FilePrivate os.FileMode = 0o600
	FileUpload  os.FileMode = 0o640
	FileCache   os.FileMode = 0o640
)

func MkdirAll(path string, mode os.FileMode) error {
	if err := os.MkdirAll(path, mode); err != nil {
		return err
	}
	_ = os.Chmod(path, mode)
	return nil
}

func WriteFile(path string, b []byte, mode os.FileMode) error {
	if err := os.WriteFile(path, b, mode); err != nil {
		return err
	}
	_ = os.Chmod(path, mode) // umask may have widened bits
	return nil
}

func OpenFile(path string, flag int, mode os.FileMode) (*os.File, error) {
	f, err := os.OpenFile(path, flag, mode)
	if err != nil {
		return nil, err
	}
	_ = f.Chmod(mode)
	return f, nil
}

func Create(path string, mode os.FileMode) (*os.File, error) {
	return OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, mode)
}

// HardenSite tightens data/session/cache/backup/upload trees under site root.
func HardenSite(root string) {
	if strings.TrimSpace(root) == "" {
		return
	}
	for _, rel := range []string{
		"data",
		"data/session",
		"data/session/admin",
		"data/session/front",
		"data/cache",
		"data/log",
		"data/backup",
		"static/backup",
		"static/upload",
	} {
		mode := DirPrivate
		if strings.HasPrefix(rel, "static/upload") {
			mode = DirUpload
		}
		_ = MkdirAll(filepath.Join(root, filepath.FromSlash(rel)), mode)
	}
	_ = filepath.WalkDir(filepath.Join(root, "data"), func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			_ = os.Chmod(path, DirPrivate)
			return nil
		}
		name := strings.ToLower(d.Name())
		switch {
		case name == ".app_salt", strings.HasSuffix(name, ".db"), strings.HasSuffix(name, ".sqlite"),
			strings.HasSuffix(name, ".json"), strings.HasSuffix(name, ".bak"):
			_ = os.Chmod(path, FilePrivate)
		case strings.HasSuffix(name, ".log"), strings.HasSuffix(name, ".html"):
			_ = os.Chmod(path, FileCache)
		}
		return nil
	})
	_ = filepath.WalkDir(filepath.Join(root, "static", "backup"), func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			_ = os.Chmod(path, DirPrivate)
			return nil
		}
		_ = os.Chmod(path, FilePrivate)
		return nil
	})
}

// HardenFile sets mode on an existing path (best-effort; no-op if missing).
func HardenFile(path string, mode os.FileMode) {
	_ = os.Chmod(path, mode)
}
