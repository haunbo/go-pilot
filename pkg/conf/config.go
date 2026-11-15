// Package conf provides layered application configuration with dotted keys.
//
// Supported sources:
//   - YAML/JSON files (optionally hot reloaded with fsnotify)
//   - environment variables (APP_SERVER_HTTP_PORT -> server.http.port)
//   - programmatic Set/Apply calls
//
// Configuration is flattened into dotted keys and can be read via GetString,
// GetInt, ... or decoded into structs with UnmarshalKey.
package conf

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/go-viper/mapstructure/v2"
	"github.com/haunbo/go-pilot/pkg/util/xcast"
	"github.com/spf13/cast"
	"gopkg.in/yaml.v3"
)

const defaultKeyDelim = "."

// ErrInvalidKey is returned when a key does not exist.
var ErrInvalidKey = errors.New("invalid key, maybe not exist in config")

// Configuration is a thread-safe layered configuration.
type Configuration struct {
	mu           sync.RWMutex
	override     map[string]any
	envOverrides map[string]any
	keyDelim     string
	keyMap       sync.Map

	hookMu    sync.RWMutex
	loaded    bool
	onChanges []func(*Configuration)
	onLoadeds []func(*Configuration)

	done      chan struct{}
	closeOnce sync.Once
	closers   []io.Closer
}

// New returns an empty Configuration.
func New() *Configuration {
	return &Configuration{
		override:     make(map[string]any),
		envOverrides: make(map[string]any),
		keyDelim:     defaultKeyDelim,
		done:         make(chan struct{}),
	}
}

// SetKeyDelim changes the key separator. It must be called before loading.
func (c *Configuration) SetKeyDelim(delim string) {
	if delim != "" {
		c.keyDelim = delim
	}
}

// Load decodes content with unmarshal and replaces the file layer.
func (c *Configuration) Load(content []byte, unmarshal Unmarshaller) error {
	conf := make(map[string]any)
	if err := unmarshal(content, &conf); err != nil {
		return err
	}
	return c.load(conf)
}

func (c *Configuration) load(conf map[string]any) error {
	c.mu.Lock()
	c.replaceLocked(conf)
	c.mu.Unlock()
	c.markLoaded()
	return nil
}

// Apply replaces the current configuration with conf and notifies change hooks.
func (c *Configuration) Apply(conf map[string]any) error {
	c.mu.Lock()
	changes := c.replaceLocked(conf)
	c.mu.Unlock()
	c.notifyChanges(changes)
	return nil
}

// replaceLocked swaps the file layer, re-applies env overrides and refreshes
// the flattened key cache. Caller must hold c.mu.
func (c *Configuration) replaceLocked(conf map[string]any) map[string]any {
	old := flattenWithSep(c.override, c.keyDelim)
	c.override = conf
	for k, v := range c.envOverrides {
		setPath(c.override, c.keyDelim, k, v)
	}
	next := flattenWithSep(c.override, c.keyDelim)
	changes := diff(old, next)
	refreshKeyMap(&c.keyMap, next)
	return changes
}

// Set sets a single key on the in-memory layer and notifies change hooks.
func (c *Configuration) Set(key string, val any) error {
	if key == "" {
		return errors.New("conf: empty key")
	}
	c.mu.Lock()
	old := flattenWithSep(c.override, c.keyDelim)
	setPath(c.override, c.keyDelim, key, val)
	next := flattenWithSep(c.override, c.keyDelim)
	changes := diff(old, next)
	refreshKeyMap(&c.keyMap, next)
	c.mu.Unlock()
	c.notifyChanges(changes)
	return nil
}

// LoadEnvironments reads os.Environ entries with prefix.
// APP_SERVER_HTTP_PORT is translated into server.http.port (prefix is removed,
// the rest is lower-cased and underscores become dots). Environment values take
// precedence over file values, including after a hot reload.
func (c *Configuration) LoadEnvironments(prefix string) {
	if prefix == "" {
		return
	}
	for _, env := range os.Environ() {
		kv := strings.SplitN(env, "=", 2)
		if len(kv) != 2 || !strings.HasPrefix(kv[0], prefix) {
			continue
		}
		key := strings.ToLower(strings.ReplaceAll(strings.TrimPrefix(kv[0], prefix), "_", "."))
		if key == "" {
			continue
		}
		c.mu.Lock()
		c.envOverrides[key] = kv[1]
		c.mu.Unlock()
		if err := c.Set(key, kv[1]); err != nil {
			logMessage("conf: load environment %s: %v", kv[0], err)
		}
	}
}

// LoadFromReader reads all bytes from reader and loads them.
func (c *Configuration) LoadFromReader(reader io.Reader, unmarshal Unmarshaller) error {
	content, err := io.ReadAll(reader)
	if err != nil {
		return err
	}
	return c.Load(content, unmarshal)
}

