package xlog

import (
	"bytes"
	"slices"
	"testing"

	"github.com/haunbo/go-pilot/pkg/conf"
)

// buildFromConfig 模拟真实用法：先 conf.Set 再 RawConfig，让旧配置同样来自配置源，
// 这样 reload 的 diff 只反映真正的改动。
func buildFromConfig(t *testing.T, key string, values map[string]any, buf *bytes.Buffer) *Logger {
	t.Helper()
	if err := conf.Set(key, values); err != nil {
		t.Fatal(err)
	}
	cfg := RawConfig(key)
	cfg.Output = buf
	return New(cfg)
}

func TestReloadAppliesLevel(t *testing.T) {
	const key = "test.logger.reload"
	buf := &bytes.Buffer{}
	logger := buildFromConfig(t, key, map[string]any{"level": "info", "format": "json"}, buf)
	SetDefault(logger)
	defer SetDefault(New(DefaultConfig()))

	if err := conf.Set(key, map[string]any{"level": "debug", "format": "json"}); err != nil {
		t.Fatal(err)
	}
	report, err := Reload(key)
	if err != nil {
		t.Fatalf("Reload: %v", err)
	}
	if !slices.Contains(report.Applied, "level") {
		t.Fatalf("applied = %v, want level", report.Applied)
	}
	if len(report.Restart) != 0 {
		t.Fatalf("restart = %v, want empty (%s)", report.Restart, report.String())
	}
	if got := logger.Level(); got != "debug" {
		t.Fatalf("level = %q, want debug", got)
	}

	// 子 logger 与默认 logger 共享同一个 atomic level，改级别后应立刻生效。
	child := logger.With(String("mod", "test"))
	child.Debug("hello-debug")
	if !bytes.Contains(buf.Bytes(), []byte("hello-debug")) {
		t.Fatalf("debug log not written after reload: %s", buf.String())
	}
}

func TestReloadReportsRestartFields(t *testing.T) {
	const key = "test.logger.restart"
	buf := &bytes.Buffer{}
	logger := buildFromConfig(t, key, map[string]any{"level": "info", "format": "json"}, buf)
	SetDefault(logger)
	defer SetDefault(New(DefaultConfig()))

	// format 与轮转参数烧在 zap core 里，改完只能重启生效。
	if err := conf.Set(key, map[string]any{
		"level":   "info",
		"format":  FormatText,
		"maxSize": 100,
	}); err != nil {
		t.Fatal(err)
	}
	report, err := Reload(key)
	if err != nil {
		t.Fatalf("Reload: %v", err)
	}
	if len(report.Restart) == 0 {
		t.Fatalf("restart = empty, want format/rotation")
	}
	for _, field := range []string{"format", "rotation"} {
		if !slices.Contains(report.Restart, field) {
			t.Fatalf("restart = %v, want %s", report.Restart, field)
		}
	}
}

func TestSetLevelRejectsUnknownLevel(t *testing.T) {
	logger := New(&Config{Level: "info", Output: &bytes.Buffer{}})
	if err := logger.SetLevel("verbose"); err == nil {
		t.Fatal("SetLevel(verbose) = nil, want error")
	}
	if got := logger.Level(); got != "info" {
		t.Fatalf("level = %q, want info", got)
	}
	if err := logger.SetLevel("warning"); err != nil {
		t.Fatalf("SetLevel(warning): %v", err)
	}
	if got := logger.Level(); got != "warn" {
		t.Fatalf("level = %q, want warn", got)
	}
}
