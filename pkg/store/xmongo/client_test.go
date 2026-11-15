package xmongo

import "testing"

func TestConfigValidation(t *testing.T) {
	cfg := DefaultConfig()
	if _, err := cfg.Build(); err == nil {
		t.Fatal("empty dsn should fail")
	}
	cfg.DSN = "://bad-uri"
	if _, err := cfg.Build(); err == nil {
		t.Fatal("invalid dsn should fail")
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.MaxPoolSize == 0 || cfg.ConnectTimeout == 0 {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
}
