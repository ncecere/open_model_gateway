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
	sender  email.Sender
	from    string
	baseURL string
	logger  *slog.Logger
}

func NewEmailSink(cfg config.SMTPConfig, baseURL string, logger *slog.Logger) AlertSink {
	sender := email.NewSMTPSender(cfg)
	if sender == nil {
		return nil
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &EmailSink{sender: sender, from: cfg.From, baseURL: strings.TrimSpace(baseURL), logger: logger}
}

func (s *EmailSink) Notify(ctx context.Context, payload AlertPayload) error {
	if s == nil || s.sender == nil {
		return nil
	}
	recipients := payload.Channels.Emails
	if len(recipients) == 0 {
		return nil
	}

	subject := fmt.Sprintf("[Budget %s] Tenant %s", strings.ToUpper(string(payload.Level)), s.tenantLabel(payload))
	body := s.formatBody(payload)
	htmlBody, err := s.renderHTML(payload)
	if err != nil && s.logger != nil {
		s.logger.Error("render budget alert email", "error", err)
	}
	msg := email.Message{
		From:     s.from,
		To:       recipients,
		Subject:  subject,
		Body:     body,
		HTMLBody: htmlBody,
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
	label := payload.TenantName
	if strings.TrimSpace(label) == "" {
		label = payload.TenantID.String()
	}
	fmt.Fprintf(&b, "Tenant: %s\n", label)
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

func (s *EmailSink) renderHTML(payload AlertPayload) (string, error) {
	data := email.BudgetAlertTemplateData{
		TenantName:       s.tenantLabel(payload),
		LevelLabel:       levelLabel(payload.Level),
		LevelClass:       levelClass(payload.Level),
		Timestamp:        payload.Timestamp.In(time.UTC).Format("Jan 02, 2006 15:04 MST"),
		CurrentSpend:     formatCurrency(payload.Status.TotalCostCents),
		BudgetLimit:      formatCurrency(payload.Status.LimitCents),
		WarningThreshold: formatPercentage(payload.WarningThreshold),
		BudgetReset:      s.formatBudgetReset(payload.BudgetReset),
		ManageLink:       s.manageLink(),
	}
	return email.RenderBudgetAlertTemplate(data)
}

func (s *EmailSink) tenantLabel(payload AlertPayload) string {
	if strings.TrimSpace(payload.TenantName) != "" {
		return payload.TenantName
	}
	return payload.TenantID.String()
}

func (s *EmailSink) manageLink() string {
	base := strings.TrimSpace(s.baseURL)
	if base == "" {
		return "#"
	}
	return strings.TrimRight(base, "/") + "/admin/ui"
}

func (s *EmailSink) formatBudgetReset(ts time.Time) string {
	if ts.IsZero() {
		return "—"
	}
	return ts.In(time.UTC).Format("Jan 02, 2006")
}

func levelClass(level AlertLevel) string {
	switch level {
	case AlertLevelExceeded:
		return "exceeded"
	case AlertLevelWarning:
		return "warning"
	default:
		return ""
	}
}

func levelLabel(level AlertLevel) string {
	switch level {
	case AlertLevelExceeded:
		return "Exceeded"
	case AlertLevelWarning:
		return "Warning"
	default:
		return "Alert"
	}
}

func formatPercentage(val float64) string {
	if val <= 0 {
		return "—"
	}
	return fmt.Sprintf("%.0f%%", val*100)
}

func formatCurrency(cents int64) string {
	return fmt.Sprintf("$%.2f", float64(cents)/100)
}
