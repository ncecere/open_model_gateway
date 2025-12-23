package billinghooks

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
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ncecere/open_model_gateway/backend/internal/config"
	"github.com/ncecere/open_model_gateway/backend/internal/db"
)

type Service struct {
	queries    *db.Queries
	client     *http.Client
	maxRetries int
	logger     *slog.Logger
}

type Webhook struct {
	ID        uuid.UUID
	TenantID  uuid.UUID
	Name      string
	URL       string
	Secret    string
	Enabled   bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Event struct {
	ID          uuid.UUID
	WebhookID   uuid.UUID
	TenantID    uuid.UUID
	PeriodStart time.Time
	PeriodEnd   time.Time
	Payload     json.RawMessage
	Success     bool
	StatusCode  *int32
	Error       string
	CreatedAt   time.Time
}

type SummaryPayload struct {
	TenantID    string            `json:"tenant_id"`
	PeriodStart time.Time         `json:"period_start"`
	PeriodEnd   time.Time         `json:"period_end"`
	GeneratedAt time.Time         `json:"generated_at"`
	Spend       SpendSummary      `json:"spend"`
	Tokens      TokenSummary      `json:"token_counts"`
	TopModels   []TopModelSummary `json:"top_models"`
}

type SpendSummary struct {
	CostCents     int64   `json:"cost_cents"`
	CostUSDMicros int64   `json:"cost_usd_micros"`
	CostUSD       float64 `json:"cost_usd"`
	Currency      string  `json:"currency"`
}

type TokenSummary struct {
	Requests     int64 `json:"requests"`
	InputTokens  int64 `json:"input_tokens"`
	OutputTokens int64 `json:"output_tokens"`
	TotalTokens  int64 `json:"total_tokens"`
}

type TopModelSummary struct {
	ModelAlias    string  `json:"model_alias"`
	Provider      string  `json:"provider"`
	Requests      int64   `json:"requests"`
	Tokens        int64   `json:"tokens"`
	CostCents     int64   `json:"cost_cents"`
	CostUSDMicros int64   `json:"cost_usd_micros"`
	CostUSD       float64 `json:"cost_usd"`
}

type CreateParams struct {
	TenantID uuid.UUID
	Name     string
	URL      string
	Secret   string
	Enabled  bool
}

type UpdateParams struct {
	Name    string
	URL     string
	Secret  string
	Enabled bool
}

func NewService(queries *db.Queries, cfg config.WebhookConfig, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 5 * time.Second
	}
	if cfg.MaxRetries <= 0 {
		cfg.MaxRetries = 1
	}
	return &Service{
		queries:    queries,
		client:     &http.Client{Timeout: cfg.Timeout},
		maxRetries: cfg.MaxRetries,
		logger:     logger,
	}
}

func (s *Service) Create(ctx context.Context, params CreateParams) (Webhook, error) {
	if s == nil || s.queries == nil {
		return Webhook{}, errors.New("billing webhooks unavailable")
	}
	if params.TenantID == uuid.Nil || strings.TrimSpace(params.URL) == "" {
		return Webhook{}, errors.New("tenant_id and url required")
	}
	row, err := s.queries.CreateBillingWebhook(ctx, db.CreateBillingWebhookParams{
		TenantID: toPgUUID(params.TenantID),
		Name:     toPgText(params.Name),
		Url:      strings.TrimSpace(params.URL),
		Secret:   toPgText(params.Secret),
		Enabled:  params.Enabled,
	})
	if err != nil {
		return Webhook{}, err
	}
	return webhookFromDB(row)
}

func (s *Service) Update(ctx context.Context, id uuid.UUID, params UpdateParams) (Webhook, error) {
	if s == nil || s.queries == nil {
		return Webhook{}, errors.New("billing webhooks unavailable")
	}
	if id == uuid.Nil {
		return Webhook{}, errors.New("invalid webhook id")
	}
	row, err := s.queries.UpdateBillingWebhook(ctx, db.UpdateBillingWebhookParams{
		ID:      toPgUUID(id),
		Name:    toPgText(params.Name),
		Url:     strings.TrimSpace(params.URL),
		Secret:  toPgText(params.Secret),
		Enabled: params.Enabled,
	})
	if err != nil {
		return Webhook{}, err
	}
	return webhookFromDB(row)
}

