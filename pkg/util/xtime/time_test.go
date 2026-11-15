package xtime

import (
	"testing"
	"time"
)

func TestBeginEndOfDay(t *testing.T) {
	loc := time.FixedZone("CST", 8*3600)
	now := time.Date(2026, 11, 15, 16, 30, 0, 0, loc)
	begin := BeginOfDay(now)
	end := EndOfDay(now)
	if begin.Hour() != 0 || begin.Minute() != 0 || begin.Second() != 0 {
		t.Fatalf("begin = %v", begin)
	}
	if end.Hour() != 23 || end.Minute() != 59 || end.Second() != 59 {
		t.Fatalf("end = %v", end)
	}
}
