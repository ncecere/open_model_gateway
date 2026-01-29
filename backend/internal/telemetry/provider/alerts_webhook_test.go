package provider

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ncecere/open_model_gateway/backend/internal/config"
)

func TestWebhookAlertSink_HMACSignature(t *testing.T) {
	secret := "provider-webhook-secret"
	var gotSig, gotVersion string
	var gotBody []byte

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotSig = r.Header.Get("X-OMG-Signature")
		gotVersion = r.Header.Get("X-OMG-Signature-Version")
		gotBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	sink := NewWebhookAlertSink(
		config.WebhookConfig{Timeout: time.Second, MaxRetries: 1, Secret: secret},
		[]string{ts.URL},
		nil,
	)

	inc := Incident{
		Identifier: Identifier{Provider: "openai", ModelAlias: "gpt-4"},
		Type:       "error_rate",
		Status:     IncidentOpen,
		OpenedAt:   time.Now(),
	}

	if err := sink.Notify(context.Background(), inc); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(gotBody)
	expectedSig := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	if gotSig != expectedSig {
		t.Fatalf("signature mismatch:\n  got:  %s\n  want: %s", gotSig, expectedSig)
	}
	if gotVersion != "v1" {
		t.Fatalf("version: expected v1, got %q", gotVersion)
	}
}

func TestWebhookAlertSink_NoSignatureWithoutSecret(t *testing.T) {
	var gotSig string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotSig = r.Header.Get("X-OMG-Signature")
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	sink := NewWebhookAlertSink(
		config.WebhookConfig{Timeout: time.Second, MaxRetries: 1},
		[]string{ts.URL},
		nil,
	)

	inc := Incident{Identifier: Identifier{Provider: "test"}, Type: "latency_p95", Status: IncidentOpen}
	sink.Notify(context.Background(), inc)

	if gotSig != "" {
		t.Fatalf("expected no signature, got %q", gotSig)
	}
}

func TestWebhookAlertSink_RetriesOnFailure(t *testing.T) {
	var attempts int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&attempts, 1)
		if count < 2 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	sink := NewWebhookAlertSink(
		config.WebhookConfig{Timeout: time.Second, MaxRetries: 3},
		[]string{ts.URL},
		nil,
	)

	inc := Incident{Identifier: Identifier{Provider: "test"}, Type: "error_rate", Status: IncidentOpen}
	if err := sink.Notify(context.Background(), inc); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := atomic.LoadInt32(&attempts); got != 2 {
		t.Fatalf("expected 2 attempts, got %d", got)
	}
}

func TestWebhookAlertSink_NilForNoURLs(t *testing.T) {
	sink := NewWebhookAlertSink(config.WebhookConfig{}, nil, nil)
	if sink != nil {
		t.Fatal("expected nil sink for no URLs")
	}
}

func TestWebhookAlertSink_NilForEmptyURLs(t *testing.T) {
	sink := NewWebhookAlertSink(config.WebhookConfig{}, []string{}, nil)
	if sink != nil {
		t.Fatal("expected nil sink for empty URLs")
	}
}

func TestWebhookAlertSink_UserAgentHeader(t *testing.T) {
	var gotUA string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	sink := NewWebhookAlertSink(
		config.WebhookConfig{Timeout: time.Second, MaxRetries: 1},
		[]string{ts.URL},
		nil,
	)

	inc := Incident{Identifier: Identifier{Provider: "test"}, Type: "error_rate", Status: IncidentOpen}
	sink.Notify(context.Background(), inc)

	if gotUA != "open-model-gateway" {
		t.Fatalf("User-Agent: expected open-model-gateway, got %q", gotUA)
	}
}
