package metadata

import (
	"testing"
	"time"
)

func TestFromMap(t *testing.T) {
	m := map[string]string{
		"deployment":             "my-deployment",
		"region":                 "us-east-1",
		"tokenizer":              "gpt-4",
		"requests_per_minute":    "1000",
		"tokens_per_minute":      "100000",
		"parallel_requests":      "50",
		"retry_max_attempts":     "3",
		"audio_voice":            "alloy",
		"audio_formats":          "mp3,wav,ogg",
		"audio_streaming":        "true",
		"api_version":            "2024-01-01",
		"bedrock_chat_format":    "messages",
		"aws_access_key_id":      "AKID",
		"vertex_edit_mode":       "inpaint",
		"groq_region":            "us-west",
		"base_url":               "http://localhost:8000",
		"custom_field":           "custom_value",
	}

	r := FromMap(m)

	// Core metadata
	if r.Core.Deployment != "my-deployment" {
		t.Errorf("expected deployment 'my-deployment', got %q", r.Core.Deployment)
	}
	if r.Core.Region != "us-east-1" {
		t.Errorf("expected region 'us-east-1', got %q", r.Core.Region)
	}
	if r.Core.Tokenizer != "gpt-4" {
		t.Errorf("expected tokenizer 'gpt-4', got %q", r.Core.Tokenizer)
	}

	// Rate limits
	if r.RateLimits.RequestsPerMinute == nil || *r.RateLimits.RequestsPerMinute != 1000 {
		t.Errorf("expected requests_per_minute 1000, got %v", r.RateLimits.RequestsPerMinute)
	}
	if r.RateLimits.TokensPerMinute == nil || *r.RateLimits.TokensPerMinute != 100000 {
		t.Errorf("expected tokens_per_minute 100000, got %v", r.RateLimits.TokensPerMinute)
	}
	if r.RateLimits.ParallelRequests == nil || *r.RateLimits.ParallelRequests != 50 {
		t.Errorf("expected parallel_requests 50, got %v", r.RateLimits.ParallelRequests)
	}

	// Retry
	if r.Retry.MaxAttempts == nil || *r.Retry.MaxAttempts != 3 {
		t.Errorf("expected max_attempts 3, got %v", r.Retry.MaxAttempts)
	}

	// Audio
	if r.Audio.Voice != "alloy" {
		t.Errorf("expected voice 'alloy', got %q", r.Audio.Voice)
	}
	if len(r.Audio.Formats) != 3 {
		t.Errorf("expected 3 formats, got %d", len(r.Audio.Formats))
	}
	if r.Audio.StreamingEnabled == nil || !*r.Audio.StreamingEnabled {
		t.Errorf("expected streaming enabled true")
	}

	// Provider-specific
	if r.Provider.Azure.APIVersion != "2024-01-01" {
		t.Errorf("expected api_version '2024-01-01', got %q", r.Provider.Azure.APIVersion)
	}
	if r.Provider.AWS.ChatFormat != "messages" {
		t.Errorf("expected chat_format 'messages', got %q", r.Provider.AWS.ChatFormat)
	}
	if r.Provider.AWS.AccessKeyID != "AKID" {
		t.Errorf("expected access_key_id 'AKID', got %q", r.Provider.AWS.AccessKeyID)
	}
	if r.Provider.Vertex.EditMode != "inpaint" {
		t.Errorf("expected edit_mode 'inpaint', got %q", r.Provider.Vertex.EditMode)
	}
	if r.Provider.Groq.Region != "us-west" {
		t.Errorf("expected groq_region 'us-west', got %q", r.Provider.Groq.Region)
	}
	if r.Provider.VLLM.BaseURL != "http://localhost:8000" {
		t.Errorf("expected base_url 'http://localhost:8000', got %q", r.Provider.VLLM.BaseURL)
	}

	// Raw access for custom fields
	if r.Get("custom_field") != "custom_value" {
		t.Errorf("expected custom_field 'custom_value', got %q", r.Get("custom_field"))
	}
}

