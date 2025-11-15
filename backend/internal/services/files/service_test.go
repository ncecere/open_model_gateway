package files

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"

	"github.com/ncecere/open_model_gateway/backend/internal/config"
	"github.com/ncecere/open_model_gateway/backend/internal/db"
	"github.com/ncecere/open_model_gateway/backend/internal/storage/blob"
)

func TestServiceListPagination(t *testing.T) {
	t.Parallel()

	tenantID := uuid.New()
	firstID := uuid.New()
	secondID := uuid.New()
	createdA := time.Now()
	createdB := createdA.Add(-time.Minute)

	stub := &stubQueries{
		listFilesFn: func(ctx context.Context, arg db.ListFilesParams) ([]db.File, error) {
			require.Equal(t, toPgUUID(tenantID), arg.TenantID)
			require.Equal(t, int32(2), arg.Limit) // limit + 1 to detect has_more
			return []db.File{
				buildDBFile(firstID, tenantID, createdA),
				buildDBFile(secondID, tenantID, createdB),
			}, nil
		},
	}

	svc := NewService(stub, nil, testFilesConfig())
	result, err := svc.List(context.Background(), tenantID, ListOptions{Limit: 1})
	require.NoError(t, err)
	require.True(t, result.HasMore)
	require.Len(t, result.Files, 1)
	require.Equal(t, firstID, result.Files[0].ID)
	require.NotNil(t, result.FirstID)
	require.NotNil(t, result.LastID)
	require.Equal(t, firstID, *result.FirstID)
	require.Equal(t, firstID, *result.LastID)
}

func TestServiceListAfterCursor(t *testing.T) {
	t.Parallel()

	tenantID := uuid.New()
	cursorID := uuid.New()
	cursorCreated := time.Now().Add(-time.Hour)

	var captured db.ListFilesParams
	stub := &stubQueries{
		getFileFn: func(ctx context.Context, arg db.GetFileParams) (db.File, error) {
			require.Equal(t, toPgUUID(tenantID), arg.TenantID)
			require.Equal(t, toPgUUID(cursorID), arg.ID)
			return buildDBFile(cursorID, tenantID, cursorCreated), nil
		},
		listFilesFn: func(ctx context.Context, arg db.ListFilesParams) ([]db.File, error) {
			captured = arg
			return []db.File{}, nil
		},
	}

	svc := NewService(stub, nil, testFilesConfig())
	_, err := svc.List(context.Background(), tenantID, ListOptions{AfterID: &cursorID})
	require.NoError(t, err)

	require.True(t, captured.AfterCreatedAt.Valid)
	require.Equal(t, cursorCreated.Unix(), captured.AfterCreatedAt.Time.Unix())

	require.True(t, captured.AfterID.Valid)
	require.Equal(t, cursorID[:], captured.AfterID.Bytes[:])
}

func TestServiceSweepExpired(t *testing.T) {
	t.Parallel()

	tenantID := uuid.New()
	expired := []db.File{
		buildDBFile(uuid.New(), tenantID, time.Now().Add(-time.Hour)),
		buildDBFile(uuid.New(), tenantID, time.Now().Add(-2*time.Hour)),
	}
	expired[0].StorageKey = "tenant/a/file-1"
	expired[1].StorageKey = "tenant/a/file-2"

	var deletedKeys []string
	var deleteCalls []db.DeleteFileParams

	stub := &stubQueries{
		listExpiredFn: func(ctx context.Context, arg db.ListExpiredFilesParams) ([]db.File, error) {
			require.Greater(t, arg.Limit, int32(0))
			return expired, nil
		},
		deleteFileFn: func(ctx context.Context, arg db.DeleteFileParams) error {
			deleteCalls = append(deleteCalls, arg)
			return nil
		},
	}
	store := &stubStore{
		deleteFn: func(ctx context.Context, key string) error {
			deletedKeys = append(deletedKeys, key)
			return nil
		},
	}

	svc := NewService(stub, store, testFilesConfig())
	require.NoError(t, svc.SweepExpired(context.Background(), 50))

	require.ElementsMatch(t, []string{"tenant/a/file-1", "tenant/a/file-2"}, deletedKeys)
	require.Len(t, deleteCalls, 2)
	for _, call := range deleteCalls {
		require.True(t, call.Reason.Valid)
		require.Equal(t, "expired", call.Reason.String)
	}
}

// Helpers

