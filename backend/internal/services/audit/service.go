package audit

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ncecere/open_model_gateway/backend/internal/db"
)

// Service provides read access to admin audit logs.
type Service struct {
	queries *db.Queries
}

func NewService(queries *db.Queries) *Service {
	return &Service{queries: queries}
}

var ErrServiceUnavailable = errors.New("audit service not initialized")

// Filter controls audit log listing.
type Filter struct {
	UserID       uuid.UUID
	Action       string
	ResourceType string
	Limit        int32
	Offset       int32
}

// LogEntry represents an audit log row.
type LogEntry struct {
	ID         uuid.UUID
	UserID     *uuid.UUID
	Action     string
	Resource   string
	ResourceID string
	Metadata   []byte
	CreatedAt  time.Time
}

func (s *Service) List(ctx context.Context, filter Filter) ([]LogEntry, error) {
	if s == nil || s.queries == nil {
		return nil, errors.New("audit service not initialized")
	}
	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	params := db.ListAuditLogsParams{
		UserIDFilter:   toNullableUUID(filter.UserID),
		ActionFilter:   toNullableText(filter.Action),
		ResourceFilter: toNullableText(filter.ResourceType),
		ListOffset:     filter.Offset,
		ListLimit:      limit,
	}
	rows, err := s.queries.ListAuditLogs(ctx, params)
	if err != nil {
		return nil, err
	}
	entries := make([]LogEntry, 0, len(rows))
	for _, row := range rows {
		id, err := uuidFromPg(row.ID)
		if err != nil {
			continue
		}
		var userPtr *uuid.UUID
		if row.UserID.Valid {
			if uid, err := uuidFromPg(row.UserID); err == nil {
				userPtr = &uid
			}
		}
		created, err := timeFromPg(row.CreatedAt)
		if err != nil {
			return nil, err
		}
		entries = append(entries, LogEntry{
			ID:         id,
			UserID:     userPtr,
			Action:     row.Action,
			Resource:   row.ResourceType,
			ResourceID: row.ResourceID,
			Metadata:   row.Metadata,
			CreatedAt:  created,
		})
	}
	return entries, nil
}

// Record inserts an audit log row.
func (s *Service) Record(ctx context.Context, params db.InsertAuditLogParams) error {
	if s == nil || s.queries == nil {
		return ErrServiceUnavailable
	}
	_, err := s.queries.InsertAuditLog(ctx, params)
	return err
}

func toNullableUUID(id uuid.UUID) pgtype.UUID {
	if id == uuid.Nil {
		return pgtype.UUID{}
	}
	return pgtype.UUID{Bytes: id, Valid: true}
}

func toNullableText(val string) pgtype.Text {
	if strings.TrimSpace(val) == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: val, Valid: true}
}

func uuidFromPg(id pgtype.UUID) (uuid.UUID, error) {
	if !id.Valid {
		return uuid.Nil, errors.New("invalid uuid")
	}
	return uuid.FromBytes(id.Bytes[:])
}

func timeFromPg(ts pgtype.Timestamptz) (time.Time, error) {
	if !ts.Valid {
		return time.Time{}, errors.New("invalid timestamp")
	}
	return ts.Time, nil
}
