// Package responsestore provides storage and retrieval of Open Responses API
// responses, enabling the previous_response_id feature.
package responsestore

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ncecere/open_model_gateway/backend/internal/db"
	"github.com/ncecere/open_model_gateway/backend/internal/logging"
)

// StoredResponse contains the fields needed to reconstruct conversation context
// from a previous response.
type StoredResponse struct {
	ID           string            `json:"id"`
	TenantID     uuid.UUID         `json:"tenant_id"`
	Model        string            `json:"model"`
	Input        json.RawMessage   `json:"input"`
	Output       json.RawMessage   `json:"output"`
	Instructions string            `json:"instructions"`
	Metadata     map[string]string `json:"metadata"`
	CreatedAt    time.Time         `json:"created_at"`
}

// Service manages response storage for the previous_response_id feature.
type Service struct {
	queries *db.Queries
	logger  *logging.Logger
	ttl     time.Duration
}

// New creates a new response store service with the given TTL for cached responses.
func New(queries *db.Queries, logger *logging.Logger, ttl time.Duration) *Service {
	if ttl <= 0 {
		ttl = 7 * 24 * time.Hour // default 7 days
	}
	return &Service{
		queries: queries,
		logger:  logger,
		ttl:     ttl,
	}
}

// Store persists a completed response for later retrieval via previous_response_id.
func (s *Service) Store(ctx context.Context, resp StoredResponse) error {
	inputBytes := resp.Input
	if inputBytes == nil {
		inputBytes = json.RawMessage("[]")
	}
	outputBytes := resp.Output
	if outputBytes == nil {
		outputBytes = json.RawMessage("[]")
	}
	metaBytes, err := json.Marshal(resp.Metadata)
	if err != nil {
		metaBytes = []byte("{}")
	}

	now := time.Now()
	return s.queries.InsertResponseCache(ctx, db.InsertResponseCacheParams{
		ID: resp.ID,
		TenantID: pgtype.UUID{
			Bytes: resp.TenantID,
			Valid: true,
		},
		Model:  resp.Model,
		Input:  inputBytes,
		Output: outputBytes,
		Instructions: pgtype.Text{
			String: resp.Instructions,
			Valid:  resp.Instructions != "",
		},
		Metadata: metaBytes,
		CreatedAt: pgtype.Timestamptz{
			Time:  now,
			Valid: true,
		},
		ExpiresAt: pgtype.Timestamptz{
			Time:  now.Add(s.ttl),
			Valid: true,
		},
	})
}

// Get retrieves a stored response by ID. Returns nil if not found or expired.
func (s *Service) Get(ctx context.Context, id string) (*StoredResponse, error) {
	row, err := s.queries.GetResponseCache(ctx, id)
	if err != nil {
		return nil, err
	}

	tenantID := uuid.UUID(row.TenantID.Bytes)
	var metadata map[string]string
	if len(row.Metadata) > 0 {
		_ = json.Unmarshal(row.Metadata, &metadata)
	}
	if metadata == nil {
		metadata = map[string]string{}
	}

	return &StoredResponse{
		ID:           row.ID,
		TenantID:     tenantID,
		Model:        row.Model,
		Input:        row.Input,
		Output:       row.Output,
		Instructions: row.Instructions.String,
		Metadata:     metadata,
		CreatedAt:    row.CreatedAt.Time,
	}, nil
}

// Cleanup removes expired response cache entries.
func (s *Service) Cleanup(ctx context.Context) error {
	return s.queries.DeleteExpiredResponseCache(ctx)
}
