package grpc_test

import (
	"context"
	"io"
	"testing"
	"time"

	clientgrpc "github.com/haunbo/go-pilot/pkg/client/grpc"
	"github.com/haunbo/go-pilot/pkg/server/xgrpc"
	"github.com/haunbo/go-pilot/pkg/xlog"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

func TestClientAgainstServer(t *testing.T) {
	discard := xlog.New(&xlog.Config{Level: "error", Format: xlog.FormatJSON, Output: io.Discard})

	serverCfg := xgrpc.DefaultConfig()
	serverCfg.Host = "127.0.0.1"
	serverCfg.Port = 0
	serverCfg.EnableAccessLog = false
	serverCfg.WithLogger(discard)

	server, err := serverCfg.Build()
	if err != nil {
		t.Fatal(err)
	}
	healthpb.RegisterHealthServer(server.Server, health.NewServer())
	go func() { _ = server.Serve() }()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = server.GracefulStop(ctx)
	}()

	clientCfg := clientgrpc.DefaultConfig()
	clientCfg.Addr = server.Addr()
	clientCfg.DialTimeout = 2 * time.Second
	clientCfg.EnableAccessLog = false
	clientCfg.WithLogger(discard)

	conn, err := clientCfg.Build()
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	resp, err := healthpb.NewHealthClient(conn).Check(ctx, &healthpb.HealthCheckRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != healthpb.HealthCheckResponse_SERVING {
		t.Fatalf("status = %v", resp.Status)
	}
}
