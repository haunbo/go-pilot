package conf_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/haunbo/go-pilot/pkg/conf"
	"gopkg.in/yaml.v3"
)

type testConfig struct {
	Name   string `json:"name"`
	Server struct {
		Host    string        `json:"host"`
		Port    int           `json:"port"`
		Timeout time.Duration `json:"timeout"`
	} `json:"server"`
	Tags []string `json:"tags"`
}

func load(t *testing.T, content string) {
	t.Helper()
	conf.Reset()
	t.Cleanup(conf.Reset)
	if err := conf.LoadFromReader(strings.NewReader(content), yaml.Unmarshal); err != nil {
		t.Fatalf("load: %v", err)
	}
}

func TestLoadGetAndUnmarshal(t *testing.T) {
	load(t, `
name: demo
server:
  host: 127.0.0.1
  port: 8090
  timeout: 5s
tags: [a, b]
`)
	if got := conf.GetString("name"); got != "demo" {
		t.Fatalf("name = %q", got)
	}
	if got := conf.GetInt("server.port"); got != 8090 {
		t.Fatalf("port = %d", got)
	}
	if got := conf.GetDuration("server.timeout"); got != 5*time.Second {
		t.Fatalf("timeout = %v", got)
	}

	var cfg testConfig
	if err := conf.UnmarshalKey("", &cfg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if cfg.Name != "demo" || cfg.Server.Port != 8090 || cfg.Server.Timeout != 5*time.Second {
		t.Fatalf("unexpected config: %+v", cfg)
	}
	if len(cfg.Tags) != 2 || cfg.Tags[1] != "b" {
		t.Fatalf("tags = %#v", cfg.Tags)
	}

	var server struct {
		Port int `json:"port"`
	}
	if err := conf.UnmarshalKey("server", &server); err != nil {
		t.Fatalf("unmarshal key: %v", err)
	}
	if server.Port != 8090 {
		t.Fatalf("server port = %d", server.Port)
	}
}

func TestSetApplyAndExists(t *testing.T) {
	load(t, "name: demo\n")
	if err := conf.Set("server.port", 9000); err != nil {
		t.Fatal(err)
	}
	if got := conf.GetInt("server.port"); got != 9000 {
		t.Fatalf("port = %d", got)
	}
	if !conf.Exists("server.port") {
		t.Fatal("server.port should exist")
	}
	if conf.Exists("server.missing") {
		t.Fatal("server.missing should not exist")
	}

	if err := conf.Apply(map[string]any{"name": "next"}); err != nil {
		t.Fatal(err)
	}
	if conf.GetString("name") != "next" {
		t.Fatalf("name = %q", conf.GetString("name"))
	}
	if conf.Exists("server.port") {
		t.Fatal("Apply should replace the previous tree")
	}
}

func TestEnvironmentOverrides(t *testing.T) {
	t.Setenv("APP_SERVER_PORT", "9100")
	t.Setenv("APP_SERVER_HOST", "0.0.0.0")
	load(t, "server:\n  port: 8090\n")
	conf.LoadEnvironments("APP_")
	if got := conf.GetInt("server.port"); got != 9100 {
		t.Fatalf("port = %d", got)
	}
	if got := conf.GetString("server.host"); got != "0.0.0.0" {
		t.Fatalf("host = %q", got)
	}
}

func TestHooks(t *testing.T) {
	conf.Reset()
	t.Cleanup(conf.Reset)
	if err := conf.LoadFromReader(strings.NewReader("a: 1\n"), yaml.Unmarshal); err != nil {
		t.Fatal(err)
	}

	loaded := 0
	conf.OnLoaded(func(*conf.Configuration) { loaded++ })
	if loaded != 1 {
		t.Fatalf("OnLoaded after load = %d, want 1", loaded)
	}

	changed := 0
	conf.OnChange(func(*conf.Configuration) { changed++ })
	if err := conf.Set("a", 2); err != nil {
		t.Fatal(err)
	}
	if changed != 1 {
		t.Fatalf("OnChange = %d, want 1", changed)
	}
	if err := conf.Set("a", 2); err != nil {
		t.Fatal(err)
	}
	if changed != 1 {
		t.Fatalf("unchanged Set should not notify, got %d", changed)
	}
}

func TestLoadFromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("server:\n  port: 8081\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	conf.Reset()
	t.Cleanup(conf.Reset)
	if err := conf.LoadFromFile(path, false); err != nil {
		t.Fatalf("load file: %v", err)
	}
	if got := conf.GetInt("server.port"); got != 8081 {
		t.Fatalf("port = %d", got)
	}
	if err := conf.LoadFromFile(filepath.Join(dir, "missing.yaml"), false); err == nil {
		t.Fatal("loading a missing file should fail")
	}
}

func TestInvalidKey(t *testing.T) {
	load(t, "a: 1\n")
	var out struct {
		B int `json:"b"`
	}
	if err := conf.UnmarshalKey("missing", &out); err == nil {
		t.Fatal("expected invalid key error")
	}
}
