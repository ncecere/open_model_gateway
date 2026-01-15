// Package base provides common HTTP patterns for provider adapters.
// It includes utilities for making HTTP requests, handling errors,
// and implementing retry logic.
package base

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/ncecere/open_model_gateway/backend/internal/apperror"
)

const (
	// DefaultTimeout is the default HTTP client timeout.
	DefaultTimeout = 60 * time.Second

	// DefaultRetryCount is the default number of retry attempts.
	DefaultRetryCount = 3

	// DefaultRetryDelay is the default delay between retries.
	DefaultRetryDelay = 100 * time.Millisecond
)

// Config holds configuration for the base adapter.
type Config struct {
	// BaseURL is the base URL for API requests (without trailing slash).
	BaseURL string

	// APIKey is the API key for authentication.
	APIKey string

	// Timeout is the HTTP client timeout.
	Timeout time.Duration

	// RetryCount is the number of retry attempts for transient failures.
	RetryCount int

	// RetryDelay is the delay between retry attempts.
	RetryDelay time.Duration

	// AuthHeader is the header name for authentication (default: "Authorization").
	AuthHeader string

	// AuthScheme is the auth scheme (default: "Bearer").
	AuthScheme string

	// ExtraHeaders are additional headers to include in all requests.
	ExtraHeaders map[string]string

	// HTTPClient is an optional custom HTTP client.
	HTTPClient *http.Client
}

// Adapter provides common HTTP functionality for provider adapters.
type Adapter struct {
	client       *http.Client
	baseURL      string
	apiKey       string
	authHeader   string
	authScheme   string
	extraHeaders map[string]string
	retryCount   int
	retryDelay   time.Duration
}

// New creates a new base adapter with the given configuration.
func New(cfg Config) *Adapter {
	if cfg.Timeout == 0 {
		cfg.Timeout = DefaultTimeout
	}
	if cfg.RetryCount == 0 {
		cfg.RetryCount = DefaultRetryCount
	}
	if cfg.RetryDelay == 0 {
		cfg.RetryDelay = DefaultRetryDelay
	}
	if cfg.AuthHeader == "" {
		cfg.AuthHeader = "Authorization"
		// Only default to Bearer scheme when using default Authorization header
		if cfg.AuthScheme == "" {
			cfg.AuthScheme = "Bearer"
		}
	}

	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: cfg.Timeout}
	}

	return &Adapter{
		client:       client,
		baseURL:      strings.TrimRight(cfg.BaseURL, "/"),
		apiKey:       strings.TrimSpace(cfg.APIKey),
		authHeader:   cfg.AuthHeader,
		authScheme:   cfg.AuthScheme,
		extraHeaders: cfg.ExtraHeaders,
		retryCount:   cfg.RetryCount,
		retryDelay:   cfg.RetryDelay,
	}
}

// Request represents an HTTP request to be sent.
type Request struct {
	Method      string
	Path        string
	Body        interface{}
	Headers     map[string]string
	Accept      string
	ContentType string
}

// DoJSON sends a JSON request and decodes the response into dest.
func (a *Adapter) DoJSON(ctx context.Context, req Request, dest interface{}) error {
	var body io.Reader
	if req.Body != nil {
		data, err := json.Marshal(req.Body)
		if err != nil {
			return apperror.WrapWithMessage("base.DoJSON", err, "failed to marshal request body")
		}
		body = bytes.NewReader(data)
	}

	if req.ContentType == "" {
		req.ContentType = "application/json"
	}
	if req.Accept == "" {
		req.Accept = "application/json"
	}

	resp, err := a.Send(ctx, req.Method, req.Path, body, req.Accept, req.ContentType, req.Headers)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		return a.DecodeError(resp)
	}

	if dest == nil {
		return nil
	}

	if err := json.NewDecoder(resp.Body).Decode(dest); err != nil {
		return apperror.WrapWithMessage("base.DoJSON", err, "failed to decode response")
	}

	return nil
}

