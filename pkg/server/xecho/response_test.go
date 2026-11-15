package xecho

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/haunbo/go-pilot/pkg/xerr"
	"github.com/labstack/echo/v4"
)

func TestSuccessEnvelopeAndRequestID(t *testing.T) {
	e := echo.New()
	e.Use(requestIDMiddleware())
	e.GET("/", func(c echo.Context) error {
		return Success(c, map[string]string{"hello": "world"})
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(headerRequestID, "req-123")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if rec.Header().Get(headerRequestID) != "req-123" {
		t.Fatalf("response request id = %q", rec.Header().Get(headerRequestID))
	}
	var body Response
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Code != 0 || body.Message != "ok" || body.RequestID != "req-123" {
		t.Fatalf("body = %+v", body)
	}
	data, ok := body.Data.(map[string]any)
	if !ok || data["hello"] != "world" {
		t.Fatalf("data = %#v", body.Data)
	}
}

func TestFailEnvelope(t *testing.T) {
	e := echo.New()
	e.Use(requestIDMiddleware())
	e.GET("/", func(c echo.Context) error {
		return Fail(c, xerr.NotFound("missing"))
	})
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d", rec.Code)
	}
	var body Response
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Code != xerr.CodeNotFound || body.Message != "missing" {
		t.Fatalf("body = %+v", body)
	}
	if body.RequestID == "" {
		t.Fatal("request id should be generated")
	}
}

func TestValidator(t *testing.T) {
	type payload struct {
		Name   string `json:"name" validate:"required" label:"名称"`
		Status int8   `json:"status" validate:"oneof=0 1" label:"状态"`
	}
	v := NewValidator("zh")
	if err := v.Validate(&payload{Name: "demo", Status: 1}); err != nil {
		t.Fatalf("valid payload rejected: %v", err)
	}
	if err := v.Validate(&payload{Status: 1}); err == nil || !strings.Contains(err.Error(), "名称") {
		t.Fatalf("missing name error = %v", err)
	}
	if err := v.Validate(&payload{Name: "demo", Status: 2}); err == nil || !strings.Contains(err.Error(), "状态") {
		t.Fatalf("invalid status error = %v", err)
	}
}

func TestErrorStatus(t *testing.T) {
	if got := errorStatus(xerr.Forbidden("no")); got != http.StatusForbidden {
		t.Fatalf("status = %d", got)
	}
	if got := errorStatus(echo.NewHTTPError(http.StatusTeapot)); got != http.StatusTeapot {
		t.Fatalf("status = %d", got)
	}
}
