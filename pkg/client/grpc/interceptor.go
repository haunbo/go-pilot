package grpc

import (
	"context"
	"fmt"
	"time"

	"github.com/haunbo/go-pilot/pkg/xlog"
	googlegrpc "google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const requestIDKey = "x-request-id"

func keepaliveParams(cfg *Config) keepalive.ClientParameters {
	return keepalive.ClientParameters{
		Time:                cfg.KeepAliveTime,
		Timeout:             cfg.KeepAliveTimeout,
		PermitWithoutStream: cfg.PermitWithoutStream,
	}
}

func accessUnaryClientInterceptor(cfg *Config) googlegrpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any, cc *googlegrpc.ClientConn, invoker googlegrpc.UnaryInvoker, opts ...googlegrpc.CallOption) error {
		start := time.Now()
		err := invoker(ctx, method, req, reply, cc, opts...)
		duration := time.Since(start)
		fields := []xlog.Field{
			xlog.String("method", method),
			xlog.String("target", cc.Target()),
			xlog.String("code", status.Code(err).String()),
			xlog.Int64("duration_ms", duration.Milliseconds()),
		}
		if id := xlog.RequestIDFromContext(ctx); id != "" {
			fields = append(fields, xlog.String("request_id", id))
		}
		if cfg.SlowThreshold > 0 && duration > cfg.SlowThreshold {
			fields = append(fields, xlog.Bool("slow", true))
		}
		if err != nil {
			fields = append(fields, xlog.FieldErr(err))
			cfg.logger.Warn("grpc access", fields...)
			return err
		}
		if cfg.AccessLogLevel == "debug" {
			cfg.logger.Debug("grpc access", fields...)
		} else {
			cfg.logger.Info("grpc access", fields...)
		}
		return nil
	}
}

func requestIDUnaryClientInterceptor() googlegrpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any, cc *googlegrpc.ClientConn, invoker googlegrpc.UnaryInvoker, opts ...googlegrpc.CallOption) error {
		ctx = injectRequestID(ctx)
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

func timeoutUnaryClientInterceptor(cfg *Config) googlegrpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any, cc *googlegrpc.ClientConn, invoker googlegrpc.UnaryInvoker, opts ...googlegrpc.CallOption) error {
		if _, ok := ctx.Deadline(); !ok {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, cfg.ReadTimeout)
			defer cancel()
		}
		start := time.Now()
		err := invoker(ctx, method, req, reply, cc, opts...)
		if cfg.SlowThreshold > 0 && time.Since(start) > cfg.SlowThreshold {
			cfg.logger.Warn("grpc slow call",
				xlog.String("method", method),
				xlog.String("target", cc.Target()),
				xlog.Int64("duration_ms", time.Since(start).Milliseconds()),
				xlog.FieldErr(err),
			)
		}
		return err
	}
}

func requestIDStreamClientInterceptor() googlegrpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *googlegrpc.StreamDesc, cc *googlegrpc.ClientConn, method string, streamer googlegrpc.Streamer, opts ...googlegrpc.CallOption) (googlegrpc.ClientStream, error) {
		return streamer(injectRequestID(ctx), desc, cc, method, opts...)
	}
}

func injectRequestID(ctx context.Context) context.Context {
	id := xlog.RequestIDFromContext(ctx)
	if id == "" {
		return ctx
	}
	md, ok := metadata.FromOutgoingContext(ctx)
	if ok {
		md = md.Copy()
	} else {
		md = metadata.MD{}
	}
	md.Set(requestIDKey, id)
	return metadata.NewOutgoingContext(ctx, md)
}

func waitReady(conn *googlegrpc.ClientConn, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	conn.Connect()
	for {
		state := conn.GetState()
		if state == connectivity.Ready {
			return nil
		}
		if !conn.WaitForStateChange(ctx, state) {
			return fmt.Errorf("grpc: connection to %s was not ready within %s (last state: %s)", conn.Target(), timeout, state)
		}
	}
}
