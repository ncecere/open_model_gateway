package batches

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ncecere/open_model_gateway/backend/internal/config"
	"github.com/ncecere/open_model_gateway/backend/internal/db"
	filesvc "github.com/ncecere/open_model_gateway/backend/internal/services/files"
)

var (
	ErrUnsupportedEndpoint = errors.New("unsupported batch endpoint")
	ErrFilePurposeMismatch = errors.New("file purpose must be batch")
)

// Service orchestrates batch metadata and ingestion.
type Service struct {
	pool    *pgxpool.Pool
	queries *db.Queries
	files   *filesvc.Service
	cfg     *config.BatchesConfig
}

func NewService(pool *pgxpool.Pool, queries *db.Queries, files *filesvc.Service, cfg *config.BatchesConfig) *Service {
	return &Service{pool: pool, queries: queries, files: files, cfg: cfg}
}

type CreateParams struct {
	TenantID         uuid.UUID
	APIKeyID         uuid.UUID
	Endpoint         string
	CompletionWindow string
	InputFileID      uuid.UUID
	Metadata         map[string]string
	MaxConcurrency   int
}

type Batch struct {
	ID                    uuid.UUID
	TenantID              uuid.UUID
	APIKeyID              uuid.UUID
	Status                string
	Endpoint              string
	CompletionWindow      string
	InputFileID           uuid.UUID
	ResultFileID          *uuid.UUID
	ErrorFileID           *uuid.UUID
	MaxConcurrency        int
	Metadata              map[string]string
	RequestCountTotal     int
	RequestCountCompleted int
	RequestCountFailed    int
	RequestCountCancelled int
	CreatedAt             time.Time
	UpdatedAt             time.Time
	InProgressAt          *time.Time
	CompletedAt           *time.Time
	CancelledAt           *time.Time
	FinalizingAt          *time.Time
	FailedAt              *time.Time
	ExpiresAt             *time.Time
}

// BatchWithTenant augments Batch records with tenant metadata for admin views.
type BatchWithTenant struct {
	Batch
	TenantName string
}

type batchInput struct {
	CustomID string          `json:"custom_id"`
	Method   string          `json:"method"`
	URL      string          `json:"url"`
	Body     json.RawMessage `json:"body"`
	Headers  json.RawMessage `json:"headers"`
}

func (s *Service) Create(ctx context.Context, params CreateParams) (Batch, error) {
	if params.TenantID == uuid.Nil || params.APIKeyID == uuid.Nil {
		return Batch{}, fmt.Errorf("tenant and api key required")
	}
	slog.Info("batch create params", "tenant_id", params.TenantID.String(), "api_key_id", params.APIKeyID.String())
	endpoint := sanitizeEndpoint(params.Endpoint)
	if endpoint == "" {
		return Batch{}, ErrUnsupportedEndpoint
	}

	reader, rec, err := s.files.Open(ctx, params.TenantID, params.InputFileID)
	if err != nil {
		return Batch{}, err
	}
	defer reader.Close()
	if rec.Purpose != filesvc.PurposeBatch {
		return Batch{}, ErrFilePurposeMismatch
	}

	entries, err := s.readBatchFile(reader, endpoint)
	if err != nil {
		return Batch{}, err
	}
	if len(entries) == 0 {
		return Batch{}, fmt.Errorf("batch file contained no valid requests")
	}
	if len(entries) > s.cfg.MaxRequests {
		return Batch{}, fmt.Errorf("batch exceeds max of %d requests", s.cfg.MaxRequests)
	}

	ttl := s.cfg.DefaultTTL
	if ttl <= 0 {
		ttl = 168 * time.Hour
	}
	if s.cfg.MaxTTL > 0 && ttl > s.cfg.MaxTTL {
		ttl = s.cfg.MaxTTL
	}
	expiresAt := time.Now().Add(ttl)

	completionWindow := strings.TrimSpace(params.CompletionWindow)
	if completionWindow == "" {
		completionWindow = "24h"
	}

	maxConcurrency := params.MaxConcurrency
	if maxConcurrency <= 0 || maxConcurrency > s.cfg.MaxConcurrency {
		maxConcurrency = s.cfg.MaxConcurrency
		if maxConcurrency <= 0 {
			maxConcurrency = 10
		}
	}

	metadata := map[string]string{}
	for k, v := range params.Metadata {
		metadata[k] = v
	}
	metadataJSON, _ := json.Marshal(metadata)

	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Batch{}, err
	}
	defer tx.Rollback(ctx)

	qtx := s.queries.WithTx(tx)

	batchRow, err := qtx.CreateBatch(ctx, db.CreateBatchParams{
		TenantID:          toPgUUID(params.TenantID),
		ApiKeyID:          toPgUUID(params.APIKeyID),
		Status:            "queued",
		Endpoint:          endpoint,
		InputFileID:       toNullableUUID(params.InputFileID),
		CompletionWindow:  pgtype.Text{String: completionWindow, Valid: true},
		MaxConcurrency:    int32(maxConcurrency),
		Metadata:          metadataJSON,
		RequestCountTotal: int32(len(entries)),
		ExpiresAt:         toPgTime(expiresAt),
	})
	if err != nil {
		return Batch{}, err
	}

	for idx, entry := range entries {
		payload, _ := json.Marshal(entry)
		_, err := qtx.InsertBatchItem(ctx, db.InsertBatchItemParams{
			BatchID:   batchRow.ID,
			ItemIndex: int64(idx),
			Status:    "queued",
			CustomID: pgtype.Text{
				String: entry.CustomID,
				Valid:  strings.TrimSpace(entry.CustomID) != "",
			},
			Input: payload,
		})
		if err != nil {
			return Batch{}, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return Batch{}, err
	}

	return toBatch(batchRow)
}

