package provider

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/ncecere/open_model_gateway/backend/internal/config"
	"github.com/ncecere/open_model_gateway/backend/internal/email"
)

// CompositeAlertSink fans out to multiple sinks.
type CompositeAlertSink struct {
	sinks []AlertSink
}

// NewCompositeAlertSink builds a fan-out sink.
func NewCompositeAlertSink(sinks ...AlertSink) *CompositeAlertSink {
	return &CompositeAlertSink{sinks: sinks}
}

// Notify fan-outs to all sinks; errors are logged but do not stop the loop.
func (c *CompositeAlertSink) Notify(ctx context.Context, inc Incident) error {
	var lastErr error
	for _, s := range c.sinks {
		if s == nil {
			continue
		}
		if err := s.Notify(ctx, inc); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

// LogAlertSink writes incidents to slog.
type LogAlertSink struct {
	logger *slog.Logger
}

func NewLogAlertSink(logger *slog.Logger) *LogAlertSink {
	return &LogAlertSink{logger: logger}
}

func (l *LogAlertSink) Notify(ctx context.Context, inc Incident) error {
	log := l.logger
	if log == nil {
		log = slog.Default()
	}
	log.InfoContext(ctx, "provider incident",
		slog.String("provider", inc.Provider),
		slog.String("alias", inc.ModelAlias),
		slog.String("type", inc.Type),
		slog.String("status", string(inc.Status)),
		slog.Int64("requests", inc.RequestCount),
		slog.Any("metadata", inc.Metadata),
	)
	return nil
}

// NoopAlertSink ignores notifications.
type NoopAlertSink struct{}

func (n NoopAlertSink) Notify(ctx context.Context, inc Incident) error { return nil }

// FuncAlertSink allows functions to act as sinks.
type FuncAlertSink func(ctx context.Context, inc Incident) error

func (f FuncAlertSink) Notify(ctx context.Context, inc Incident) error {
	if f == nil {
		return fmt.Errorf("nil alert func")
	}
	return f(ctx, inc)
}

// EmailAlertSink reuses SMTP config to deliver provider alerts.
type EmailAlertSink struct {
	sender  email.Sender
	from    string
	baseURL string
	logger  *slog.Logger
	to      []string
}

func NewEmailAlertSink(cfg config.SMTPConfig, baseURL string, recipients []string, logger *slog.Logger) AlertSink {
	sender := email.NewSMTPSender(cfg)
	if sender == nil {
		return nil
	}
	if logger == nil {
		logger = slog.Default()
	}
	if len(recipients) == 0 {
		recipients = []string{cfg.From}
	}
	return &EmailAlertSink{sender: sender, from: cfg.From, baseURL: strings.TrimSpace(baseURL), logger: logger, to: recipients}
}

func (s *EmailAlertSink) Notify(ctx context.Context, inc Incident) error {
	if s == nil || s.sender == nil {
		return nil
	}
	if len(s.to) == 0 {
		return nil
	}
	subject := fmt.Sprintf("[Provider Alert] %s/%s %s", inc.Provider, inc.ModelAlias, inc.Type)
	htmlBody, _ := s.renderHTML(inc)
	body := s.formatText(inc)
	msg := email.Message{
		From:     s.from,
		To:       s.to,
		Subject:  subject,
		Body:     body,
		HTMLBody: htmlBody,
	}
	if err := s.sender.Send(ctx, msg); err != nil && s.logger != nil {
		s.logger.Error("send provider alert email", "error", err)
		return err
	}
	return nil
}

func (s *EmailAlertSink) renderHTML(inc Incident) (string, error) {
	data := email.ProviderAlertTemplateData{
		Provider:          inc.Provider,
		Alias:             inc.ModelAlias,
		IncidentType:      inc.Type,
		IncidentTypeLabel: strings.ToUpper(strings.ReplaceAll(inc.Type, "_", " ")),
		Timestamp:         inc.OpenedAt.Format("Jan 02, 2006 15:04 MST"),
		RequestCount:      fmt.Sprintf("%d", inc.RequestCount),
		WindowLabel:       fmt.Sprintf("%ds", inc.WindowSeconds),
		ObservedValue:     s.observedValue(inc),
		ThresholdValue:    s.thresholdValue(inc),
		SampleError:       inc.SampleError,
		ManageLink:        s.manageLink(),
	}
	return email.RenderProviderAlertTemplate(data)
}

func (s *EmailAlertSink) formatText(inc Incident) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Provider: %s\n", inc.Provider)
	fmt.Fprintf(&b, "Alias: %s\n", inc.ModelAlias)
	fmt.Fprintf(&b, "Incident: %s\n", inc.Type)
	fmt.Fprintf(&b, "Opened: %s\n", inc.OpenedAt.Format(time.RFC3339))
	fmt.Fprintf(&b, "Requests: %d\n", inc.RequestCount)
	if inc.SampleError != "" {
		fmt.Fprintf(&b, "Sample error: %s\n", inc.SampleError)
	}
	fmt.Fprintf(&b, "Observed: %s\n", s.observedValue(inc))
	fmt.Fprintf(&b, "Threshold: %s\n", s.thresholdValue(inc))
	return b.String()
}

func (s *EmailAlertSink) observedValue(inc Incident) string {
	switch inc.Type {
	case "latency_p95":
		return fmt.Sprintf("%d ms", inc.LatencyP95Ms)
	case "error_rate":
		if inc.RequestCount == 0 {
			return "0%"
		}
		return fmt.Sprintf("%.2f%%", float64(inc.ErrorCount)/float64(inc.RequestCount)*100)
	case "timeout_rate":
		if inc.RequestCount == 0 {
			return "0%"
		}
		return fmt.Sprintf("%.2f%%", float64(inc.TimeoutCount)/float64(inc.RequestCount)*100)
	default:
		return "n/a"
	}
}

func (s *EmailAlertSink) thresholdValue(inc Incident) string {
	if inc.Metadata == nil {
		return ""
	}
	if v, ok := inc.Metadata["threshold_ms"]; ok {
		return fmt.Sprintf("%v ms", v)
	}
	if v, ok := inc.Metadata["threshold"]; ok {
		return fmt.Sprintf("%v", v)
	}
	return ""
}

func (s *EmailAlertSink) manageLink() string {
	base := strings.TrimSpace(s.baseURL)
	if base == "" {
		return "#"
	}
	return strings.TrimRight(base, "/") + "/admin/ui"
}

// WebhookAlertSink delivers provider incidents via JSON webhook.
type WebhookAlertSink struct {
	client     *http.Client
	secret     string
	maxRetries int
	urls       []string
	logger     *slog.Logger
}

func NewWebhookAlertSink(cfg config.WebhookConfig, urls []string, logger *slog.Logger) AlertSink {
	if len(urls) == 0 {
		return nil
	}
	if logger == nil {
		logger = slog.Default()
	}
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	maxRetries := cfg.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 3
	}
	return &WebhookAlertSink{
		client:     &http.Client{Timeout: timeout},
		secret:     strings.TrimSpace(cfg.Secret),
		maxRetries: maxRetries,
		urls:       urls,
		logger:     logger,
	}
}

