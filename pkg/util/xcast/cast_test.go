package xcast

import "testing"

func TestToInt64Slice(t *testing.T) {
	got := ToInt64Slice([]any{1, "2", 3.0})
	want := []int64{1, 2, 3}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got[%d] = %d, want %d", i, got[i], want[i])
		}
	}
}
