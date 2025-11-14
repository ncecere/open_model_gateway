package admin

import (
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/ncecere/open_model_gateway/backend/internal/app"
	"github.com/ncecere/open_model_gateway/backend/internal/config"
	"github.com/ncecere/open_model_gateway/backend/internal/db"
	"github.com/ncecere/open_model_gateway/backend/internal/httpserver/httputil"
	adminconfigsvc "github.com/ncecere/open_model_gateway/backend/internal/services/adminconfig"
)

func registerAdminSettingsRoutes(router fiber.Router, container *app.Container) {
	handler := &settingsHandler{container: container}
	group := router.Group("/settings")
	group.Get("/files", handler.getFileSettings)
	group.Put("/files", handler.updateFileSettings)
	group.Get("/batches", handler.getBatchSettings)
	group.Put("/batches", handler.updateBatchSettings)
	group.Get("/alerts", handler.getAlertSettings)
	group.Put("/alerts", handler.updateAlertSettings)
	group.Post("/alerts/test-email", handler.sendTestAlertEmail)
}

type settingsHandler struct {
	container *app.Container
}

func (h *settingsHandler) getFileSettings(c *fiber.Ctx) error {
	if err := requireAnyRole(c, h.container, db.MembershipRoleAdmin); err != nil {
		return err
	}
	svc := h.service()
	if svc == nil {
		return httputil.WriteError(c, fiber.StatusNotImplemented, "settings service unavailable")
	}
	settings := svc.CurrentFileSettings()
	return c.JSON(toFileSettingsPayload(settings))
}

func (h *settingsHandler) updateFileSettings(c *fiber.Ctx) error {
	if err := requireAnyRole(c, h.container, db.MembershipRoleAdmin); err != nil {
		return err
	}
	svc := h.service()
	if svc == nil {
		return httputil.WriteError(c, fiber.StatusNotImplemented, "settings service unavailable")
	}
	adminID, ok := adminUserIDFromContext(c.UserContext())
	if !ok {
		return httputil.WriteError(c, fiber.StatusUnauthorized, "admin identity missing")
	}
	var payload fileSettingsPayload
	if err := c.BodyParser(&payload); err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid request body")
	}
	if err := validateFileSettings(payload); err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, err.Error())
	}
	updated, err := svc.UpdateFileSettings(c.Context(), adminconfigsvc.FileSettings{
		MaxSizeMB:         payload.MaxSizeMB,
		DefaultTTLSeconds: payload.DefaultTTLSeconds,
		MaxTTLSeconds:     payload.MaxTTLSeconds,
	}, adminID)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	if err := recordAudit(c, h.container, "settings.files.update", "files_settings", "global", fiber.Map{
		"max_size_mb":         updated.MaxSizeMB,
		"default_ttl_seconds": updated.DefaultTTLSeconds,
		"max_ttl_seconds":     updated.MaxTTLSeconds,
	}); err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(toFileSettingsPayload(updated))
}

func (h *settingsHandler) getBatchSettings(c *fiber.Ctx) error {
	if err := requireAnyRole(c, h.container, db.MembershipRoleAdmin); err != nil {
		return err
	}
	svc := h.service()
	if svc == nil {
		return httputil.WriteError(c, fiber.StatusNotImplemented, "settings service unavailable")
	}
	settings := svc.CurrentBatchSettings()
	return c.JSON(toBatchSettingsPayload(settings))
}

func (h *settingsHandler) updateBatchSettings(c *fiber.Ctx) error {
	if err := requireAnyRole(c, h.container, db.MembershipRoleAdmin); err != nil {
		return err
	}
	svc := h.service()
	if svc == nil {
		return httputil.WriteError(c, fiber.StatusNotImplemented, "settings service unavailable")
	}
	adminID, ok := adminUserIDFromContext(c.UserContext())
	if !ok {
		return httputil.WriteError(c, fiber.StatusUnauthorized, "admin identity missing")
	}
	var payload batchSettingsPayload
	if err := c.BodyParser(&payload); err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid request body")
	}
	if err := validateBatchSettings(payload); err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, err.Error())
	}
	updated, err := svc.UpdateBatchSettings(c.Context(), adminconfigsvc.BatchSettings{
		MaxRequests:       payload.MaxRequests,
		MaxConcurrency:    payload.MaxConcurrency,
		DefaultTTLSeconds: payload.DefaultTTLSeconds,
		MaxTTLSeconds:     payload.MaxTTLSeconds,
	}, adminID)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	if err := recordAudit(c, h.container, "settings.batches.update", "batches_settings", "global", fiber.Map{
		"max_requests":        updated.MaxRequests,
		"max_concurrency":     updated.MaxConcurrency,
		"default_ttl_seconds": updated.DefaultTTLSeconds,
		"max_ttl_seconds":     updated.MaxTTLSeconds,
	}); err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(toBatchSettingsPayload(updated))
}

