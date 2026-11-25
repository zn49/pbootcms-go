package admin

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"strings"

	"gbootcms/internal/app"
	"gbootcms/internal/fsperm"
)

const aiSecretNS = "|pbootcms_ai_secret_v1"

func appSecretKey(root string) string {
	saltFile := filepath.Join(root, "data", ".app_salt")
	salt, err := os.ReadFile(saltFile)
	if err != nil || len(strings.TrimSpace(string(salt))) == 0 {
		b := make([]byte, 16)
		_, _ = rand.Read(b)
		s := hex.EncodeToString(b)
		_ = fsperm.MkdirAll(filepath.Dir(saltFile), fsperm.DirPrivate)
		_ = fsperm.WriteFile(saltFile, []byte(s), fsperm.FilePrivate)
		salt = []byte(s)
	} else {
		fsperm.HardenFile(saltFile, fsperm.FilePrivate)
	}
	sum := sha256.Sum256([]byte(strings.TrimSpace(string(salt)) + aiSecretNS))
	return hex.EncodeToString(sum[:])[:32]
}

func aesEncrypt(root, plain string) string {
	if plain == "" {
		return ""
	}
	key := []byte(appSecretKey(root))
	block, err := aes.NewCipher(key)
	if err != nil {
		return ""
	}
	iv := make([]byte, aes.BlockSize)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return ""
	}
	src := pkcs7Pad([]byte(plain), aes.BlockSize)
	dst := make([]byte, len(src))
	cipher.NewCBCEncrypter(block, iv).CryptBlocks(dst, src)
	return base64.StdEncoding.EncodeToString(append(iv, dst...))
}

func aesDecrypt(root, encoded string) string {
	if encoded == "" {
		return ""
	}
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil || len(raw) < aes.BlockSize+1 {
		return ""
	}
	key := []byte(appSecretKey(root))
	block, err := aes.NewCipher(key)
	if err != nil {
		return ""
	}
	iv, cipherText := raw[:aes.BlockSize], raw[aes.BlockSize:]
	if len(cipherText)%aes.BlockSize != 0 {
		return ""
	}
	dst := make([]byte, len(cipherText))
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(dst, cipherText)
	out, err := pkcs7Unpad(dst)
	if err != nil {
		return ""
	}
	return string(out)
}

func pkcs7Pad(b []byte, block int) []byte {
	n := block - len(b)%block
	out := make([]byte, len(b)+n)
	copy(out, b)
	for i := len(b); i < len(out); i++ {
		out[i] = byte(n)
	}
	return out
}

func pkcs7Unpad(b []byte) ([]byte, error) {
	if len(b) == 0 {
		return nil, io.ErrUnexpectedEOF
	}
	n := int(b[len(b)-1])
	if n < 1 || n > len(b) {
		return nil, io.ErrUnexpectedEOF
	}
	for i := len(b) - n; i < len(b); i++ {
		if b[i] != byte(n) {
			return nil, io.ErrUnexpectedEOF
		}
	}
	return b[:len(b)-n], nil
}

func maskSecret(key string) string {
	if key == "" {
		return ""
	}
	if len(key) <= 8 {
		return strings.Repeat("*", len(key))
	}
	return key[:4] + "****" + key[len(key)-4:]
}

func isMaskedSecret(val string) bool {
	return strings.Contains(val, "****")
}

func aiKeyPlain(a *app.App) string {
	v := strings.TrimSpace(a.DB.Conf("ai_api_key"))
	if v == "" {
		return ""
	}
	if p := aesDecrypt(a.Cfg.SiteRoot, v); p != "" {
		return p
	}
	// plaintext leftover from installs that never encrypted
	if _, err := base64.StdEncoding.DecodeString(v); err != nil || len(v) < 24 {
		return v
	}
	return ""
}
