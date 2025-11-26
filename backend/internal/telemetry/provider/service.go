package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/ncecere/open_model_gateway/backend/internal/config"
	"github.com/ncecere/open_model_gateway/backend/internal/db"
	"sync"
)

// Service orchestrates provider telemetry evaluation, persistence, and alerting hooks.
type Service struct {
	recorder   *Recorder
	evaluator  *Evaluator
	queries    *db.Queries
	alertSink  AlertSink
	dispatcher *IncidentDispatcher
	window     time.Duration
	retention  int
	logger     *slog.Logger
	downweight bool
	now        func() time.Time
	mu         sync.RWMutex
	degraded   map[string]struct{}
}

// AlertSink dispatches incident notifications.
type AlertSink interface {
	Notify(ctx context.Context, incident Incident) error
}

// DegradedRoutes returns a snapshot of current degraded route keys.
func (s *Service) DegradedRoutes() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]string, 0, len(s.degraded))
	for k := range s.degraded {
		out = append(out, k)
	}
	return out
}

// NewService wires the telemetry components into a background-friendly service.
func NewService(rec *Recorder, queries *db.Queries, cfg config.TelemetryConfig, budgetCfg config.BudgetConfig, alertSink AlertSink, logger *slog.Logger) *Service {
	thresh := func(id Identifier) SLIThresholds {
		return thresholdsFor(id, cfg.Provider.Defaults, cfg.Provider.Overrides)
	}
	return &Service{
		recorder:   rec,
		evaluator:  NewEvaluator(rec, thresh),
		queries:    queries,
		alertSink:  alertSink,
		dispatcher: NewIncidentDispatcher(ProviderAlertConfig(cfg, budgetCfg), alertSink),
		window:     cfg.Provider.WindowSize,
		retention:  cfg.Provider.IncidentRetentionDays,
		logger:     logger,
		downweight: cfg.Provider.DownweightWhenDegraded,
		now:        time.Now,
		degraded:   make(map[string]struct{}),
	}
}

// Tick runs a single evaluation cycle and returns newly opened incidents.
func (s *Service) Tick(ctx context.Context) ([]Incident, error) {
	if s == nil || s.recorder == nil || s.queries == nil {
		return nil, fmt.Errorf("telemetry service not initialized")
	}
	incidents, err := s.evaluator.Evaluate(ctx, s.window)
	if err != nil {
		return nil, err
	}
	var opened []Incident
	for _, inc := range incidents {
		dbIncident, err := s.persistIncident(ctx, inc)
		if err != nil {
			s.log().Error("persist incident", slog.String("provider", inc.Provider), slog.String("alias", inc.ModelAlias), slog.String("type", inc.Type), slog.String("err", err.Error()))
			continue
		}
		if inc.Status == IncidentOpen {
			opened = append(opened, dbIncident)
			if s.dispatcher != nil {
				if err := s.dispatcher.Dispatch(ctx, dbIncident); err != nil {
					s.log().Warn("alert dispatch failed", slog.String("provider", inc.Provider), slog.String("alias", inc.ModelAlias), slog.String("type", inc.Type), slog.String("err", err.Error()))
				}
			}
		}
	}
	s.updateDegraded(incidents)
	if s.retention > 0 {
		if err := s.queries.PurgeProviderIncidents(ctx, s.retention); err != nil {
			s.log().Warn("purge incidents failed", slog.String("err", err.Error()))
		}
	}
	return opened, nil
}

// IsDegraded returns true when a route key is marked unhealthy.
func (s *Service) IsDegraded(alias, routeKey string) bool {
	if s == nil {
		return false
	}
	key := alias + "::" + routeKey
	s.mu.RLock()
	_, ok := s.degraded[key]
	s.mu.RUnlock()
	return ok
}

func (s *Service) updateDegraded(incidents []Incident) {
	s.mu.Lock()
	defer s.mu.Unlock()
	next := make(map[string]struct{})
	for _, inc := range incidents {
		if inc.Status != IncidentOpen {
			continue
		}
		routeKey := inc.Route
		if routeKey == "" {
			continue
		}
		key := inc.ModelAlias + "::" + routeKey
		next[key] = struct{}{}
	}
	s.degraded = next
}

