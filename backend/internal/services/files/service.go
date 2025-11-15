package files

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ncecere/open_model_gateway/backend/internal/config"
	"github.com/ncecere/open_model_gateway/backend/internal/db"
	"github.com/ncecere/open_model_gateway/backend/internal/storage/blob"
)

const (
	StatusUploading = "uploading"
	StatusUploaded  = "uploaded"
	StatusProcessed = "processed"
	StatusError     = "error"
	StatusDeleted   = "deleted"

	PurposeFineTune         = "fine-tune"
	PurposeBatch            = "batch"
	PurposeAssist           = "assistants"
	PurposeAssistantsOutput = "assistants_output"
	PurposeVision           = "vision"
	PurposeModeration       = "moderation"
	PurposeResponses        = "responses"
	PurposeFineTuneResults  = "fine-tune-results"
)

var allowedPurposes = map[string]struct{}{
	PurposeFineTune:         {},
	PurposeBatch:            {},
	PurposeAssist:           {},
	PurposeAssistantsOutput: {},
	PurposeVision:           {},
	PurposeModeration:       {},
	PurposeResponses:        {},
	PurposeFineTuneResults:  {},
}

// Service coordinates file metadata + blob storage.
type Service struct {
	queries fileQueries
	store   blob.Store
	cfg     *config.FilesConfig
}

type fileQueries interface {
	CreateFile(context.Context, db.CreateFileParams) (db.File, error)
	GetFile(context.Context, db.GetFileParams) (db.File, error)
	DeleteFile(context.Context, db.DeleteFileParams) error
	ListFiles(context.Context, db.ListFilesParams) ([]db.File, error)
	ListFilesAdmin(context.Context, db.ListFilesAdminParams) ([]db.ListFilesAdminRow, error)
	GetFileByID(context.Context, pgtype.UUID) (db.File, error)
	GetTenantByID(context.Context, pgtype.UUID) (db.Tenant, error)
	ListExpiredFiles(context.Context, db.ListExpiredFilesParams) ([]db.File, error)
}

func NewService(queries fileQueries, store blob.Store, cfg *config.FilesConfig) *Service {
	return &Service{queries: queries, store: store, cfg: cfg}
}

// Upload stores the file and records metadata for the tenant.
type UploadParams struct {
	TenantID    uuid.UUID
	Filename    string
	Purpose     string
	ContentType string
	ContentLen  int64
	TTL         time.Duration
	Reader      io.Reader
}

type FileRecord struct {
	ID              uuid.UUID
	TenantID        uuid.UUID
	Filename        string
	Purpose         string
	ContentType     string
	Bytes           int64
	StorageKey      string
	StorageBackend  string
	Encrypted       bool
	Checksum        string
	ExpiresAt       time.Time
	CreatedAt       time.Time
	DeletedAt       *time.Time
	Status          string
	StatusDetails   string
	StatusUpdatedAt time.Time
}

type FileWithTenant struct {
	FileRecord
	TenantName string
}

type ListOptions struct {
	Purpose string
	Limit   int32
	AfterID *uuid.UUID
}

type ListResult struct {
	Files   []FileRecord
	HasMore bool
	FirstID *uuid.UUID
	LastID  *uuid.UUID
}

func (s *Service) Upload(ctx context.Context, params UploadParams) (FileRecord, error) {
	if strings.TrimSpace(params.Filename) == "" {
		return FileRecord{}, fmt.Errorf("filename required")
	}
	if _, ok := allowedPurposes[params.Purpose]; !ok {
		return FileRecord{}, fmt.Errorf("unsupported purpose %q", params.Purpose)
	}
	maxBytes := int64(s.cfg.MaxSizeMB) * 1024 * 1024
	if params.ContentLen > 0 && params.ContentLen > maxBytes {
		return FileRecord{}, fmt.Errorf("file exceeds max size of %d MB", s.cfg.MaxSizeMB)
	}
	ttl := params.TTL
	if ttl <= 0 {
		ttl = s.cfg.DefaultTTL
	}
	if ttl > s.cfg.MaxTTL {
		ttl = s.cfg.MaxTTL
	}

	hash := sha256.New()
	reader := io.TeeReader(params.Reader, hash)
	key := fmt.Sprintf("tenant/%s/%s", params.TenantID.String(), uuid.New().String())
	info, err := s.store.Put(ctx, key, reader, blob.PutOptions{
		ContentType: params.ContentType,
		Metadata: map[string]string{
			"purpose":  params.Purpose,
			"filename": params.Filename,
		},
	})
	if err != nil {
		return FileRecord{}, err
	}

	checksum := hex.EncodeToString(hash.Sum(nil))
	expiresAt := time.Now().Add(ttl)
	bytesStored := params.ContentLen
	if bytesStored <= 0 {
		bytesStored = info.Size
	}

	record, err := s.queries.CreateFile(ctx, db.CreateFileParams{
		TenantID:       toPgUUID(params.TenantID),
		Filename:       params.Filename,
		Purpose:        params.Purpose,
		ContentType:    params.ContentType,
		Bytes:          bytesStored,
		StorageBackend: strings.TrimSpace(s.cfg.Storage),
		StorageKey:     key,
		Checksum:       pgtype.Text{String: checksum, Valid: true},
		Encrypted:      info.Encrypted,
		ExpiresAt:      toPgTime(expiresAt),
		Status:         StatusUploaded,
	})
	if err != nil {
		_ = s.store.Delete(ctx, key)
		return FileRecord{}, err
	}
	return toFileRecord(record)
}

