package usagepipeline

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/ncecere/open_model_gateway/backend/internal/config"
)

func TestWebhookSinkNotify(t *testing.T) {
	var received webhookPayload
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	sink := NewWebhookSink(config.WebhookConfig{Timeout: time.Second, MaxRetries: 1}, nil)
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
	if received.TenantID != payload.TenantID.String() {
		t.Fatalf("tenant mismatch")
	}
	if received.Level != string(payload.Level) {
		t.Fatalf("level mismatch")
	}
}
