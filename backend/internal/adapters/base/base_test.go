package base

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ncecere/open_model_gateway/backend/internal/apperror"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
	}{
		{
			name: "default values",
			cfg: Config{
				BaseURL: "https://api.example.com",
				APIKey:  "test-key",
			},
		},
		{
			name: "custom values",
			cfg: Config{
				BaseURL:    "https://api.example.com/",
				APIKey:     "  test-key  ",
				Timeout:    30 * time.Second,
				RetryCount: 5,
				RetryDelay: 200 * time.Millisecond,
				AuthHeader: "X-API-Key",
				AuthScheme: "",
				ExtraHeaders: map[string]string{
					"X-Custom": "value",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := New(tt.cfg)
			if adapter == nil {
				t.Fatal("expected non-nil adapter")
			}
			if adapter.client == nil {
				t.Error("expected non-nil client")
			}
		})
	}
}

func TestAdapter_DoJSON(t *testing.T) {
	type testRequest struct {
		Name string `json:"name"`
	}
	type testResponse struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}

	t.Run("successful request", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("expected POST, got %s", r.Method)
			}
			if r.Header.Get("Authorization") != "Bearer test-key" {
				t.Errorf("expected Bearer auth, got %s", r.Header.Get("Authorization"))
			}
			if r.Header.Get("Content-Type") != "application/json" {
				t.Errorf("expected application/json content type, got %s", r.Header.Get("Content-Type"))
			}

			var req testRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Errorf("failed to decode request: %v", err)
			}
			if req.Name != "test" {
				t.Errorf("expected name=test, got %s", req.Name)
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(testResponse{ID: 1, Name: "test"})
		}))
		defer server.Close()

		adapter := New(Config{
			BaseURL: server.URL,
			APIKey:  "test-key",
		})

		var resp testResponse
		err := adapter.DoJSON(context.Background(), Request{
			Method: http.MethodPost,
			Path:   "/test",
			Body:   testRequest{Name: "test"},
		}, &resp)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.ID != 1 {
			t.Errorf("expected ID=1, got %d", resp.ID)
		}
		if resp.Name != "test" {
			t.Errorf("expected Name=test, got %s", resp.Name)
		}
	})

	t.Run("error response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(APIError{
				Error: struct {
					Message string `json:"message"`
					Type    string `json:"type"`
					Code    string `json:"code"`
				}{
					Message: "invalid request",
					Type:    "invalid_request_error",
				},
			})
		}))
		defer server.Close()

		adapter := New(Config{
			BaseURL: server.URL,
			APIKey:  "test-key",
		})

		var resp testResponse
		err := adapter.DoJSON(context.Background(), Request{
			Method: http.MethodPost,
			Path:   "/test",
			Body:   testRequest{Name: "test"},
		}, &resp)

		if err == nil {
			t.Fatal("expected error")
		}
		if !errors.Is(err, apperror.ErrBadRequest) {
			t.Errorf("expected ErrBadRequest, got %v", err)
		}
	})
}

func TestAdapter_NewRequest(t *testing.T) {
	t.Run("bearer auth", func(t *testing.T) {
		adapter := New(Config{
			BaseURL:    "https://api.example.com",
			APIKey:     "test-key",
			AuthHeader: "Authorization",
			AuthScheme: "Bearer",
		})

		req, err := adapter.NewRequest(context.Background(), http.MethodGet, "/test", nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if req.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("expected Bearer auth, got %s", req.Header.Get("Authorization"))
		}
	})

	t.Run("custom auth header", func(t *testing.T) {
		adapter := New(Config{
			BaseURL:    "https://api.example.com",
			APIKey:     "test-key",
			AuthHeader: "X-API-Key",
			AuthScheme: "",
		})

		req, err := adapter.NewRequest(context.Background(), http.MethodGet, "/test", nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if req.Header.Get("X-API-Key") != "test-key" {
			t.Errorf("expected X-API-Key=test-key, got %s", req.Header.Get("X-API-Key"))
		}
	})

	t.Run("extra headers", func(t *testing.T) {
		adapter := New(Config{
			BaseURL: "https://api.example.com",
			APIKey:  "test-key",
			ExtraHeaders: map[string]string{
				"X-Custom-Header": "custom-value",
			},
		})

		req, err := adapter.NewRequest(context.Background(), http.MethodGet, "/test", nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if req.Header.Get("X-Custom-Header") != "custom-value" {
			t.Errorf("expected custom header, got %s", req.Header.Get("X-Custom-Header"))
		}
	})

	t.Run("path normalization", func(t *testing.T) {
		adapter := New(Config{
			BaseURL: "https://api.example.com/",
			APIKey:  "test-key",
		})

		req, err := adapter.NewRequest(context.Background(), http.MethodGet, "test", nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if req.URL.String() != "https://api.example.com/test" {
			t.Errorf("expected normalized URL, got %s", req.URL.String())
		}
	})
}

func TestAdapter_DecodeError(t *testing.T) {
	makeAPIError := func(msg string) map[string]interface{} {
		return map[string]interface{}{
			"error": map[string]string{
				"message": msg,
				"type":    "error",
			},
		}
	}

	tests := []struct {
		name       string
		statusCode int
		body       interface{}
		wantErr    error
	}{
		{
			name:       "bad request",
			statusCode: http.StatusBadRequest,
			body:       makeAPIError("bad request"),
			wantErr:    apperror.ErrBadRequest,
		},
		{
			name:       "unauthorized",
			statusCode: http.StatusUnauthorized,
			body:       makeAPIError("unauthorized"),
			wantErr:    apperror.ErrUnauthorized,
		},
		{
			name:       "forbidden",
			statusCode: http.StatusForbidden,
			body:       makeAPIError("forbidden"),
			wantErr:    apperror.ErrForbidden,
		},
		{
			name:       "not found",
			statusCode: http.StatusNotFound,
			body:       makeAPIError("not found"),
			wantErr:    apperror.ErrNotFound,
		},
		{
			name:       "rate limited",
			statusCode: http.StatusTooManyRequests,
			body:       makeAPIError("rate limited"),
			wantErr:    apperror.ErrRateLimited,
		},
		{
			name:       "service unavailable",
			statusCode: http.StatusServiceUnavailable,
			body:       makeAPIError("service unavailable"),
			wantErr:    apperror.ErrServiceUnavailable,
		},
		{
			name:       "internal error",
			statusCode: http.StatusInternalServerError,
			body:       makeAPIError("internal error"),
			wantErr:    apperror.ErrInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				json.NewEncoder(w).Encode(tt.body)
			}))
			defer server.Close()

			adapter := New(Config{
				BaseURL: server.URL,
				APIKey:  "test-key",
			})

			resp, err := adapter.Send(context.Background(), http.MethodGet, "/test", nil, "", "", nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			defer resp.Body.Close()

			decodeErr := adapter.DecodeError(resp)
			if !errors.Is(decodeErr, tt.wantErr) {
				t.Errorf("expected %v, got %v", tt.wantErr, decodeErr)
			}
		})
	}
}

