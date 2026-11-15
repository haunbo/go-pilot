// Package xcrypto provides the random byte source used to build identifiers.
package xcrypto

import (
	"crypto/rand"
	"errors"
	"fmt"
)

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
