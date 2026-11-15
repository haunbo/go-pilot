package redis

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/haunbo/go-pilot/pkg/xlog"
	goredis "github.com/redis/go-redis/v9"
)

func xlogDefault() *xlog.Logger {
	return xlog.With(xlog.String("mod", "client.redis"))
}

type accessHook struct {
	name   string
	addr   string
	logger *xlog.Logger
}

func newAccessHook(name, addr string, logger *xlog.Logger) *accessHook {
	return &accessHook{name: name, addr: addr, logger: logger}
}

func (h *accessHook) DialHook(next goredis.DialHook) goredis.DialHook { return next }

func (h *accessHook) ProcessHook(next goredis.ProcessHook) goredis.ProcessHook {
	return func(ctx context.Context, cmd goredis.Cmder) error {
		start := time.Now()
		err := next(ctx, cmd)
		h.log(ctx, cmd, time.Since(start), err)
		return err
	}
}

func (h *accessHook) ProcessPipelineHook(next goredis.ProcessPipelineHook) goredis.ProcessPipelineHook {
	return func(ctx context.Context, cmds []goredis.Cmder) error {
		start := time.Now()
		err := next(ctx, cmds)
		cost := time.Since(start)
		for _, cmd := range cmds {
			h.log(ctx, cmd, cost, cmd.Err())
		}
		return err
	}
}

func (h *accessHook) log(ctx context.Context, cmd goredis.Cmder, cost time.Duration, err error) {
	extra := map[string]any{
		"key":  h.name,
		"addr": h.addr,
		"name": cmd.Name(),
		"req":  formatArgs(cmd),
		"cost": cost.Milliseconds(),
	}
	if err != nil {
		extra["error"] = err.Error()
		if errors.Is(err, goredis.Nil) {
			h.logger.Warn("redis access", xlog.FieldExtra(extra))
			return
		}
		h.logger.Error("redis access", xlog.FieldExtra(extra))
		return
	}
	h.logger.Info("redis access", xlog.FieldExtra(extra))
}

func formatArgs(cmd goredis.Cmder) string {
	args := cmd.Args()
	if len(args) > 0 {
		if name, ok := args[0].(string); ok {
			switch strings.ToLower(name) {
			case "auth", "hello", "acl":
				return name + " ******"
			}
		}
	}
	return fmt.Sprint(args)
}

type debugHook struct {
	name   string
	addr   string
	logger *xlog.Logger
}

func newDebugHook(name, addr string, logger *xlog.Logger) *debugHook {
	return &debugHook{name: name, addr: addr, logger: logger}
}

func (h *debugHook) DialHook(next goredis.DialHook) goredis.DialHook { return next }

func (h *debugHook) ProcessHook(next goredis.ProcessHook) goredis.ProcessHook {
	return func(ctx context.Context, cmd goredis.Cmder) error {
		start := time.Now()
		err := next(ctx, cmd)
		h.log(ctx, cmd, time.Since(start), err)
		return err
	}
}

func (h *debugHook) ProcessPipelineHook(next goredis.ProcessPipelineHook) goredis.ProcessPipelineHook {
	return func(ctx context.Context, cmds []goredis.Cmder) error {
		start := time.Now()
		err := next(ctx, cmds)
		cost := time.Since(start)
		for _, cmd := range cmds {
			h.log(ctx, cmd, cost, cmd.Err())
		}
		return err
	}
}

func (h *debugHook) log(_ context.Context, cmd goredis.Cmder, cost time.Duration, err error) {
	fields := []xlog.Field{
		xlog.String("key", h.name),
		xlog.String("addr", h.addr),
		xlog.String("cmd", cmd.Name()),
		xlog.String("req", formatArgs(cmd)),
		xlog.Int64("cost", cost.Milliseconds()),
	}
	if err != nil {
		fields = append(fields, xlog.FieldErr(err))
	}
	h.logger.Debug("redis debug", fields...)
}
