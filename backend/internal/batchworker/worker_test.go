package batchworker

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ncecere/open_model_gateway/backend/internal/app"
	"github.com/ncecere/open_model_gateway/backend/internal/config"
	"github.com/ncecere/open_model_gateway/backend/internal/db"
	"github.com/ncecere/open_model_gateway/backend/internal/requestctx"
	batchsvc "github.com/ncecere/open_model_gateway/backend/internal/services/batches"
	filesvc "github.com/ncecere/open_model_gateway/backend/internal/services/files"
	"github.com/ncecere/open_model_gateway/backend/internal/storage/blob"
)

func TestFileTTLCoversNilAndExpired(t *testing.T) {
	var batch batchsvc.Batch
	if ttl := fileTTL(batch); ttl != 0 {
		t.Fatalf("expected zero ttl for nil expires, got %s", ttl)
	}

	expired := time.Now().Add(-time.Hour)
	batch.ExpiresAt = &expired
	if ttl := fileTTL(batch); ttl != 0 {
		t.Fatalf("expected zero ttl for expired batch, got %s", ttl)
	}

	future := time.Now().Add(time.Hour)
	batch.ExpiresAt = &future
	ttl := fileTTL(batch)
	if ttl <= 0 || ttl > time.Hour+time.Second {
		t.Fatalf("unexpected ttl value: %s", ttl)
	}
}

func TestEncodeErrorPayloadFormatsOpenAIShape(t *testing.T) {
	payload := encodeErrorPayload("test_code", "boom")
	var body map[string]openAIError
	if err := json.Unmarshal(payload, &body); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	errObj := body["error"]
	if errObj.Type != "batch_error" {
		t.Fatalf("unexpected error type %s", errObj.Type)
	}
	if errObj.Code != "test_code" || errObj.Message != "boom" {
		t.Fatalf("unexpected payload %+v", errObj)
	}
}

func TestMapStatusToCode(t *testing.T) {
	cases := map[int]string{
		fiber.StatusBadRequest:          "invalid_request_error",
		fiber.StatusForbidden:           "permission_error",
		fiber.StatusTooManyRequests:     "rate_limit_error",
		fiber.StatusServiceUnavailable:  "service_unavailable",
		fiber.StatusInternalServerError: "provider_error",
	}
	for status, want := range cases {
		if got := mapStatusToCode(status); got != want {
			t.Fatalf("status %d: want %s got %s", status, want, got)
		}
	}
}

func TestParseBatchFileRefsCombinesSingleAndArray(t *testing.T) {
	single := json.RawMessage(`"file-1"`)
	multi := json.RawMessage(`["file-2","file-3"]`)

	refs, err := parseBatchFileRefs(single, multi)
	if err != nil {
		t.Fatalf("parse refs: %v", err)
	}
	want := []string{"file-1", "file-2", "file-3"}
	if len(refs) != len(want) {
		t.Fatalf("expected %d refs got %d", len(want), len(refs))
	}
	for i := range want {
		if refs[i] != want[i] {
			t.Fatalf("refs mismatch at %d: want %s got %s", i, want[i], refs[i])
		}
	}
}

func TestLoadBatchImageInputsSuccess(t *testing.T) {
	tenantID := uuid.New()
	fileID := uuid.New()
	storageKey := "test/object"
	data := []byte("fake image data")

	fileRecord := db.File{
		ID:             toPgUUID(fileID),
		TenantID:       toPgUUID(tenantID),
		Filename:       "image.png",
		Purpose:        "batch",
		ContentType:    "image/png",
		Bytes:          int64(len(data)),
		StorageBackend: "test",
		StorageKey:     storageKey,
		ExpiresAt:      pgtype.Timestamptz{Time: time.Now().Add(time.Hour), Valid: true},
		CreatedAt:      pgtype.Timestamptz{Time: time.Now(), Valid: true},
		Status:         "uploaded",
		StatusUpdatedAt: pgtype.Timestamptz{
			Time:  time.Now(),
			Valid: true,
		},
	}

	queries := &fakeFileQueries{files: map[uuid.UUID]db.File{fileID: fileRecord}}
	store := &fakeBlobStore{objects: map[string][]byte{storageKey: data}}
	cfg := &config.FilesConfig{
		MaxSizeMB:  16,
		DefaultTTL: time.Hour,
		MaxTTL:     24 * time.Hour,
	}
	fileSvc := filesvc.NewService(queries, store, cfg)

	worker := &Worker{container: &app.Container{Files: fileSvc}}
	rc := &requestctx.Context{TenantID: tenantID}

	inputs, errPayload := worker.loadBatchImageInputs(context.Background(), rc, []string{fileID.String()}, 2)
	if errPayload != nil {
		t.Fatalf("unexpected error payload: %s", errPayload)
	}
	if len(inputs) != 1 {
		t.Fatalf("expected single input, got %d", len(inputs))
	}
	if got, want := string(inputs[0].Data), string(data); got != want {
		t.Fatalf("input data mismatch: want %q got %q", want, got)
	}
	if got, want := inputs[0].ContentType, "image/png"; got != want {
		t.Fatalf("expected content type %s got %s", want, got)
	}
}