// LoadFromFile loads a YAML or JSON file. When watch is true the file is
// monitored and reloaded on change; registered OnChange hooks are called.
func (c *Configuration) LoadFromFile(path string, watch bool) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	var unmarshal Unmarshaller
	switch strings.ToLower(filepath.Ext(abs)) {
	case ".yaml", ".yml":
		unmarshal = yaml.Unmarshal
	case ".json":
		unmarshal = json.Unmarshal
	default:
		return fmt.Errorf("conf: unsupported config type %q", filepath.Ext(abs))
	}
	ds, err := NewFileDataSource(abs, watch)
	if err != nil {
		return err
	}
	return c.LoadFromDataSource(ds, unmarshal)
}

// LoadFromDataSource loads from ds and keeps watching when supported.
func (c *Configuration) LoadFromDataSource(ds DataSource, unmarshal Unmarshaller) error {
	content, err := ds.ReadConfig()
	if err != nil {
		_ = ds.Close()
		return err
	}
	if err := c.Load(content, unmarshal); err != nil {
		_ = ds.Close()
		return err
	}

	c.mu.Lock()
	c.closers = append(c.closers, ds)
	done := c.done
	c.mu.Unlock()

	changed := ds.IsConfigChanged()
	if changed == nil {
		return nil
	}
	go func() {
		for {
			select {
			case <-done:
				return
			case _, ok := <-changed:
				if !ok {
					return
				}
				content, err := ds.ReadConfig()
				if err != nil {
					logMessage("conf: read config for reload failed: %v", err)
					continue
				}
				changed, err := c.reflush(content, unmarshal)
				if err != nil {
					logMessage("conf: reload config failed: %v", err)
					continue
				}
				logMessage("conf: config reloaded, %d key(s) changed", changed)
			}
		}
	}()
	return nil
}

func (c *Configuration) reflush(content []byte, unmarshal Unmarshaller) (int, error) {
	conf := make(map[string]any)
	if err := unmarshal(content, &conf); err != nil {
		return 0, err
	}
	c.mu.Lock()
	changes := c.replaceLocked(conf)
	c.mu.Unlock()
	c.notifyChanges(changes)
	return len(changes), nil
}

// Close stops watchers and closes all attached data sources.
func (c *Configuration) Close() error {
	var errs []error
	c.closeOnce.Do(func() {
		close(c.done)
		c.mu.Lock()
		closers := c.closers
		c.closers = nil
		c.mu.Unlock()
		for _, closer := range closers {
			if err := closer.Close(); err != nil {
				errs = append(errs, err)
			}
		}
	})
	return errors.Join(errs...)
}

// OnChange registers a hook called after a successful reload that changed keys.
func (c *Configuration) OnChange(fn func(*Configuration)) {
	c.hookMu.Lock()
	c.onChanges = append(c.onChanges, fn)
	c.hookMu.Unlock()
}

// OnLoaded registers a hook called once configuration has been loaded.
// If already loaded, fn runs immediately.
func (c *Configuration) OnLoaded(fn func(*Configuration)) {
	c.hookMu.Lock()
	if c.loaded {
		c.hookMu.Unlock()
		fn(c)
		return
	}
	c.onLoadeds = append(c.onLoadeds, fn)
	c.hookMu.Unlock()
}

// Loaded reports whether at least one successful load happened.
func (c *Configuration) Loaded() bool {
	c.hookMu.RLock()
	defer c.hookMu.RUnlock()
	return c.loaded
}

func (c *Configuration) markLoaded() {
	c.hookMu.Lock()
	if c.loaded {
		c.hookMu.Unlock()
		return
	}
	c.loaded = true
	hooks := append([]func(*Configuration){}, c.onLoadeds...)
	c.hookMu.Unlock()
	for _, fn := range hooks {
		fn(c)
	}
}

func (c *Configuration) notifyChanges(changes map[string]any) {
	if len(changes) == 0 {
		return
	}
	c.hookMu.RLock()
	hooks := append([]func(*Configuration){}, c.onChanges...)
	c.hookMu.RUnlock()
	for _, fn := range hooks {
		fn(c)
	}
}

// Get returns the raw value associated with key.
func (c *Configuration) Get(key string) any {
	if key == "" {
		return nil
	}
	if v, ok := c.keyMap.Load(key); ok {
		return v
	}
	c.mu.RLock()
	value, _ := lookupPath(c.override, strings.Split(key, c.keyDelim))
	c.mu.RUnlock()
	c.keyMap.Store(key, value)
	return value
}

// Exists reports whether key exists and is non-nil.
func (c *Configuration) Exists(key string) bool {
	return c.Get(key) != nil
}

// GetString returns the value as string.
func (c *Configuration) GetString(key string) string { return cast.ToString(c.Get(key)) }

// GetBool returns the value as bool.
func (c *Configuration) GetBool(key string) bool { return cast.ToBool(c.Get(key)) }

