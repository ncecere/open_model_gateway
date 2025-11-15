package usagepipeline

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ncecere/open_model_gateway/backend/internal/db"
	"github.com/ncecere/open_model_gateway/backend/internal/requestctx"
)

// AlertDispatcher coordinates alert cooldown tracking + persistence.
type AlertDispatcher struct {
	sink    AlertSink
	queries *db.Queries

	stateMu        sync.Mutex
	state          map[uuid.UUID]alertSnapshot
	guardrailMu    sync.Mutex
	guardrailState map[uuid.UUID]time.Time
}

func NewAlertDispatcher(queries *db.Queries, sink AlertSink) *AlertDispatcher {
	if sink == nil {
		sink = NewLogAlertSink(nil)
	}
	return &AlertDispatcher{
		sink:           sink,
		queries:        queries,
		state:          make(map[uuid.UUID]alertSnapshot),
		guardrailState: make(map[uuid.UUID]time.Time),
	}
}

func (a *AlertDispatcher) Dispatch(ctx context.Context, rec Record, status BudgetStatus, ts time.Time) error {
	rc := rec.Context
	if rc == nil {
		return nil
	}

	channels := AlertChannels{
		Emails:   rc.AlertEmails,
		Webhooks: rc.AlertWebhooks,
	}

	if !rc.AlertsEnabled || (len(channels.Emails) == 0 && len(channels.Webhooks) == 0) {
		return a.updateAlertState(ctx, rc, AlertLevelNone, time.Time{})
	}

	level := AlertLevelNone
	if status.Exceeded {
		level = AlertLevelExceeded
	} else if status.Warning {
		level = AlertLevelWarning
	} else {
		return a.updateAlertState(ctx, rc, AlertLevelNone, time.Time{})
	}

	state := a.loadAlertState(rc.TenantID, rc.AlertLastLevel, rc.AlertLastSent)
	cooldown := rc.AlertCooldown
	if cooldown <= 0 {
		cooldown = time.Hour
	}

	if !state.Sent.IsZero() {
		elapsed := ts.Sub(state.Sent)
		if elapsed < cooldown && alertSeverity(level) <= alertSeverity(state.Level) {
			return nil
		}
	}

	payload := AlertPayload{
		TenantID:     rc.TenantID,
		Level:        level,
		Status:       status,
		Channels:     channels,
		Timestamp:    ts,
		APIKeyPrefix: rc.APIKeyPrefix,
		ModelAlias:   rec.Alias,
	}

	err := a.sink.Notify(ctx, payload)
	a.recordAlertEvent(ctx, payload, err == nil, err)
	if err != nil {
		return err
	}

	return a.updateAlertState(ctx, rc, level, ts)
}

func (a *AlertDispatcher) loadAlertState(tenantID uuid.UUID, fallbackLevel string, fallbackTime time.Time) alertSnapshot {
	a.stateMu.Lock()
	defer a.stateMu.Unlock()

	if state, ok := a.state[tenantID]; ok {
		return state
	}

	return alertSnapshot{
		Level: parseAlertLevel(fallbackLevel),
		Sent:  fallbackTime,
	}
}

func (a *AlertDispatcher) storeAlertState(tenantID uuid.UUID, snapshot alertSnapshot) {
	a.stateMu.Lock()
	defer a.stateMu.Unlock()

	if snapshot.Level == AlertLevelNone || snapshot.Sent.IsZero() {
		delete(a.state, tenantID)
		return
	}

	a.state[tenantID] = snapshot
}

func (a *AlertDispatcher) updateAlertState(ctx context.Context, rc *requestctx.Context, level AlertLevel, ts time.Time) error {
	if rc == nil {
		return nil
	}

	snapshot := alertSnapshot{Level: level, Sent: ts}
	a.storeAlertState(rc.TenantID, snapshot)

	if !rc.HasBudgetOverride {
		return nil
	}

	params := db.UpdateTenantBudgetAlertStateParams{
		TenantID: toPgUUID(rc.TenantID),
	}
	if level == AlertLevelNone || ts.IsZero() {
		params.LastAlertAt = pgtype.Timestamptz{}
		params.LastAlertLevel = pgtype.Text{}
	} else {
		params.LastAlertAt = pgtype.Timestamptz{Time: ts, Valid: true}
		params.LastAlertLevel = pgtype.Text{String: string(level), Valid: true}
	}

	return a.queries.UpdateTenantBudgetAlertState(ctx, params)
}

