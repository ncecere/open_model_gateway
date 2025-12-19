package usagepipeline

import (
	"context"
	"errors"
	"log/slog"
	"math"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	decimal "github.com/shopspring/decimal"

	"github.com/ncecere/open_model_gateway/backend/internal/config"
	"github.com/ncecere/open_model_gateway/backend/internal/db"
	"github.com/ncecere/open_model_gateway/backend/internal/models"
	"github.com/ncecere/open_model_gateway/backend/internal/observability"
	"github.com/ncecere/open_model_gateway/backend/internal/pricing"
	"github.com/ncecere/open_model_gateway/backend/internal/requestctx"
)

// Logger persists request and usage records while enforcing configured budgets.
type Logger struct {
	recorder *UsageRecorder
	budgets  *BudgetEvaluator
	alerts   *AlertDispatcher
	metrics  *observability.Provider

	pricing          *pricing.Cache
	remainderMu      sync.Mutex
	tenantRemainders map[uuid.UUID]decimal.Decimal
	events           chan usageEvent
	cancel           context.CancelFunc
	processorWG      sync.WaitGroup
	shuttingDown     atomic.Bool
}

type usageEvent struct {
	record     Record
	timestamp  time.Time
	costCents  int64
	costMicros int64
	status     BudgetStatus
	attempts   int
}

const (
	defaultQueueSize      = 1024
	maxPersistAttempts    = 3
	persistRetryBackoff   = 150 * time.Millisecond
	defaultPersistTimeout = 5 * time.Second
)

type alertSnapshot struct {
	Level AlertLevel
	Sent  time.Time
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
func NewLogger(ctx context.Context, pool *pgxpool.Pool, queries *db.Queries, cfg config.BudgetConfig, sink AlertSink, metrics *observability.Provider) *Logger {
	if ctx == nil {
		ctx = context.Background()
	}
	runCtx, cancel := context.WithCancel(ctx)
	l := &Logger{
		recorder:         NewUsageRecorder(pool, queries),
		budgets:          NewBudgetEvaluator(cfg, queries, metrics),
		alerts:           NewAlertDispatcher(queries, sink),
		metrics:          metrics,
		pricing:          pricing.NewCache(),
		tenantRemainders: make(map[uuid.UUID]decimal.Decimal),
		events:           make(chan usageEvent, defaultQueueSize),
		cancel:           cancel,
	}
	l.processorWG.Add(1)
	go func() {
		defer l.processorWG.Done()
		l.run(runCtx)
	}()
	return l
}

// LoadCatalog seeds or refreshes the in-memory pricing cache.
func (l *Logger) LoadCatalog(entries []config.ModelCatalogEntry) {
	if l == nil {
		return
	}
	if l.pricing == nil {
		l.pricing = pricing.NewCache()
	}
	l.pricing.Load(entries)
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

	schedule := l.budgets.Schedule(rec.Context)
	total, err := l.budgets.SumUsage(ctx, rec.Context.TenantID, ts, schedule)
	if err != nil {
		return BudgetStatus{}, err
	}
	if rec.Success {
		total += costCents
	}

	exceeded := total >= limit
	warning := !exceeded && overThreshold(total, limit, rec.Context.WarningThreshold)

	status := BudgetStatus{
		TotalCostCents: total,
		LimitCents:     limit,
		Warning:        warning,
		Exceeded:       exceeded,
	}

	l.enqueueEvent(usageEvent{
		record:     rec,
		timestamp:  ts,
		costCents:  costCents,
		costMicros: costMicros,
		status:     status,
	})
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
	if l == nil || l.pricing == nil {
		return decimal.Zero
	}
	params := pricing.Params{
		PromptTokens:     int64(usage.PromptTokens),
		CompletionTokens: int64(usage.CompletionTokens),
	}
	return l.pricing.Cost(alias, params)
}

func (l *Logger) enqueueEvent(evt usageEvent) {
	if l == nil {
		return
	}
	if l.shuttingDown.Load() {
		go l.persistEvent(context.Background(), evt)
		return
	}
	select {
	case l.events <- evt:
	default:
		// Backpressure: persist synchronously to avoid dropping records.
		go l.persistEvent(context.Background(), evt)
	}
}

func (l *Logger) run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			l.drainQueue()
			return
		case evt := <-l.events:
			l.persistEvent(context.Background(), evt)
		}
	}
}

func (l *Logger) drainQueue() {
	for {
		select {
		case evt := <-l.events:
			l.persistEvent(context.Background(), evt)
		default:
			return
		}
	}
}

func (l *Logger) persistEvent(ctx context.Context, evt usageEvent) {
	if l == nil {
		return
	}
	persistCtx, cancel := context.WithTimeout(ctx, defaultPersistTimeout)
	defer cancel()
	if err := l.recorder.Persist(persistCtx, evt.record, evt.timestamp, evt.costCents, evt.costMicros); err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			// attempt retry regardless
		}
		if evt.attempts < maxPersistAttempts && !l.shuttingDown.Load() {
			evt.attempts++
			time.AfterFunc(persistRetryBackoff*time.Duration(evt.attempts), func() {
				l.enqueueEvent(evt)
			})
		} else {
			slog.Error("persist usage record", slog.String("alias", evt.record.Alias), slog.String("provider", evt.record.Provider), slog.String("error", err.Error()))
		}
		return
	}

	if l.metrics != nil {
		tenantLabel := evt.record.Context.TenantID.String()
		l.metrics.RecordAPILatency(tenantLabel, evt.record.Alias, evt.record.Provider, evt.record.Status, evt.record.Latency)
		if evt.record.Success {
			l.metrics.RecordTokens(tenantLabel, evt.record.Alias, evt.record.Provider, int64(evt.record.Usage.PromptTokens), int64(evt.record.Usage.CompletionTokens))
		}
	}

	if err := l.alerts.Dispatch(context.Background(), evt.record, evt.status, evt.timestamp); err != nil {
		slog.Error("dispatch budget alert", slog.String("tenant_id", evt.record.Context.TenantID.String()), slog.String("error", err.Error()))
	}
}

// RunBackground starts the usage log processor.
// Shutdown stops the logger background processor and flushes pending events.
func (l *Logger) Shutdown(ctx context.Context) error {
	if l == nil {
		return nil
	}
	l.shuttingDown.Store(true)
	if l.cancel != nil {
		l.cancel()
	}
	done := make(chan struct{})
	go func() {
		l.processorWG.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
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
