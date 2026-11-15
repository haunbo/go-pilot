// Package xmongo wraps the official MongoDB driver with configuration loading,
// connection pooling and startup health checks.
package xmongo

import (
	"errors"
	"time"

	"github.com/haunbo/go-pilot/pkg/conf"
	"github.com/haunbo/go-pilot/pkg/xlog"
)

// Config controls a MongoDB client.
type Config struct {
	Name                   string        `json:"name"`
	DSN                    string        `json:"dsn"`
	MinPoolSize            uint64        `json:"minPoolSize"`
	MaxPoolSize            uint64        `json:"maxPoolSize"`
	ConnectTimeout         time.Duration `json:"connectTimeout"`
	ServerSelectionTimeout time.Duration `json:"serverSelectionTimeout"`
}

// DefaultConfig returns safe defaults.
func DefaultConfig() *Config {
	return &Config{
		MinPoolSize:            0,
		MaxPoolSize:            100,
		ConnectTimeout:         5 * time.Second,
		ServerSelectionTimeout: 5 * time.Second,
	}
}

// RawConfig loads MongoDB config from the dotted key.
func RawConfig(key string) *Config {
	cfg := DefaultConfig()
	if err := conf.UnmarshalKey(key, cfg); err != nil && !errors.Is(err, conf.ErrInvalidKey) {
		xlog.Warn("load mongo config failed, using defaults", xlog.String("key", key), xlog.FieldErr(err))
	}
	cfg.Name = key
	return cfg
}

// StdConfig loads mongo.<name>.
func StdConfig(name string) *Config {
	return RawConfig("mongo." + name)
}

// Build connects to MongoDB and verifies the connection with PING.
func (c *Config) Build() (*Client, error) {
	return newClient(c)
}

// MustBuild builds the client and panics on error.
func (c *Config) MustBuild() *Client {
	client, err := c.Build()
	if err != nil {
		panic(err)
	}
	return client
}