func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	if s == nil || s.queries == nil {
		return errors.New("billing webhooks unavailable")
	}
	if id == uuid.Nil {
		return errors.New("invalid webhook id")
	}
	return s.queries.DeleteBillingWebhook(ctx, toPgUUID(id))
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (Webhook, error) {
	if s == nil || s.queries == nil {
		return Webhook{}, errors.New("billing webhooks unavailable")
	}
	row, err := s.queries.GetBillingWebhook(ctx, toPgUUID(id))
	if err != nil {
		return Webhook{}, err
	}
	return webhookFromDB(row)
}

func (s *Service) ListAdmin(ctx context.Context) ([]Webhook, error) {
	if s == nil || s.queries == nil {
		return nil, errors.New("billing webhooks unavailable")
	}
	rows, err := s.queries.ListBillingWebhooksAdmin(ctx)
	if err != nil {
		return nil, err
	}
	return webhooksFromDB(rows)
}

func (s *Service) ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]Webhook, error) {
	if s == nil || s.queries == nil {
		return nil, errors.New("billing webhooks unavailable")
	}
	rows, err := s.queries.ListBillingWebhooksByTenant(ctx, toPgUUID(tenantID))
	if err != nil {
		return nil, err
	}
	return webhooksFromDB(rows)
}

func (s *Service) ListEvents(ctx context.Context, webhookID uuid.UUID, limit, offset int32) ([]Event, error) {
	if s == nil || s.queries == nil {
		return nil, errors.New("billing webhooks unavailable")
	}
	if limit <= 0 {
		limit = 25
	}
	rows, err := s.queries.ListBillingWebhookEvents(ctx, db.ListBillingWebhookEventsParams{
		WebhookID: toPgUUID(webhookID),
		Limit:     limit,
		Offset:    offset,
	})
	if err != nil {
		return nil, err
	}
	return eventsFromDB(rows)
}

func (s *Service) DispatchSummary(ctx context.Context, tenantID uuid.UUID, start, end time.Time) ([]Event, error) {
	if s == nil || s.queries == nil {
		return nil, errors.New("billing webhooks unavailable")
	}
	webhooks, err := s.ListByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if len(webhooks) == 0 {
		return nil, nil
	}
	summary, err := s.buildSummary(ctx, tenantID, start, end)
	if err != nil {
		return nil, err
	}
	payload, err := json.Marshal(summary)
	if err != nil {
		return nil, err
	}

	events := make([]Event, 0, len(webhooks))
	for _, webhook := range webhooks {
		if !webhook.Enabled {
			continue
		}
		event, err := s.deliver(ctx, webhook, payload, start, end)
		if err != nil {
			s.logger.Warn("billing webhook failed", slog.String("webhook_id", webhook.ID.String()), slog.String("error", err.Error()))
		}
		events = append(events, event)
	}
	return events, nil
}

func (s *Service) RetryEvent(ctx context.Context, eventID uuid.UUID) (Event, error) {
	if s == nil || s.queries == nil {
		return Event{}, errors.New("billing webhooks unavailable")
	}
	row, err := s.queries.GetBillingWebhookEvent(ctx, toPgUUID(eventID))
	if err != nil {
		return Event{}, err
	}
	webhookRow, err := s.queries.GetBillingWebhook(ctx, row.WebhookID)
	if err != nil {
		return Event{}, err
	}
	webhook, err := webhookFromDB(webhookRow)
	if err != nil {
		return Event{}, err
	}
	start, err := timeFromPg(row.PeriodStart)
	if err != nil {
		return Event{}, err
	}
	end, err := timeFromPg(row.PeriodEnd)
	if err != nil {
		return Event{}, err
	}
	return s.deliver(ctx, webhook, row.Payload, start, end)
}

