package xlog

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
)

func TestLoggerJSONAndRequestID(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&Config{Format: FormatJSON, Level: "debug", Output: &buf, Dir: "-"})
	ctx := WithRequestID(context.Background(), "req-123")
	logger.WithContext(ctx).Info("hello", String("key", "value"), FieldCost(1500*1000*1000))

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("decode log: %v (%s)", err, buf.String())
	}
	if entry["detail"] != "hello" || entry["level"] != "info" {
		t.Fatalf("unexpected entry: %#v", entry)
	}
	if entry["request_id"] != "req-123" {
		t.Fatalf("request_id = %v", entry["request_id"])
	}
	if entry["key"] != "value" {
		t.Fatalf("key = %v", entry["key"])
	}
	if entry["cost"] != "1500.000" {
		t.Fatalf("cost = %v", entry["cost"])
	}
}

func TestLevelFiltering(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&Config{Format: FormatJSON, Level: "error", Output: &buf})
	logger.Info("dropped")
	if buf.Len() != 0 {
		t.Fatalf("info log should be filtered, got %s", buf.String())
	}
	logger.Error("kept")
	if buf.Len() == 0 {
		t.Fatal("error log should be written")
	}
}

func TestFieldExtra(t *testing.T) {
	field := FieldExtra(map[string]any{"b": 2, "a": 1})
	var got map[string]any
	if err := json.Unmarshal([]byte(field.String), &got); err != nil {
		t.Fatal(err)
	}
	if got["a"] != float64(1) || got["b"] != float64(2) {
		t.Fatalf("extra = %#v", got)
	}
}
