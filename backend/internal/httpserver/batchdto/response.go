package batchdto

import (
	"time"

	batchsvc "github.com/ncecere/open_model_gateway/backend/internal/services/batches"
)

// Counts captures aggregate request progress for a batch.
type Counts struct {
	Total     int `json:"total"`
	Completed int `json:"completed"`
	Failed    int `json:"failed"`
	Cancelled int `json:"cancelled"`
}

// Batch represents the API-friendly batch payload shared by admin and user routes.
type Batch struct {
	ID               string            `json:"id"`
	TenantID         string            `json:"tenant_id"`
	TenantName       string            `json:"tenant_name,omitempty"`
	APIKeyID         string            `json:"api_key_id"`
	Endpoint         string            `json:"endpoint"`
	Status           string            `json:"status"`
	CompletionWindow string            `json:"completion_window"`
	MaxConcurrency   int               `json:"max_concurrency"`
	Metadata         map[string]string `json:"metadata,omitempty"`
	InputFileID      string            `json:"input_file_id"`
	OutputFileID     *string           `json:"output_file_id,omitempty"`
	ErrorFileID      *string           `json:"error_file_id,omitempty"`
	CreatedAt        time.Time         `json:"created_at"`
	UpdatedAt        time.Time         `json:"updated_at"`
	InProgressAt     *time.Time        `json:"in_progress_at,omitempty"`
	CompletedAt      *time.Time        `json:"completed_at,omitempty"`
	CancelledAt      *time.Time        `json:"cancelled_at,omitempty"`
	FinalizingAt     *time.Time        `json:"finalizing_at,omitempty"`
	FailedAt         *time.Time        `json:"failed_at,omitempty"`
	ExpiresAt        *time.Time        `json:"expires_at,omitempty"`
	Counts           Counts            `json:"counts"`
}

// FromBatch converts the service layer batch into an API response.
func FromBatch(batch batchsvc.Batch) Batch {
	var metadata map[string]string
	if len(batch.Metadata) > 0 {
		metadata = make(map[string]string, len(batch.Metadata))
		for k, v := range batch.Metadata {
			metadata[k] = v
		}
	}

	resp := Batch{
		ID:               batch.ID.String(),
		TenantID:         batch.TenantID.String(),
		APIKeyID:         batch.APIKeyID.String(),
		Endpoint:         batch.Endpoint,
		Status:           batch.Status,
		CompletionWindow: batch.CompletionWindow,
		MaxConcurrency:   batch.MaxConcurrency,
		Metadata:         metadata,
		InputFileID:      batch.InputFileID.String(),
		CreatedAt:        batch.CreatedAt,
		UpdatedAt:        batch.UpdatedAt,
		Counts: Counts{
			Total:     batch.RequestCountTotal,
			Completed: batch.RequestCountCompleted,
			Failed:    batch.RequestCountFailed,
			Cancelled: batch.RequestCountCancelled,
		},
	}
	if batch.ResultFileID != nil {
		id := batch.ResultFileID.String()
		resp.OutputFileID = &id
	}
	if batch.ErrorFileID != nil {
		id := batch.ErrorFileID.String()
		resp.ErrorFileID = &id
	}
	if batch.InProgressAt != nil {
		resp.InProgressAt = batch.InProgressAt
	}
	if batch.CompletedAt != nil {
		resp.CompletedAt = batch.CompletedAt
	}
	if batch.CancelledAt != nil {
		resp.CancelledAt = batch.CancelledAt
	}
	if batch.FinalizingAt != nil {
		resp.FinalizingAt = batch.FinalizingAt
	}
	if batch.FailedAt != nil {
		resp.FailedAt = batch.FailedAt
	}
	if batch.ExpiresAt != nil {
		resp.ExpiresAt = batch.ExpiresAt
	}
	return resp
}

// FromBatchWithTenant includes tenant metadata when available.
func FromBatchWithTenant(batch batchsvc.BatchWithTenant) Batch {
	resp := FromBatch(batch.Batch)
	resp.TenantName = batch.TenantName
	return resp
}