func (s *Service) deliver(ctx context.Context, webhook Webhook, payload []byte, start, end time.Time) (Event, error) {
	result := Event{
		WebhookID:   webhook.ID,
		TenantID:    webhook.TenantID,
		PeriodStart: start,
		PeriodEnd:   end,
		Payload:     payload,
	}
	statusCode, err := s.postWithRetries(ctx, webhook, payload)
	result.Success = err == nil
	if statusCode > 0 {
		code := int32(statusCode)
		result.StatusCode = &code
	}
	if err != nil {
		result.Error = err.Error()
	}
	row, insertErr := s.queries.InsertBillingWebhookEvent(ctx, db.InsertBillingWebhookEventParams{
		WebhookID:   toPgUUID(webhook.ID),
		TenantID:    toPgUUID(webhook.TenantID),
		PeriodStart: toPgTime(start),
		PeriodEnd:   toPgTime(end),
		Payload:     payload,
		Success:     result.Success,
		StatusCode:  toPgInt32(result.StatusCode),
		Error:       toPgText(result.Error),
	})
	if insertErr != nil {
		return result, insertErr
	}
	final, err := eventFromDB(row)
	if err != nil {
		return result, err
	}
	return final, err
}

func (s *Service) postWithRetries(ctx context.Context, webhook Webhook, payload []byte) (int, error) {
	var lastErr error
	var lastStatus int
	for attempt := 1; attempt <= s.maxRetries; attempt++ {
		status, err := s.post(ctx, webhook, payload)
		lastStatus = status
		if err == nil {
			return status, nil
		}
		lastErr = err
		delay := time.Duration(attempt) * 250 * time.Millisecond
		select {
		case <-ctx.Done():
			return lastStatus, ctx.Err()
		case <-time.After(delay):
		}
	}
	return lastStatus, lastErr
}

func (s *Service) post(ctx context.Context, webhook Webhook, payload []byte) (int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhook.URL, bytes.NewReader(payload))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "open-model-gateway")
	if sig := signPayload(webhook.Secret, payload); sig != "" {
		req.Header.Set("X-OMG-Signature", sig)
		req.Header.Set("X-OMG-Signature-Version", "v1")
		req.Header.Set("X-OMG-Timestamp", time.Now().UTC().Format(time.RFC3339))
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusBadRequest {
		return resp.StatusCode, fmt.Errorf("status %d", resp.StatusCode)
	}
	return resp.StatusCode, nil
}

func (s *Service) buildSummary(ctx context.Context, tenantID uuid.UUID, start, end time.Time) (SummaryPayload, error) {
	totals, err := s.queries.SumUsageForTenant(ctx, db.SumUsageForTenantParams{
		TenantID: toPgUUID(tenantID),
		Ts:       toPgTime(start),
		Ts_2:     toPgTime(end),
	})
	if err != nil {
		return SummaryPayload{}, err
	}
	topModels, err := s.queries.AggregateUsageByModelForTenant(ctx, db.AggregateUsageByModelForTenantParams{
		TenantID: toPgUUID(tenantID),
		Ts:       toPgTime(start),
		Ts_2:     toPgTime(end),
		Limit:    5,
	})
	if err != nil {
		return SummaryPayload{}, err
	}
	models := make([]TopModelSummary, 0, len(topModels))
	for _, model := range topModels {
		models = append(models, TopModelSummary{
			ModelAlias:    model.ModelAlias,
			Provider:      model.Provider,
			Requests:      model.Requests,
			Tokens:        model.Tokens,
			CostCents:     model.CostCents,
			CostUSDMicros: model.CostUsdMicros,
			CostUSD:       microsToUSD(model.CostUsdMicros),
		})
	}
	return SummaryPayload{
		TenantID:    tenantID.String(),
		PeriodStart: start,
		PeriodEnd:   end,
		GeneratedAt: time.Now().UTC(),
		Spend: SpendSummary{
			CostCents:     totals.TotalCostCents,
			CostUSDMicros: totals.TotalCostUsdMicros,
			CostUSD:       microsToUSD(totals.TotalCostUsdMicros),
			Currency:      "USD",
		},
		Tokens: TokenSummary{
			Requests:     int64FromAny(totals.TotalRequests),
			InputTokens:  int64FromAny(totals.TotalInputTokens),
			OutputTokens: int64FromAny(totals.TotalOutputTokens),
			TotalTokens:  int64FromAny(totals.TotalInputTokens) + int64FromAny(totals.TotalOutputTokens),
		},
		TopModels: models,
	}, nil
}

func signPayload(secret string, payload []byte) string {
	clean := strings.TrimSpace(secret)
	if clean == "" {
		return ""
	}
	mac := hmac.New(sha256.New, []byte(clean))
	mac.Write(payload)
	sum := mac.Sum(nil)
	return "sha256=" + hex.EncodeToString(sum)
}

