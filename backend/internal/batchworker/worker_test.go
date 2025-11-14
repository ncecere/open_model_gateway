package batchworker

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"

	batchsvc "github.com/ncecere/open_model_gateway/backend/internal/services/batches"
)

func TestFileTTLCoversNilAndExpired(t *testing.T) {
	var batch batchsvc.Batch
	if ttl := fileTTL(batch); ttl != 0 {
		t.Fatalf("expected zero ttl for nil expires, got %s", ttl)
	}

	expired := time.Now().Add(-time.Hour)
	batch.ExpiresAt = &expired
	if ttl := fileTTL(batch); ttl != 0 {
		t.Fatalf("expected zero ttl for expired batch, got %s", ttl)
	}

	future := time.Now().Add(time.Hour)
	batch.ExpiresAt = &future
	ttl := fileTTL(batch)
	if ttl <= 0 || ttl > time.Hour+time.Second {
		t.Fatalf("unexpected ttl value: %s", ttl)
	}
}

func TestEncodeErrorPayloadFormatsOpenAIShape(t *testing.T) {
	payload := encodeErrorPayload("test_code", "boom")
	var body map[string]openAIError
	if err := json.Unmarshal(payload, &body); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	errObj := body["error"]
	if errObj.Type != "batch_error" {
		t.Fatalf("unexpected error type %s", errObj.Type)
	}
	if errObj.Code != "test_code" || errObj.Message != "boom" {
		t.Fatalf("unexpected payload %+v", errObj)
	}
}

func TestMapStatusToCode(t *testing.T) {
	cases := map[int]string{
		fiber.StatusBadRequest:          "invalid_request_error",
		fiber.StatusForbidden:           "permission_error",
		fiber.StatusTooManyRequests:     "rate_limit_error",
		fiber.StatusServiceUnavailable:  "service_unavailable",
		fiber.StatusInternalServerError: "provider_error",
	}
	for status, want := range cases {
		if got := mapStatusToCode(status); got != want {
			t.Fatalf("status %d: want %s got %s", status, want, got)
		}
	}
}
