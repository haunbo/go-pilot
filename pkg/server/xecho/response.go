package xecho

import (
	"net/http"

	"github.com/haunbo/go-pilot/pkg/xerr"
	"github.com/labstack/echo/v4"
)

// Response is the unified API envelope.
type Response struct {
	Code      int    `json:"code"`
	Message   string `json:"message"`
	Data      any    `json:"data,omitempty"`
	RequestID string `json:"requestId,omitempty"`
}

// Success writes a 200 response with business code 0.
func Success(c echo.Context, data any) error {
	return c.JSON(http.StatusOK, Response{
		Code:      xerr.CodeOK,
		Message:   "ok",
		Data:      data,
		RequestID: RequestID(c),
	})
}

// Fail writes an error response derived from err.
func Fail(c echo.Context, err error) error {
	appErr := xerr.From(err)
	if appErr == nil {
		return Success(c, nil)
	}
	return c.JSON(appErr.Status, Response{
		Code:      appErr.Code,
		Message:   appErr.Message,
		RequestID: RequestID(c),
	})
}

// Write writes a response envelope with an explicit status and business code.
func Write(c echo.Context, status, code int, message string, data any) error {
	return c.JSON(status, Response{
		Code:      code,
		Message:   message,
		Data:      data,
		RequestID: RequestID(c),
	})
}