func parseAlertLevel(value string) AlertLevel {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case string(AlertLevelExceeded):
		return AlertLevelExceeded
	case string(AlertLevelWarning):
		return AlertLevelWarning
	case string(AlertLevelGuardrail):
		return AlertLevelGuardrail
	default:
		return AlertLevelNone
	}
}

func alertSeverity(level AlertLevel) int {
	switch level {
	case AlertLevelExceeded:
		return 2
	case AlertLevelWarning:
		return 1
	case AlertLevelGuardrail:
		return 3
	default:
		return 0
	}
}

type GuardrailAlertInfo struct {
	Stage      string
	Action     string
	Category   string
	Violations []string
	ModelAlias string
}

func (a *AlertDispatcher) DispatchGuardrail(ctx context.Context, rc *requestctx.Context, info GuardrailAlertInfo) error {
	if a == nil || rc == nil {
		return nil
	}
	channels := AlertChannels{Emails: rc.AlertEmails, Webhooks: rc.AlertWebhooks}
	if !rc.AlertsEnabled || (len(channels.Emails) == 0 && len(channels.Webhooks) == 0) {
		return nil
	}
	cooldown := rc.AlertCooldown
	if cooldown <= 0 {
		cooldown = time.Hour
	}
	now := time.Now().UTC()
	last := a.lastGuardrailAlert(rc.TenantID)
	if !last.IsZero() && now.Sub(last) < cooldown {
		return nil
	}
	payload := AlertPayload{
		TenantID:     rc.TenantID,
		Level:        AlertLevelGuardrail,
		Channels:     channels,
		Timestamp:    now,
		APIKeyPrefix: rc.APIKeyPrefix,
		ModelAlias:   info.ModelAlias,
		Guardrail: &GuardrailAlert{
			Stage:      info.Stage,
			Action:     info.Action,
			Category:   info.Category,
			Violations: info.Violations,
		},
	}
	if err := a.sink.Notify(ctx, payload); err != nil {
		a.recordAlertEvent(ctx, payload, false, err)
		return err
	}
	a.recordAlertEvent(ctx, payload, true, nil)
	a.setLastGuardrailAlert(rc.TenantID, now)
	return nil
}

func (a *AlertDispatcher) lastGuardrailAlert(tenantID uuid.UUID) time.Time {
	a.guardrailMu.Lock()
	defer a.guardrailMu.Unlock()
	return a.guardrailState[tenantID]
}

func (a *AlertDispatcher) setLastGuardrailAlert(tenantID uuid.UUID, ts time.Time) {
	a.guardrailMu.Lock()
	defer a.guardrailMu.Unlock()
	if ts.IsZero() {
		delete(a.guardrailState, tenantID)
		return
	}
	a.guardrailState[tenantID] = ts
}

func (a *AlertDispatcher) recordAlertEvent(ctx context.Context, payload AlertPayload, success bool, notifyErr error) {
	if a == nil || a.queries == nil {
		return
	}
	channelsJSON, err := json.Marshal(payload.Channels)
	if err != nil {
		return
	}
	eventJSON, err := json.Marshal(alertEventBody{
		TenantID:     payload.TenantID.String(),
		Level:        payload.Level,
		Status:       payload.Status,
		APIKeyPrefix: payload.APIKeyPrefix,
		ModelAlias:   payload.ModelAlias,
		Timestamp:    payload.Timestamp.UTC(),
		Guardrail:    payload.Guardrail,
	})
	if err != nil {
		return
	}
	var errText pgtype.Text
	if notifyErr != nil {
		errText = toPgText(notifyErr.Error())
	}
	params := db.InsertBudgetAlertEventParams{
		TenantID: toPgUUID(payload.TenantID),
		Level:    string(payload.Level),
		Channels: channelsJSON,
		Payload:  eventJSON,
		Success:  success,
		Error:    errText,
	}
	_ = a.queries.InsertBudgetAlertEvent(ctx, params)
}

type alertEventBody struct {
	TenantID     string          `json:"tenant_id"`
	Level        AlertLevel      `json:"level"`
	Status       BudgetStatus    `json:"status"`
	APIKeyPrefix string          `json:"api_key_prefix"`
	ModelAlias   string          `json:"model_alias"`
	Timestamp    time.Time       `json:"timestamp"`
	Guardrail    *GuardrailAlert `json:"guardrail,omitempty"`
}