func (h *settingsHandler) getAlertSettings(c *fiber.Ctx) error {
	if err := requireAnyRole(c, h.container, db.MembershipRoleAdmin); err != nil {
		return err
	}
	svc := h.service()
	if svc == nil {
		return httputil.WriteError(c, fiber.StatusNotImplemented, "settings service unavailable")
	}
	settings := svc.CurrentAlertSettings()
	return c.JSON(toAlertSettingsPayload(settings))
}

func (h *settingsHandler) updateAlertSettings(c *fiber.Ctx) error {
	if err := requireAnyRole(c, h.container, db.MembershipRoleAdmin); err != nil {
		return err
	}
	svc := h.service()
	if svc == nil {
		return httputil.WriteError(c, fiber.StatusNotImplemented, "settings service unavailable")
	}
	adminID, ok := adminUserIDFromContext(c.UserContext())
	if !ok {
		return httputil.WriteError(c, fiber.StatusUnauthorized, "admin identity missing")
	}
	var payload alertSettingsPayload
	if err := c.BodyParser(&payload); err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid request body")
	}
	settings, err := svc.UpdateAlertSettings(c.Context(), alertPayloadToConfig(payload), adminID)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, err.Error())
	}
	h.container.UpdateBudgetConfig(h.container.Config.Budgets)
	if err := recordAudit(c, h.container, "settings.alerts.update", "alert_settings", "global", fiber.Map{
		"smtp_host":            settings.SMTP.Host,
		"smtp_port":            settings.SMTP.Port,
		"smtp_username":        settings.SMTP.Username,
		"smtp_from":            settings.SMTP.From,
		"smtp_use_tls":         settings.SMTP.UseTLS,
		"smtp_skip_tls_verify": settings.SMTP.SkipTLSVerify,
		"webhook_timeout":      settings.Webhook.Timeout.String(),
		"webhook_max_retries":  settings.Webhook.MaxRetries,
	}); err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(toAlertSettingsPayload(settings))
}

func (h *settingsHandler) sendTestAlertEmail(c *fiber.Ctx) error {
	if err := requireAnyRole(c, h.container, db.MembershipRoleAdmin); err != nil {
		return err
	}
	svc := h.service()
	if svc == nil {
		return httputil.WriteError(c, fiber.StatusNotImplemented, "settings service unavailable")
	}
	var payload struct {
		Email string `json:"email"`
	}
	if err := c.BodyParser(&payload); err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid request body")
	}
	if err := svc.SendAlertTestEmail(c.Context(), payload.Email); err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, err.Error())
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *settingsHandler) service() *adminconfigsvc.Service {
	if h.container == nil {
		return nil
	}
	return h.container.AdminConfig
}

type fileSettingsPayload struct {
	MaxSizeMB         int `json:"max_size_mb"`
	DefaultTTLSeconds int `json:"default_ttl_seconds"`
	MaxTTLSeconds     int `json:"max_ttl_seconds"`
}

func toFileSettingsPayload(input adminconfigsvc.FileSettings) fileSettingsPayload {
	return fileSettingsPayload{
		MaxSizeMB:         input.MaxSizeMB,
		DefaultTTLSeconds: input.DefaultTTLSeconds,
		MaxTTLSeconds:     input.MaxTTLSeconds,
	}
}

