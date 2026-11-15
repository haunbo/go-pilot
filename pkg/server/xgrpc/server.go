package xgrpc

import (
	"context"
	"errors"
	"net"
	"os"

	"github.com/haunbo/go-pilot/pkg/xlog"
	"google.golang.org/grpc"
)

// Server wraps grpc.Server and owns its listener.
type Server struct {
	*grpc.Server
	config   *Config
	listener net.Listener
}

// Serve starts serving gRPC requests until the server stops.
func (s *Server) Serve() error {
	s.config.logger.Info("grpc server started", xlog.String("addr", s.Addr()))
	err := s.Server.Serve(s.listener)
	if err != nil && !errors.Is(err, grpc.ErrServerStopped) {
		return err
	}
	return nil
}

// Stop stops the server immediately.
func (s *Server) Stop() error {
	s.Server.Stop()
	return nil
}

// GracefulStop waits for active RPCs or stops when ctx is done.
func (s *Server) GracefulStop(ctx context.Context) error {
	done := make(chan struct{})
	go func() {
		s.Server.GracefulStop()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		s.Server.Stop()
		return ctx.Err()
	}
}

// Addr returns the actual listen address.
func (s *Server) Addr() string {
	return s.listener.Addr().String()
}

// Healthz reports process health.
func (s *Server) Healthz() bool { return true }

// Info describes the server instance.
type Info struct {
	Name     string `json:"name"`
	Scheme   string `json:"scheme"`
	Address  string `json:"address"`
	Hostname string `json:"hostname"`
}

// Info returns service metadata.
func (s *Server) Info() *Info {
	hostname, _ := os.Hostname()
	return &Info{
		Name:     "grpc",
		Scheme:   "grpc",
		Address:  s.Addr(),
		Hostname: hostname,
	}
}

// Config returns the server config.
func (s *Server) Config() *Config { return s.config }
