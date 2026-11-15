package xcrypto

import "testing"

func TestMd5(t *testing.T) {
	if got := Md5("hello"); got != "5d41402abc4b2a76b9719d911017c592" {
		t.Fatalf("Md5() = %s", got)
	}
}

func TestRandomString(t *testing.T) {
	a, b := RandomString(16), RandomString(16)
	if len(a) != 16 || len(b) != 16 {
		t.Fatalf("length = %d, %d", len(a), len(b))
	}
	if a == b {
		t.Fatal("random strings should be unique")
	}
}

func TestSHA256(t *testing.T) {
	if got := SHA256("hello"); got != "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824" {
		t.Fatalf("SHA256() = %s", got)
	}
}

func TestHmacSHA256(t *testing.T) {
	if got := HmacSHA256("key", "data"); got == "" || len(got) != 64 {
		t.Fatalf("HmacSHA256() = %q", got)
	}
}
