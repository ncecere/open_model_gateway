package public

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	batchsvc "github.com/ncecere/open_model_gateway/backend/internal/services/batches"
)

func TestValidateBatchMetadata(t *testing.T) {
	tests := []struct {
		name    string
		meta    map[string]string
		wantErr bool
	}{
		{name: "empty", meta: map[string]string{}, wantErr: false},
		{name: "valid", meta: map[string]string{"label": "ok"}, wantErr: false},
		{name: "too many", meta: buildMetadataEntries(17), wantErr: true},
		{name: "blank key", meta: map[string]string{" ": "value"}, wantErr: true},
		{name: "key too long", meta: map[string]string{strings.Repeat("k", 65): "value"}, wantErr: true},
		{name: "value too long", meta: map[string]string{"label": strings.Repeat("a", 513)}, wantErr: true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			err := validateBatchMetadata(tt.meta)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func buildMetadataEntries(count int) map[string]string {
	result := make(map[string]string, count)
	for i := 0; i < count; i++ {
		result[uuid.NewString()] = "v"
	}
	return result
}

func TestToOpenAIBatchIncludesErrorsAndTimestamps(t *testing.T) {
	now := time.Unix(1730000000, 0).UTC()
	canceling := now.Add(time.Minute)
	expired := now.Add(2 * time.Hour)
	inputID := uuid.New()
	resultID := uuid.New()
	errID := uuid.New()
	apiKeyID := uuid.New()
	tenantID := uuid.New()

	batch := batchsvc.Batch{
		ID:                    uuid.New(),
		TenantID:              tenantID,
		APIKeyID:              apiKeyID,
		Status:                "failed",
		Endpoint:              "/v1/chat/completions",
		CompletionWindow:      "24h",
		InputFileID:           inputID,
		ResultFileID:          &resultID,
		ErrorFileID:           &errID,
		Metadata:              map[string]string{"label": "nightly"},
		RequestCountTotal:     2,
		RequestCountCompleted: 1,
		RequestCountFailed:    1,
		CreatedAt:             now,
		UpdatedAt:             now,
		InProgressAt:          &now,
		CompletedAt:           nil,
		CancelledAt:           nil,
		CancellingAt:          &canceling,
		FinalizingAt:          &now,
		FailedAt:              &now,
		ExpiresAt:             &expired,
		ExpiredAt:             &expired,
		Errors: []batchsvc.BatchError{
			{Code: "empty_file", Message: "file empty"},
		},
	}

	resp := toOpenAIBatch(batch)
	require.NotNil(t, resp.Errors)
	require.Equal(t, "list", resp.Errors.Object)
	require.Len(t, resp.Errors.Data, 1)
	require.Equal(t, "empty_file", resp.Errors.Data[0].Code)
	require.NotNil(t, resp.CancellingAt)
	require.Equal(t, canceling.Unix(), *resp.CancellingAt)
	require.NotNil(t, resp.ExpiredAt)
	require.Equal(t, expired.Unix(), *resp.ExpiredAt)
}
