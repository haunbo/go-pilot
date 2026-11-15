// Package xmath contains small generic numeric helpers.
package xmath

// Integer is an integer constraint.
type Integer interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr
}

// Number is a numeric constraint.
type Number interface {
	Integer | ~float32 | ~float64
}

// Ordered is an ordered constraint.
type Ordered interface {
	Integer | ~float32 | ~float64 | ~string
}

// Abs returns the absolute value of n.
func Abs[T Number](n T) T {
	if n < 0 {
		return -n
	}
	return n
}

// Clamp limits v to the inclusive [min, max] range.
func Clamp[T Ordered](v, min, max T) T {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

// Percent returns part/total*100. It returns 0 when total is zero.
func Percent[T Number](part, total T) float64 {
	if total == 0 {
		return 0
	}
	return float64(part) / float64(total) * 100
}
