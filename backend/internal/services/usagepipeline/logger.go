package usagepipeline

import (
	"context"
	"errors"
	"log/slog"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	decimal "github.com/shopspring/decimal"

	"github.com/ncecere/open_model_gateway/backend/internal/config"
	"github.com/ncecere/open_model_gateway/backend/internal/db"
	"github.com/ncecere/open_model_gateway/backend/internal/models"
	"github.com/ncecere/open_model_gateway/backend/internal/observability"
	"github.com/ncecere/open_model_gateway/backend/internal/requestctx"
)

// Logger persists request and usage records while enforcing configured budgets.
type Logger struct {
	recorder *UsageRecorder
	budgets  *BudgetEvaluator
	alerts   *AlertDispatcher
	metrics  *observability.Provider

	priceMu          sync.RWMutex
	prices           map[string]priceInfo
	remainderMu      sync.Mutex
	tenantRemainders map[uuid.UUID]decimal.Decimal
}

type alertSnapshot struct {
	Level AlertLevel
	Sent  time.Time
}

type priceInfo struct {
	Input    decimal.Decimal
	Output   decimal.Decimal
	Currency string
}

func dollarsToCents(value float64) int64 {
	return int64(math.Round(value * 100))
}

// Record captures the outcome of a provider request.
type Record struct {
	Context           *requestctx.Context
	Alias             string
	Provider          string
	Usage             models.Usage
	Latency           time.Duration
	Status            int
	ErrorCode         string
	IdempotencyKey    string
	TraceID           string
	Timestamp         time.Time
	Success           bool
	OverrideCostCents *int64
}

// BudgetStatus reflects the tenant's budget posture after a request.
type BudgetStatus struct {
	TotalCostCents int64
	LimitCents     int64
	Warning        bool
	Exceeded       bool
}

// NewLogger constructs a usage logger using the shared pool and queries.
func NewLogger(pool *pgxpool.Pool, queries *db.Queries, cfg config.BudgetConfig, sink AlertSink, metrics *observability.Provider) *Logger {
	return &Logger{
		recorder:         NewUsageRecorder(pool, queries),
		budgets:          NewBudgetEvaluator(cfg, queries),
		alerts:           NewAlertDispatcher(queries, sink),
		metrics:          metrics,
		prices:           make(map[string]priceInfo),
		tenantRemainders: make(map[uuid.UUID]decimal.Decimal),
	}
}

// LoadCatalog seeds or refreshes the in-memory pricing cache.
func (l *Logger) LoadCatalog(entries []config.ModelCatalogEntry) {
	l.priceMu.Lock()
	defer l.priceMu.Unlock()

	for _, entry := range entries {
		l.prices[entry.Alias] = priceInfo{
			Input:    decimal.NewFromFloat(entry.PriceInput),
			Output:   decimal.NewFromFloat(entry.PriceOutput),
			Currency: entry.Currency,
		}
	}
}

// CheckBudget verifies whether the tenant can proceed before processing the request.
func (l *Logger) CheckBudget(ctx context.Context, rc *requestctx.Context, now time.Time) (BudgetStatus, error) {
	return l.budgets.Check(ctx, rc, now)
}

// Record persists request/usage details and returns the post-request budget status.
func (l *Logger) Record(ctx context.Context, rec Record) (BudgetStatus, error) {
	if rec.Context == nil {
		return BudgetStatus{}, errors.New("request context missing")
	}
	if rec.Alias == "" || rec.Provider == "" {
		return BudgetStatus{}, errors.New("model alias and provider required")
	}

	ts := rec.Timestamp
	if ts.IsZero() {
		ts = time.Now().UTC()
	}

	limit := l.budgets.EffectiveLimit(rec.Context)

	var costCents int64
	var costMicros int64
	if rec.Success {
		if rec.OverrideCostCents != nil {
			costCents = *rec.OverrideCostCents
			costMicros = *rec.OverrideCostCents * 10000 // convert cents to micros (1 cent = 10,000 micros)
		} else {
			costUSD := l.costFor(rec.Alias, rec.Usage)
			costCents = l.allocateCostCents(rec.Context.TenantID, costUSD)
			costMicros = usdToMicros(costUSD)
		}
	}

	if err := l.recorder.Persist(ctx, rec, ts, costCents, costMicros); err != nil {
		return BudgetStatus{}, err
	}
	if l.metrics != nil {
		tenantLabel := rec.Context.TenantID.String()
		l.metrics.RecordAPILatency(tenantLabel, rec.Alias, rec.Provider, rec.Status, rec.Latency)
		if rec.Success {
			l.metrics.RecordTokens(tenantLabel, rec.Alias, rec.Provider, int64(rec.Usage.PromptTokens), int64(rec.Usage.CompletionTokens))
		}
	}

	schedule := l.budgets.Schedule(rec.Context)
	total, err := l.budgets.SumUsage(ctx, rec.Context.TenantID, ts, schedule)
	if err != nil {
		return BudgetStatus{}, err
	}

	exceeded := total >= limit
	warning := !exceeded && overThreshold(total, limit, rec.Context.WarningThreshold)

	status := BudgetStatus{
		TotalCostCents: total,
		LimitCents:     limit,
		Warning:        warning,
		Exceeded:       exceeded,
	}

	if err := l.alerts.Dispatch(ctx, rec, status, ts); err != nil {
		slog.Error("dispatch budget alert", slog.String("tenant_id", rec.Context.TenantID.String()), slog.String("error", err.Error()))
	}

	return status, nil
}

