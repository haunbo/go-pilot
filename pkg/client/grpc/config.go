// Package grpc provides a gRPC client wrapper with timeouts, access logs,
// request-id propagation and optional TLS.
package grpc

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/haunbo/go-pilot/pkg/conf"
	"github.com/haunbo/go-pilot/pkg/xlog"
	googlegrpc "google.golang.org/grpc"
	"google.golang.org/grpc/balancer/roundrobin"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

// ClientConn is an alias of grpc.ClientConn.
type ClientConn = googlegrpc.ClientConn

// Config controls a gRPC client connection.
type Config struct {
	Addr                string        `json:"addr"`
	DialTimeout         time.Duration `json:"dialTimeout"`
	ReadTimeout         time.Duration `json:"readTimeout"`
	SlowThreshold       time.Duration `json:"slowThreshold"`
	EnableAccessLog     bool          `json:"enableAccessLog"`
	AccessLogLevel      string        `json:"accessLogLevel"` // info or debug
	BalancerName        string        `json:"balancerName"`
	KeepAliveTime       time.Duration `json:"keepAliveTime"`
	KeepAliveTimeout    time.Duration `json:"keepAliveTimeout"`
	PermitWithoutStream bool          `json:"permitWithoutStream"`
	EnableTLS           bool          `json:"enableTLS"`
	ServerName          string        `json:"serverName"`
	CaFile              string        `json:"caFile"`
	CertFile            string        `json:"certFile"`
	PrivateFile         string        `json:"privateFile"`
	InsecureSkipVerify  bool          `json:"insecureSkipVerify"`

	logger      *xlog.Logger
	dialOptions []googlegrpc.DialOption
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		DialTimeout:         3 * time.Second,
		ReadTimeout:         1 * time.Second,
		SlowThreshold:       600 * time.Millisecond,
		EnableAccessLog:     true,
		AccessLogLevel:      "info",
		BalancerName:        roundrobin.Name,
		KeepAliveTime:       5 * time.Minute,
		KeepAliveTimeout:    20 * time.Second,
		PermitWithoutStream: true,
		logger:              xlog.With(xlog.String("mod", "client.grpc")),
	}
}

// RawConfig loads grpc.<name> and falls back to defaults when missing.
func RawConfig(key string) *Config {
	cfg := DefaultConfig()
	if err := conf.UnmarshalKey(key, cfg); err != nil && !errors.Is(err, conf.ErrInvalidKey) {
		xlog.Warn("load grpc client config failed, using defaults", xlog.String("key", key), xlog.FieldErr(err))
	}
	if cfg.logger == nil {
		cfg.logger = xlog.With(xlog.String("mod", "client.grpc"))
	}
	return cfg
}

// StdConfig loads grpc.<name>.
func StdConfig(name string) *Config {
	return RawConfig("grpc." + name)
}

// WithLogger sets the client logger.
func (c *Config) WithLogger(logger *xlog.Logger) *Config {
	if logger != nil {
		c.logger = logger
	}
	return c
}

// WithDialOption appends gRPC dial options.
func (c *Config) WithDialOption(options ...googlegrpc.DialOption) *Config {
	c.dialOptions = append(c.dialOptions, options...)
	return c
}

// Build creates the client connection. When DialTimeout is greater than zero,
// Build waits until the connection is ready or the timeout expires. Set
// DialTimeout to zero for a lazy connection.
func (c *Config) Build() (*googlegrpc.ClientConn, error) {
	if c.Addr == "" {
		return nil, fmt.Errorf("grpc: addr is required")
	}
	if c.logger == nil {
		c.logger = xlog.With(xlog.String("mod", "client.grpc"))
	}
	if c.BalancerName == "" {
		c.BalancerName = roundrobin.Name
	}

	creds, err := c.transportCredentials()
	if err != nil {
		return nil, err
	}
	options := []googlegrpc.DialOption{
		googlegrpc.WithTransportCredentials(creds),
		googlegrpc.WithDefaultServiceConfig(fmt.Sprintf(`{"loadBalancingPolicy":%q}`, c.BalancerName)),
	}
	if c.KeepAliveTime > 0 {
		options = append(options, googlegrpc.WithKeepaliveParams(keepaliveParams(c)))
	}
	options = append(options, c.dialOptions...)

	unary := make([]googlegrpc.UnaryClientInterceptor, 0, 3)
	if c.EnableAccessLog {
		unary = append(unary, accessUnaryClientInterceptor(c))
	}
	unary = append(unary, requestIDUnaryClientInterceptor())
	if c.ReadTimeout > 0 {
		unary = append(unary, timeoutUnaryClientInterceptor(c))
	}
	if len(unary) > 0 {
		options = append(options, googlegrpc.WithChainUnaryInterceptor(unary...))
	}

	options = append(options, googlegrpc.WithChainStreamInterceptor(requestIDStreamClientInterceptor()))

	conn, err := googlegrpc.NewClient(c.Addr, options...)
	if err != nil {
		return nil, fmt.Errorf("grpc: new client: %w", err)
	}
	if c.DialTimeout > 0 {
		if err := waitReady(conn, c.DialTimeout); err != nil {
			_ = conn.Close()
			return nil, err
		}
	}
	return conn, nil
}

// MustBuild builds the connection and panics on error.
func (c *Config) MustBuild() *googlegrpc.ClientConn {
	conn, err := c.Build()
	if err != nil {
		panic(err)
	}
	return conn
}

func (c *Config) transportCredentials() (credentials.TransportCredentials, error) {
	if !c.EnableTLS {
		return insecure.NewCredentials(), nil
	}
	tlsConfig := &tls.Config{
		MinVersion:         tls.VersionTLS12,
		ServerName:         c.ServerName,
		InsecureSkipVerify: c.InsecureSkipVerify,
	}
	if c.CaFile != "" {
		caPEM, err := os.ReadFile(c.CaFile)
		if err != nil {
			return nil, fmt.Errorf("grpc: read ca: %w", err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(caPEM) {
			return nil, fmt.Errorf("grpc: invalid ca pem")
		}
		tlsConfig.RootCAs = pool
	}
	if c.CertFile != "" || c.PrivateFile != "" {
		cert, err := tls.LoadX509KeyPair(c.CertFile, c.PrivateFile)
		if err != nil {
			return nil, fmt.Errorf("grpc: load client key pair: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}
	return credentials.NewTLS(tlsConfig), nil
}
