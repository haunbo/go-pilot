package xecho

import (
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/haunbo/go-pilot/pkg/xlog"
	"github.com/labstack/echo/v4"
)

func TestServerServeAndStop(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Host = "127.0.0.1"
	cfg.Port = 0
	cfg.EnableAccessLog = false
	cfg.WithLogger(xlog.New(&xlog.Config{Level: "error", Format: xlog.FormatJSON, Output: io.Discard}))

	server, err := cfg.Build()
	if err != nil {
		t.Fatal(err)
	}
	server.GET("/ping", func(c echo.Context) error {
		return c.String(http.StatusOK, "pong")
	})

	done := make(chan error, 1)
	go func() { done <- server.Serve() }()

	var response *http.Response
	client := &http.Client{Timeout: time.Second}
	for i := 0; i < 50; i++ {
		response, err = client.Get("http://" + server.Addr() + "/ping")
		if err == nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err != nil {
		_ = server.Stop()
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(response.Body)
	_ = response.Body.Close()
	if string(body) != "pong" {
		t.Fatalf("body = %q", body)
	}
	if err := server.Stop(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("serve error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("server did not stop")
	}
}
