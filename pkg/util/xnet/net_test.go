package xnet

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestLocalListener(t *testing.T) {
	l, err := LocalListener()
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	if err := Dial(l.Addr().String(), time.Second); err != nil {
		t.Fatalf("dial listener: %v", err)
	}
}

func TestHTTPPing(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()
	if err := HTTPPing(context.Background(), srv.URL, time.Second); err != nil {
		t.Fatal(err)
	}
}

func TestGetIPs(t *testing.T) {
	if _, err := GetIPs(); err != nil {
		t.Fatal(err)
	}
	if _, err := GetMacAddrs(); err != nil {
		t.Fatal(err)
	}
}