func TestToMap(t *testing.T) {
	rpm := int32(1000)
	attempts := 3
	streaming := true
	backoff := 100 * time.Millisecond

	r := Route{
		Core: CoreMetadata{
			Deployment: "my-deployment",
			Region:     "us-east-1",
		},
		RateLimits: RateLimitMetadata{
			RequestsPerMinute: &rpm,
		},
		Retry: RetryMetadata{
			MaxAttempts:    &attempts,
			InitialBackoff: &backoff,
		},
		Audio: AudioMetadata{
			Voice:            "nova",
			Formats:          []string{"mp3", "wav"},
			StreamingEnabled: &streaming,
		},
		Provider: ProviderMetadata{
			Azure: AzureMetadata{
				APIVersion: "2024-02-01",
			},
		},
		raw: map[string]string{
			"custom_field": "preserved",
		},
	}

	m := r.ToMap()

	if m["deployment"] != "my-deployment" {
		t.Errorf("expected deployment 'my-deployment', got %q", m["deployment"])
	}
	if m["region"] != "us-east-1" {
		t.Errorf("expected region 'us-east-1', got %q", m["region"])
	}
	if m["requests_per_minute"] != "1000" {
		t.Errorf("expected requests_per_minute '1000', got %q", m["requests_per_minute"])
	}
	if m["retry_max_attempts"] != "3" {
		t.Errorf("expected retry_max_attempts '3', got %q", m["retry_max_attempts"])
	}
	if m["retry_initial_backoff_ms"] != "100" {
		t.Errorf("expected retry_initial_backoff_ms '100', got %q", m["retry_initial_backoff_ms"])
	}
	if m["audio_voice"] != "nova" {
		t.Errorf("expected audio_voice 'nova', got %q", m["audio_voice"])
	}
	if m["audio_formats"] != "mp3,wav" {
		t.Errorf("expected audio_formats 'mp3,wav', got %q", m["audio_formats"])
	}
	if m["audio_streaming"] != "true" {
		t.Errorf("expected audio_streaming 'true', got %q", m["audio_streaming"])
	}
	if m["api_version"] != "2024-02-01" {
		t.Errorf("expected api_version '2024-02-01', got %q", m["api_version"])
	}
	if m["custom_field"] != "preserved" {
		t.Errorf("expected custom_field 'preserved', got %q", m["custom_field"])
	}
}

func TestFromMapNil(t *testing.T) {
	r := FromMap(nil)
	if r.Core.Deployment != "" {
		t.Errorf("expected empty deployment, got %q", r.Core.Deployment)
	}
	if r.Get("nonexistent") != "" {
		t.Errorf("expected empty for nonexistent key")
	}
}

func TestRoundTrip(t *testing.T) {
	original := map[string]string{
		"deployment":           "prod-deploy",
		"region":               "eu-west-1",
		"requests_per_minute":  "500",
		"audio_voice":          "shimmer",
		"bedrock_chat_format":  "titan",
		"api_version":          "2024-03-01",
		"custom_extension":     "extension_value",
	}

	r := FromMap(original)
	result := r.ToMap()

	for k, v := range original {
		if result[k] != v {
			t.Errorf("round-trip failed for %q: expected %q, got %q", k, v, result[k])
		}
	}
}

func TestBoolParsing(t *testing.T) {
	tests := []struct {
		input    string
		expected *bool
	}{
		{"true", boolPtr(true)},
		{"false", boolPtr(false)},
		{"yes", boolPtr(true)},
		{"no", boolPtr(false)},
		{"1", boolPtr(true)},
		{"0", boolPtr(false)},
		{"enabled", boolPtr(true)},
		{"disabled", boolPtr(false)},
		{"", nil},
		{"invalid", nil},
	}

	for _, tc := range tests {
		result := parseBoolPtr(tc.input)
		if tc.expected == nil {
			if result != nil {
				t.Errorf("parseBoolPtr(%q): expected nil, got %v", tc.input, *result)
			}
		} else if result == nil {
			t.Errorf("parseBoolPtr(%q): expected %v, got nil", tc.input, *tc.expected)
		} else if *result != *tc.expected {
			t.Errorf("parseBoolPtr(%q): expected %v, got %v", tc.input, *tc.expected, *result)
		}
	}
}

func boolPtr(v bool) *bool {
	return &v
}
