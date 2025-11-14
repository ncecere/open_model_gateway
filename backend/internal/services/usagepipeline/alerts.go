package usagepipeline

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

type AlertLevel string

const (
	AlertLevelNone     AlertLevel = "none"
	AlertLevelWarning  AlertLevel = "warning"
	AlertLevelExceeded AlertLevel = "exceeded"
)

type AlertChannels struct {
	Emails   []string
	Webhooks []string
}

type AlertPayload struct {
	TenantID     uuid.UUID
	Level        AlertLevel
	Status       BudgetStatus
	Channels     AlertChannels
	Timestamp    time.Time
	APIKeyPrefix string
	ModelAlias   string
}

type AlertSink interface {
	Notify(ctx context.Context, payload AlertPayload) error
}

type LogAlertSink struct {
	logger *slog.Logger
}

func NewLogAlertSink(logger *slog.Logger) *LogAlertSink {
	if logger == nil {
		logger = slog.Default()
	}
	return &LogAlertSink{logger: logger}
}

func (s *LogAlertSink) Notify(ctx context.Context, payload AlertPayload) error {
	if s == nil || s.logger == nil {
		return nil
	}

	s.logger.WarnContext(ctx, "budget alert",
		slog.String("tenant_id", payload.TenantID.String()),
		slog.String("level", string(payload.Level)),
		slog.Int64("total_cost_cents", payload.Status.TotalCostCents),
		slog.Int64("limit_cents", payload.Status.LimitCents),
		slog.Bool("warning", payload.Status.Warning),
		slog.Bool("exceeded", payload.Status.Exceeded),
		slog.String("api_key_prefix", payload.APIKeyPrefix),
		slog.String("model_alias", payload.ModelAlias),
		slog.Any("emails", payload.Channels.Emails),
		slog.Any("webhooks", payload.Channels.Webhooks),
		slog.Time("timestamp", payload.Timestamp.UTC()),
	)
	return nil
}
