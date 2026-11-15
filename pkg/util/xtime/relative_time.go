package xtime

import "time"

var initTime = time.Now().AddDate(-1, -1, -1)

// Now returns a relative duration since process start, useful for elapsed calculation.
func Now() time.Duration {
	return time.Since(initTime)
}

// Since returns the duration elapsed since d.
func Since(d time.Duration) time.Duration {
	return time.Since(initTime) - d
}

// BeginOfDay returns 00:00:00 of t in its location.
func BeginOfDay(t time.Time) time.Time {
	year, month, day := t.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, t.Location())
}

// EndOfDay returns 23:59:59.999999999 of t in its location.
func EndOfDay(t time.Time) time.Time {
	return BeginOfDay(t).AddDate(0, 0, 1).Add(-time.Nanosecond)
}
