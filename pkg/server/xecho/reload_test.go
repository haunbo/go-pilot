package xecho

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/haunbo/go-pilot/pkg/conf"
	"github.com/haunbo/go-pilot/pkg/xlog"
	"github.com/labstack/echo/v4"
)

// newReloadTestServer 按真实用法构建：配置来自 conf，端口用 0 避免占用固定端口。
func newReloadTestServer(t *testing.T, buf *bytes.Buffer, key string, values map[string]any) *Server {
	t.Helper()
	if err := conf.Set(key, values); err != nil {
		t.Fatal(err)
	}
	logger := xlog.New(&xlog.Config{Level: "debug", Format: xlog.FormatJSON, Output: buf})
	server, err := RawConfig(key).WithLogger(logger).Build()
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	t.Cleanup(func() { _ = server.Stop() })
	server.GET("/ping", func(c echo.Context) error {
		return c.String(http.StatusOK, "pong")
	})
	return server
}

var reloadBaseConfig = map[string]any{
	"host":                      "127.0.0.1",
	"port":                      0,
	"enableAccessLog":           true,
	"slowQueryThresholdInMilli": 500,
	"validationLocale":          "zh",
}

func reloadValuesWith(overrides map[string]any) map[string]any {
	out := make(map[string]any, len(reloadBaseConfig)+len(overrides))
	for k, v := range reloadBaseConfig {
		out[k] = v
	}
	for k, v := range overrides {
		out[k] = v
	}
	return out
}

func TestReloadTogglesAccessLog(t *testing.T) {
	const key = "server.reloadaccesslog"
	buf := &bytes.Buffer{}
	server := newReloadTestServer(t, buf, key, reloadValuesWith(nil))

	server.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/ping", nil))
	if !bytes.Contains(buf.Bytes(), []byte(`"access"`)) {
		t.Fatalf("access log missing before reload: %s", buf.String())
	}

	buf.Reset()
	if err := conf.Set(key, reloadValuesWith(map[string]any{"enableAccessLog": false})); err != nil {
		t.Fatal(err)
	}
	report, err := server.Reload()
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if !slices.Contains(report.Applied, "enableAccessLog") {
		t.Fatalf("applied = %v, want enableAccessLog", report.Applied)
	}
	if len(report.Restart) != 0 {
		t.Fatalf("restart = %v, want empty", report.Restart)
	}

	server.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/ping", nil))
	if bytes.Contains(buf.Bytes(), []byte(`"access"`)) {
		t.Fatalf("access log still written after disable: %s", buf.String())
	}

	// 错误必须照常记录，不受访问日志开关影响。
	buf.Reset()
	server.GET("/boom", func(c echo.Context) error { panic("boom") })
	server.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/boom", nil))
	if !bytes.Contains(buf.Bytes(), []byte("boom")) {
		t.Fatalf("panic not logged after disable: %s", buf.String())
	}
}

func TestReloadReportsPortChangeNeedsRestart(t *testing.T) {
	const key = "server.reloadport"
	server := newReloadTestServer(t, &bytes.Buffer{}, key, reloadValuesWith(nil))

	before := server.Addr()
	if err := conf.Set(key, reloadValuesWith(map[string]any{"port": 18081})); err != nil {
		t.Fatal(err)
	}
	report, err := server.Reload()
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if !slices.Contains(report.Restart, "port") {
		t.Fatalf("restart = %v, want port", report.Restart)
	}
	if got := server.Addr(); got != before {
		t.Fatalf("listener moved from %s to %s without restart", before, got)
	}
}

func TestReloadWithoutConfigKeyFails(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Port = 0
	server, err := cfg.Build()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = server.Stop() }()

	if _, err := server.Reload(); err == nil {
		t.Fatal("Reload() = nil, want error for a config built by hand")
	}
}
