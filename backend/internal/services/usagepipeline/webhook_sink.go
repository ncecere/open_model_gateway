package usagepipeline

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/ncecere/open_model_gateway/backend/internal/config"
)

// WebhookSink delivers alerts to arbitrary HTTP endpoints.
type WebhookSink struct {
	client     *http.Client
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
		if err := s.post(ctx, url, body); err != nil {
			lastErr = err
			delay := time.Duration(attempt) * 250 * time.Millisecond
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
			continue
		}
		return nil
	}
	return lastErr
}

func (s *WebhookSink) post(ctx context.Context, url string, body []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("status %d", resp.StatusCode)
	}
	return nil
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
