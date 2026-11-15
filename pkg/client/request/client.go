package request

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/haunbo/go-pilot/pkg/xlog"
)

// Error is returned for network failures and non-2xx responses.
type Error struct {
	StatusCode int
	Body       []byte
	Cause      error
}

// Error implements error.
func (e *Error) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("request failed: %v", e.Cause)
	}
	body := string(e.Body)
	if len(body) > 512 {
		body = body[:512] + "..."
	}
	return fmt.Sprintf("request failed with status %d: %s", e.StatusCode, body)
}

// Unwrap returns the underlying network error.
func (e *Error) Unwrap() error { return e.Cause }

// IsStatus reports whether err is a request Error with the given status.
func IsStatus(err error, status int) bool {
	var reqErr *Error
	return errors.As(err, &reqErr) && reqErr.StatusCode == status
}

// Response is a fully read HTTP response.
type Response struct {
	StatusCode int
	Header     http.Header
	Body       []byte
	Duration   time.Duration
}

// IsSuccess reports whether the status is 2xx.
func (r *Response) IsSuccess() bool {
	return r != nil && r.StatusCode >= 200 && r.StatusCode < 300
}

// Decode unmarshals the response body into v. *[]byte and *string are
// supported directly.
func (r *Response) Decode(v any) error {
	if v == nil || len(r.Body) == 0 {
		return nil
	}
	switch target := v.(type) {
	case *[]byte:
		*target = append((*target)[:0], r.Body...)
		return nil
	case *string:
		*target = string(r.Body)
		return nil
	default:
		return json.Unmarshal(r.Body, v)
	}
}

// Client sends HTTP requests.
type Client struct {
	cfg  *Config
	base *url.URL
	http *http.Client
}

func newClient(cfg *Config, base *url.URL) *Client {
	return &Client{
		cfg:  cfg,
		base: base,
		http: &http.Client{Timeout: cfg.Timeout},
	}
}

// Config returns the client config.
func (c *Client) Config() *Config { return c.cfg }

// Request builds a request.
func (c *Client) Request(method, path string) *Request {
	return &Request{client: c, method: method, path: path, headers: map[string]string{}}
}

// Get sends a GET request.
func (c *Client) Get(ctx context.Context, path string, out any) (*Response, error) {
	return c.Request(http.MethodGet, path).Do(ctx, out)
}

// Post sends a POST request with a JSON body.
func (c *Client) Post(ctx context.Context, path string, body, out any) (*Response, error) {
	return c.Request(http.MethodPost, path).SetBody(body).Do(ctx, out)
}

// Put sends a PUT request with a JSON body.
func (c *Client) Put(ctx context.Context, path string, body, out any) (*Response, error) {
	return c.Request(http.MethodPut, path).SetBody(body).Do(ctx, out)
}

// Patch sends a PATCH request with a JSON body.
func (c *Client) Patch(ctx context.Context, path string, body, out any) (*Response, error) {
	return c.Request(http.MethodPatch, path).SetBody(body).Do(ctx, out)
}

// Delete sends a DELETE request.
func (c *Client) Delete(ctx context.Context, path string, out any) (*Response, error) {
	return c.Request(http.MethodDelete, path).Do(ctx, out)
}

// Request describes one HTTP call.
type Request struct {
	client  *Client
	method  string
	path    string
	body    any
	headers map[string]string
	query   url.Values
}

// SetHeader adds a request header.
func (r *Request) SetHeader(key, value string) *Request {
	r.headers[key] = value
	return r
}

// SetQuery adds a query parameter.
func (r *Request) SetQuery(key, value string) *Request {
	if r.query == nil {
		r.query = url.Values{}
	}
	r.query.Set(key, value)
	return r
}

// SetBody sets the request body. Structs and maps are JSON encoded; []byte,
// string and io.Reader are sent as-is.
func (r *Request) SetBody(body any) *Request {
	r.body = body
	return r
}

type payload struct {
	data        []byte
	contentType string
}

func encodePayload(body any) (*payload, error) {
	if body == nil {
		return nil, nil
	}
	switch value := body.(type) {
	case []byte:
		return &payload{data: value}, nil
	case string:
		return &payload{data: []byte(value)}, nil
	case io.Reader:
		data, err := io.ReadAll(value)
		if err != nil {
			return nil, err
		}
		return &payload{data: data}, nil
	default:
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("request: encode body: %w", err)
		}
		return &payload{data: data, contentType: "application/json"}, nil
	}
}

// Do executes the request and decodes a 2xx response into out.
func (r *Request) Do(ctx context.Context, out any) (*Response, error) {
	if r.method == "" {
		return nil, errors.New("request: empty method")
	}
	target, err := r.buildURL()
	if err != nil {
		return nil, err
	}
	payload, err := encodePayload(r.body)
	if err != nil {
		return nil, err
	}

	attempts := max(r.client.cfg.RetryCount+1, 1)
	var lastErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		resp, err := r.execute(ctx, target, payload)
		if err != nil {
			lastErr = err
			if attempt < attempts && r.retryableMethod() && ctx.Err() == nil {
				if sleepErr := sleep(ctx, r.backoff(attempt)); sleepErr != nil {
					return nil, sleepErr
				}
				continue
			}
			return nil, &Error{Cause: err}
		}

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			if err := resp.Decode(out); err != nil {
				return resp, fmt.Errorf("request: decode response: %w", err)
			}
			return resp, nil
		}
		lastErr = &Error{StatusCode: resp.StatusCode, Body: resp.Body}
		if attempt < attempts && r.retryableStatus(resp.StatusCode) && r.retryableMethod() {
			if sleepErr := sleep(ctx, r.retryAfter(resp, attempt)); sleepErr != nil {
				return resp, sleepErr
			}
			continue
		}
		return resp, lastErr
	}
	return nil, lastErr
}

