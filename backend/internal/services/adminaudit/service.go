package adminaudit

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ncecere/open_model_gateway/backend/internal/db"
	auditservice "github.com/ncecere/open_model_gateway/backend/internal/services/audit"
)

// Recorder defines the minimal audit recorder contract.
type Recorder interface {
	Record(ctx context.Context, params db.InsertAuditLogParams) error
}

// Service wraps audit recording with metadata helpers.
type Service struct {
	audit Recorder
}

func NewService(audit Recorder) *Service {
	return &Service{audit: audit}
}

// Record inserts an audit entry with JSON metadata.
func (s *Service) Record(ctx context.Context, userID uuid.UUID, action, resourceType, resourceID string, metadata any) error {
	if s == nil || s.audit == nil {
		return auditservice.ErrServiceUnavailable
	}
	metaBytes := []byte("{}")
	if metadata != nil {
		data, err := json.Marshal(metadata)
		if err != nil {
			return err
		}
		metaBytes = data
	}
	return s.audit.Record(ctx, db.InsertAuditLogParams{
		UserID:       pgtype.UUID{Bytes: userID, Valid: true},
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Metadata:     metaBytes,
	})
}
