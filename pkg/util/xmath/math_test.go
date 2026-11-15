package xmath

import "testing"

func TestAbs(t *testing.T) {
	if got := Abs(-3); got != 3 {
		t.Fatalf("Abs(-3) = %d", got)
	}
	if got := Abs(2.5); got != 2.5 {
		t.Fatalf("Abs(2.5) = %v", got)
	}
}

func TestClamp(t *testing.T) {
	if got := Clamp(10, 0, 5); got != 5 {
		t.Fatalf("Clamp = %d", got)
	}
	if got := Clamp(-1, 0, 5); got != 0 {
		t.Fatalf("Clamp = %d", got)
	}
}

func TestPercent(t *testing.T) {
	if got := Percent(1, 4); got != 25 {
		t.Fatalf("Percent = %v", got)
	}
	if got := Percent(1, 0); got != 0 {
		t.Fatalf("Percent zero total = %v", got)
	}
}
