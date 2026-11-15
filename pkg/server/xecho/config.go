// Package xecho provides an opinionated Echo HTTP server with structured
// access logs, panic recovery, request IDs, validation and CORS.
package xecho

import (
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/haunbo/go-pilot/pkg/conf"
	"github.com/haunbo/go-pilot/pkg/xlog"
	"github.com/labstack/echo/v4/middleware"
)

// Config controls the HTTP server.
type Config struct {
	Host                      string        `json:"host"`
	Port                      int           `json:"port"`
	Debug                     bool          `json:"debug"`
	ServiceAddress            string        `json:"serviceAddress"`
	CertFile                  string        `json:"certFile"`
	PrivateFile               string        `json:"privateFile"`
	EnableTLS                 bool          `json:"enableTLS"`
	ReadTimeout               time.Duration `json:"readTimeout"`
	WriteTimeout              time.Duration `json:"writeTimeout"`
	IdleTimeout               time.Duration `json:"idleTimeout"`
	SlowQueryThresholdInMilli int64         `json:"slowQueryThresholdInMilli"`
	EnableAccessLog           bool          `json:"enableAccessLog"`
	AllowOrigins              []string      `json:"allowOrigins"`
	BodyLimit                 string        `json:"bodyLimit"`
	ValidationLocale          string        `json:"validationLocale"`

	logger *xlog.Logger
	key    string // dotted config key, kept so Server.Reload can re-read it
}

// DefaultConfig returns sensible defaults for a backend API.
func DefaultConfig() *Config {
	return &Config{
		Host:                      "",
		Port:                      8080,
		ReadTimeout:               15 * time.Second,
		WriteTimeout:              15 * time.Second,
		IdleTimeout:               60 * time.Second,
		SlowQueryThresholdInMilli: 500,
		EnableAccessLog:           true,
		AllowOrigins:              []string{"*"},
		BodyLimit:                 "10M",
		ValidationLocale:          "en",
		logger:                    xlog.With(xlog.String("mod", "echo.server")),
	}
}

// RawConfig loads a server config from the dotted key, falling back to defaults.
func RawConfig(key string) *Config {
	cfg := DefaultConfig()
	cfg.key = key
	if err := conf.UnmarshalKey(key, cfg); err != nil && !errors.Is(err, conf.ErrInvalidKey) {
		xlog.Warn("load http server config failed, using defaults", xlog.String("key", key), xlog.FieldErr(err))
	}
	if cfg.logger == nil {
		cfg.logger = xlog.With(xlog.String("mod", "echo.server"))
	}
	return cfg
}

// StdConfig loads server.<name>.
func StdConfig(name string) *Config {
	return RawConfig("server." + name)
}

// Key returns the dotted config key this config was read from, or "" when the
// config was built by hand.
func (c *Config) Key() string { return c.key }

// WithLogger sets the logger used by middleware.
func (c *Config) WithLogger(logger *xlog.Logger) *Config {
	if logger != nil {
		c.logger = logger
	}
	return c
}

// WithHost overrides the listen host.
func (c *Config) WithHost(host string) *Config {
	c.Host = host
	return c
}

// WithPort overrides the listen port.
func (c *Config) WithPort(port int) *Config {
	c.Port = port
	return c
}

// Address returns host:port.
func (c *Config) Address() string {
	return net.JoinHostPort(c.Host, fmt.Sprintf("%d", c.Port))
}

// MustBuild builds the server and panics on error.
func (c *Config) MustBuild() *Server {
	s, err := c.Build()
	if err != nil {
		panic(err)
	}
	return s
}

// Build creates the server and installs global middleware.
func (c *Config) Build() (*Server, error) {
	if c.logger == nil {
		c.logger = xlog.With(xlog.String("mod", "echo.server"))
	}
	server, err := newServer(c)
	if err != nil {
		return nil, err
	}
	server.Use(requestIDMiddleware())
	// recoverMiddleware 也负责访问日志，它按请求读取当前配置，
	// 因此 enableAccessLog / slowQueryThresholdInMilli 改动后无需重启。
	server.Use(recoverMiddleware(server))
	server.Use(corsMiddleware(c.AllowOrigins))
	if c.BodyLimit != "" {
		server.Use(middleware.BodyLimit(c.BodyLimit))
	}
	return server, nil
}
