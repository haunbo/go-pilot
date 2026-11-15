package xstring

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"net"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Strings is a helper type for string slices.
type Strings []string

// Split splits raw by sep and returns Strings.
func Split(raw, sep string) Strings {
	return Strings(strings.Split(raw, sep))
}

// KickEmpty removes empty elements.
func KickEmpty(ss []string) Strings {
	ret := make([]string, 0, len(ss))
	for _, str := range ss {
		if str != "" {
			ret = append(ret, str)
		}
	}
	return Strings(ret)
}

// Kick removes elements matched by remove.
func Kick(ss []string, remove func(item string) bool) Strings {
	out := make([]string, 0, len(ss))
	for _, str := range ss {
		if !remove(str) {
			out = append(out, str)
		}
	}
	return out
}

// AnyBlank reports whether ss contains an empty element.
func AnyBlank(ss []string) bool {
	for _, str := range ss {
		if str == "" {
			return true
		}
	}
	return false
}

// Any reports whether any element matches.
func Any(ss []string, match func(item string) bool) bool {
	for _, str := range ss {
		if match(str) {
			return true
		}
	}
	return false
}

// All reports whether all elements match. An empty slice returns false.
func All(ss []string, match func(item string) bool) bool {
	if len(ss) == 0 {
		return false
	}
	for _, str := range ss {
		if !match(str) {
			return false
		}
	}
	return true
}

// HeadT returns the first element and the rest.
func (ss Strings) HeadT() (string, Strings) {
	if len(ss) > 0 {
		return ss[0], ss[1:]
	}
	return "", Strings{}
}

// Head returns the first element or an empty string.
func (ss Strings) Head() string {
	if len(ss) > 0 {
		return ss[0]
	}
	return ""
}

// Head2 returns the first two elements.
func (ss Strings) Head2() (h0, h1 string) {
	if len(ss) > 0 {
		h0 = ss[0]
	}
	if len(ss) > 1 {
		h1 = ss[1]
	}
	return
}

// Head3 returns the first three elements.
func (ss Strings) Head3() (h0, h1, h2 string) {
	if len(ss) > 0 {
		h0 = ss[0]
	}
	if len(ss) > 1 {
		h1 = ss[1]
	}
	if len(ss) > 2 {
		h2 = ss[2]
	}
	return
}

// Head4 returns the first four elements.
func (ss Strings) Head4() (h0, h1, h2, h3 string) {
	if len(ss) > 0 {
		h0 = ss[0]
	}
	if len(ss) > 1 {
		h1 = ss[1]
	}
	if len(ss) > 2 {
		h2 = ss[2]
	}
	if len(ss) > 3 {
		h3 = ss[3]
	}
	return
}

// Pointer returns a pointer to str.
func Pointer(str string) *string {
	return &str
}

// IndexString returns the index of str, or -1.
func IndexString(ss []string, str string) int {
	for i, s := range ss {
		if s == str {
			return i
		}
	}
	return -1
}

// HasString reports whether ss contains str.
func HasString(ss []string, str string) bool {
	return IndexString(ss, str) > -1
}

// Reverse reverses a UTF-8 string.
func Reverse(s string) string {
	size := len(s)
	buf := make([]byte, size)
	for start := 0; start < size; {
		r, n := utf8.DecodeRuneInString(s[start:])
		start += n
		utf8.EncodeRune(buf[size-start:], r)
	}
	return string(buf)
}

// Addr2Hex converts an IPv4 address string to hex.
func Addr2Hex(str string) (string, error) {
	ipStr, portStr, err := net.SplitHostPort(str)
	if err != nil {
		return "", err
	}
	ip := net.ParseIP(ipStr).To4()
	if ip == nil {
		return "", fmt.Errorf("invalid ipv4 address: %s", ipStr)
	}
	port, err := strconv.ParseUint(portStr, 10, 16)
	if err != nil {
		return "", err
	}
	buf := make([]byte, 2)
	binary.BigEndian.PutUint16(buf, uint16(port))
	ip = append(ip, buf...)
	return hex.EncodeToString(ip), nil
}

// Hex2Addr converts a hex address back to host:port.
func Hex2Addr(str string) (string, error) {
	buf, err := hex.DecodeString(str)
	if err != nil {
		return "", err
	}
	if len(buf) < 6 {
		return "", fmt.Errorf("bad hex string length")
	}
	return fmt.Sprintf("%s:%d", net.IP(buf[:4]).String(), binary.BigEndian.Uint16(buf[4:6])), nil
}

// Default returns value when it is non-empty, otherwise fallback.
func Default(value, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}

// Truncate limits s to max runes.
func Truncate(s string, max int) string {
	if max <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max])
}

// JoinNonEmpty joins non-empty values with sep.
func JoinNonEmpty(sep string, values ...string) string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value != "" {
			out = append(out, value)
		}
	}
	return strings.Join(out, sep)
}

// FunctionName returns the fully qualified name of fn, or "" for non-functions.
func FunctionName(fn any) string {
	if fn == nil {
		return ""
	}
	value := reflect.ValueOf(fn)
	if value.Kind() != reflect.Func {
		return ""
	}
	function := runtime.FuncForPC(value.Pointer())
	if function == nil {
		return ""
	}
	return function.Name()
}