// SLIResult summarizes current window metrics and thresholds.
type SLIResult struct {
	Identifier
	WindowSeconds int
	WindowStart   time.Time
	WindowEnd     time.Time
	RequestCount  int64
	ErrorCount    int64
	TimeoutCount  int64
	LatencyP95Ms  int64
	ErrorRate     float64
	TimeoutRate   float64
	Thresholds    SLIThresholds
}

// CurrentSLIs returns the latest SLI rollups for the configured window.
func (s *Service) CurrentSLIs(ctx context.Context) ([]SLIResult, error) {
	if s == nil || s.recorder == nil || s.evaluator == nil {
		return nil, fmt.Errorf("telemetry service not initialized")
	}
	now := s.now().UTC()
	summaries, err := s.recorder.AggregateWindow(ctx, s.window, now)
	if err != nil {
		return nil, err
	}
	windowStart := now.Add(-s.window).UTC()
	results := make([]SLIResult, 0, len(summaries))
	for id, sum := range summaries {
		p95 := percentileFromBuckets(sum.LatencyBuckets, sum.LatencyCount)
		errorRate := rateFloat(sum.Errors, sum.Requests)
		timeoutRate := rateFloat(sum.Timeouts, sum.Requests)
		results = append(results, SLIResult{
			Identifier:    id,
			WindowSeconds: int(s.window.Seconds()),
			WindowStart:   windowStart,
			WindowEnd:     now,
			RequestCount:  sum.Requests,
			ErrorCount:    sum.Errors,
			TimeoutCount:  sum.Timeouts,
			LatencyP95Ms:  p95,
			ErrorRate:     errorRate,
			TimeoutRate:   timeoutRate,
			Thresholds:    s.evaluator.thresholds(id),
		})
	}
	return results, nil
}

func (s *Service) persistIncident(ctx context.Context, inc Incident) (Incident, error) {
	meta, _ := json.Marshal(inc.Metadata)
	params := db.UpsertProviderIncidentParams{
		Provider:        inc.Provider,
		ModelAlias:      inc.ModelAlias,
		IncidentType:    inc.Type,
		Status:          string(inc.Status),
		WindowSeconds:   int32(inc.WindowSeconds),
		OpenedAt:        ToPgTime(inc.OpenedAt),
		ResolvedAt:      ToPgTimePtr(inc.ResolvedAt),
		WindowStartedAt: ToPgTime(inc.WindowStart),
		WindowEndedAt:   ToPgTime(inc.WindowEnd),
		SampleError:     ToPgText(inc.SampleError),
		RequestCount:    int32(inc.RequestCount),
		ErrorCount:      int32(inc.ErrorCount),
		TimeoutCount:    int32(inc.TimeoutCount),
		LatencyP95Ms:    ToPgInt4(inc.LatencyP95Ms),
		Metadata:        meta,
	}
	row, err := s.queries.UpsertProviderIncident(ctx, params)
	if err != nil {
		return inc, err
	}
	incDB := inc
	incDB.ID = row.ID.String()
	incDB.OpenedAt = row.OpenedAt.Time
	if row.ResolvedAt.Valid {
		resolved := row.ResolvedAt.Time
		incDB.ResolvedAt = &resolved
	}
	return incDB, nil
}

func (s *Service) log() *slog.Logger {
	if s.logger != nil {
		return s.logger
	}
	return slog.Default()
}

func rateFloat(num, denom int64) float64 {
	if denom == 0 {
		return 0
	}
	return float64(num) / float64(denom)
}

func ToPgTime(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}

func ToPgTimePtr(t *time.Time) pgtype.Timestamptz {
	if t == nil {
		return pgtype.Timestamptz{Valid: false}
	}
	return pgtype.Timestamptz{Time: *t, Valid: true}
}

func ToPgText(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: s, Valid: true}
}

func ToPgInt4(v int64) pgtype.Int4 {
	return pgtype.Int4{Int32: int32(v), Valid: v != 0}
}

// RedisAvailable returns nil when the recorder's Redis is reachable.
func (s *Service) RedisAvailable(ctx context.Context) error {
	if s == nil || s.recorder == nil || s.recorder.client == nil {
		return fmt.Errorf("redis client missing")
	}
	return s.recorder.client.Ping(ctx).Err()
}
