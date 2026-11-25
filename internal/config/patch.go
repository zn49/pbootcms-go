package config

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"os"

	"gbootcms/internal/fsperm"
)

// Patch merges fields into config JSON on disk (preserves unrelated keys / relative paths).
func Patch(path string, fields map[string]any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	b = bytes.TrimPrefix(b, []byte{0xEF, 0xBB, 0xBF})
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	for k, v := range fields {
		raw[k] = v
	}
	nb, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return err
	}
	return fsperm.WriteFile(path, append(nb, '\n'), fsperm.FilePrivate)
}

// RandomSalt returns 16 hex chars.
func RandomSalt() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

// RandomDBFile returns "<hex>.db".
func RandomDBFile() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:]) + ".db"
}
