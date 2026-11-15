package xstring

import "testing"

func TestStringHelpers(t *testing.T) {
	if got := Default("", "fallback"); got != "fallback" {
		t.Fatalf("Default = %q", got)
	}
	if got := Truncate("你好世界", 2); got != "你好" {
		t.Fatalf("Truncate = %q", got)
	}
	if got := JoinNonEmpty(",", "a", "", "b"); got != "a,b" {
		t.Fatalf("JoinNonEmpty = %q", got)
	}
	if got := FunctionName(TestStringHelpers); got == "" {
		t.Fatal("FunctionName returned empty")
	}
}
