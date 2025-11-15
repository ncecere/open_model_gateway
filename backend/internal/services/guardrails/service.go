package guardrails

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ncecere/open_model_gateway/backend/internal/db"
)

type Service struct {
	queries *db.Queries
}

type Policy struct {
	TenantID uuid.UUID      `json:"tenant_id,omitempty"`
	APIKeyID uuid.UUID      `json:"api_key_id,omitempty"`
	Config   map[string]any `json:"config"`
}

type Event struct {
	ID         uuid.UUID      `json:"id"`
	TenantID   *uuid.UUID     `json:"tenant_id,omitempty"`
	TenantName string         `json:"tenant_name,omitempty"`
	APIKeyID   *uuid.UUID     `json:"api_key_id,omitempty"`
	APIKeyName string         `json:"api_key_name,omitempty"`
	ModelAlias string         `json:"model_alias,omitempty"`
	Action     string         `json:"action"`
	Category   string         `json:"category,omitempty"`
	Stage      string         `json:"stage,omitempty"`
	Violations []string       `json:"violations,omitempty"`
	Details    map[string]any `json:"details"`
	CreatedAt  time.Time      `json:"created_at"`
}

type ListEventsParams struct {
	TenantID uuid.UUID
	APIKeyID uuid.UUID
	Action   string
	Stage    string
	Category string
	Start    time.Time
	End      time.Time
	Limit    int32
	Offset   int32
}

type ListEventsResult struct {
	Events     []Event `json:"events"`
	Total      int64   `json:"total"`
	NextOffset int32   `json:"next_offset"`
}

func NewService(queries *db.Queries) *Service {
	return &Service{queries: queries}
}

func (s *Service) GetTenantPolicy(ctx context.Context, tenantID uuid.UUID) (Policy, error) {
	if s == nil || s.queries == nil {
		return Policy{}, nil
	}
	row, err := s.queries.GetTenantGuardrailPolicy(ctx, toPgUUID(tenantID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Policy{TenantID: tenantID, Config: map[string]any{}}, nil
		}
		return Policy{}, err
	}
	return convertPolicy(row)
}

func (s *Service) UpsertTenantPolicy(ctx context.Context, tenantID uuid.UUID, config map[string]any) (Policy, error) {
	if config == nil {
		config = map[string]any{}
	}
	payload, _ := json.Marshal(config)
	row, err := s.queries.UpsertTenantGuardrailPolicy(ctx, db.UpsertTenantGuardrailPolicyParams{
		TenantID:   toPgUUID(tenantID),
		ConfigJson: payload,
	})
	if err != nil {
		return Policy{}, err
	}
	return convertPolicy(row)
}

func (s *Service) DeleteTenantPolicy(ctx context.Context, tenantID uuid.UUID) error {
	if s == nil || s.queries == nil {
		return nil
	}
	return s.queries.DeleteTenantGuardrailPolicy(ctx, toPgUUID(tenantID))
}

func (s *Service) GetAPIKeyPolicy(ctx context.Context, apiKeyID uuid.UUID) (Policy, error) {
	if s == nil || s.queries == nil {
		return Policy{}, nil
	}
	row, err := s.queries.GetAPIKeyGuardrailPolicy(ctx, toPgUUID(apiKeyID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Policy{APIKeyID: apiKeyID, Config: map[string]any{}}, nil
		}
		return Policy{}, err
	}
	return convertPolicy(row)
}

func (s *Service) UpsertAPIKeyPolicy(ctx context.Context, apiKeyID uuid.UUID, config map[string]any) (Policy, error) {
	if config == nil {
		config = map[string]any{}
	}
	payload, _ := json.Marshal(config)
	row, err := s.queries.UpsertAPIKeyGuardrailPolicy(ctx, db.UpsertAPIKeyGuardrailPolicyParams{
		ApiKeyID:   toPgUUID(apiKeyID),
		ConfigJson: payload,
	})
	if err != nil {
		return Policy{}, err
	}
	return convertPolicy(row)
}

func (s *Service) DeleteAPIKeyPolicy(ctx context.Context, apiKeyID uuid.UUID) error {
	if s == nil || s.queries == nil {
		return nil
	}
	return s.queries.DeleteAPIKeyGuardrailPolicy(ctx, toPgUUID(apiKeyID))
}

func (s *Service) EffectivePolicy(ctx context.Context, tenantID, apiKeyID uuid.UUID) (map[string]any, error) {
	merged := map[string]any{}
	if tenantID != uuid.Nil {
		policy, err := s.GetTenantPolicy(ctx, tenantID)
		if err != nil {
			return nil, err
		}
		merged = mergeConfigMaps(merged, policy.Config)
	}
	if apiKeyID != uuid.Nil {
		policy, err := s.GetAPIKeyPolicy(ctx, apiKeyID)
		if err != nil {
			return nil, err
		}
		merged = mergeConfigMaps(merged, policy.Config)
	}
	return merged, nil
}

