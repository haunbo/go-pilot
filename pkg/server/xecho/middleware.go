package xecho

import (
	"errors"
	"fmt"
	"net/http"
	"runtime"
	"slices"
	"strings"
	"time"

	"github.com/haunbo/go-pilot/pkg/util/xstring"
	"github.com/haunbo/go-pilot/pkg/xerr"
	"github.com/haunbo/go-pilot/pkg/xlog"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

const headerRequestID = "X-Request-ID"

func requestIDMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			id := c.Request().Header.Get(headerRequestID)
			if id == "" {
				id = xstring.GenerateUUID()
			}
			c.Set("request_id", id)
			c.Response().Header().Set(headerRequestID, id)
			c.SetRequest(c.Request().WithContext(xlog.WithRequestID(c.Request().Context(), id)))
			return next(c)
		}
	}
}

func recoverMiddleware(logger *xlog.Logger, slowQueryThresholdInMilli int64) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) (err error) {
			start := time.Now()
			defer func() {
				rec := recover()
				if rec != nil {
					switch v := rec.(type) {
					case error:
						err = v
					default:
						err = fmt.Errorf("%v", v)
					}
				}

				fields := []xlog.Field{
					xlog.String("method", c.Request().Method),
					xlog.String("path", c.Request().URL.Path),
					xlog.String("host", c.Request().Host),
					xlog.String("ip", c.RealIP()),
					xlog.Int64("latency", time.Since(start).Milliseconds()),
				}
				if id := RequestID(c); id != "" {
					fields = append(fields, xlog.String("request_id", id))
				}
				status := c.Response().Status
				if err != nil {
					status = errorStatus(err)
				}
				if rec != nil {
					status = http.StatusInternalServerError
				}
				extra := map[string]any{
					"status": status,
				}
				if rec != nil {
					extra["stack"] = goroutineStack()
				}
				if slowQueryThresholdInMilli > 0 {
					if cost := time.Since(start).Milliseconds(); cost > slowQueryThresholdInMilli {
						extra["slow"] = cost
					}
				}
				fields = append(fields, xlog.FieldExtra(extra))
				if err != nil {
					if status >= http.StatusInternalServerError {
						logger.Error(err.Error(), fields...)
					} else {
						logger.Warn(err.Error(), fields...)
					}
					return
				}
				if isHealthPath(c.Request().URL.Path) {
					return
				}
				logger.Info("access", fields...)
			}()

			err = next(c)
			return
		}
	}
}

func goroutineStack() string {
	buf := make([]byte, 4096)
	n := runtime.Stack(buf, true)
	return string(buf[:n])
}

func corsMiddleware(origins []string) echo.MiddlewareFunc {
	if len(origins) == 0 {
		origins = []string{"*"}
	}
	allowCredentials := !slices.Contains(origins, "*")
	return middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: origins,
		AllowHeaders: []string{
			echo.HeaderAccept,
			echo.HeaderContentType,
			echo.HeaderAuthorization,
			"X-Requested-With",
			"X-Request-ID",
			"X-CSRF-Token",
		},
		ExposeHeaders:    []string{"X-Request-ID", "Content-Length"},
		AllowMethods:     []string{http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions},
		AllowCredentials: allowCredentials,
		MaxAge:           86400,
	})
}

// isHealthPath reports whether path should be skipped by access logging.
func isHealthPath(path string) bool {
	return path == "/healthz" || path == "/readyz"
}

func errorStatus(err error) int {
	var httpErr *echo.HTTPError
	if errors.As(err, &httpErr) {
		return httpErr.Code
	}
	return xerr.From(err).Status
}

// RequestID returns the request id stored by the middleware.
func RequestID(c echo.Context) string {
	if c == nil {
		return ""
	}
	if id, ok := c.Get("request_id").(string); ok && id != "" {
		return id
	}
	if c.Request() != nil {
		if id := xlog.RequestIDFromContext(c.Request().Context()); id != "" {
			return id
		}
		return strings.TrimSpace(c.Request().Header.Get(headerRequestID))
	}
	return ""
}
