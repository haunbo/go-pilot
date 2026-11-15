package xgrpc

import (
	"context"
	"fmt"
	"runtime/debug"
	"time"

	"github.com/haunbo/go-pilot/pkg/xlog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

const requestIDKey = "x-request-id"

func accessUnaryServerInterceptor(logger *xlog.Logger, cfg *Config) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		start := time.Now()
		ctx = enrichContext(ctx)
		defer func() {
			if rec := recover(); rec != nil {
				err = status.Error(codes.Internal, "internal server error")
				logger.Error("grpc panic",
					xlog.String("method", info.FullMethod),
					xlog.String("panic", fmt.Sprint(rec)),
					xlog.String("stack", string(debug.Stack())),
				)
			}
			logServerAccess(logger, cfg, ctx, info.FullMethod, start, err)
		}()
		return handler(ctx, req)
	}
}

func accessStreamServerInterceptor(logger *xlog.Logger, cfg *Config) grpc.StreamServerInterceptor {
	return func(srv any, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
		start := time.Now()
		wrapped := &wrappedServerStream{ServerStream: stream, ctx: enrichContext(stream.Context())}
		defer func() {
			if rec := recover(); rec != nil {
				err = status.Error(codes.Internal, "internal server error")
				logger.Error("grpc panic",
					xlog.String("method", info.FullMethod),
					xlog.String("panic", fmt.Sprint(rec)),
					xlog.String("stack", string(debug.Stack())),
				)
			}
			logServerAccess(logger, cfg, wrapped.ctx, info.FullMethod, start, err)
		}()
		return handler(srv, wrapped)
	}
}

func logServerAccess(logger *xlog.Logger, cfg *Config, ctx context.Context, method string, start time.Time, err error) {
	duration := time.Since(start)
	code := status.Code(err)
	fields := []xlog.Field{
		xlog.String("method", method),
		xlog.String("code", code.String()),
		xlog.Int64("duration_ms", duration.Milliseconds()),
	}
	if id := xlog.RequestIDFromContext(ctx); id != "" {
		fields = append(fields, xlog.String("request_id", id))
	}
	if p, ok := peer.FromContext(ctx); ok && p.Addr != nil {
		fields = append(fields, xlog.String("peer", p.Addr.String()))
	}
	if cfg.SlowQueryThresholdInMilli > 0 && duration.Milliseconds() > cfg.SlowQueryThresholdInMilli {
		fields = append(fields, xlog.Bool("slow", true))
	}
	if err != nil {
		fields = append(fields, xlog.FieldErr(err))
		logger.Warn("grpc access", fields...)
		return
	}
	if cfg.EnableAccessLog {
		logger.Info("grpc access", fields...)
	}
}

func enrichContext(ctx context.Context) context.Context {
	if id := xlog.RequestIDFromContext(ctx); id != "" {
		return ctx
	}
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ctx
	}
	for _, key := range []string{requestIDKey, "request-id"} {
		if values := md.Get(key); len(values) > 0 && values[0] != "" {
			return xlog.WithRequestID(ctx, values[0])
		}
	}
	return ctx
}

type wrappedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrappedServerStream) Context() context.Context { return w.ctx }