type stubQueries struct {
	createFileFn  func(context.Context, db.CreateFileParams) (db.File, error)
	getFileFn     func(context.Context, db.GetFileParams) (db.File, error)
	deleteFileFn  func(context.Context, db.DeleteFileParams) error
	listFilesFn   func(context.Context, db.ListFilesParams) ([]db.File, error)
	listAdminFn   func(context.Context, db.ListFilesAdminParams) ([]db.ListFilesAdminRow, error)
	getByIDFn     func(context.Context, pgtype.UUID) (db.File, error)
	getTenantFn   func(context.Context, pgtype.UUID) (db.Tenant, error)
	listExpiredFn func(context.Context, db.ListExpiredFilesParams) ([]db.File, error)
}

func (s *stubQueries) CreateFile(ctx context.Context, arg db.CreateFileParams) (db.File, error) {
	if s.createFileFn != nil {
		return s.createFileFn(ctx, arg)
	}
	return db.File{}, nil
}

func (s *stubQueries) GetFile(ctx context.Context, arg db.GetFileParams) (db.File, error) {
	if s.getFileFn != nil {
		return s.getFileFn(ctx, arg)
	}
	return db.File{}, nil
}

func (s *stubQueries) DeleteFile(ctx context.Context, arg db.DeleteFileParams) error {
	if s.deleteFileFn != nil {
		return s.deleteFileFn(ctx, arg)
	}
	return nil
}

func (s *stubQueries) ListFiles(ctx context.Context, arg db.ListFilesParams) ([]db.File, error) {
	if s.listFilesFn != nil {
		return s.listFilesFn(ctx, arg)
	}
	return nil, nil
}

func (s *stubQueries) ListFilesAdmin(ctx context.Context, arg db.ListFilesAdminParams) ([]db.ListFilesAdminRow, error) {
	if s.listAdminFn != nil {
		return s.listAdminFn(ctx, arg)
	}
	return nil, nil
}

func (s *stubQueries) GetFileByID(ctx context.Context, id pgtype.UUID) (db.File, error) {
	if s.getByIDFn != nil {
		return s.getByIDFn(ctx, id)
	}
	return db.File{}, nil
}

func (s *stubQueries) GetTenantByID(ctx context.Context, id pgtype.UUID) (db.Tenant, error) {
	if s.getTenantFn != nil {
		return s.getTenantFn(ctx, id)
	}
	return db.Tenant{}, nil
}

func (s *stubQueries) ListExpiredFiles(ctx context.Context, arg db.ListExpiredFilesParams) ([]db.File, error) {
	if s.listExpiredFn != nil {
		return s.listExpiredFn(ctx, arg)
	}
	return nil, nil
}

type stubStore struct {
	deleteFn func(context.Context, string) error
}

func (s *stubStore) Put(context.Context, string, io.Reader, blob.PutOptions) (blob.ObjectInfo, error) {
	return blob.ObjectInfo{}, nil
}

func (s *stubStore) Get(context.Context, string) (io.ReadCloser, blob.ObjectInfo, error) {
	return io.NopCloser(bytes.NewReader(nil)), blob.ObjectInfo{}, nil
}

func (s *stubStore) Delete(ctx context.Context, key string) error {
	if s.deleteFn != nil {
		return s.deleteFn(ctx, key)
	}
	return nil
}

func buildDBFile(id uuid.UUID, tenantID uuid.UUID, created time.Time) db.File {
	return db.File{
		ID:              toPgUUID(id),
		TenantID:        toPgUUID(tenantID),
		Filename:        "file.txt",
		Purpose:         PurposeFineTune,
		ContentType:     "text/plain",
		Bytes:           10,
		StorageBackend:  "local",
		StorageKey:      "key",
		Checksum:        pgtype.Text{String: "abc", Valid: true},
		Encrypted:       false,
		Metadata:        []byte("{}"),
		ExpiresAt:       toPgTime(created.Add(24 * time.Hour)),
		CreatedAt:       toPgTime(created),
		DeletedAt:       pgtype.Timestamptz{Valid: false},
		Status:          StatusUploaded,
		StatusDetails:   pgtype.Text{Valid: false},
		StatusUpdatedAt: toPgTime(created),
	}
}

func testFilesConfig() *config.FilesConfig {
	return &config.FilesConfig{
		MaxSizeMB:      200,
		DefaultTTL:     24 * time.Hour,
		MaxTTL:         30 * 24 * time.Hour,
		Storage:        "local",
		SweepInterval:  15 * time.Minute,
		SweepBatchSize: 200,
	}
}
