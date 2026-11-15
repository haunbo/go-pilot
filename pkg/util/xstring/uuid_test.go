package xstring

import (
	"strings"
	"testing"
)

func TestGenerateUUID(t *testing.T) {
	id := GenerateUUID()
	if len(id) != 36 {
		t.Fatalf("uuid length = %d, want 36: %q", len(id), id)
	}
	if id[14] != '4' {
		t.Fatalf("uuid version = %c, want 4", id[14])
	}
	if !strings.Contains("89ab", string(id[19])) {
		t.Fatalf("uuid variant = %c, want 8/9/a/b", id[19])
	}
	if GenerateUUID() == id {
		t.Fatal("two calls returned the same uuid")
	}
}