func (s *Service) List(ctx context.Context, tenantID uuid.UUID, limit, offset int32) ([]Batch, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.queries.ListBatches(ctx, db.ListBatchesParams{
		TenantID: toPgUUID(tenantID),
		Limit:    limit,
		Offset:   offset,
	})
	if err != nil {
		return nil, err
	}
	out := make([]Batch, 0, len(rows))
	for _, row := range rows {
		rec, err := toBatch(row)
		if err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	return out, nil
}

// ListAll returns paginated batches across all tenants with optional filters.
func (s *Service) ListAll(ctx context.Context, tenantID *uuid.UUID, statuses []string, search string, limit, offset int32) ([]BatchWithTenant, int64, error) {
	if s.queries == nil {
		return nil, 0, errors.New("batch queries unavailable")
	}
	if limit <= 0 {
		limit = 50
	}
	params := db.ListBatchesAdminParams{
		TenantID:   toOptionalUUID(tenantID),
		Statuses:   statuses,
		PageLimit:  limit,
		PageOffset: offset,
	}
	if trimmed := strings.TrimSpace(search); trimmed != "" {
		params.Search = pgtype.Text{String: trimmed, Valid: true}
	}
	rows, err := s.queries.ListBatchesAdmin(ctx, params)
	if err != nil {
		return nil, 0, err
	}
	result := make([]BatchWithTenant, 0, len(rows))
	var total int64
	for _, row := range rows {
		batch, convErr := toBatchFromAdminRow(row)
		if convErr != nil {
			return nil, 0, convErr
		}
		result = append(result, BatchWithTenant{
			Batch:      batch,
			TenantName: row.TenantName,
		})
		total = row.TotalCount
	}
	return result, total, nil
}

func (s *Service) Get(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) (Batch, error) {
	row, err := s.queries.GetBatch(ctx, db.GetBatchParams{
		TenantID: toPgUUID(tenantID),
		ID:       toPgUUID(id),
	})
	if err != nil {
		return Batch{}, err
	}
	return toBatch(row)
}

func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (Batch, error) {
	row, err := s.queries.GetBatchByID(ctx, toPgUUID(id))
	if err != nil {
		return Batch{}, err
	}
	return toBatch(row)
}

func (s *Service) CancelByID(ctx context.Context, id uuid.UUID) (Batch, error) {
	record, err := s.GetByID(ctx, id)
	if err != nil {
		return Batch{}, err
	}
	return s.Cancel(ctx, record.TenantID, id)
}

func (s *Service) Cancel(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) (Batch, error) {
	row, err := s.queries.CancelBatch(ctx, db.CancelBatchParams{
		TenantID: toPgUUID(tenantID),
		ID:       toPgUUID(id),
		Status:   "cancelled",
	})
	if err != nil {
		return Batch{}, err
	}
	return toBatch(row)
}

func (s *Service) readBatchFile(r io.Reader, endpoint string) ([]batchInput, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	var inputs []batchInput
	line := 0
	for scanner.Scan() {
		line++
		raw := strings.TrimSpace(scanner.Text())
		if raw == "" {
			continue
		}
		var entry batchInput
		if err := json.Unmarshal([]byte(raw), &entry); err != nil {
			return nil, fmt.Errorf("line %d: %w", line, err)
		}
		if strings.ToUpper(entry.Method) != "POST" {
			return nil, fmt.Errorf("line %d: method must be POST", line)
		}
		if sanitizeEndpoint(entry.URL) != endpoint {
			return nil, fmt.Errorf("line %d: url mismatch with endpoint", line)
		}
		if len(entry.Body) == 0 {
			return nil, fmt.Errorf("line %d: body required", line)
		}
		inputs = append(inputs, entry)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return inputs, nil
}

func sanitizeEndpoint(endpoint string) string {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return ""
	}
	if !strings.HasPrefix(endpoint, "/") {
		endpoint = "/" + endpoint
	}
	switch endpoint {
	case "/v1/chat/completions", "/v1/embeddings", "/v1/images/generations":
		return endpoint
	default:
		return ""
	}
}

func toBatch(row db.Batch) (Batch, error) {
	id, err := fromPgUUID(row.ID)
	if err != nil {
		return Batch{}, err
	}
	tenantID, err := fromPgUUID(row.TenantID)
	if err != nil {
		return Batch{}, err
	}
	apiKeyID, err := fromPgUUID(row.ApiKeyID)
	if err != nil {
		return Batch{}, err
	}

	batch := Batch{
		ID:                    id,
		TenantID:              tenantID,
		APIKeyID:              apiKeyID,
		Status:                row.Status,
		Endpoint:              row.Endpoint,
		CompletionWindow:      row.CompletionWindow.String,
		MaxConcurrency:        int(row.MaxConcurrency),
		Metadata:              map[string]string{},
		RequestCountTotal:     int(row.RequestCountTotal),
		RequestCountCompleted: int(row.RequestCountCompleted),
		RequestCountFailed:    int(row.RequestCountFailed),
		RequestCountCancelled: int(row.RequestCountCancelled),
		CreatedAt:             row.CreatedAt.Time,
		UpdatedAt:             row.UpdatedAt.Time,
	}

	if len(row.Metadata) > 0 {
		_ = json.Unmarshal(row.Metadata, &batch.Metadata)
	}
	if row.InputFileID.Valid {
		if val, err := fromPgUUID(row.InputFileID); err == nil {
			batch.InputFileID = val
		}
	}
	if row.ResultFileID.Valid {
		if val, err := fromPgUUID(row.ResultFileID); err == nil {
			batch.ResultFileID = &val
		}
	}
	if row.ErrorFileID.Valid {
		if val, err := fromPgUUID(row.ErrorFileID); err == nil {
			batch.ErrorFileID = &val
		}
	}
	if row.InProgressAt.Valid {
		t := row.InProgressAt.Time
		batch.InProgressAt = &t
	}
	if row.CompletedAt.Valid {
		t := row.CompletedAt.Time
		batch.CompletedAt = &t
	}
	if row.CancelledAt.Valid {
		t := row.CancelledAt.Time
		batch.CancelledAt = &t
	}
	if row.FinalizingAt.Valid {
		t := row.FinalizingAt.Time
		batch.FinalizingAt = &t
	}
	if row.FailedAt.Valid {
		t := row.FailedAt.Time
		batch.FailedAt = &t
	}
	if row.ExpiresAt.Valid {
		t := row.ExpiresAt.Time
		batch.ExpiresAt = &t
	}

	return batch, nil
}

func toBatchFromAdminRow(row db.ListBatchesAdminRow) (Batch, error) {
	return toBatch(db.Batch{
		ID:                    row.ID,
		TenantID:              row.TenantID,
		ApiKeyID:              row.ApiKeyID,
		Status:                row.Status,
		Endpoint:              row.Endpoint,
		InputFileID:           row.InputFileID,
		ResultFileID:          row.ResultFileID,
		ErrorFileID:           row.ErrorFileID,
		CompletionWindow:      row.CompletionWindow,
		MaxConcurrency:        row.MaxConcurrency,
		Metadata:              row.Metadata,
		RequestCountTotal:     row.RequestCountTotal,
		RequestCountCompleted: row.RequestCountCompleted,
		RequestCountFailed:    row.RequestCountFailed,
		RequestCountCancelled: row.RequestCountCancelled,
		CreatedAt:             row.CreatedAt,
		UpdatedAt:             row.UpdatedAt,
		InProgressAt:          row.InProgressAt,
		CompletedAt:           row.CompletedAt,
		CancelledAt:           row.CancelledAt,
		FinalizingAt:          row.FinalizingAt,
		FailedAt:              row.FailedAt,
		ExpiresAt:             row.ExpiresAt,
	})
}

// ClaimNextBatch finds the oldest queued batch and marks it in progress atomically.
func (s *Service) ClaimNextBatch(ctx context.Context) (Batch, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Batch{}, err
	}
	defer tx.Rollback(ctx)

	qtx := s.queries.WithTx(tx)
	row, err := qtx.GetOldestQueuedBatch(ctx)
	if err != nil {
		return Batch{}, err
	}
	claimed, err := qtx.MarkBatchInProgress(ctx, row.ID)
	if err != nil {
		return Batch{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Batch{}, err
	}
	return toBatch(claimed)
}

// ClaimNextItem locks and transitions the next queued batch item to running state.
func (s *Service) ClaimNextItem(ctx context.Context, batchID uuid.UUID) (db.BatchItem, error) {
	return s.queries.ClaimNextBatchItem(ctx, toPgUUID(batchID))
}

// CompleteItem marks the specified batch item as completed and stores the response payload.
func (s *Service) CompleteItem(ctx context.Context, itemID uuid.UUID, payload []byte) error {
	return s.queries.CompleteBatchItem(ctx, db.CompleteBatchItemParams{
		ID:       toPgUUID(itemID),
		Response: payload,
	})
}

// FailItem records a failed batch item with the provided error payload.
func (s *Service) FailItem(ctx context.Context, itemID uuid.UUID, payload []byte) error {
	return s.queries.FailBatchItem(ctx, db.FailBatchItemParams{
		ID:    toPgUUID(itemID),
		Error: payload,
	})
}

// IncrementCounts adjusts the aggregate batch counters.
func (s *Service) IncrementCounts(ctx context.Context, batchID uuid.UUID, completed, failed, cancelled int) error {
	return s.queries.IncrementBatchCounts(ctx, db.IncrementBatchCountsParams{
		ID:                    toPgUUID(batchID),
		RequestCountCompleted: int32(completed),
		RequestCountFailed:    int32(failed),
		RequestCountCancelled: int32(cancelled),
	})
}

// FinalizeBatch updates the terminal status and links output/error files.
func (s *Service) FinalizeBatch(ctx context.Context, batchID uuid.UUID, status string, resultFileID, errorFileID *uuid.UUID) (Batch, error) {
	params := db.MarkBatchFinalStatusParams{
		ID:     toPgUUID(batchID),
		Status: status,
	}
	if resultFileID != nil {
		params.ResultFileID = toNullableUUID(*resultFileID)
	}
	if errorFileID != nil {
		params.ErrorFileID = toNullableUUID(*errorFileID)
	}
	row, err := s.queries.MarkBatchFinalStatus(ctx, params)
	if err != nil {
		return Batch{}, err
	}
	return toBatch(row)
}

func toPgUUID(id uuid.UUID) pgtype.UUID {
	var out pgtype.UUID
	if id == uuid.Nil {
		return pgtype.UUID{Valid: false}
	}

	copy(out.Bytes[:], id[:])
	out.Valid = true
	return out
}

func toNullableUUID(id uuid.UUID) pgtype.UUID {
	if id == uuid.Nil {
		return pgtype.UUID{Valid: false}
	}
	return toPgUUID(id)
}

func toOptionalUUID(id *uuid.UUID) pgtype.UUID {
	if id == nil {
		return pgtype.UUID{Valid: false}
	}
	return toPgUUID(*id)
}

func toPgTime(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}

func fromPgUUID(id pgtype.UUID) (uuid.UUID, error) {
	if !id.Valid {
		return uuid.UUID{}, fmt.Errorf("uuid invalid")
	}
	return uuid.FromBytes(id.Bytes[:])
}
