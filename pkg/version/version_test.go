package version

import "testing"

func TestVersion(t *testing.T) {
	if Version != "1.0.0" {
		t.Fatalf("Version = %q", Version)
	}
	if String() == "" {
		t.Fatal("String() returned empty")
	}
}
