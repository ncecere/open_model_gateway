package usagepipeline

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/ncecere/open_model_gateway/backend/internal/config"
)

// WebhookSink delivers alerts to arbitrary HTTP endpoints.
type WebhookSink struct {
	client     *http.Client
	secret     string
	maxRetries int
	logger     *slog.Logger
}

func NewWebhookSink(cfg config.WebhookConfig, logger *slog.Logger) AlertSink {
	if logger == nil {
		logger = slog.Default()
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 5 * time.Second
	}
	if cfg.MaxRetries <= 0 {
		cfg.MaxRetries = 1
	}
	return &WebhookSink{
		client:     &http.Client{Timeout: cfg.Timeout},
		secret:     strings.TrimSpace(cfg.Secret),
		maxRetries: cfg.MaxRetries,
		logger:     logger,
	}
}

func (s *WebhookSink) Notify(ctx context.Context, payload AlertPayload) error {
	if s == nil {
		return nil
	}
	urls := payload.Channels.Webhooks
	if len(urls) == 0 {
		return nil
	}

	body, err := json.Marshal(webhookPayload{
		TenantID:       payload.TenantID.String(),
		Level:          string(payload.Level),
		LimitCents:     payload.Status.LimitCents,
		TotalCostCents: payload.Status.TotalCostCents,
		Warning:        payload.Status.Warning,
		Exceeded:       payload.Status.Exceeded,
		APIKeyPrefix:   payload.APIKeyPrefix,
		ModelAlias:     payload.ModelAlias,
		Timestamp:      payload.Timestamp.UTC(),
	})
	if err != nil {
		return err
	}

	var errs []error
	for _, target := range urls {
		if strings.TrimSpace(target) == "" {
			continue
		}
		if err := s.postWithRetries(ctx, target, body); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", target, err))
		}
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

func (s *WebhookSink) postWithRetries(ctx context.Context, url string, body []byte) error {
	var lastErr error
	for attempt := 1; attempt <= s.maxRetries; attempt++ {
		start := time.Now()
		statusCode, err := s.post(ctx, url, body)
		elapsed := time.Since(start)
		if err == nil {
			s.logger.Info("budget webhook delivered",
				slog.String("url", url),
				slog.Int("status", statusCode),
				slog.Duration("latency", elapsed),
				slog.Int("attempt", attempt),
			)
			return nil
		}
		lastErr = err
		s.logger.Warn("budget webhook delivery failed",
			slog.String("url", url),
			slog.Int("status", statusCode),
			slog.String("error", err.Error()),
			slog.Duration("latency", elapsed),
			slog.Int("attempt", attempt),
			slog.Int("max_retries", s.maxRetries),
		)
		if attempt < s.maxRetries {
			base := time.Duration(attempt) * 250 * time.Millisecond
			jitter := time.Duration(rand.Int63n(int64(base) / 4))
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(base + jitter):
			}
		}
	}
	return lastErr
}

func (s *WebhookSink) post(ctx context.Context, url string, body []byte) (int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "open-model-gateway")
	if sig := s.signPayload(body); sig != "" {
		req.Header.Set("X-OMG-Signature", sig)
		req.Header.Set("X-OMG-Signature-Version", "v1")
		req.Header.Set("X-OMG-Timestamp", time.Now().UTC().Format(time.RFC3339))
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return resp.StatusCode, fmt.Errorf("status %d", resp.StatusCode)
	}
	return resp.StatusCode, nil
}

func (s *WebhookSink) signPayload(payload []byte) string {
	if s.secret == "" {
		return ""
	}
	mac := hmac.New(sha256.New, []byte(s.secret))
	mac.Write(payload)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

type webhookPayload struct {
	TenantID       string    `json:"tenant_id"`
	Level          string    `json:"level"`
	LimitCents     int64     `json:"limit_cents"`
	TotalCostCents int64     `json:"total_cost_cents"`
	Warning        bool      `json:"warning"`
	Exceeded       bool      `json:"exceeded"`
	APIKeyPrefix   string    `json:"api_key_prefix"`
	ModelAlias     string    `json:"model_alias"`
	Timestamp      time.Time `json:"timestamp"`
}
