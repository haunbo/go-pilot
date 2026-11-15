package conf_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/haunbo/go-pilot/pkg/conf"
)

func TestFileWatchReloads(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("server:\n  port: 8080\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	conf.Reset()
	t.Cleanup(conf.Reset)
	if err := conf.LoadFromFile(path, true); err != nil {
		t.Fatal(err)
	}

	changed := make(chan struct{}, 1)
	conf.OnChange(func(*conf.Configuration) {
		select {
		case changed <- struct{}{}:
		default:
		}
	})

	if err := os.WriteFile(path, []byte("server:\n  port: 9090\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	select {
	case <-changed:
	case <-time.After(3 * time.Second):
		t.Fatal("config change was not detected")
	}
	if got := conf.GetInt("server.port"); got != 9090 {
		t.Fatalf("port = %d", got)
	}
}
