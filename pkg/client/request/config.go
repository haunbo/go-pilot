// Package request is a small JSON-friendly HTTP client with retries, auth,
// slow-request logging and request-id propagation.
package request

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/haunbo/go-pilot/pkg/conf"
	"github.com/haunbo/go-pilot/pkg/xlog"
)

// AuthConfig configures common authentication.
type AuthConfig struct {
	// Type is bearer, basic or apikey.
	Type string `json:"type"`
	// Token is used by bearer.
	Token string `json:"token"`
	// Username and Password are used by basic.
	Username string `json:"username"`
	Password string `json:"password"`
	// Header is the header name used by apikey (default X-API-Key).
	Header string `json:"header"`
	// QueryParam switches apikey to a query parameter when set.
	QueryParam string `json:"queryParam"`
}

// Config controls an HTTP client.
type Config struct {
	BaseURL            string            `json:"baseURL"`
	Timeout            time.Duration     `json:"timeout"`
	RetryCount         int               `json:"retryCount"`
	RetryWaitTime      time.Duration     `json:"retryWaitTime"`
	MaxRetryWaitTime   time.Duration     `json:"maxRetryWaitTime"`
	RetryNonIdempotent bool              `json:"retryNonIdempotent"`
	SlowThreshold      time.Duration     `json:"slowThreshold"`
	MaxResponseBytes   int64             `json:"maxResponseBytes"`
	Headers            map[string]string `json:"headers"`
	Auth               AuthConfig        `json:"auth"`
	Debug              bool              `json:"debug"`
	RequestIDHeader    string            `json:"requestIDHeader"`

	logger *xlog.Logger
}

// DefaultConfig returns safe defaults.
func DefaultConfig() *Config {
	return &Config{
		Timeout:          10 * time.Second,
		RetryWaitTime:    200 * time.Millisecond,
		MaxRetryWaitTime: 5 * time.Second,
		SlowThreshold:    500 * time.Millisecond,
		MaxResponseBytes: 10 << 20,
		RequestIDHeader:  "X-Request-ID",
		logger:           xlog.With(xlog.String("mod", "client.request")),
	}
}

// RawConfig loads request.<name> from conf and falls back to defaults.
func RawConfig(key string) *Config {
	cfg := DefaultConfig()
	if err := conf.UnmarshalKey(key, cfg); err != nil && !errors.Is(err, conf.ErrInvalidKey) {
		xlog.Warn("load request config failed, using defaults", xlog.String("key", key), xlog.FieldErr(err))
	}
	if cfg.logger == nil {
		cfg.logger = xlog.With(xlog.String("mod", "client.request"))
	}
	return cfg
}

// StdConfig loads request.<name>.
func StdConfig(name string) *Config {
	return RawConfig("request." + name)
}

// WithLogger sets the client logger.
func (c *Config) WithLogger(logger *xlog.Logger) *Config {
	if logger != nil {
		c.logger = logger
	}
	return c
}

// Build validates cfg and creates a Client.
func (c *Config) Build() (*Client, error) {
	if c.Timeout <= 0 {
		c.Timeout = 10 * time.Second
	}
	if c.RetryWaitTime <= 0 {
		c.RetryWaitTime = 200 * time.Millisecond
	}
	if c.MaxRetryWaitTime <= 0 {
		c.MaxRetryWaitTime = 5 * time.Second
	}
	if c.MaxResponseBytes <= 0 {
		c.MaxResponseBytes = 10 << 20
	}
	if c.RequestIDHeader == "" {
		c.RequestIDHeader = "X-Request-ID"
	}
	if c.logger == nil {
		c.logger = xlog.With(xlog.String("mod", "client.request"))
	}

	var base *url.URL
	if c.BaseURL != "" {
		parsed, err := url.Parse(c.BaseURL)
		if err != nil || !parsed.IsAbs() {
			return nil, fmt.Errorf("request: invalid base url %q", c.BaseURL)
		}
		base = parsed
	}

	switch strings.ToLower(c.Auth.Type) {
	case "", "bearer", "basic", "apikey":
	default:
		return nil, fmt.Errorf("request: unsupported auth type %q", c.Auth.Type)
	}
	return newClient(c, base), nil
}

// MustBuild builds the client and panics on error.
func (c *Config) MustBuild() *Client {
	client, err := c.Build()
	if err != nil {
		panic(err)
	}
	return client
}
