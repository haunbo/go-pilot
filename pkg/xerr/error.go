// Package xerr defines business errors with an HTTP status and a stable code.
package xerr

import (
	"errors"
	"fmt"
	"net/http"
)

// Error is an application error safe to return to API clients.
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Status  int    `json:"-"`
	cause   error
}

// Error implements error.
func (e *Error) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.cause)
	}
	return e.Message
}

// Unwrap returns the wrapped cause.
func (e *Error) Unwrap() error { return e.cause }

// HTTPStatus returns the HTTP status code.
func (e *Error) HTTPStatus() int { return e.Status }

// New creates an Error.
func New(code, status int, message string) *Error {
	return &Error{Code: code, Status: status, Message: message}
}

// Wrap creates an Error with a cause.
func Wrap(err error, code, status int, message string) *Error {
	return &Error{Code: code, Status: status, Message: message, cause: err}
}

// From converts any error to *Error. Unknown errors become a generic 500.
func From(err error) *Error {
	if err == nil {
		return nil
	}
	if appErr, ok := errors.AsType[*Error](err); ok {
		return appErr
	}
	return &Error{
		Code:    CodeInternal,
		Status:  http.StatusInternalServerError,
		Message: "internal server error",
		cause:   err,
	}
}

// BadRequest creates a 400 error.
func BadRequest(message string) *Error {
	return New(CodeInvalidParams, http.StatusBadRequest, message)
}

// InvalidParams creates a 400 error with the invalid parameters code.
func InvalidParams(message string) *Error {
	return New(CodeInvalidParams, http.StatusBadRequest, message)
}

// Unauthorized creates a 401 error.
func Unauthorized(message string) *Error {
	return New(CodeUnauthorized, http.StatusUnauthorized, message)
}

// Forbidden creates a 403 error.
func Forbidden(message string) *Error {
	return New(CodeForbidden, http.StatusForbidden, message)
}

// NotFound creates a 404 error.
func NotFound(message string) *Error {
	return New(CodeNotFound, http.StatusNotFound, message)
}

// Conflict creates a 409 error.
func Conflict(message string) *Error {
	return New(CodeConflict, http.StatusConflict, message)
}

// Internal creates a 500 error.
func Internal(message string) *Error {
	return New(CodeInternal, http.StatusInternalServerError, message)
}

// InternalWrap creates a 500 error with a cause.
func InternalWrap(err error, message string) *Error {
	return Wrap(err, CodeInternal, http.StatusInternalServerError, message)
}
