package auth

import (
	"crypto/md5"
	"encoding/hex"
)

// Encrypt is PHP encrypt_string: md5(md5(plain)).
func Encrypt(plain string) string {
	sum := md5.Sum([]byte(plain))
	inner := hex.EncodeToString(sum[:])
	sum = md5.Sum([]byte(inner))
	return hex.EncodeToString(sum[:])
}
