package usagepipeline

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

type AlertLevel string

const (
	AlertLevelNone      AlertLevel = "none"
	AlertLevelWarning   AlertLevel = "warning"
	AlertLevelExceeded  AlertLevel = "exceeded"
	AlertLevelGuardrail AlertLevel = "guardrail"
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
	Guardrail    *GuardrailAlert
}

type GuardrailAlert struct {
	Stage      string   `json:"stage"`
	Action     string   `json:"action"`
	Category   string   `json:"category,omitempty"`
	Violations []string `json:"violations,omitempty"`
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
	logger := s.logger
	if payload.Guardrail != nil {
		logger.WarnContext(ctx, "guardrail alert",
			slog.String("tenant_id", payload.TenantID.String()),
			slog.String("stage", payload.Guardrail.Stage),
			slog.String("action", payload.Guardrail.Action),
			slog.Any("violations", payload.Guardrail.Violations),
			slog.String("api_key_prefix", payload.APIKeyPrefix),
			slog.String("model_alias", payload.ModelAlias),
			slog.Any("emails", payload.Channels.Emails),
			slog.Any("webhooks", payload.Channels.Webhooks),
			slog.Time("timestamp", payload.Timestamp.UTC()),
		)
		return nil
	}

	logger.WarnContext(ctx, "budget alert",
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
