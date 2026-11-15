// Package xgrpc provides an opinionated gRPC server with access logging,
// panic recovery, request-id propagation and optional TLS/mTLS.
package xgrpc

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"os"

	"github.com/haunbo/go-pilot/pkg/conf"
	"github.com/haunbo/go-pilot/pkg/xlog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// Config controls the gRPC server.
type Config struct {
	Host                      string `json:"host"`
	Port                      int    `json:"port"`
	Network                   string `json:"network"`
	EnableAccessLog           bool   `json:"enableAccessLog"`
	SlowQueryThresholdInMilli int64  `json:"slowQueryThresholdInMilli"`
	MaxRecvMsgSize            int    `json:"maxRecvMsgSize"`
	MaxSendMsgSize            int    `json:"maxSendMsgSize"`
	EnableTLS                 bool   `json:"enableTLS"`
	CaFile                    string `json:"caFile"`
	CertFile                  string `json:"certFile"`
	PrivateFile               string `json:"privateFile"`

	logger             *xlog.Logger
	serverOptions      []grpc.ServerOption
	unaryInterceptors  []grpc.UnaryServerInterceptor
	streamInterceptors []grpc.StreamServerInterceptor
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		Port:                      9092,
		Network:                   "tcp",
		EnableAccessLog:           true,
		SlowQueryThresholdInMilli: 500,
		logger:                    xlog.With(xlog.String("mod", "grpc.server")),
	}
}

// RawConfig loads server.<name> and falls back to defaults when missing.
func RawConfig(key string) *Config {
	cfg := DefaultConfig()
	if err := conf.UnmarshalKey(key, cfg); err != nil && !errors.Is(err, conf.ErrInvalidKey) {
		xlog.Warn("load grpc server config failed, using defaults", xlog.String("key", key), xlog.FieldErr(err))
	}
	if cfg.logger == nil {
		cfg.logger = xlog.With(xlog.String("mod", "grpc.server"))
	}
	return cfg
}

// StdConfig loads server.<name>.
func StdConfig(name string) *Config {
	return RawConfig("server." + name)
}

// WithLogger sets the server logger.
func (c *Config) WithLogger(logger *xlog.Logger) *Config {
	if logger != nil {
		c.logger = logger
	}
	return c
}

// WithServerOption appends gRPC server options. Do not pass interceptor
// options here; use WithUnaryInterceptor or WithStreamInterceptor instead.
func (c *Config) WithServerOption(options ...grpc.ServerOption) *Config {
	c.serverOptions = append(c.serverOptions, options...)
	return c
}

// WithUnaryInterceptor appends unary server interceptors.
func (c *Config) WithUnaryInterceptor(interceptors ...grpc.UnaryServerInterceptor) *Config {
	c.unaryInterceptors = append(c.unaryInterceptors, interceptors...)
	return c
}

// WithStreamInterceptor appends stream server interceptors.
func (c *Config) WithStreamInterceptor(interceptors ...grpc.StreamServerInterceptor) *Config {
	c.streamInterceptors = append(c.streamInterceptors, interceptors...)
	return c
}

// Address returns host:port.
func (c *Config) Address() string {
	return net.JoinHostPort(c.Host, fmt.Sprintf("%d", c.Port))
}

// Build creates the server and listens on the configured address.
func (c *Config) Build() (*Server, error) {
	if c.logger == nil {
		c.logger = xlog.With(xlog.String("mod", "grpc.server"))
	}
	if c.Network == "" {
		c.Network = "tcp"
	}

	options := append([]grpc.ServerOption{}, c.serverOptions...)
	if c.MaxRecvMsgSize > 0 {
		options = append(options, grpc.MaxRecvMsgSize(c.MaxRecvMsgSize))
	}
	if c.MaxSendMsgSize > 0 {
		options = append(options, grpc.MaxSendMsgSize(c.MaxSendMsgSize))
	}
	if c.EnableTLS {
		creds, err := c.transportCredentials()
		if err != nil {
			return nil, err
		}
		options = append(options, grpc.Creds(creds))
	}

	unary := append([]grpc.UnaryServerInterceptor{accessUnaryServerInterceptor(c.logger, c)}, c.unaryInterceptors...)
	stream := append([]grpc.StreamServerInterceptor{accessStreamServerInterceptor(c.logger, c)}, c.streamInterceptors...)
	options = append(options,
		grpc.ChainUnaryInterceptor(unary...),
		grpc.ChainStreamInterceptor(stream...),
	)

	listener, err := net.Listen(c.Network, c.Address())
	if err != nil {
		return nil, fmt.Errorf("xgrpc: listen %s: %w", c.Address(), err)
	}
	if tcpAddr, ok := listener.Addr().(*net.TCPAddr); ok {
		c.Port = tcpAddr.Port
	}
	return &Server{Server: grpc.NewServer(options...), config: c, listener: listener}, nil
}

// MustBuild builds the server and panics on error.
func (c *Config) MustBuild() *Server {
	server, err := c.Build()
	if err != nil {
		panic(err)
	}
	return server
}

func (c *Config) transportCredentials() (credentials.TransportCredentials, error) {
	cert, err := tls.LoadX509KeyPair(c.CertFile, c.PrivateFile)
	if err != nil {
		return nil, fmt.Errorf("xgrpc: load key pair: %w", err)
	}
	tlsConfig := &tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{cert},
	}
	if c.CaFile != "" {
		caPEM, err := os.ReadFile(c.CaFile)
		if err != nil {
			return nil, fmt.Errorf("xgrpc: read ca: %w", err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(caPEM) {
			return nil, fmt.Errorf("xgrpc: invalid ca pem")
		}
		tlsConfig.ClientCAs = pool
		tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
	}
	return credentials.NewTLS(tlsConfig), nil
}
