package xstring

import (
	"strings"
	"testing"
)

func TestGenerateUUID(t *testing.T) {
	id := GenerateUUID()
	if len(id) != 36 {
		t.Fatalf("uuid length = %d, want 36", len(id))
	}
	if id[14] != '4' {
		t.Fatalf("uuid version = %c, want 4", id[14])
	}
	if !strings.Contains("89ab", string(id[19])) {
		t.Fatalf("uuid variant = %c, want 8/9/a/b", id[19])
	}
	if GenerateUUID() == id {
		t.Fatal("uuid should be unique")
	}
}

func TestReverse(t *testing.T) {
	if got := Reverse("你好abc"); got != "cba好你" {
		t.Fatalf("Reverse() = %q", got)
	}
}

func TestToCamelCase(t *testing.T) {
	if got := ToCamelCase("hello world"); got != "helloWorld" {
		t.Fatalf("ToCamelCase() = %q", got)
	}
}
