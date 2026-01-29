package requestctx

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestWithContext_FromContext_RoundTrip(t *testing.T) {
	rc := &Context{
		TenantID:     uuid.New(),
		APIKeyID:     uuid.New(),
		APIKeyPrefix: "test-prefix",
		Scopes:       []string{"chat", "embeddings"},
	}
	ctx := WithContext(context.Background(), rc)
	got, ok := FromContext(ctx)
	if !ok {
		t.Fatal("expected FromContext to return true")
	}
	if got.TenantID != rc.TenantID {
		t.Fatalf("TenantID mismatch: %v != %v", got.TenantID, rc.TenantID)
	}
	if got.APIKeyPrefix != "test-prefix" {
		t.Fatalf("APIKeyPrefix mismatch: %q", got.APIKeyPrefix)
	}
}

func TestFromContext_Missing(t *testing.T) {
	_, ok := FromContext(context.Background())
	if ok {
		t.Fatal("expected false for missing context")
	}
}

func TestFromContext_NilContext(t *testing.T) {
	_, ok := FromContext(nil)
	if ok {
		t.Fatal("expected false for nil context")
	}
}

func TestWithContext_NilParent(t *testing.T) {
	rc := &Context{TenantID: uuid.New()}
	ctx := WithContext(nil, rc)
	got, ok := FromContext(ctx)
	if !ok {
		t.Fatal("expected FromContext to return true with nil parent")
	}
	if got.TenantID != rc.TenantID {
		t.Fatal("TenantID mismatch")
	}
}

func TestFiberLocalsKey_NonEmpty(t *testing.T) {
	key := FiberLocalsKey()
	if key == "" {
		t.Fatal("expected non-empty locals key")
	}
}
