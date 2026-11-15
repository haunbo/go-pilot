// Package xrdb wraps GORM with configuration loading and connection pooling.
package xrdb

import (
	"errors"
	"time"

	"github.com/haunbo/go-pilot/pkg/conf"
	"github.com/haunbo/go-pilot/pkg/xlog"
	"github.com/spf13/cast"
)

// Type is a supported relational database type.
type Type string

const (
	// MySQL uses gorm.io/driver/mysql.
	MySQL Type = "mysql"
	// Postgres uses gorm.io/driver/postgres.
	Postgres Type = "postgres"
	// SQLServer uses gorm.io/driver/sqlserver.
	SQLServer Type = "sqlserver"
	// SQLite uses gorm.io/driver/sqlite.
	SQLite Type = "sqlite"
)

// Config controls a GORM client. Type can be mysql, postgres, sqlserver or sqlite.
type Config struct {
	Name            string        `json:"name"`
	DSN             string        `json:"dsn"`
	Type            Type          `json:"type"`
	Debug           bool          `json:"debug"`
	MaxIdleConns    int           `json:"maxIdleConns"`
	MaxOpenConns    int           `json:"maxOpenConns"`
	ConnMaxLifetime time.Duration `json:"connMaxLifetime"`
	SlowThreshold   time.Duration `json:"slowThreshold"`
	LogLevel        string        `json:"logLevel"` // silent, error, warn, info
}

// DefaultConfig returns defaults suited to a web backend.
func DefaultConfig() *Config {
	return &Config{
		Type:            MySQL,
		MaxIdleConns:    10,
		MaxOpenConns:    50,
		ConnMaxLifetime: cast.ToDuration("300s"),
		SlowThreshold:   cast.ToDuration("200ms"),
		LogLevel:        "warn",
	}
}

// RawConfig loads a database config from the dotted key.
func RawConfig(key string) *Config {
	cfg := DefaultConfig()
	if err := conf.UnmarshalKey(key, cfg); err != nil && !errors.Is(err, conf.ErrInvalidKey) {
		xlog.Warn("load database config failed, using defaults", xlog.String("key", key), xlog.FieldErr(err))
	}
	if cfg.Name == "" {
		cfg.Name = key
	}
	return cfg
}

// StdConfig loads database.<name>.
func StdConfig(name string) *Config {
	return RawConfig("database." + name)
}

// Build opens the database, configures the pool and verifies connectivity.
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
