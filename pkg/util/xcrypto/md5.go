package xcrypto

import (
	"crypto/md5"
	"encoding/hex"
)

// Md5 returns the lowercase hex md5 digest of str.
func Md5(str string) string {
	sum := md5.Sum([]byte(str))
	return hex.EncodeToString(sum[:])
}
