package adminconfig

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ncecere/open_model_gateway/backend/internal/config"
	"github.com/ncecere/open_model_gateway/backend/internal/db"
	"github.com/ncecere/open_model_gateway/backend/internal/email"
	batchsvc "github.com/ncecere/open_model_gateway/backend/internal/services/batches"
	filesvc "github.com/ncecere/open_model_gateway/backend/internal/services/files"
	usagepipeline "github.com/ncecere/open_model_gateway/backend/internal/services/usagepipeline"
)

const (
	filesSettingKey   = "files_config"
	batchesSettingKey = "batches_config"
	alertsSettingKey  = "alert_transport_config"
)

type Service struct {
	queries *db.Queries
	cfg     *config.Config
	files   *filesvc.Service
	batches *batchsvc.Service
}

func NewService(queries *db.Queries, cfg *config.Config, files *filesvc.Service, batches *batchsvc.Service) *Service {
	return &Service{queries: queries, cfg: cfg, files: files, batches: batches}
}

type FileSettings struct {
	MaxSizeMB         int `json:"max_size_mb"`
	DefaultTTLSeconds int `json:"default_ttl_seconds"`
	MaxTTLSeconds     int `json:"max_ttl_seconds"`
}

type BatchSettings struct {
	MaxRequests       int `json:"max_requests"`
	MaxConcurrency    int `json:"max_concurrency"`
	DefaultTTLSeconds int `json:"default_ttl_seconds"`
	MaxTTLSeconds     int `json:"max_ttl_seconds"`
}

type AlertSettings struct {
	SMTP    config.SMTPConfig    `json:"smtp"`
	Webhook config.WebhookConfig `json:"webhook"`
}

func (s *Service) CurrentFileSettings() FileSettings {
	return FileSettings{
		MaxSizeMB:         s.cfg.Files.MaxSizeMB,
		DefaultTTLSeconds: int(s.cfg.Files.DefaultTTL / time.Second),
		MaxTTLSeconds:     int(s.cfg.Files.MaxTTL / time.Second),
	}
}

func (s *Service) CurrentBatchSettings() BatchSettings {
	return BatchSettings{
		MaxRequests:       s.cfg.Batches.MaxRequests,
		MaxConcurrency:    s.cfg.Batches.MaxConcurrency,
		DefaultTTLSeconds: int(s.cfg.Batches.DefaultTTL / time.Second),
		MaxTTLSeconds:     int(s.cfg.Batches.MaxTTL / time.Second),
	}
}

func (s *Service) CurrentAlertSettings() AlertSettings {
	return AlertSettings{
		SMTP:    s.cfg.Budgets.Alert.SMTP,
		Webhook: s.cfg.Budgets.Alert.Webhook,
	}
}

func (s *Service) UpdateFileSettings(ctx context.Context, req FileSettings, updatedBy uuid.UUID) (FileSettings, error) {
	if req.MaxSizeMB <= 0 {
		return FileSettings{}, errors.New("max size must be positive")
	}
	if req.DefaultTTLSeconds <= 0 || req.MaxTTLSeconds <= 0 {
		return FileSettings{}, errors.New("ttls must be positive")
	}
	if req.DefaultTTLSeconds > req.MaxTTLSeconds {
		return FileSettings{}, errors.New("default ttl cannot exceed max ttl")
	}

	s.cfg.Files.MaxSizeMB = req.MaxSizeMB
	s.cfg.Files.DefaultTTL = time.Duration(req.DefaultTTLSeconds) * time.Second
	s.cfg.Files.MaxTTL = time.Duration(req.MaxTTLSeconds) * time.Second

	payload, _ := json.Marshal(req)
	if _, err := s.queries.UpsertSystemSetting(ctx, db.UpsertSystemSettingParams{
		Key:       filesSettingKey,
		Value:     payload,
		UpdatedBy: toPgUUID(updatedBy),
	}); err != nil {
		return FileSettings{}, err
	}
	return req, nil
}

