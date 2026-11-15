package nacos

import "testing"

func TestConfigValidation(t *testing.T) {
	if _, err := New(Config{}, false); err == nil {
		t.Fatal("empty server should fail")
	}
	if _, err := New(Config{ServerAddr: "127.0.0.1"}, false); err == nil {
		t.Fatal("empty dataID should fail")
	}
}
