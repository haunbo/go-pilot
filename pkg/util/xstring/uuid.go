package xstring

import (
	"fmt"

	"github.com/haunbo/go-pilot/pkg/util/xcrypto"
)

// GenerateUUID returns a random UUID-like string.
func GenerateUUID() string {
	buf, err := xcrypto.Bytes(16)
	if err != nil {
		panic(fmt.Errorf("failed to read random bytes: %w", err))
	}
	buf[6] = (buf[6] & 0x0f) | 0x40 // version 4
	buf[8] = (buf[8] & 0x3f) | 0x80 // variant 10
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%12x",
		buf[0:4], buf[4:6], buf[6:8], buf[8:10], buf[10:16])
}

// GenerateShortUUID returns the first 8 characters of a UUID.
func GenerateShortUUID() string {
	return GenerateUUID()[:8]
}
