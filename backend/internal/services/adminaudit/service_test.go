package adminaudit

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"

	"github.com/ncecere/open_model_gateway/backend/internal/db"
)

type stubRecorder struct {
	params db.InsertAuditLogParams
	err    error
}

func (s *stubRecorder) Record(ctx context.Context, params db.InsertAuditLogParams) error {
	s.params = params
	return s.err
}

func TestServiceRecord_Success(t *testing.T) {
	stub := &stubRecorder{}
	svc := NewService(stub)

	userID := uuid.New()
	meta := map[string]string{"foo": "bar"}
	if err := svc.Record(context.Background(), userID, "action", "resource", "id-123", meta); err != nil {
		t.Fatalf("Record() error = %v", err)
	}

	if stub.params.Action != "action" || stub.params.ResourceType != "resource" || stub.params.ResourceID != "id-123" {
		t.Fatalf("unexpected params: %+v", stub.params)
	}
	if stub.params.UserID.Bytes != userID {
		t.Fatalf("user id not propagated")
	}
	var decoded map[string]string
	if err := json.Unmarshal(stub.params.Metadata, &decoded); err != nil {
		t.Fatalf("metadata not valid json: %v", err)
	}
	if decoded["foo"] != "bar" {
		t.Fatalf("metadata mismatch: %+v", decoded)
	}
}

func TestServiceRecord_Unavailable(t *testing.T) {
	svc := NewService(nil)
	err := svc.Record(context.Background(), uuid.New(), "a", "b", "c", nil)
	if err == nil {
		t.Fatal("expected error when recorder is nil")
	}
}
