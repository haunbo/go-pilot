package redis

import "testing"

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Addr == "" || cfg.PoolSize <= 0 {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
}

func TestBuildRejectsEmptyAddr(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Addr = ""
	if _, err := cfg.Build(); err == nil {
		t.Fatal("empty addr should fail")
	}
}