func (r *Request) execute(ctx context.Context, target *url.URL, payload *payload) (*Response, error) {
	var reader io.Reader
	if payload != nil {
		reader = bytes.NewReader(payload.data)
	}
	req, err := http.NewRequestWithContext(ctx, r.method, target.String(), reader)
	if err != nil {
		return nil, err
	}
	if payload != nil {
		req.ContentLength = int64(len(payload.data))
	}

	for key, value := range r.client.cfg.Headers {
		req.Header.Set(key, value)
	}
	for key, value := range r.headers {
		req.Header.Set(key, value)
	}
	if payload != nil && payload.contentType != "" && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", payload.contentType)
	}
	r.applyAuth(req)

	if r.client.cfg.RequestIDHeader != "" && req.Header.Get(r.client.cfg.RequestIDHeader) == "" {
		if id := xlog.RequestIDFromContext(ctx); id != "" {
			req.Header.Set(r.client.cfg.RequestIDHeader, id)
		}
	}

	start := time.Now()
	resp, err := r.client.http.Do(req)
	duration := time.Since(start)
	if err != nil {
		r.client.log(r.method, target.String(), 0, duration, err)
		return nil, err
	}
	defer resp.Body.Close()

	limit := r.client.cfg.MaxResponseBytes
	body, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > limit {
		return nil, fmt.Errorf("request: response exceeds %d bytes", limit)
	}
	response := &Response{
		StatusCode: resp.StatusCode,
		Header:     resp.Header.Clone(),
		Body:       body,
		Duration:   duration,
	}
	r.client.log(r.method, target.String(), resp.StatusCode, duration, nil)
	return response, nil
}

func (r *Request) buildURL() (*url.URL, error) {
	parsed, err := url.Parse(r.path)
	if err != nil {
		return nil, fmt.Errorf("request: parse path: %w", err)
	}
	if parsed.IsAbs() {
		return mergeQuery(parsed, r.query), nil
	}
	if r.client.base == nil {
		return nil, errors.New("request: base url is required for relative paths")
	}
	return mergeQuery(r.client.base.ResolveReference(parsed), r.query), nil
}

func mergeQuery(target *url.URL, query url.Values) *url.URL {
	if len(query) == 0 {
		return target
	}
	merged := target.Query()
	for key, values := range query {
		for _, value := range values {
			merged.Add(key, value)
		}
	}
	target.RawQuery = merged.Encode()
	return target
}

func (r *Request) applyAuth(req *http.Request) {
	auth := r.client.cfg.Auth
	if req.Header.Get("Authorization") != "" {
		return
	}
	switch strings.ToLower(auth.Type) {
	case "bearer":
		if auth.Token != "" {
			req.Header.Set("Authorization", "Bearer "+auth.Token)
		}
	case "basic":
		if auth.Username != "" || auth.Password != "" {
			req.SetBasicAuth(auth.Username, auth.Password)
		}
	case "apikey":
		header := auth.Header
		if header == "" {
			header = "X-API-Key"
		}
		if auth.QueryParam != "" {
			query := req.URL.Query()
			query.Set(auth.QueryParam, auth.Token)
			req.URL.RawQuery = query.Encode()
			return
		}
		if auth.Token != "" {
			req.Header.Set(header, auth.Token)
		}
	}
}

func (r *Request) retryableMethod() bool {
	if r.client.cfg.RetryNonIdempotent {
		return true
	}
	switch strings.ToUpper(r.method) {
	case http.MethodGet, http.MethodHead, http.MethodPut, http.MethodDelete, http.MethodOptions:
		return true
	default:
		return false
	}
}

func (r *Request) retryableStatus(status int) bool {
	return status == http.StatusTooManyRequests || status >= http.StatusInternalServerError
}

func (r *Request) retryAfter(resp *Response, attempt int) time.Duration {
	if resp.StatusCode == http.StatusTooManyRequests {
		if seconds, err := strconv.Atoi(strings.TrimSpace(resp.Header.Get("Retry-After"))); err == nil && seconds > 0 {
			return time.Duration(seconds) * time.Second
		}
	}
	return r.backoff(attempt)
}

func (r *Request) backoff(attempt int) time.Duration {
	delay := r.client.cfg.RetryWaitTime
	for i := 1; i < attempt; i++ {
		delay *= 2
		if delay >= r.client.cfg.MaxRetryWaitTime {
			delay = r.client.cfg.MaxRetryWaitTime
			break
		}
	}
	if delay <= 0 {
		return 0
	}
	jitterMax := int64(delay / 2)
	if jitterMax <= 0 {
		return delay
	}
	return delay/2 + time.Duration(rand.Int64N(jitterMax+1))
}

func (c *Client) log(method, target string, status int, duration time.Duration, err error) {
	fields := []xlog.Field{
		xlog.String("method", method),
		xlog.String("url", target),
		xlog.Int("status", status),
		xlog.Int64("duration_ms", duration.Milliseconds()),
	}
	if err != nil {
		fields = append(fields, xlog.FieldErr(err))
	}
	if c.cfg.Debug {
		c.cfg.logger.Info("http request", fields...)
	}
	if c.cfg.SlowThreshold > 0 && duration > c.cfg.SlowThreshold {
		c.cfg.logger.Warn("http request slow", fields...)
	}
}

func sleep(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return nil
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