func webhooksFromDB(rows []db.BillingWebhook) ([]Webhook, error) {
	result := make([]Webhook, 0, len(rows))
	for _, row := range rows {
		hook, err := webhookFromDB(row)
		if err != nil {
			return nil, err
		}
		result = append(result, hook)
	}
	return result, nil
}

func webhookFromDB(row db.BillingWebhook) (Webhook, error) {
	id, err := uuidFromPg(row.ID)
	if err != nil {
		return Webhook{}, err
	}
	tenantID, err := uuidFromPg(row.TenantID)
	if err != nil {
		return Webhook{}, err
	}
	createdAt, err := timeFromPg(row.CreatedAt)
	if err != nil {
		return Webhook{}, err
	}
	updatedAt, err := timeFromPg(row.UpdatedAt)
	if err != nil {
		return Webhook{}, err
	}
	return Webhook{
		ID:        id,
		TenantID:  tenantID,
		Name:      textFromPg(row.Name),
		URL:       row.Url,
		Secret:    textFromPg(row.Secret),
		Enabled:   row.Enabled,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}, nil
}

func eventsFromDB(rows []db.BillingWebhookEvent) ([]Event, error) {
	result := make([]Event, 0, len(rows))
	for _, row := range rows {
		event, err := eventFromDB(row)
		if err != nil {
			return nil, err
		}
		result = append(result, event)
	}
	return result, nil
}

func eventFromDB(row db.BillingWebhookEvent) (Event, error) {
	id, err := uuidFromPg(row.ID)
	if err != nil {
		return Event{}, err
	}
	webhookID, err := uuidFromPg(row.WebhookID)
	if err != nil {
		return Event{}, err
	}
	tenantID, err := uuidFromPg(row.TenantID)
	if err != nil {
		return Event{}, err
	}
	start, err := timeFromPg(row.PeriodStart)
	if err != nil {
		return Event{}, err
	}
	end, err := timeFromPg(row.PeriodEnd)
	if err != nil {
		return Event{}, err
	}
	createdAt, err := timeFromPg(row.CreatedAt)
	if err != nil {
		return Event{}, err
	}
	return Event{
		ID:          id,
		WebhookID:   webhookID,
		TenantID:    tenantID,
		PeriodStart: start,
		PeriodEnd:   end,
		Payload:     row.Payload,
		Success:     row.Success,
		StatusCode:  int32PtrFromPg(row.StatusCode),
		Error:       textFromPg(row.Error),
		CreatedAt:   createdAt,
	}, nil
}

func toPgUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: true}
}

func toPgTime(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}

func toPgText(value string) pgtype.Text {
	clean := strings.TrimSpace(value)
	if clean == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: clean, Valid: true}
}

func toPgInt32(value *int32) pgtype.Int4 {
	if value == nil {
		return pgtype.Int4{}
	}
	return pgtype.Int4{Int32: *value, Valid: true}
}

func uuidFromPg(id pgtype.UUID) (uuid.UUID, error) {
	if !id.Valid {
		return uuid.Nil, errors.New("invalid uuid")
	}
	return id.Bytes, nil
}

func timeFromPg(ts pgtype.Timestamptz) (time.Time, error) {
	if !ts.Valid {
		return time.Time{}, errors.New("invalid timestamp")
	}
	return ts.Time, nil
}

func textFromPg(value pgtype.Text) string {
	if !value.Valid {
		return ""
	}
	return value.String
}

func int32PtrFromPg(value pgtype.Int4) *int32 {
	if !value.Valid {
		return nil
	}
	val := value.Int32
	return &val
}

func microsToUSD(micros int64) float64 {
	return float64(micros) / 1_000_000
}

func int64FromAny(value interface{}) int64 {
	switch v := value.(type) {
	case int64:
		return v
	case int32:
		return int64(v)
	case int:
		return int64(v)
	case float64:
		return int64(v)
	case float32:
		return int64(v)
	case string:
		parsed, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
		if err == nil {
			return parsed
		}
	case []byte:
		parsed, err := strconv.ParseInt(strings.TrimSpace(string(v)), 10, 64)
		if err == nil {
			return parsed
		}
	}
	return 0
}