func (s *Service) Delete(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) error {
	rec, err := s.queries.GetFile(ctx, db.GetFileParams{
		TenantID: toPgUUID(tenantID),
		ID:       toPgUUID(id),
	})
	if err != nil {
		return err
	}
	if err := s.store.Delete(ctx, rec.StorageKey); err != nil {
		return err
	}
	return s.queries.DeleteFile(ctx, db.DeleteFileParams{
		TenantID: toPgUUID(tenantID),
		ID:       toPgUUID(id),
		Reason:   pgtype.Text{String: "deleted by tenant request", Valid: true},
	})
}

// List returns tenant files ordered by created_at DESC using cursor pagination.
func (s *Service) List(ctx context.Context, tenantID uuid.UUID, opts ListOptions) (ListResult, error) {
	limit := opts.Limit
	if limit <= 0 {
		limit = 100
	}
	if limit > 100 {
		limit = 100
	}

	afterCreated := pgtype.Timestamptz{Valid: false}
	afterIDParam := pgtype.UUID{Valid: false}
	if opts.AfterID != nil {
		row, err := s.queries.GetFile(ctx, db.GetFileParams{
			TenantID: toPgUUID(tenantID),
			ID:       toPgUUID(*opts.AfterID),
		})
		if err != nil {
			return ListResult{}, err
		}
		afterCreated = row.CreatedAt
		afterIDParam = row.ID
	}

	rows, err := s.queries.ListFiles(ctx, db.ListFilesParams{
		TenantID:       toPgUUID(tenantID),
		Limit:          limit + 1,
		Purpose:        toPgText(opts.Purpose),
		AfterCreatedAt: afterCreated,
		AfterID:        afterIDParam,
	})
	if err != nil {
		return ListResult{}, err
	}

	hasMore := int32(len(rows)) > limit
	if hasMore {
		rows = rows[:len(rows)-1]
	}

	files := make([]FileRecord, 0, len(rows))
	for _, row := range rows {
		rec, err := toFileRecord(row)
		if err != nil {
			return ListResult{}, err
		}
		files = append(files, rec)
	}

	var firstID, lastID *uuid.UUID
	if len(files) > 0 {
		first := files[0].ID
		last := files[len(files)-1].ID
		firstID = &first
		lastID = &last
	}

	return ListResult{
		Files:   files,
		HasMore: hasMore,
		FirstID: firstID,
		LastID:  lastID,
	}, nil
}

// ListAll returns paginated files across all tenants for admin auditing.
func (s *Service) ListAll(ctx context.Context, tenantID *uuid.UUID, purpose, search, state string, limit, offset int32) ([]FileWithTenant, int64, error) {
	if s.queries == nil {
		return nil, 0, errors.New("file queries unavailable")
	}
	if limit <= 0 {
		limit = 50
	}
	params := db.ListFilesAdminParams{
		TenantID:   toOptionalUUID(tenantID),
		Purpose:    toPgText(purpose),
		Search:     toPgText(search),
		State:      strings.ToLower(strings.TrimSpace(state)),
		PageLimit:  limit,
		PageOffset: offset,
	}
	if params.State == "" {
		params.State = "active"
	}
	rows, err := s.queries.ListFilesAdmin(ctx, params)
	if err != nil {
		return nil, 0, err
	}
	result := make([]FileWithTenant, 0, len(rows))
	var total int64
	for _, row := range rows {
		record, convErr := toFileWithTenant(row)
		if convErr != nil {
			return nil, 0, convErr
		}
		result = append(result, record)
		total = row.TotalCount
	}
	return result, total, nil
}

// Open returns the blob reader + metadata for downloading.
func (s *Service) Open(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) (io.ReadCloser, FileRecord, error) {
	row, err := s.queries.GetFile(ctx, db.GetFileParams{
		TenantID: toPgUUID(tenantID),
		ID:       toPgUUID(id),
	})
	if err != nil {
		return nil, FileRecord{}, err
	}
	rec, err := toFileRecord(row)
	if err != nil {
		return nil, FileRecord{}, err
	}
	reader, _, err := s.store.Get(ctx, rec.StorageKey)
	if err != nil {
		return nil, FileRecord{}, err
	}
	return reader, rec, nil
}

// GetByID fetches a file record regardless of tenant.
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (FileRecord, error) {
	row, err := s.queries.GetFileByID(ctx, toPgUUID(id))
	if err != nil {
		return FileRecord{}, err
	}
	return toFileRecord(row)
}

