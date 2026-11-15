package conf

import (
	"io"
	"time"
)

var defaultConfiguration = New()

// Default returns the process-wide configuration.
func Default() *Configuration { return defaultConfiguration }

// LoadFromFile loads path into the default configuration.
func LoadFromFile(path string, watch bool) error {
	return defaultConfiguration.LoadFromFile(path, watch)
}

// LoadFromReader loads config into the default configuration.
func LoadFromReader(r io.Reader, unmarshal Unmarshaller) error {
	return defaultConfiguration.LoadFromReader(r, unmarshal)
}

// LoadFromDataSource loads ds into the default configuration.
func LoadFromDataSource(ds DataSource, unmarshal Unmarshaller) error {
	return defaultConfiguration.LoadFromDataSource(ds, unmarshal)
}

// LoadEnvironments loads environment overrides into the default configuration.
func LoadEnvironments(prefix string) { defaultConfiguration.LoadEnvironments(prefix) }

// Get returns a raw config value.
func Get(key string) any { return defaultConfiguration.Get(key) }

// Exists reports whether key exists.
func Exists(key string) bool { return defaultConfiguration.Exists(key) }

// Set sets a config value.
func Set(key string, val any) error { return defaultConfiguration.Set(key, val) }

// Apply replaces the default configuration.
func Apply(conf map[string]any) error { return defaultConfiguration.Apply(conf) }

// UnmarshalKey decodes key into rawVal.
func UnmarshalKey(key string, rawVal any, opts ...GetOption) error {
	return defaultConfiguration.UnmarshalKey(key, rawVal, opts...)
}

// GetString returns a string value.
func GetString(key string) string { return defaultConfiguration.GetString(key) }

// GetBool returns a bool value.
func GetBool(key string) bool { return defaultConfiguration.GetBool(key) }

// GetInt returns an int value.
func GetInt(key string) int { return defaultConfiguration.GetInt(key) }

// GetInt64 returns an int64 value.
func GetInt64(key string) int64 { return defaultConfiguration.GetInt64(key) }

// GetFloat64 returns a float64 value.
func GetFloat64(key string) float64 { return defaultConfiguration.GetFloat64(key) }

// GetDuration returns a time.Duration value.
func GetDuration(key string) time.Duration { return defaultConfiguration.GetDuration(key) }

// GetStringSlice returns a []string value.
func GetStringSlice(key string) []string { return defaultConfiguration.GetStringSlice(key) }

// GetStringMap returns a map[string]any value.
func GetStringMap(key string) map[string]any { return defaultConfiguration.GetStringMap(key) }

// OnChange registers a reload hook on the default configuration.
func OnChange(fn func(*Configuration)) { defaultConfiguration.OnChange(fn) }

// OnLoaded registers a load hook on the default configuration.
func OnLoaded(fn func(*Configuration)) { defaultConfiguration.OnLoaded(fn) }

// Traverse flattens the default configuration.
func Traverse(sep string) map[string]any { return defaultConfiguration.Traverse(sep) }

// Debug prints the default configuration.
func Debug(sep string) { defaultConfiguration.Debug(sep) }

// Close stops watchers of the default configuration.
func Close() error { return defaultConfiguration.Close() }

// Reset replaces the default configuration with an empty one.
func Reset() {
	_ = defaultConfiguration.Close()
	defaultConfiguration = New()
}