func (l *Logger) insertRequest(ctx context.Context, q *db.Queries, rec Record, ts time.Time, costCents int64, costMicros int64) error {
	latency := rec.Latency.Milliseconds()
	if latency < 0 {
		latency = 0
	}

	_, err := q.InsertRequestRecord(ctx, db.InsertRequestRecordParams{
		TenantID:       toPgUUID(rec.Context.TenantID),
		ApiKeyID:       toPgNullableUUID(rec.Context.APIKeyID),
		Ts:             pgtype.Timestamptz{Time: ts, Valid: true},
		ModelAlias:     rec.Alias,
		Provider:       rec.Provider,
		LatencyMs:      int32(latency),
		Status:         int32(rec.Status),
		ErrorCode:      toPgText(rec.ErrorCode),
		InputTokens:    int64(rec.Usage.PromptTokens),
		OutputTokens:   int64(rec.Usage.CompletionTokens),
		CostCents:      costCents,
		CostUsdMicros:  costMicros,
		IdempotencyKey: toPgText(rec.IdempotencyKey),
		TraceID:        toPgText(rec.TraceID),
	})
	return err
}

func (l *Logger) insertUsage(ctx context.Context, q *db.Queries, rec Record, ts time.Time, costCents int64, costMicros int64) error {
	_, err := q.InsertUsageRecord(ctx, db.InsertUsageRecordParams{
		TenantID:      toPgUUID(rec.Context.TenantID),
		ApiKeyID:      toPgNullableUUID(rec.Context.APIKeyID),
		Ts:            pgtype.Timestamptz{Time: ts, Valid: true},
		ModelAlias:    rec.Alias,
		Provider:      rec.Provider,
		InputTokens:   int64(rec.Usage.PromptTokens),
		OutputTokens:  int64(rec.Usage.CompletionTokens),
		Requests:      1,
		CostCents:     costCents,
		CostUsdMicros: costMicros,
	})
	return err
}

// SetConfig swaps the budget configuration at runtime.
func (l *Logger) SetConfig(cfg config.BudgetConfig) {
	l.budgets.SetConfig(cfg)
}

func (l *Logger) costFor(alias string, usage models.Usage) decimal.Decimal {
	price := l.priceFor(alias)
	if price.Input.IsZero() && price.Output.IsZero() {
		return decimal.Zero
	}

	prompt := decimal.NewFromInt(int64(usage.PromptTokens))
	completion := decimal.NewFromInt(int64(usage.CompletionTokens))

	thousand := decimal.NewFromInt(1000)
	promptCost := price.Input.Mul(prompt).Div(thousand)
	completionCost := price.Output.Mul(completion).Div(thousand)
	totalUSD := promptCost.Add(completionCost)
	if totalUSD.IsNegative() {
		return decimal.Zero
	}
	return totalUSD
}

func (l *Logger) priceFor(alias string) priceInfo {
	l.priceMu.RLock()
	if info, ok := l.prices[alias]; ok {
		l.priceMu.RUnlock()
		return info
	}
	l.priceMu.RUnlock()
	return priceInfo{}
}

func (l *Logger) allocateCostCents(tenantID uuid.UUID, usd decimal.Decimal) int64 {
	if usd.IsZero() {
		return 0
	}
	cents := usd.Mul(decimal.NewFromInt(100))
	l.remainderMu.Lock()
	defer l.remainderMu.Unlock()

	remainder := l.tenantRemainders[tenantID]
	total := remainder.Add(cents)
	whole := total.Truncate(0)
	l.tenantRemainders[tenantID] = total.Sub(whole)
	if whole.IsZero() {
		return 0
	}
	return whole.IntPart()
}

func usdToMicros(value decimal.Decimal) int64 {
	if value.IsZero() {
		return 0
	}
	micros := value.Mul(decimal.NewFromInt(1_000_000)).Round(0)
	return micros.IntPart()
}

func overThreshold(total, limit int64, threshold float64) bool {
	if limit <= 0 || threshold <= 0 {
		return false
	}
	if threshold >= 1 {
		threshold = 0.99
	}
	return float64(total) >= threshold*float64(limit)
}

// periodBounds function moved to time_windows.go

func toInt64(val interface{}) int64 {
	switch v := val.(type) {
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
	case decimal.Decimal:
		return v.IntPart()
	case nil:
		return 0
	default:
		return 0
	}
}

func toPgUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: true}
}

func toPgNullableUUID(id uuid.UUID) pgtype.UUID {
	if id == uuid.Nil {
		return pgtype.UUID{}
	}
	return toPgUUID(id)
}

func toPgText(value string) pgtype.Text {
	if strings.TrimSpace(value) == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: value, Valid: true}
}