func TestLoadBatchImageInputsRejectsLargeFiles(t *testing.T) {
	tenantID := uuid.New()
	fileID := uuid.New()
	storageKey := "oversized/object"
	data := bytes.Repeat([]byte("a"), maxBatchImageBytes+1)

	fileRecord := db.File{
		ID:             toPgUUID(fileID),
		TenantID:       toPgUUID(tenantID),
		Filename:       "huge.png",
		Purpose:        "batch",
		ContentType:    "image/png",
		Bytes:          int64(len(data)),
		StorageBackend: "test",
		StorageKey:     storageKey,
		ExpiresAt:      pgtype.Timestamptz{Time: time.Now().Add(time.Hour), Valid: true},
		CreatedAt:      pgtype.Timestamptz{Time: time.Now(), Valid: true},
		Status:         "uploaded",
		StatusUpdatedAt: pgtype.Timestamptz{
			Time:  time.Now(),
			Valid: true,
		},
	}

	queries := &fakeFileQueries{files: map[uuid.UUID]db.File{fileID: fileRecord}}
	store := &fakeBlobStore{objects: map[string][]byte{storageKey: data}}
	cfg := &config.FilesConfig{
		MaxSizeMB:  64,
		DefaultTTL: time.Hour,
		MaxTTL:     24 * time.Hour,
	}
	fileSvc := filesvc.NewService(queries, store, cfg)

	worker := &Worker{container: &app.Container{Files: fileSvc}}
	rc := &requestctx.Context{TenantID: tenantID}

	_, errPayload := worker.loadBatchImageInputs(context.Background(), rc, []string{fileID.String()}, 2)
	if errPayload == nil {
		t.Fatalf("expected error payload for oversized file")
	}
	errObj := decodeErrorPayload(t, errPayload)
	if errObj.Code != "invalid_request_error" {
		t.Fatalf("unexpected error code %s", errObj.Code)
	}
	if !strings.Contains(errObj.Message, "exceeds") {
		t.Fatalf("unexpected error message %s", errObj.Message)
	}
}

func decodeErrorPayload(t *testing.T, payload []byte) openAIError {
	t.Helper()
	var body errorPayload
	if err := json.Unmarshal(payload, &body); err != nil {
		t.Fatalf("decode error payload: %v", err)
	}
	return body.Error
}

type fakeFileQueries struct {
	files map[uuid.UUID]db.File
}

func (f *fakeFileQueries) CreateFile(ctx context.Context, arg db.CreateFileParams) (db.File, error) {
	return db.File{}, errors.New("not implemented")
}

func (f *fakeFileQueries) GetFile(ctx context.Context, arg db.GetFileParams) (db.File, error) {
	id, err := fromPgUUID(arg.ID)
	if err != nil {
		return db.File{}, err
	}
	tenantID, err := fromPgUUID(arg.TenantID)
	if err != nil {
		return db.File{}, err
	}
	file, ok := f.files[id]
	if !ok {
		return db.File{}, fmt.Errorf("file %s not found", id)
	}
	fileTenant, err := fromPgUUID(file.TenantID)
	if err != nil {
		return db.File{}, err
	}
	if fileTenant != tenantID {
		return db.File{}, fmt.Errorf("tenant mismatch")
	}
	return file, nil
}

func (f *fakeFileQueries) DeleteFile(ctx context.Context, arg db.DeleteFileParams) error {
	return errors.New("not implemented")
}

func (f *fakeFileQueries) ListFiles(ctx context.Context, arg db.ListFilesParams) ([]db.File, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeFileQueries) ListFilesAdmin(ctx context.Context, arg db.ListFilesAdminParams) ([]db.ListFilesAdminRow, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeFileQueries) GetFileByID(ctx context.Context, id pgtype.UUID) (db.File, error) {
	u, err := fromPgUUID(id)
	if err != nil {
		return db.File{}, err
	}
	file, ok := f.files[u]
	if !ok {
		return db.File{}, fmt.Errorf("file %s not found", u)
	}
	return file, nil
}

func (f *fakeFileQueries) GetTenantByID(ctx context.Context, id pgtype.UUID) (db.Tenant, error) {
	return db.Tenant{}, errors.New("not implemented")
}

func (f *fakeFileQueries) ListExpiredFiles(ctx context.Context, arg db.ListExpiredFilesParams) ([]db.File, error) {
	return nil, errors.New("not implemented")
}

type fakeBlobStore struct {
	objects map[string][]byte
}

func (s *fakeBlobStore) Put(ctx context.Context, key string, body io.Reader, opts blob.PutOptions) (blob.ObjectInfo, error) {
	return blob.ObjectInfo{}, errors.New("not implemented")
}

func (s *fakeBlobStore) Get(ctx context.Context, key string) (io.ReadCloser, blob.ObjectInfo, error) {
	data, ok := s.objects[key]
	if !ok {
		return nil, blob.ObjectInfo{}, fmt.Errorf("object %s not found", key)
	}
	return io.NopCloser(bytes.NewReader(data)), blob.ObjectInfo{Key: key, Size: int64(len(data))}, nil
}

func (s *fakeBlobStore) Delete(ctx context.Context, key string) error {
	return errors.New("not implemented")
}
