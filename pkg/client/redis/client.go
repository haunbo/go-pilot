package redis

import (
	"context"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// ErrNotFound is returned when a key does not exist.
var ErrNotFound = goredis.Nil

// Client wraps a go-redis client.
type Client struct {
	master *goredis.Client
	config *Config
}

func newClient(cfg *Config) (*Client, error) {
	if cfg == nil {
		return nil, fmt.Errorf("redis: nil config")
	}
	if cfg.Addr == "" {
		return nil, fmt.Errorf("redis: empty addr for %q", cfg.name)
	}
	if cfg.logger == nil {
		cfg.logger = xlogDefault()
	}

	client := goredis.NewClient(&goredis.Options{
		Addr:                  cfg.Addr,
		Username:              cfg.Username,
		Password:              cfg.Password,
		DB:                    cfg.DB,
		MaxRetries:            cfg.MaxRetries,
		DialTimeout:           cfg.DialTimeout,
		ReadTimeout:           cfg.ReadTimeout,
		WriteTimeout:          cfg.WriteTimeout,
		PoolSize:              cfg.PoolSize,
		MinIdleConns:          cfg.MinIdleConns,
		ContextTimeoutEnabled: cfg.ContextTimeoutEnabled,
	})

	timeout := cfg.DialTimeout
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout+time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("redis: ping %s: %w", cfg.Addr, err)
	}

	if cfg.Debug {
		client.AddHook(newDebugHook(cfg.name, cfg.Addr, cfg.logger))
	}
	if cfg.EnableAccessLog {
		client.AddHook(newAccessHook(cfg.name, cfg.Addr, cfg.logger))
	}
	return &Client{master: client, config: cfg}, nil
}

// RDB returns the underlying go-redis client.
func (c *Client) RDB() *goredis.Client { return c.master }

// Config returns the client config.
func (c *Client) Config() *Config { return c.config }

// Ping checks connectivity.
func (c *Client) Ping(ctx context.Context) error {
	return c.master.Ping(ctx).Err()
}

// Close closes the connection pool.
func (c *Client) Close() error {
	if c.master == nil {
		return nil
	}
	return c.master.Close()
}
