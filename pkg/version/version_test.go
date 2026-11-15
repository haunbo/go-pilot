package version

import (
	"runtime/debug"
	"strings"
	"testing"
)

func TestVersion(t *testing.T) {
	if Version != "1.0.0" {
		t.Fatalf("Version = %q", Version)
	}
	if String() == "" {
		t.Fatal("String() returned empty")
	}
}

// When go-pilot is the main module (as in this repository) the build info has no
// release tag, so String() must fall back to the Version constant instead of
// returning "(devel)".
func TestStringFallsBackToConstant(t *testing.T) {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		t.Skip("build info not available")
	}
	if info.Main.Path != modulePath {
		t.Skipf("not running inside the go-pilot module: main=%s", info.Main.Path)
	}
	if got := String(); got != Version {
		t.Fatalf("String() = %q, want %q", got, Version)
	}
}

func TestStringNeverReturnsDevel(t *testing.T) {
	if got := String(); strings.Contains(got, "devel") {
		t.Fatalf("String() = %q, must not leak the (devel) placeholder", got)
	}
}