func (w *WebhookAlertSink) Notify(ctx context.Context, inc Incident) error {
	if w == nil || w.client == nil || len(w.urls) == 0 {
		return nil
	}
	payload := map[string]any{
		"provider":       inc.Provider,
		"alias":          inc.ModelAlias,
		"incident_type":  inc.Type,
		"status":         inc.Status,
		"opened_at":      inc.OpenedAt,
		"window_seconds": inc.WindowSeconds,
		"request_count":  inc.RequestCount,
		"error_count":    inc.ErrorCount,
		"timeout_count":  inc.TimeoutCount,
		"latency_p95_ms": inc.LatencyP95Ms,
		"metadata":       inc.Metadata,
		"sample_error":   inc.SampleError,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	var lastErr error
	for _, u := range w.urls {
		if err := w.postWithRetries(ctx, strings.TrimSpace(u), body); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

func (w *WebhookAlertSink) postWithRetries(ctx context.Context, url string, body []byte) error {
	var lastErr error
	for attempt := 1; attempt <= w.maxRetries; attempt++ {
		start := time.Now()
		statusCode, err := w.post(ctx, url, body)
		elapsed := time.Since(start)
		if err == nil {
			w.logger.Info("provider webhook delivered",
				slog.String("url", url),
				slog.Int("status", statusCode),
				slog.Duration("latency", elapsed),
				slog.Int("attempt", attempt),
			)
			return nil
		}
		lastErr = err
		w.logger.Warn("provider webhook delivery failed",
			slog.String("url", url),
			slog.Int("status", statusCode),
			slog.String("error", err.Error()),
			slog.Duration("latency", elapsed),
			slog.Int("attempt", attempt),
			slog.Int("max_retries", w.maxRetries),
		)
		if attempt < w.maxRetries {
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

func (w *WebhookAlertSink) post(ctx context.Context, url string, body []byte) (int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "open-model-gateway")
	if sig := w.signPayload(body); sig != "" {
		req.Header.Set("X-OMG-Signature", sig)
		req.Header.Set("X-OMG-Signature-Version", "v1")
		req.Header.Set("X-OMG-Timestamp", time.Now().UTC().Format(time.RFC3339))
	}
	resp, err := w.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return resp.StatusCode, fmt.Errorf("status %d", resp.StatusCode)
	}
	return resp.StatusCode, nil
}

func (w *WebhookAlertSink) signPayload(payload []byte) string {
	if w.secret == "" {
		return ""
	}
	mac := hmac.New(sha256.New, []byte(w.secret))
	mac.Write(payload)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}
