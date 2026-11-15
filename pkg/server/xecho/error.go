package xecho

import (
	"errors"
	"net/http"

	"github.com/haunbo/go-pilot/pkg/xerr"
	"github.com/labstack/echo/v4"
)

func (s *Server) handleError(err error, c echo.Context) {
	if err == nil || c.Response().Committed {
		return
	}

	var httpErr *echo.HTTPError
	if errors.As(err, &httpErr) {
		status := httpErr.Code
		message := http.StatusText(status)
		if msg, ok := httpErr.Message.(string); ok && msg != "" {
			message = msg
		}
		_ = Write(c, status, status, message, nil)
		return
	}

	appErr := xerr.From(err)
	_ = Write(c, appErr.Status, appErr.Code, appErr.Message, nil)
}
