package groq

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ncecere/open_model_gateway/backend/internal/models"
	"github.com/ncecere/open_model_gateway/backend/internal/providers/fixtures"
)

func TestAdapterChat(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); !strings.HasPrefix(got, "Bearer test") {
			t.Fatalf("missing auth header: %s", got)
		}
		w.Header().Set("Content-Type", "application/json")
		data, err := fixtures.Read("groq_chat_completion.json")
		if err != nil {
			t.Fatalf("fixture: %v", err)
		}
		_, _ = w.Write(data)
	}
	adapter := newTestAdapter(t, handler)

	req := models.ChatRequest{
		Model: "llama-3.3-70b-versatile",
		Messages: []models.ChatMessage{
			{Role: "user", Content: "hi"},
		},
	}
	resp, err := adapter.Chat(context.Background(), req)
	if err != nil {
		t.Fatalf("chat: %v", err)
	}
	if resp.Usage.TotalTokens != 18 {
		t.Fatalf("expected usage total 18, got %d", resp.Usage.TotalTokens)
	}
	if len(resp.Choices) != 1 || resp.Choices[0].Message.Content != "hello from groq" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestAdapterChatStream(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		chunks := []string{
			`{"id":"chatcmpl-test","object":"chat.completion.chunk","model":"llama-3.3-70b-versatile","created":1730241104,"choices":[{"index":0,"delta":{"role":"assistant","content":"hello"},"finish_reason":null}]}`,
			`{"id":"chatcmpl-test","object":"chat.completion.chunk","model":"llama-3.3-70b-versatile","created":1730241104,"choices":[{"index":0,"delta":{"role":"assistant","content":" world"},"finish_reason":"stop"}],"usage":{"prompt_tokens":12,"completion_tokens":6,"total_tokens":18}}`,
			`[DONE]`,
		}
		for _, chunk := range chunks {
			_, _ = w.Write([]byte("data: " + chunk + "\n\n"))
		}
	}
	adapter := newTestAdapter(t, handler)

	stream, closeFn, err := adapter.ChatStream(context.Background(), models.ChatRequest{
		Model: "llama",
		Messages: []models.ChatMessage{
			{Role: "user", Content: "hello"},
		},
	})
	if err != nil {
		t.Fatalf("chat stream: %v", err)
	}
	defer closeFn()

	var builder strings.Builder
	for chunk := range stream {
		if len(chunk.Choices) > 0 {
			builder.WriteString(chunk.Choices[0].Delta.Content)
		}
		if chunk.Usage != nil {
			if chunk.Usage.TotalTokens != 18 {
				t.Fatalf("unexpected usage %+v", chunk.Usage)
			}
			break
		}
	}
	if builder.String() != "hello world" {
		t.Fatalf("unexpected content %q", builder.String())
	}
}

func TestAdapterHealthCheckError(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"message":"bad key","type":"invalid_api_key"}}`))
	}
	adapter := newTestAdapter(t, handler)
	err := adapter.HealthCheck(context.Background())
	if err == nil || !strings.Contains(err.Error(), "bad key") {
		t.Fatalf("expected auth error, got %v", err)
	}
}

func newTestAdapter(t *testing.T, handler http.HandlerFunc) *Adapter {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(handler))
	t.Cleanup(server.Close)
	adapter, err := New(Options{
		APIKey:  "test",
		BaseURL: server.URL,
		HTTPClient: &http.Client{
			Timeout: 2 * time.Second,
		},
	})
	if err != nil {
		t.Fatalf("new adapter: %v", err)
	}
	return adapter
}