func TestDecodeAPIError(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		headers    map[string]string
		body       string
		wantErr    error
		wantMsg    string
	}{
		{
			name:       "429 with Retry-After",
			statusCode: http.StatusTooManyRequests,
			headers:    map[string]string{"Retry-After": "30"},
			body:       `{"error":{"message":"rate limited"}}`,
			wantErr:    apperror.ErrRateLimited,
		},
		{
			name:       "429 without Retry-After",
			statusCode: http.StatusTooManyRequests,
			body:       `{}`,
			wantErr:    apperror.ErrRateLimited,
		},
		{
			name:       "503 overloaded",
			statusCode: http.StatusServiceUnavailable,
			body:       `{}`,
			wantErr:    apperror.ErrServiceUnavailable,
		},
		{
			name:       "529 Anthropic overloaded",
			statusCode: 529,
			body:       `{"error":{"message":"overloaded"}}`,
			wantErr:    apperror.ErrServiceUnavailable,
		},
		{
			name:       "400 with JSON error body",
			statusCode: http.StatusBadRequest,
			body:       `{"error":{"message":"invalid model"}}`,
			wantErr:    apperror.ErrBadRequest,
			wantMsg:    "invalid model",
		},
		{
			name:       "400 with plain text body",
			statusCode: http.StatusBadRequest,
			body:       `bad request`,
			wantErr:    apperror.ErrBadRequest,
			wantMsg:    "bad request",
		},
		{
			name:       "401 unauthorized",
			statusCode: http.StatusUnauthorized,
			body:       `{"error":{"message":"invalid key"}}`,
			wantErr:    apperror.ErrUnauthorized,
		},
		{
			name:       "500 with empty body",
			statusCode: http.StatusInternalServerError,
			body:       ``,
			wantErr:    apperror.ErrInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				for k, v := range tt.headers {
					w.Header().Set(k, v)
				}
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.body))
			}))
			defer server.Close()

			resp, err := http.Get(server.URL)
			if err != nil {
				t.Fatalf("unexpected http error: %v", err)
			}

			decodeErr := DecodeAPIError("test-provider", resp)
			if !errors.Is(decodeErr, tt.wantErr) {
				t.Errorf("expected %v, got %v", tt.wantErr, decodeErr)
			}
			if tt.wantMsg != "" {
				var appErr *apperror.Error
				if errors.As(decodeErr, &appErr) {
					if appErr.Message != tt.wantMsg {
						t.Errorf("expected message %q, got %q", tt.wantMsg, appErr.Message)
					}
				}
			}
		})
	}
}

func TestAdapter_Retry(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()

	adapter := New(Config{
		BaseURL:    server.URL,
		APIKey:     "test-key",
		RetryCount: 3,
		RetryDelay: 10 * time.Millisecond,
	})

	var resp map[string]string
	err := adapter.DoJSON(context.Background(), Request{
		Method: http.MethodGet,
		Path:   "/test",
	}, &resp)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
	if resp["status"] != "ok" {
		t.Errorf("expected status=ok, got %s", resp["status"])
	}
}
