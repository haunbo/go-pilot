package xmap

import (
	"reflect"
	"testing"
)

func TestMergeStringMap(t *testing.T) {
	dst := map[string]any{
		"a": map[string]any{"x": 1, "y": 2},
		"b": "keep",
	}
	src := map[string]any{
		"a": map[string]any{"y": 3, "z": 4},
		"b": "replace",
		"c": true,
	}
	MergeStringMap(dst, src)
	want := map[string]any{
		"a": map[string]any{"x": 1, "y": 3, "z": 4},
		"b": "replace",
		"c": true,
	}
	if !reflect.DeepEqual(dst, want) {
		t.Fatalf("MergeStringMap = %#v", dst)
	}
}

func TestFlattenUnflatten(t *testing.T) {
	src := map[string]any{
		"server": map[string]any{"port": 8080, "tls": map[string]any{"enabled": true}},
	}
	flat := Flatten(src, ".")
	if flat["server.port"] != 8080 || flat["server.tls.enabled"] != true {
		t.Fatalf("Flatten = %#v", flat)
	}
	nested := Unflatten(flat, ".")
	if !reflect.DeepEqual(nested, src) {
		t.Fatalf("Unflatten = %#v", nested)
	}
}

func TestDeepSearchAndGet(t *testing.T) {
	m := map[string]any{"a": map[string]any{"b": 1}}
	child := DeepSearchInMap(m, "a")
	if child["b"] != 1 {
		t.Fatalf("child = %#v", child)
	}
	if value, ok := Get(m, "a", "b"); !ok || value != 1 {
		t.Fatalf("Get = %v, %v", value, ok)
	}
	if _, ok := Get(m, "a", "missing"); ok {
		t.Fatal("missing key should not exist")
	}
}

func TestInsensitiviseMap(t *testing.T) {
	m := map[string]any{"A": 1, "Nested": map[string]any{"B": 2}}
	InsensitiviseMap(m)
	if m["a"] != 1 {
		t.Fatalf("m = %#v", m)
	}
	nested := m["nested"].(map[string]any)
	if nested["b"] != 2 {
		t.Fatalf("nested = %#v", nested)
	}
}