func validateFileSettings(payload fileSettingsPayload) error {
	if payload.MaxSizeMB <= 0 {
		return fmt.Errorf("max_size_mb must be positive")
	}
	if payload.DefaultTTLSeconds <= 0 || payload.MaxTTLSeconds <= 0 {
		return fmt.Errorf("ttls must be positive")
	}
	if payload.DefaultTTLSeconds > payload.MaxTTLSeconds {
		return fmt.Errorf("default ttl cannot exceed max ttl")
	}
	return nil
}

type batchSettingsPayload struct {
	MaxRequests       int `json:"max_requests"`
	MaxConcurrency    int `json:"max_concurrency"`
	DefaultTTLSeconds int `json:"default_ttl_seconds"`
	MaxTTLSeconds     int `json:"max_ttl_seconds"`
}

func toBatchSettingsPayload(input adminconfigsvc.BatchSettings) batchSettingsPayload {
	return batchSettingsPayload{
		MaxRequests:       input.MaxRequests,
		MaxConcurrency:    input.MaxConcurrency,
		DefaultTTLSeconds: input.DefaultTTLSeconds,
		MaxTTLSeconds:     input.MaxTTLSeconds,
	}
}

func validateBatchSettings(payload batchSettingsPayload) error {
	if payload.MaxRequests <= 0 || payload.MaxConcurrency <= 0 {
		return fmt.Errorf("limits must be positive")
	}
	if payload.DefaultTTLSeconds <= 0 || payload.MaxTTLSeconds <= 0 {
		return fmt.Errorf("ttls must be positive")
	}
	if payload.DefaultTTLSeconds > payload.MaxTTLSeconds {
		return fmt.Errorf("default ttl cannot exceed max ttl")
	}
	return nil
}

type smtpSettingsPayload struct {
	Host                 string `json:"host"`
	Port                 int    `json:"port"`
	Username             string `json:"username"`
	Password             string `json:"password"`
	From                 string `json:"from"`
	UseTLS               bool   `json:"use_tls"`
	SkipTLSVerify        bool   `json:"skip_tls_verify"`
	ConnectTimeoutSecond int    `json:"connect_timeout_seconds"`
}

type webhookSettingsPayload struct {
	TimeoutSeconds int `json:"timeout_seconds"`
	MaxRetries     int `json:"max_retries"`
}

type alertSettingsPayload struct {
	SMTP    smtpSettingsPayload    `json:"smtp"`
	Webhook webhookSettingsPayload `json:"webhook"`
}

func toAlertSettingsPayload(input adminconfigsvc.AlertSettings) alertSettingsPayload {
	return alertSettingsPayload{
		SMTP: smtpSettingsPayload{
			Host:                 input.SMTP.Host,
			Port:                 input.SMTP.Port,
			Username:             input.SMTP.Username,
			Password:             input.SMTP.Password,
			From:                 input.SMTP.From,
			UseTLS:               input.SMTP.UseTLS,
			SkipTLSVerify:        input.SMTP.SkipTLSVerify,
			ConnectTimeoutSecond: int(input.SMTP.ConnectTimeout / time.Second),
		},
		Webhook: webhookSettingsPayload{
			TimeoutSeconds: int(input.Webhook.Timeout / time.Second),
			MaxRetries:     input.Webhook.MaxRetries,
		},
	}
}

func alertPayloadToConfig(payload alertSettingsPayload) adminconfigsvc.AlertSettings {
	return adminconfigsvc.AlertSettings{
		SMTP: config.SMTPConfig{
			Host:           strings.TrimSpace(payload.SMTP.Host),
			Port:           payload.SMTP.Port,
			Username:       strings.TrimSpace(payload.SMTP.Username),
			Password:       payload.SMTP.Password,
			From:           strings.TrimSpace(payload.SMTP.From),
			UseTLS:         payload.SMTP.UseTLS,
			SkipTLSVerify:  payload.SMTP.SkipTLSVerify,
			ConnectTimeout: time.Duration(payload.SMTP.ConnectTimeoutSecond) * time.Second,
		},
		Webhook: config.WebhookConfig{
			Timeout:    time.Duration(payload.Webhook.TimeoutSeconds) * time.Second,
			MaxRetries: payload.Webhook.MaxRetries,
		},
	}
}
