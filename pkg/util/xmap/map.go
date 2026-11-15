// Package xmap contains helpers for nested map[string]any values, commonly
// used with JSON/YAML configuration and API payloads.
package xmap

import (
	"fmt"
	"strings"
)

// MergeStringMap recursively merges src into dst. A value of a different type
// replaces the destination value.
func MergeStringMap(dst, src map[string]any) {
	for key, srcValue := range src {
		dstValue, ok := dst[key]
		if !ok {
			dst[key] = srcValue
			continue
		}
		dstMap, dstOK := dstValue.(map[string]any)
		srcMap, srcOK := srcValue.(map[string]any)
		if dstOK && srcOK {
			MergeStringMap(dstMap, srcMap)
			continue
		}
		dst[key] = srcValue
	}
}

// ToMapStringInterface converts map[any]any, as produced by
// some YAML decoders, into map[string]any.
func ToMapStringInterface(src map[any]any) map[string]any {
	dst := make(map[string]any, len(src))
	for key, value := range src {
		dst[fmt.Sprintf("%v", key)] = value
	}
	return dst
}

// DeepSearchInMap returns the nested map at the given path. It returns an empty
// map when the path does not exist and never mutates the source map.
func DeepSearchInMap(m map[string]any, paths ...string) map[string]any {
	current := m
	for _, path := range paths {
		next, ok := current[path].(map[string]any)
		if !ok {
			return map[string]any{}
		}
		current = next
	}
	return current
}

// Get returns the value at a path.
func Get(m map[string]any, paths ...string) (any, bool) {
	var current any = m
	for _, path := range paths {
		next, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		current, ok = next[path]
		if !ok {
			return nil, false
		}
	}
	return current, true
}

// Flatten converts nested maps into a single map with joined keys.
func Flatten(m map[string]any, sep string) map[string]any {
	out := make(map[string]any)
	flatten("", m, sep, out)
	return out
}

func flatten(prefix string, m map[string]any, sep string, out map[string]any) {
	for key, value := range m {
		full := key
		if prefix != "" {
			full = prefix + sep + key
		}
		if child, ok := asStringMap(value); ok {
			flatten(full, child, sep, out)
			continue
		}
		out[full] = value
	}
}

// Unflatten converts a flattened map back into a nested map.
func Unflatten(m map[string]any, sep string) map[string]any {
	out := make(map[string]any)
	for key, value := range m {
		parts := strings.Split(key, sep)
		current := out
		for _, part := range parts[:len(parts)-1] {
			next, ok := current[part].(map[string]any)
			if !ok {
				next = make(map[string]any)
				current[part] = next
			}
			current = next
		}
		current[parts[len(parts)-1]] = value
	}
	return out
}

// InsensitiviseMap recursively lower-cases map keys.
func InsensitiviseMap(m map[string]any) {
	for key, value := range m {
		if child, ok := asStringMap(value); ok {
			InsensitiviseMap(child)
			m[key] = child
		}
		lower := strings.ToLower(key)
		if lower == key {
			continue
		}
		delete(m, key)
		m[lower] = value
	}
}

func asStringMap(value any) (map[string]any, bool) {
	switch m := value.(type) {
	case map[string]any:
		return m, true
	case map[any]any:
		return ToMapStringInterface(m), true
	default:
		return nil, false
	}
}

// IsMap reports whether value is a string-keyed or interface-keyed map.
func IsMap(value any) bool {
	_, ok := asStringMap(value)
	return ok
}
