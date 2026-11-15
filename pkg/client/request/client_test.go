package request

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/haunbo/go-pilot/pkg/xlog"
)

func TestGetJSONAndAuth(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"message":"ok"}`))
	}))
	defer srv.Close()

	cfg := DefaultConfig()
	cfg.BaseURL = srv.URL
	cfg.Auth = AuthConfig{Type: "bearer", Token: "token"}
	client := cfg.MustBuild()

	var out struct {
		Message string `json:"message"`
	}
	if _, err := client.Get(context.Background(), "/ping", &out); err != nil {
		t.Fatal(err)
	}
	if out.Message != "ok" {
		t.Fatalf("message = %q", out.Message)
	}
	if gotAuth != "Bearer token" {
		t.Fatalf("auth = %q", gotAuth)
	}
}

func TestRetryGET(t *testing.T) {
	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&attempts, 1) < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	cfg := DefaultConfig()
	cfg.BaseURL = srv.URL
	cfg.RetryCount = 2
	cfg.RetryWaitTime = time.Millisecond
	client := cfg.MustBuild()

	var out map[string]bool
	if _, err := client.Get(context.Background(), "/", &out); err != nil {
		t.Fatal(err)
	}
	if attempts != 3 {
		t.Fatalf("attempts = %d", attempts)
	}
}

func TestNoRetryPOST(t *testing.T) {
	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	cfg := DefaultConfig()
	cfg.BaseURL = srv.URL
	cfg.RetryCount = 3
	cfg.RetryWaitTime = time.Millisecond
	client := cfg.MustBuild()

	_, err := client.Post(context.Background(), "/", map[string]string{"a": "b"}, nil)
	if err == nil || !IsStatus(err, http.StatusInternalServerError) {
		t.Fatalf("error = %v", err)
	}
	if attempts != 1 {
		t.Fatalf("attempts = %d", attempts)
	}
}

func TestTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
	}))
	defer srv.Close()

	cfg := DefaultConfig()
	cfg.BaseURL = srv.URL
	cfg.Timeout = 5 * time.Millisecond
	client := cfg.MustBuild()
	_, err := client.Get(context.Background(), "/", nil)
	if err == nil {
		t.Fatal("expected timeout error")
	}
	var reqErr *Error
	if !errors.As(err, &reqErr) || reqErr.Cause == nil {
		t.Fatalf("error = %v", err)
	}
}

func TestRequestID(t *testing.T) {
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("X-Request-ID")
	}))
	defer srv.Close()

	cfg := DefaultConfig()
	cfg.BaseURL = srv.URL
	client := cfg.MustBuild()
	ctx := xlog.WithRequestID(context.Background(), "req-1")
	if _, err := client.Get(ctx, "/", nil); err != nil {
		t.Fatal(err)
	}
	if got != "req-1" {
		t.Fatalf("request id = %q", got)
	}
}

func TestResponseDecodeBytes(t *testing.T) {
	r := &Response{Body: []byte("hello")}
	var data []byte
	if err := r.Decode(&data); err != nil || string(data) != "hello" {
		t.Fatalf("decode = %q, %v", data, err)
	}
}
