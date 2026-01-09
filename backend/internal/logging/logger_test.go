package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name   string
		cfg    Config
		format string
	}{
		{
			name:   "default config",
			cfg:    Config{},
			format: "text",
		},
		{
			name:   "json format",
			cfg:    Config{Format: "json", Level: "debug"},
			format: "json",
		},
		{
			name:   "text format with warn level",
			cfg:    Config{Format: "text", Level: "warn"},
			format: "text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			tt.cfg.Output = &buf

			logger := New(tt.cfg)
			if logger == nil {
				t.Fatal("expected non-nil logger")
			}

			// Use Warn level to ensure output for all log level configs
			logger.Warn("test message")

			output := buf.String()
			if output == "" {
				t.Error("expected log output")
			}
		})
	}
}

func TestDefault(t *testing.T) {
	logger := Default()
	if logger == nil {
		t.Fatal("expected non-nil logger")
	}
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"info", slog.LevelInfo},
		{"warn", slog.LevelWarn},
		{"warning", slog.LevelWarn},
		{"error", slog.LevelError},
		{"unknown", slog.LevelInfo},
		{"", slog.LevelInfo},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := parseLevel(tt.input); got != tt.expected {
				t.Errorf("parseLevel(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestContextLogger(t *testing.T) {
	var buf bytes.Buffer
	logger := New(Config{
		Level:  "debug",
		Format: "json",
		Output: &buf,
	})

	ctx := context.Background()
	ctx = WithContext(ctx, logger)

	tenantID := uuid.New()
	userID := uuid.New()
	apiKeyID := uuid.New()
	requestID := "req-12345"

	ctx = WithTenantID(ctx, tenantID)
	ctx = WithUserID(ctx, userID)
	ctx = WithAPIKeyID(ctx, apiKeyID)
	ctx = WithRequestID(ctx, requestID)

	ctxLogger := ContextLogger(ctx)
	ctxLogger.Info("test with context")

	output := buf.String()

	if !strings.Contains(output, tenantID.String()) {
		t.Error("expected tenant_id in output")
	}
	if !strings.Contains(output, userID.String()) {
		t.Error("expected user_id in output")
	}
	if !strings.Contains(output, apiKeyID.String()) {
		t.Error("expected api_key_id in output")
	}
	if !strings.Contains(output, requestID) {
		t.Error("expected request_id in output")
	}
}

func TestFromContext_Default(t *testing.T) {
	ctx := context.Background()
	logger := FromContext(ctx)
	if logger == nil {
		t.Error("expected non-nil logger from empty context")
	}
}

func TestLoggerWith(t *testing.T) {
	var buf bytes.Buffer
	logger := New(Config{
		Level:  "debug",
		Format: "json",
		Output: &buf,
	})

	childLogger := logger.With("key", "value")
	childLogger.Info("test message")

	output := buf.String()
	if !strings.Contains(output, `"key":"value"`) {
		t.Errorf("expected key=value in output, got: %s", output)
	}
}

func TestLoggerWithGroup(t *testing.T) {
	var buf bytes.Buffer
	logger := New(Config{
		Level:  "debug",
		Format: "json",
		Output: &buf,
	})

	groupLogger := logger.WithGroup("request")
	groupLogger.Info("test message", "path", "/api/test")

	output := buf.String()
	if !strings.Contains(output, "request") {
		t.Errorf("expected group in output, got: %s", output)
	}
}

func TestFieldHelpers(t *testing.T) {
	t.Run("TenantID", func(t *testing.T) {
		id := uuid.New()
		attr := TenantID(id)
		if attr.Key != "tenant_id" {
			t.Errorf("expected key=tenant_id, got %s", attr.Key)
		}
		if attr.Value.String() != id.String() {
			t.Errorf("expected value=%s, got %s", id.String(), attr.Value.String())
		}
	})

	t.Run("UserID", func(t *testing.T) {
		id := uuid.New()
		attr := UserID(id)
		if attr.Key != "user_id" {
			t.Errorf("expected key=user_id, got %s", attr.Key)
		}
	})

	t.Run("APIKeyID", func(t *testing.T) {
		id := uuid.New()
		attr := APIKeyID(id)
		if attr.Key != "api_key_id" {
			t.Errorf("expected key=api_key_id, got %s", attr.Key)
		}
	})

	t.Run("RequestID", func(t *testing.T) {
		attr := RequestID("req-123")
		if attr.Key != "request_id" {
			t.Errorf("expected key=request_id, got %s", attr.Key)
		}
		if attr.Value.String() != "req-123" {
			t.Errorf("expected value=req-123, got %s", attr.Value.String())
		}
	})

	t.Run("Model", func(t *testing.T) {
		attr := Model("gpt-4")
		if attr.Key != "model" {
			t.Errorf("expected key=model, got %s", attr.Key)
		}
	})

	t.Run("Provider", func(t *testing.T) {
		attr := Provider("openai")
		if attr.Key != "provider" {
			t.Errorf("expected key=provider, got %s", attr.Key)
		}
	})

	t.Run("Op", func(t *testing.T) {
		attr := Op("chat.create")
		if attr.Key != "op" {
			t.Errorf("expected key=op, got %s", attr.Key)
		}
	})

	t.Run("Error", func(t *testing.T) {
		err := errors.New("test error")
		attr := Error(err)
		if attr.Key != "error" {
			t.Errorf("expected key=error, got %s", attr.Key)
		}
		if attr.Value.String() != "test error" {
			t.Errorf("expected value=test error, got %s", attr.Value.String())
		}
	})

	t.Run("Error_nil", func(t *testing.T) {
		attr := Error(nil)
		if attr.Key != "" {
			t.Errorf("expected empty key for nil error, got %s", attr.Key)
		}
	})

	t.Run("DurationMs", func(t *testing.T) {
		attr := DurationMs(150)
		if attr.Key != "duration_ms" {
			t.Errorf("expected key=duration_ms, got %s", attr.Key)
		}
		if attr.Value.Int64() != 150 {
			t.Errorf("expected value=150, got %d", attr.Value.Int64())
		}
	})

	t.Run("StatusCode", func(t *testing.T) {
		attr := StatusCode(200)
		if attr.Key != "status_code" {
			t.Errorf("expected key=status_code, got %s", attr.Key)
		}
	})

	t.Run("TokensInput", func(t *testing.T) {
		attr := TokensInput(100)
		if attr.Key != "tokens_input" {
			t.Errorf("expected key=tokens_input, got %s", attr.Key)
		}
	})

	t.Run("TokensOutput", func(t *testing.T) {
		attr := TokensOutput(50)
		if attr.Key != "tokens_output" {
			t.Errorf("expected key=tokens_output, got %s", attr.Key)
		}
	})

	t.Run("Path", func(t *testing.T) {
		attr := Path("/v1/chat/completions")
		if attr.Key != "path" {
			t.Errorf("expected key=path, got %s", attr.Key)
		}
	})

	t.Run("Method", func(t *testing.T) {
		attr := Method("POST")
		if attr.Key != "method" {
			t.Errorf("expected key=method, got %s", attr.Key)
		}
	})
}

func TestJSONOutput(t *testing.T) {
	var buf bytes.Buffer
	logger := New(Config{
		Level:  "info",
		Format: "json",
		Output: &buf,
	})

	logger.Info("test message", "key", "value")

	var logEntry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
		t.Fatalf("failed to parse JSON log: %v", err)
	}

	if logEntry["msg"] != "test message" {
		t.Errorf("expected msg=test message, got %v", logEntry["msg"])
	}
	if logEntry["key"] != "value" {
		t.Errorf("expected key=value, got %v", logEntry["key"])
	}
	if logEntry["level"] != "INFO" {
		t.Errorf("expected level=INFO, got %v", logEntry["level"])
	}
}
