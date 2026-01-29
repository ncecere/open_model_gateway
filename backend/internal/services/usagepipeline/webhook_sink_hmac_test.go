package usagepipeline

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/ncecere/open_model_gateway/backend/internal/config"
)

func TestWebhookSink_HMACSignature(t *testing.T) {
	secret := "test-webhook-secret"
	var gotSig, gotVersion, gotTimestamp string
	var gotBody []byte

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotSig = r.Header.Get("X-OMG-Signature")
		gotVersion = r.Header.Get("X-OMG-Signature-Version")
		gotTimestamp = r.Header.Get("X-OMG-Timestamp")
		gotBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	sink := NewWebhookSink(config.WebhookConfig{
		Timeout:    time.Second,
		MaxRetries: 1,
		Secret:     secret,
	}, nil)

	payload := AlertPayload{
		TenantID:  uuid.New(),
		Level:     AlertLevelWarning,
		Status:    BudgetStatus{LimitCents: 10000, TotalCostCents: 8000, Warning: true},
		Channels:  AlertChannels{Webhooks: []string{ts.URL}},
		Timestamp: time.Now(),
	}

	if err := sink.Notify(context.Background(), payload); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify HMAC
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(gotBody)
	expectedSig := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	if gotSig != expectedSig {
		t.Fatalf("signature mismatch:\n  got:  %s\n  want: %s", gotSig, expectedSig)
	}
	if gotVersion != "v1" {
		t.Fatalf("version: expected v1, got %q", gotVersion)
	}
	if gotTimestamp == "" {
		t.Fatal("expected non-empty timestamp header")
	}
}

func TestWebhookSink_NoSignatureWithoutSecret(t *testing.T) {
	var gotSig string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotSig = r.Header.Get("X-OMG-Signature")
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	sink := NewWebhookSink(config.WebhookConfig{
		Timeout:    time.Second,
		MaxRetries: 1,
	}, nil)

	payload := AlertPayload{
		TenantID: uuid.New(),
		Level:    AlertLevelWarning,
		Status:   BudgetStatus{LimitCents: 100},
		Channels: AlertChannels{Webhooks: []string{ts.URL}},
	}

	if err := sink.Notify(context.Background(), payload); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotSig != "" {
		t.Fatalf("expected no signature, got %q", gotSig)
	}
}

func TestWebhookSink_RetriesOnFailure(t *testing.T) {
	var attempts int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&attempts, 1)
		if count < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	sink := NewWebhookSink(config.WebhookConfig{
		Timeout:    time.Second,
		MaxRetries: 3,
	}, nil)

	payload := AlertPayload{
		TenantID: uuid.New(),
		Level:    AlertLevelExceeded,
		Status:   BudgetStatus{LimitCents: 100, Exceeded: true},
		Channels: AlertChannels{Webhooks: []string{ts.URL}},
	}

	if err := sink.Notify(context.Background(), payload); err != nil {
		t.Fatalf("unexpected error after retries: %v", err)
	}
	if got := atomic.LoadInt32(&attempts); got != 3 {
		t.Fatalf("expected 3 attempts, got %d", got)
	}
}

func TestWebhookSink_PayloadFields(t *testing.T) {
	var received webhookPayload
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&received)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	tenantID := uuid.New()
	payload := AlertPayload{
		TenantID:     tenantID,
		Level:        AlertLevelExceeded,
		Status:       BudgetStatus{LimitCents: 5000, TotalCostCents: 5100, Exceeded: true},
		Channels:     AlertChannels{Webhooks: []string{ts.URL}},
		APIKeyPrefix: "sk-test",
		ModelAlias:   "gpt-4",
		Timestamp:    time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC),
	}

	sink := NewWebhookSink(config.WebhookConfig{Timeout: time.Second, MaxRetries: 1}, nil)
	sink.Notify(context.Background(), payload)

	if received.TenantID != tenantID.String() {
		t.Fatalf("tenant mismatch")
	}
	if received.Level != "exceeded" {
		t.Fatalf("level: expected exceeded, got %q", received.Level)
	}
	if !received.Exceeded {
		t.Fatal("expected exceeded=true")
	}
	if received.APIKeyPrefix != "sk-test" {
		t.Fatalf("api key prefix mismatch: %q", received.APIKeyPrefix)
	}
	if received.ModelAlias != "gpt-4" {
		t.Fatalf("model alias mismatch: %q", received.ModelAlias)
	}
}
