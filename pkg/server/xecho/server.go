package xecho

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/haunbo/go-pilot/pkg/xlog"
	"github.com/labstack/echo/v4"
)

// ServiceInfo describes a running service instance.
type ServiceInfo struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Scheme   string `json:"scheme"`
	Address  string `json:"address"`
	Hostname string `json:"hostname"`
}

// Server wraps echo.Echo.
type Server struct {
	*echo.Echo
	config   *Config
	listener net.Listener
}

func newServer(config *Config) (*Server, error) {
	listener, err := listen(config)
	if err != nil {
		return nil, err
	}
	if tcpAddr, ok := listener.Addr().(*net.TCPAddr); ok {
		config.Port = tcpAddr.Port
	}

	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.Validator = NewValidator(config.ValidationLocale)

	s := &Server{Echo: e, config: config, listener: listener}
	e.HTTPErrorHandler = s.handleError
	e.Server.ReadTimeout = config.ReadTimeout
	e.Server.WriteTimeout = config.WriteTimeout
	e.Server.IdleTimeout = config.IdleTimeout
	e.Server.ReadHeaderTimeout = config.ReadTimeout
	return s, nil
}

func listen(config *Config) (net.Listener, error) {
	if !config.EnableTLS {
		l, err := net.Listen("tcp", config.Address())
		if err != nil {
			return nil, fmt.Errorf("listen %s: %w", config.Address(), err)
		}
		return l, nil
	}
	cert, err := tls.LoadX509KeyPair(config.CertFile, config.PrivateFile)
	if err != nil {
		return nil, fmt.Errorf("load tls key pair: %w", err)
	}
	l, err := tls.Listen("tcp", config.Address(), &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	})
	if err != nil {
		return nil, fmt.Errorf("listen tls %s: %w", config.Address(), err)
	}
	return l, nil
}

// Serve starts serving HTTP until the server stops.
func (s *Server) Serve() error {
	s.Debug = s.config.Debug
	if s.Debug {
		for _, route := range s.Routes() {
			fmt.Printf("[echo] %-7s %s\n", route.Method, route.Path)
		}
	}
	s.config.logger.Info("http server started",
		xlog.String("addr", s.listener.Addr().String()),
		xlog.Bool("tls", s.config.EnableTLS),
	)
	s.Echo.Listener = s.listener
	err := s.Echo.Start("")
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// Stop closes the listener and active connections immediately.
func (s *Server) Stop() error {
	return s.Echo.Close()
}

// GracefulStop stops accepting new connections and waits for active requests.
func (s *Server) GracefulStop(ctx context.Context) error {
	return s.Echo.Shutdown(ctx)
}

// Healthz reports process health.
func (s *Server) Healthz() bool { return true }

// Info returns service metadata.
func (s *Server) Info() *ServiceInfo {
	address := s.listener.Addr().String()
	if s.config.ServiceAddress != "" {
		address = s.config.ServiceAddress
	}
	hostname, err := os.Hostname()
	if err != nil {
		hostname = ""
	}
	scheme := "http"
	if s.config.EnableTLS {
		scheme = "https"
	}
	return &ServiceInfo{
		ID:       hostname + "-" + address,
		Name:     "http",
		Scheme:   scheme,
		Address:  address,
		Hostname: hostname,
	}
}

// Addr returns the actual listener address, useful when port 0 is configured.
func (s *Server) Addr() string {
	return s.listener.Addr().String()
}

// Uptime is a small helper for health endpoints.
func (s *Server) Uptime(now time.Time, startedAt time.Time) time.Duration {
	return now.Sub(startedAt)
}