func (s *Service) GetWithTenant(ctx context.Context, id uuid.UUID) (FileWithTenant, error) {
	row, err := s.queries.GetFileByID(ctx, toPgUUID(id))
	if err != nil {
		return FileWithTenant{}, err
	}
	record, err := toFileRecord(row)
	if err != nil {
		return FileWithTenant{}, err
	}
	tenantName := ""
	if tenant, terr := s.queries.GetTenantByID(ctx, row.TenantID); terr == nil {
		tenantName = tenant.Name
	} else if !errors.Is(terr, pgx.ErrNoRows) {
		return FileWithTenant{}, terr
	}
	return FileWithTenant{FileRecord: record, TenantName: tenantName}, nil
}

// DeleteByID marks a file as deleted for auditing purposes.
func (s *Service) DeleteByID(ctx context.Context, id uuid.UUID) error {
	record, err := s.queries.GetFileByID(ctx, toPgUUID(id))
	if err != nil {
		return err
	}
	return s.queries.DeleteFile(ctx, db.DeleteFileParams{
		TenantID: record.TenantID,
		ID:       record.ID,
		Reason:   pgtype.Text{String: "deleted by admin", Valid: true},
	})
}

// SweepExpired removes expired files periodically.
func (s *Service) SweepExpired(ctx context.Context, batchSize int32) error {
	if batchSize <= 0 {
		batchSize = 100
	}
	expired, err := s.queries.ListExpiredFiles(ctx, db.ListExpiredFilesParams{
		ExpiresAt: toPgTime(time.Now()),
		Limit:     batchSize,
	})
	if err != nil {
		return err
	}
	for _, rec := range expired {
		_ = s.store.Delete(ctx, rec.StorageKey)
		_ = s.queries.DeleteFile(ctx, db.DeleteFileParams{
			TenantID: rec.TenantID,
			ID:       rec.ID,
			Reason:   pgtype.Text{String: "expired", Valid: true},
		})
	}
	return nil
}

func toFileRecord(row db.File) (FileRecord, error) {
	id, err := fromPgUUID(row.ID)
	if err != nil {
		return FileRecord{}, err
	}
	tenant, err := fromPgUUID(row.TenantID)
	if err != nil {
		return FileRecord{}, err
	}
	checksum := ""
	if row.Checksum.Valid {
		checksum = row.Checksum.String
	}
	var deletedAt *time.Time
	if row.DeletedAt.Valid {
		t := row.DeletedAt.Time
		deletedAt = &t
	}
	return FileRecord{
		ID:              id,
		TenantID:        tenant,
		Filename:        row.Filename,
		Purpose:         row.Purpose,
		ContentType:     row.ContentType,
		Bytes:           row.Bytes,
		StorageKey:      row.StorageKey,
		StorageBackend:  row.StorageBackend,
		Encrypted:       row.Encrypted,
		Checksum:        checksum,
		ExpiresAt:       row.ExpiresAt.Time,
		CreatedAt:       row.CreatedAt.Time,
		DeletedAt:       deletedAt,
		Status:          row.Status,
		StatusDetails:   fromPgText(row.StatusDetails),
		StatusUpdatedAt: row.StatusUpdatedAt.Time,
	}, nil
}

func toFileWithTenant(row db.ListFilesAdminRow) (FileWithTenant, error) {
	record, err := toFileRecord(db.File{
		ID:             row.ID,
		TenantID:       row.TenantID,
		Filename:       row.Filename,
		Purpose:        row.Purpose,
		ContentType:    row.ContentType,
		Bytes:          row.Bytes,
		StorageBackend: row.StorageBackend,
		StorageKey:     row.StorageKey,
		Checksum:       row.Checksum,
		Encrypted:      row.Encrypted,
		Metadata:       row.Metadata,
		ExpiresAt:      row.ExpiresAt,
		CreatedAt:      row.CreatedAt,
		DeletedAt:      row.DeletedAt,
	})
	if err != nil {
		return FileWithTenant{}, err
	}
	return FileWithTenant{FileRecord: record, TenantName: row.TenantName}, nil
}

func toPgUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: true}
}

func toOptionalUUID(id *uuid.UUID) pgtype.UUID {
	if id == nil {
		return pgtype.UUID{Valid: false}
	}
	return pgtype.UUID{Bytes: *id, Valid: true}
}

func toPgText(value string) pgtype.Text {
	value = strings.TrimSpace(value)
	if value == "" || strings.EqualFold(value, "all") {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: value, Valid: true}
}

func fromPgUUID(value pgtype.UUID) (uuid.UUID, error) {
	if !value.Valid {
		return uuid.UUID{}, fmt.Errorf("uuid is null")
	}
	return value.Bytes, nil
}

func fromPgText(value pgtype.Text) string {
	if !value.Valid {
		return ""
	}
	return value.String
}

func toPgTime(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}
