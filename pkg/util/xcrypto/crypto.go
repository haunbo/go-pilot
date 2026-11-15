package xcrypto

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
)

const randomAlphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// Bytes returns length cryptographically random bytes.
func Bytes(length int) ([]byte, error) {
	if length <= 0 {
		return []byte{}, nil
	}
	key := make([]byte, length)
	n, err := rand.Read(key)
	if err != nil {
		return nil, fmt.Errorf("read random source: %w", err)
	}
	if n < length {
		return nil, errors.New("entropy exhausted")
	}
	return key, nil
}

// RandomString returns a cryptographically random alphanumeric string of
// exactly n characters.
func RandomString(n int) string {
	if n <= 0 {
		return ""
	}
	out := make([]byte, n)
	max := big.NewInt(int64(len(randomAlphabet)))
	for i := range out {
		index, err := rand.Int(rand.Reader, max)
		if err != nil {
			panic(err)
		}
		out[i] = randomAlphabet[index.Int64()]
	}
	return string(out)
}

// SHA256 returns the lowercase hex SHA-256 digest of value.
func SHA256(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

// HmacSHA256 returns the lowercase hex HMAC-SHA256 of data using key.
func HmacSHA256(key, data string) string {
	mac := hmac.New(sha256.New, []byte(key))
	_, _ = mac.Write([]byte(data))
	return hex.EncodeToString(mac.Sum(nil))
}