// Send sends an HTTP request and returns the response.
func (a *Adapter) Send(ctx context.Context, method, path string, body io.Reader, accept, contentType string, headers map[string]string) (*http.Response, error) {
	req, err := a.NewRequest(ctx, method, path, body)
	if err != nil {
		return nil, err
	}

	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if accept != "" {
		req.Header.Set("Accept", accept)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	return a.Do(req)
}

// NewRequest creates a new HTTP request with authentication headers.
func (a *Adapter) NewRequest(ctx context.Context, method, path string, body io.Reader) (*http.Request, error) {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	endpoint := a.baseURL + path

	req, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return nil, apperror.WrapWithMessage("base.NewRequest", err, "failed to create request")
	}

	if a.apiKey != "" {
		if a.authScheme != "" {
			req.Header.Set(a.authHeader, a.authScheme+" "+a.apiKey)
		} else {
			req.Header.Set(a.authHeader, a.apiKey)
		}
	}

	for k, v := range a.extraHeaders {
		req.Header.Set(k, v)
	}

	return req, nil
}

// Do executes an HTTP request with optional retries for transient failures.
func (a *Adapter) Do(req *http.Request) (*http.Response, error) {
	var resp *http.Response
	var err error

	if req.Body != nil && req.GetBody == nil {
		bodyBytes, readErr := io.ReadAll(req.Body)
		if readErr != nil {
			return nil, apperror.WrapWithMessage("base.Do", readErr, "failed to read request body for retries")
		}
		_ = req.Body.Close()
		req.GetBody = func() (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader(bodyBytes)), nil
		}
		req.Body, _ = req.GetBody()
	}

	for attempt := 0; attempt <= a.retryCount; attempt++ {
		if attempt > 0 && req.GetBody != nil {
			req.Body, err = req.GetBody()
			if err != nil {
				return nil, apperror.WrapWithMessage("base.Do", err, "failed to reset request body for retry")
			}
		}

		resp, err = a.client.Do(req)
		if err != nil {
			// Network errors are retryable
			if attempt < a.retryCount {
				time.Sleep(a.retryDelay * time.Duration(attempt+1))
				continue
			}
			return nil, apperror.WrapWithMessage("base.Do", err, "request failed after retries")
		}

		// Check for retryable status codes
		if resp.StatusCode == http.StatusTooManyRequests ||
			resp.StatusCode == http.StatusServiceUnavailable ||
			resp.StatusCode == http.StatusGatewayTimeout {
			resp.Body.Close()
			if attempt < a.retryCount {
				time.Sleep(a.retryDelay * time.Duration(attempt+1))
				continue
			}
		}

		break
	}

	return resp, nil
}

// DecodeError reads and parses an error response.
func (a *Adapter) DecodeError(resp *http.Response) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return mapStatusToError(resp.StatusCode, "failed to read error response")
	}

	// Try to parse as JSON error
	var apiErr APIError
	if err := json.Unmarshal(body, &apiErr); err == nil && apiErr.Error.Message != "" {
		return mapStatusToError(resp.StatusCode, apiErr.Error.Message)
	}

	// Fall back to raw body
	if len(body) > 0 {
		return mapStatusToError(resp.StatusCode, string(body))
	}

	return mapStatusToError(resp.StatusCode, http.StatusText(resp.StatusCode))
}

// APIError represents a standard API error response.
type APIError struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error"`
}

// mapStatusToError maps HTTP status codes to apperror types.
func mapStatusToError(status int, message string) error {
	op := "api"
	switch status {
	case http.StatusBadRequest:
		return apperror.BadRequest(op, message)
	case http.StatusUnauthorized:
		return apperror.Unauthorized(op, message)
	case http.StatusForbidden:
		return apperror.Forbidden(op, message)
	case http.StatusNotFound:
		return apperror.NotFound(op, message)
	case http.StatusConflict:
		return apperror.Conflict(op, message)
	case http.StatusTooManyRequests:
		return apperror.RateLimited(op, message)
	case http.StatusServiceUnavailable, http.StatusBadGateway, http.StatusGatewayTimeout:
		return apperror.ServiceUnavailable(op, message)
	default:
		return apperror.Internal(op, fmt.Sprintf("status %d: %s", status, message))
	}
}

// Client returns the underlying HTTP client.
func (a *Adapter) Client() *http.Client {
	return a.client
}

// BaseURL returns the base URL.
func (a *Adapter) BaseURL() string {
	return a.baseURL
}
