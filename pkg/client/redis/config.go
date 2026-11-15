// Package redis wraps go-redis with config loading, health checks and
// optional access/debug logging.
package redis

import (
	"errors"
	"time"

	"github.com/haunbo/go-pilot/pkg/conf"
	"github.com/haunbo/go-pilot/pkg/xlog"
	"github.com/spf13/cast"
)

// Config controls a Redis client.
type Config struct {
	Addr     string `json:"addr"`
	Username string `json:"username"`
	Password string `json:"password"`
	// DB is the Redis database index. Cross-database keys are not recommended.
	DB int `json:"db"`
	// PoolSize is the maximum number of connections.
	PoolSize int `json:"poolSize"`
	// MinIdleConns keeps warm connections ready.
	MinIdleConns int `json:"minIdleConns"`
	// MaxRetries is the maximum number of retries; -1 disables retries.
	MaxRetries int `json:"maxRetries"`
	// DialTimeout is the timeout for establishing new connections.
	DialTimeout time.Duration `json:"dialTimeout"`
	// ReadTimeout is the socket read timeout.
	ReadTimeout time.Duration `json:"readTimeout"`
	// WriteTimeout is the socket write timeout.
	WriteTimeout time.Duration `json:"writeTimeout"`
	// ContextTimeoutEnabled makes the client respect context deadlines.
	ContextTimeoutEnabled bool `json:"contextTimeoutEnabled"`

	// Debug logs every command at debug level.
	Debug bool `json:"debug"`
	// EnableAccessLog logs every command with latency and errors.
	EnableAccessLog bool `json:"enableAccessLog"`

	logger *xlog.Logger
	name   string
}

// DefaultConfig returns defaults for a local Redis.
func DefaultConfig() *Config {
	return &Config{
		name:                  "default",
		Addr:                  "127.0.0.1:6379",
		DB:                    0,
		PoolSize:              200,
		MinIdleConns:          20,
		MaxRetries:            3,
		DialTimeout:           cast.ToDuration("3s"),
		ReadTimeout:           cast.ToDuration("1s"),
		WriteTimeout:          cast.ToDuration("1s"),
		ContextTimeoutEnabled: true,
		logger:                xlog.With(xlog.String("mod", "client.redis")),
	}
}

// RawConfig loads redis config from the dotted key, falling back to defaults
// when the section is missing.
func RawConfig(key string) *Config {
	cfg := DefaultConfig()
	if err := conf.UnmarshalKey(key, cfg); err != nil && !errors.Is(err, conf.ErrInvalidKey) {
		xlog.Warn("load redis config failed, using defaults", xlog.String("key", key), xlog.FieldErr(err))
	}
	cfg.name = key
	if cfg.logger == nil {
		cfg.logger = xlog.With(xlog.String("mod", "client.redis"))
	}
	return cfg
}

// StdConfig loads redis.<name>.
func StdConfig(name string) *Config {
	return RawConfig("redis." + name)
}

// Name returns the config key used to load this client.
func (c *Config) Name() string { return c.name }

// WithLogger overrides the logger used by access/debug hooks.
func (c *Config) WithLogger(logger *xlog.Logger) *Config {
	if logger != nil {
		c.logger = logger
	}
	return c
}

// Build creates the client and verifies connectivity with PING.
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
