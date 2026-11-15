// Package xlog is a small structured logging facade built on zap.
package xlog

import (
	"errors"
	"io"
	"path/filepath"

	"github.com/haunbo/go-pilot/pkg/conf"
)

const (
	// FormatText outputs console friendly text.
	FormatText = "text"
	// FormatJSON outputs one JSON object per line.
	FormatJSON = "json"
)

// Config controls logger output.
type Config struct {
	Dir        string    `json:"dir"`        // "-" or "" writes to stdout
	Name       string    `json:"name"`       // file name when Dir is set
	Level      string    `json:"level"`      // debug, info, warn, error, panic, fatal
	Compress   bool      `json:"compress"`   // compress rotated files
	MaxBackup  int       `json:"maxBackup"`  // number of rotated files to keep
	MaxSize    int       `json:"maxSize"`    // MB before rotation
	MaxAge     int       `json:"maxAge"`     // days to keep rotated files
	Format     string    `json:"format"`     // json or text
	CallerSkip int       `json:"callerSkip"` // extra caller frames to skip
	Output     io.Writer `json:"-"`          // optional explicit writer (tests, custom sinks)

	key string // dotted config key this snapshot was read from
}

// DefaultConfig returns production friendly defaults.
func DefaultConfig() *Config {
	return &Config{
		Dir:       "-",
		Name:      "app.log",
		Level:     "info",
		MaxSize:   500,
		MaxAge:    7,
		MaxBackup: 10,
		Format:    FormatJSON,
	}
}

// Filename returns the configured log file path.
func (c *Config) Filename() string {
	if c.Dir == "" || c.Dir == "-" {
		return ""
	}
	return filepath.Join(c.Dir, c.Name)
}

// RawConfig loads a logger config from the dotted key.
func RawConfig(key string) *Config {
	cfg := DefaultConfig()
	cfg.key = key
	if err := conf.UnmarshalKey(key, cfg); err != nil && !errors.Is(err, conf.ErrInvalidKey) {
		Warn("load logger config failed, using defaults", String("key", key), FieldErr(err))
	}
	return cfg
}

// StdConfig loads logger.<name>.
func StdConfig(name string) *Config {
	return RawConfig("logger." + name)
}

// Key returns the dotted config key this config was read from, or "" for
// configs built by hand.
func (c *Config) Key() string { return c.key }

// Build creates a Logger from the config.
func (c *Config) Build() *Logger {
	return New(c)
}