// GetInt returns the value as int.
func (c *Configuration) GetInt(key string) int { return cast.ToInt(c.Get(key)) }

// GetInt64 returns the value as int64.
func (c *Configuration) GetInt64(key string) int64 { return cast.ToInt64(c.Get(key)) }

// GetFloat64 returns the value as float64.
func (c *Configuration) GetFloat64(key string) float64 { return cast.ToFloat64(c.Get(key)) }

// GetTime returns the value as time.Time.
func (c *Configuration) GetTime(key string) time.Time { return cast.ToTime(c.Get(key)) }

// GetDuration returns the value as time.Duration.
func (c *Configuration) GetDuration(key string) time.Duration { return cast.ToDuration(c.Get(key)) }

// GetStringSlice returns the value as []string.
func (c *Configuration) GetStringSlice(key string) []string { return cast.ToStringSlice(c.Get(key)) }

// GetInt64Slice returns the value as []int64.
func (c *Configuration) GetInt64Slice(key string) []int64 { return xcast.ToInt64Slice(c.Get(key)) }

// GetSlice returns the value as []any.
func (c *Configuration) GetSlice(key string) []any { return cast.ToSlice(c.Get(key)) }

// GetStringMap returns the value as map[string]any.
func (c *Configuration) GetStringMap(key string) map[string]any {
	return cast.ToStringMap(c.Get(key))
}

// GetStringMapString returns the value as map[string]string.
func (c *Configuration) GetStringMapString(key string) map[string]string {
	return cast.ToStringMapString(c.Get(key))
}

// UnmarshalKey decodes key into rawVal. Use key "" to decode the whole tree.
func (c *Configuration) UnmarshalKey(key string, rawVal any, opts ...GetOption) error {
	options := defaultGetOptions
	for _, opt := range opts {
		opt(&options)
	}

	var value any
	if key == "" {
		c.mu.RLock()
		value = c.override
		c.mu.RUnlock()
	} else {
		value = c.Get(key)
		if value == nil {
			return fmt.Errorf("%w: %s", ErrInvalidKey, key)
		}
	}

	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToTimeDurationHookFunc(),
			mapstructure.StringToSliceHookFunc(","),
		),
		Result:           rawVal,
		TagName:          options.TagName,
		Squash:           true,
		WeaklyTypedInput: true,
	})
	if err != nil {
		return err
	}
	return decoder.Decode(value)
}

// Traverse flattens the whole configuration into dotted keys.
func (c *Configuration) Traverse(sep string) map[string]any {
	if sep == "" {
		sep = c.keyDelim
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return flattenWithSep(c.override, sep)
}

// Debug prints the current configuration as indented JSON.
func (c *Configuration) Debug(sep string) {
	data, err := json.MarshalIndent(c.Traverse(sep), "", "  ")
	if err != nil {
		logMessage("conf: %v", err)
		return
	}
	logMessage("conf:\n%s", data)
}

func setPath(m map[string]any, delim, key string, val any) {
	paths := strings.Split(key, delim)
	for _, p := range paths[:len(paths)-1] {
		next, ok := m[p].(map[string]any)
		if !ok {
			next = make(map[string]any)
			m[p] = next
		}
		m = next
	}
	m[paths[len(paths)-1]] = val
}

func lookupPath(m map[string]any, paths []string) (any, bool) {
	var current any = m
	for _, p := range paths {
		mm, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		current, ok = mm[p]
		if !ok {
			return nil, false
		}
	}
	return current, true
}

func flattenWithSep(m map[string]any, sep string) map[string]any {
	out := make(map[string]any)
	var walk func(prefix string, data map[string]any)
	walk = func(prefix string, data map[string]any) {
		for k, v := range data {
			key := k
			if prefix != "" {
				key = prefix + sep + k
			}
			if child, ok := asStringMap(v); ok {
				walk(key, child)
				continue
			}
			out[key] = v
		}
	}
	walk("", m)
	return out
}

func asStringMap(v any) (map[string]any, bool) {
	switch m := v.(type) {
	case map[string]any:
		return m, true
	case map[any]any:
		out := make(map[string]any, len(m))
		for k, v := range m {
			out[fmt.Sprintf("%v", k)] = v
		}
		return out, true
	default:
		return nil, false
	}
}

func diff(old, next map[string]any) map[string]any {
	changes := make(map[string]any)
	for k, v := range next {
		if ov, ok := old[k]; !ok || !reflect.DeepEqual(ov, v) {
			changes[k] = v
		}
	}
	for k := range old {
		if _, ok := next[k]; !ok {
			changes[k] = nil
		}
	}
	return changes
}

func refreshKeyMap(keyMap *sync.Map, next map[string]any) {
	keyMap.Range(func(key, _ any) bool {
		keyMap.Delete(key)
		return true
	})
	for k, v := range next {
		keyMap.Store(k, v)
	}
}
