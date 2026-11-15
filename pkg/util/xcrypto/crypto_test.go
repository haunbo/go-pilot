package xcrypto

import "testing"

func TestBytes(t *testing.T) {
	a, err := Bytes(16)
	if err != nil {
		t.Fatalf("Bytes(16): %v", err)
	}
	if len(a) != 16 {
		t.Fatalf("len = %d, want 16", len(a))
	}

	b, err := Bytes(16)
	if err != nil {
		t.Fatalf("Bytes(16): %v", err)
	}
	if string(a) == string(b) {
		t.Fatal("two calls returned identical bytes")
	}
}

func TestBytesNonPositive(t *testing.T) {
	for _, n := range []int{0, -1} {
		b, err := Bytes(n)
		if err != nil {
			t.Fatalf("Bytes(%d): %v", n, err)
		}
		if len(b) != 0 {
			t.Fatalf("Bytes(%d) = %d bytes, want 0", n, len(b))
		}
	}
}
