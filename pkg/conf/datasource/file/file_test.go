package file_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/haunbo/go-pilot/pkg/conf"
	filesource "github.com/haunbo/go-pilot/pkg/conf/datasource/file"
	"gopkg.in/yaml.v3"
)

func TestNewAndLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("server:\n  port: 8080\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	source, err := filesource.New(path, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := conf.LoadFromDataSource(source, yaml.Unmarshal); err != nil {
		t.Fatal(err)
	}
	if got := conf.GetInt("server.port"); got != 8080 {
		t.Fatalf("port = %d", got)
	}
}

func TestNewMissingFile(t *testing.T) {
	if _, err := filesource.New(filepath.Join(t.TempDir(), "missing.yaml"), false); err == nil {
		t.Fatal("missing file should fail")
	}
}