func (s *Service) RecordEvent(ctx context.Context, tenantID, apiKeyID uuid.UUID, alias, action, category string, details map[string]any) error {
	if s == nil || s.queries == nil {
		return nil
	}
	if details == nil {
		details = map[string]any{}
	}
	payload, _ := json.Marshal(details)
	_, err := s.queries.InsertGuardrailEvent(ctx, db.InsertGuardrailEventParams{
		TenantID:   toPgUUID(tenantID),
		ApiKeyID:   toPgUUID(apiKeyID),
		ModelAlias: toPgText(alias),
		Action:     strings.TrimSpace(action),
		Category:   toPgText(category),
		Details:    payload,
	})
	return err
}

func (s *Service) ListEvents(ctx context.Context, params ListEventsParams) (ListEventsResult, error) {
	if s == nil || s.queries == nil {
		return ListEventsResult{}, nil
	}
	limit := params.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	offset := params.Offset
	if offset < 0 {
		offset = 0
	}
	rows, err := s.queries.ListGuardrailEvents(ctx, db.ListGuardrailEventsParams{
		Column1: toPgUUID(params.TenantID),
		Column2: toPgUUID(params.APIKeyID),
		Column3: strings.TrimSpace(params.Action),
		Column4: strings.TrimSpace(params.Stage),
		Column5: strings.TrimSpace(params.Category),
		Column6: toPgTimestamptz(params.Start),
		Column7: toPgTimestamptz(params.End),
		Limit:   limit,
		Offset:  offset,
	})
	if err != nil {
		return ListEventsResult{}, err
	}
	events := make([]Event, 0, len(rows))
	var total int64
	for _, row := range rows {
		evt, convertErr := convertEventRow(row)
		if convertErr != nil {
			return ListEventsResult{}, convertErr
		}
		events = append(events, evt)
		total = row.TotalRows
	}
	return ListEventsResult{
		Events:     events,
		Total:      total,
		NextOffset: offset + int32(len(events)),
	}, nil
}

func mergeConfigMaps(base, override map[string]any) map[string]any {
	out := make(map[string]any, len(base)+len(override))
	for k, v := range base {
		out[k] = v
	}
	for k, v := range override {
		if existing, ok := out[k]; ok {
			if existingMap, ok := existing.(map[string]any); ok {
				if vMap, ok := v.(map[string]any); ok {
					out[k] = mergeConfigMaps(existingMap, vMap)
					continue
				}
			}
		}
		out[k] = v
	}
	return out
}

func toPgText(value string) pgtype.Text {
	value = strings.TrimSpace(value)
	if value == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: value, Valid: true}
}

func convertPolicy(row db.GuardrailPolicy) (Policy, error) {
	policy := Policy{Config: map[string]any{}}
	if row.TenantID.Valid {
		if id, err := uuid.FromBytes(row.TenantID.Bytes[:]); err == nil {
			policy.TenantID = id
		}
	}
	if row.ApiKeyID.Valid {
		if id, err := uuid.FromBytes(row.ApiKeyID.Bytes[:]); err == nil {
			policy.APIKeyID = id
		}
	}
	if len(row.ConfigJson) > 0 {
		_ = json.Unmarshal(row.ConfigJson, &policy.Config)
	}
	return policy, nil
}

func toPgUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: id != uuid.Nil}
}

func convertEventRow(row db.ListGuardrailEventsRow) (Event, error) {
	var evt Event
	if !row.ID.Valid {
		return evt, errors.New("event id missing")
	}
	id, err := uuid.FromBytes(row.ID.Bytes[:])
	if err != nil {
		return evt, err
	}
	evt.ID = id
	if tenantID, err := uuidPtrFromPg(row.TenantID); err == nil {
		evt.TenantID = tenantID
	} else {
		return evt, err
	}
	if apiKeyID, err := uuidPtrFromPg(row.ApiKeyID); err == nil {
		evt.APIKeyID = apiKeyID
	} else {
		return evt, err
	}
	evt.TenantName = row.TenantName.String
	evt.APIKeyName = row.ApiKeyName.String
	if row.ModelAlias.Valid {
		evt.ModelAlias = row.ModelAlias.String
	}
	evt.Action = strings.TrimSpace(row.Action)
	if row.Category.Valid {
		evt.Category = row.Category.String
	}
	if row.CreatedAt.Valid {
		evt.CreatedAt = row.CreatedAt.Time
	}
	if len(row.Details) > 0 {
		_ = json.Unmarshal(row.Details, &evt.Details)
	}
	if evt.Details == nil {
		evt.Details = map[string]any{}
	}
	if stage, ok := evt.Details["stage"].(string); ok {
		evt.Stage = stage
	}
	if rawViolations, ok := evt.Details["violations"]; ok {
		evt.Violations = toStringSlice(rawViolations)
	}
	return evt, nil
}

func uuidPtrFromPg(value pgtype.UUID) (*uuid.UUID, error) {
	if !value.Valid {
		return nil, nil
	}
	id, err := uuid.FromBytes(value.Bytes[:])
	if err != nil {
		return nil, err
	}
	return &id, nil
}

func toStringSlice(value any) []string {
	switch v := value.(type) {
	case []string:
		return v
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if str, ok := item.(string); ok {
				out = append(out, str)
			}
		}
		return out
	default:
		return nil
	}
}

func toPgTimestamptz(t time.Time) pgtype.Timestamptz {
	if t.IsZero() {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: t, Valid: true}
}
