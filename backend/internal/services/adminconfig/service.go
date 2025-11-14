package adminconfig

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ncecere/open_model_gateway/backend/internal/config"
	"github.com/ncecere/open_model_gateway/backend/internal/db"
	batchsvc "github.com/ncecere/open_model_gateway/backend/internal/services/batches"
	filesvc "github.com/ncecere/open_model_gateway/backend/internal/services/files"
	"github.com/ncecere/open_model_gateway/backend/internal/services/usagepipeline"
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

func (s *Service) SendAlertTestEmail(ctx context.Context, to string) error {
	to = strings.TrimSpace(to)
	if to == "" {
		return errors.New("email required")
	}
	sink := usagepipeline.NewSMTPSink(s.cfg.Budgets.Alert.SMTP, slog.Default())
	if sink == nil {
		return errors.New("smtp not configured")
	}
	payload := usagepipeline.AlertPayload{
		TenantID:  uuid.Nil,
		Level:     usagepipeline.AlertLevelWarning,
		Status:    usagepipeline.BudgetStatus{},
		Channels:  usagepipeline.AlertChannels{Emails: []string{to}},
		Timestamp: time.Now().UTC(),
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	return sink.Notify(ctx, payload)
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
