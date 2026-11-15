package xerr

import (
	"errors"
	"net/http"
	"testing"
)

func TestFrom(t *testing.T) {
	appErr := NotFound("user not found")
	if got := From(appErr); got != appErr {
		t.Fatal("From should return the same *Error")
	}
	got := From(errors.New("db down"))
	if got.Code != CodeInternal || got.Status != http.StatusInternalServerError {
		t.Fatalf("unexpected error: %+v", got)
	}
	if got.Error() != "internal server error: db down" {
		t.Fatalf("Error() = %q", got.Error())
	}
}

func TestWrap(t *testing.T) {
	cause := errors.New("redis timeout")
	err := Wrap(cause, CodeInternal, http.StatusInternalServerError, "request failed")
	if !errors.Is(err, cause) {
		t.Fatal("wrapped error should unwrap to cause")
	}
	if err.Error() != "request failed: redis timeout" {
		t.Fatalf("Error() = %q", err.Error())
	}
}