func (s *Service) UpdateBatchSettings(ctx context.Context, req BatchSettings, updatedBy uuid.UUID) (BatchSettings, error) {
	if req.MaxRequests <= 0 || req.MaxConcurrency <= 0 {
		return BatchSettings{}, errors.New("limits must be positive")
	}
	if req.DefaultTTLSeconds <= 0 || req.MaxTTLSeconds <= 0 {
		return BatchSettings{}, errors.New("ttls must be positive")
	}
	if req.DefaultTTLSeconds > req.MaxTTLSeconds {
		return BatchSettings{}, errors.New("default ttl cannot exceed max ttl")
	}

	s.cfg.Batches.MaxRequests = req.MaxRequests
	s.cfg.Batches.MaxConcurrency = req.MaxConcurrency
	s.cfg.Batches.DefaultTTL = time.Duration(req.DefaultTTLSeconds) * time.Second
	s.cfg.Batches.MaxTTL = time.Duration(req.MaxTTLSeconds) * time.Second

	payload, _ := json.Marshal(req)
	if _, err := s.queries.UpsertSystemSetting(ctx, db.UpsertSystemSettingParams{
		Key:       batchesSettingKey,
		Value:     payload,
		UpdatedBy: toPgUUID(updatedBy),
	}); err != nil {
		return BatchSettings{}, err
	}
	return req, nil
}

func (s *Service) UpdateAlertSettings(ctx context.Context, req AlertSettings, updatedBy uuid.UUID) (AlertSettings, error) {
	if req.SMTP.Port < 0 {
		return AlertSettings{}, errors.New("smtp port must be positive")
	}
	if req.SMTP.ConnectTimeout < 0 {
		return AlertSettings{}, errors.New("smtp timeout must be non-negative")
	}
	if req.Webhook.Timeout < 0 {
		return AlertSettings{}, errors.New("webhook timeout must be non-negative")
	}
	if req.Webhook.MaxRetries < 0 {
		return AlertSettings{}, errors.New("webhook max retries must be non-negative")
	}

	s.cfg.Budgets.Alert.SMTP = req.SMTP
	s.cfg.Budgets.Alert.Webhook = req.Webhook

	payload, _ := json.Marshal(req)
	if _, err := s.queries.UpsertSystemSetting(ctx, db.UpsertSystemSettingParams{
		Key:       alertsSettingKey,
		Value:     payload,
		UpdatedBy: toPgUUID(updatedBy),
	}); err != nil {
		return AlertSettings{}, err
	}

	return req, nil
}

func (s *Service) SendTestEmail(ctx context.Context, to string, kind string) error {
	to = strings.TrimSpace(to)
	if to == "" {
		return errors.New("email required")
	}
	emailType := strings.ToLower(strings.TrimSpace(kind))
	if emailType == "" {
		emailType = "budget"
	}
	switch emailType {
	case "invite":
		return s.sendSampleInvite(ctx, to)
	case "smtp", "transport":
		return s.sendSampleTransport(ctx, to)
	case "budget":
		return s.sendSampleBudgetAlert(ctx, to)
	default:
		return fmt.Errorf("unknown test email type %q", kind)
	}
}

