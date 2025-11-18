package usagepipeline

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/ncecere/open_model_gateway/backend/internal/config"
	"github.com/ncecere/open_model_gateway/backend/internal/email"
)

// EmailSink delivers alert payloads via the shared email sender.
type EmailSink struct {
	sender email.Sender
	from   string
	logger *slog.Logger
}

func NewEmailSink(cfg config.SMTPConfig, logger *slog.Logger) AlertSink {
	sender := email.NewSMTPSender(cfg)
	if sender == nil {
		return nil
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &EmailSink{sender: sender, from: cfg.From, logger: logger}
}

func (s *EmailSink) Notify(ctx context.Context, payload AlertPayload) error {
	if s == nil || s.sender == nil {
		return nil
	}
	recipients := payload.Channels.Emails
	if len(recipients) == 0 {
		return nil
	}

	subject := fmt.Sprintf("[Budget %s] Tenant %s", strings.ToUpper(string(payload.Level)), payload.TenantID)
	body := s.formatBody(payload)
	msg := email.Message{
		From:    s.from,
		To:      recipients,
		Subject: subject,
		Body:    body,
	}
	if err := s.sender.Send(ctx, msg); err != nil && s.logger != nil {
		s.logger.Error("send budget alert email", "error", err)
		return err
	}
	return nil
}

func (s *EmailSink) formatBody(payload AlertPayload) string {
	limit := formatCurrency(payload.Status.LimitCents)
	spend := formatCurrency(payload.Status.TotalCostCents)

	var b strings.Builder
	fmt.Fprintf(&b, "Tenant ID: %s\n", payload.TenantID)
	fmt.Fprintf(&b, "Level: %s\n", strings.ToUpper(string(payload.Level)))
	fmt.Fprintf(&b, "Spend: %s / %s\n", spend, limit)
	fmt.Fprintf(&b, "Exceeded: %t\n", payload.Status.Exceeded)
	fmt.Fprintf(&b, "Warning: %t\n", payload.Status.Warning)
	if payload.APIKeyPrefix != "" {
		fmt.Fprintf(&b, "API Key Prefix: %s\n", payload.APIKeyPrefix)
	}
	if payload.ModelAlias != "" {
		fmt.Fprintf(&b, "Model Alias: %s\n", payload.ModelAlias)
	}
	fmt.Fprintf(&b, "Timestamp: %s\n", payload.Timestamp.UTC().Format(time.RFC3339))
	return b.String()
}

func formatCurrency(cents int64) string {
	return fmt.Sprintf("$%.2f", float64(cents)/100)
}