func (s *Service) sendSampleBudgetAlert(ctx context.Context, to string) error {
	sink := usagepipeline.NewEmailSink(s.cfg.Budgets.Alert.SMTP, s.cfg.Public.BaseURL, nil)
	if sink == nil {
		return errors.New("smtp not configured")
	}
	payload := usagepipeline.AlertPayload{
		TenantID:         uuid.Nil,
		TenantName:       "Sample Tenant",
		Level:            usagepipeline.AlertLevelWarning,
		Status:           usagepipeline.BudgetStatus{TotalCostCents: 4500, LimitCents: 10000, Warning: true},
		Channels:         usagepipeline.AlertChannels{Emails: []string{to}},
		Timestamp:        time.Now().UTC(),
		WarningThreshold: 0.8,
		BudgetReset:      time.Now().Add(24 * time.Hour),
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	return sink.Notify(ctx, payload)
}

func (s *Service) sendSampleInvite(ctx context.Context, to string) error {
	sender, err := s.smtpSender()
	if err != nil {
		return err
	}
	base := strings.TrimRight(strings.TrimSpace(s.cfg.Public.BaseURL), "/")
	if base == "" {
		return errors.New("public.base_url must be configured to send invite previews")
	}
	portal := fmt.Sprintf("%s/admin/ui", base)
	htmlBody, err := email.RenderAdminInviteTemplate(email.AdminInviteTemplateData{
		RecipientName: "Sample User",
		PortalURL:     portal,
	})
	if err != nil {
		return err
	}
	plain := fmt.Sprintf(
		"Hi Sample User,\n\nYou've been invited to Open Model Gateway. Visit %s to accept the invite and sign in with your organization's SSO provider or assigned credentials.\n",
		portal,
	)
	msg := email.Message{
		From:     strings.TrimSpace(s.cfg.Budgets.Alert.SMTP.From),
		To:       []string{to},
		Subject:  "You're invited to Open Model Gateway",
		Body:     plain,
		HTMLBody: htmlBody,
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	return sender.Send(ctx, msg)
}

func (s *Service) sendSampleTransport(ctx context.Context, to string) error {
	sender, err := s.smtpSender()
	if err != nil {
		return err
	}
	ts := time.Now().UTC()
	htmlBody, err := email.RenderTestEmailTemplate(email.TestEmailTemplateData{
		Recipient: to,
		Timestamp: ts.Format("Jan 02, 2006 15:04 MST"),
	})
	if err != nil {
		return err
	}
	plain := fmt.Sprintf("SMTP test email\n\nRecipient: %s\nDispatched at: %s\n", to, ts.Format(time.RFC3339))
	msg := email.Message{
		From:     strings.TrimSpace(s.cfg.Budgets.Alert.SMTP.From),
		To:       []string{to},
		Subject:  "Open Model Gateway SMTP Test",
		Body:     plain,
		HTMLBody: htmlBody,
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	return sender.Send(ctx, msg)
}

func (s *Service) smtpSender() (email.Sender, error) {
	sender := email.NewSMTPSender(s.cfg.Budgets.Alert.SMTP)
	if sender == nil {
		return nil, errors.New("smtp not configured")
	}
	return sender, nil
}

func toPgUUID(id uuid.UUID) pgtype.UUID {
	if id == uuid.Nil {
		return pgtype.UUID{Valid: false}
	}
	var out pgtype.UUID
	copy(out.Bytes[:], id[:])
	out.Valid = true
	return out
}

// ApplyOverrides loads stored file/batch settings and mutates cfg accordingly.
func ApplyOverrides(ctx context.Context, queries *db.Queries, cfg *config.Config) {
	if cfg == nil || queries == nil {
		return
	}
	if setting, err := queries.GetSystemSetting(ctx, filesSettingKey); err == nil {
		var payload FileSettings
		if err := json.Unmarshal(setting.Value, &payload); err == nil {
			if payload.MaxSizeMB > 0 {
				cfg.Files.MaxSizeMB = payload.MaxSizeMB
			}
			if payload.DefaultTTLSeconds > 0 {
				cfg.Files.DefaultTTL = time.Duration(payload.DefaultTTLSeconds) * time.Second
			}
			if payload.MaxTTLSeconds > 0 {
				cfg.Files.MaxTTL = time.Duration(payload.MaxTTLSeconds) * time.Second
			}
		}
	}
	if setting, err := queries.GetSystemSetting(ctx, batchesSettingKey); err == nil {
		var payload BatchSettings
		if err := json.Unmarshal(setting.Value, &payload); err == nil {
			if payload.MaxRequests > 0 {
				cfg.Batches.MaxRequests = payload.MaxRequests
			}
			if payload.MaxConcurrency > 0 {
				cfg.Batches.MaxConcurrency = payload.MaxConcurrency
			}
			if payload.DefaultTTLSeconds > 0 {
				cfg.Batches.DefaultTTL = time.Duration(payload.DefaultTTLSeconds) * time.Second
			}
			if payload.MaxTTLSeconds > 0 {
				cfg.Batches.MaxTTL = time.Duration(payload.MaxTTLSeconds) * time.Second
			}
		}
	} else if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		_ = err
	}
	if setting, err := queries.GetSystemSetting(ctx, alertsSettingKey); err == nil {
		var payload AlertSettings
		if err := json.Unmarshal(setting.Value, &payload); err == nil {
			cfg.Budgets.Alert.SMTP = payload.SMTP
			cfg.Budgets.Alert.Webhook = payload.Webhook
		}
	}
}
