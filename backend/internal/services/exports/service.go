package exports

import (
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/xitongsys/parquet-go/parquet"
	"github.com/xitongsys/parquet-go-source/local"
	"github.com/xitongsys/parquet-go/writer"

	"github.com/ncecere/open_model_gateway/backend/internal/db"
	filesvc "github.com/ncecere/open_model_gateway/backend/internal/services/files"
	"github.com/ncecere/open_model_gateway/backend/internal/timeutil"
)

const (
	StatusQueued     = "queued"
	StatusProcessing = "processing"
	StatusReady      = "ready"
	StatusFailed     = "failed"
)

const (
	FormatCSV     = "csv"
	FormatParquet = "parquet"
)

const (
	GranularityDaily   = "daily"
	GranularityMonthly = "monthly"
)

const (
	defaultExportPeriod = "30d"
	maxExportRangeDays  = 90
)

var ErrInvalidExport = errors.New("invalid usage export request")

// Service coordinates usage export records and the generated files.
type Service struct {
	queries   *db.Queries
	files     *filesvc.Service
	location  *time.Location
	maxWindow time.Duration
}

type CreateParams struct {
	Scope        string
	RequestedBy  uuid.UUID
	FileTenantID uuid.UUID
	TenantIDs    []uuid.UUID
	Period       string
	Start        *time.Time
	End          *time.Time
	Granularity  string
	Format       string
	Timezone     string
}

type Export struct {
	ID           uuid.UUID
	Scope        string
	Status       string
	Format       string
	Granularity  string
	Timezone     string
	PeriodStart  time.Time
	PeriodEnd    time.Time
	TenantIDs    []uuid.UUID
	RequestedBy  *uuid.UUID
	FileID       *uuid.UUID
	FileTenantID *uuid.UUID
	RowCount     *int32
	Error        string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	StartedAt    *time.Time
	CompletedAt  *time.Time
}

type exportRow struct {
	Bucket       time.Time
	TenantID     string
	TenantName   string
	ModelAlias   string
	Provider     string
	Requests     int64
	InputTokens  int64
	OutputTokens int64
	TotalTokens  int64
	CostCents    int64
	CostUSDMicros int64
}

type parquetExportRow struct {
	Bucket        string  `parquet:"name=bucket, type=BYTE_ARRAY, convertedtype=UTF8"`
	TenantID      string  `parquet:"name=tenant_id, type=BYTE_ARRAY, convertedtype=UTF8"`
	TenantName    string  `parquet:"name=tenant_name, type=BYTE_ARRAY, convertedtype=UTF8"`
	ModelAlias    string  `parquet:"name=model_alias, type=BYTE_ARRAY, convertedtype=UTF8"`
	Provider      string  `parquet:"name=provider, type=BYTE_ARRAY, convertedtype=UTF8"`
	Requests      int64   `parquet:"name=requests, type=INT64"`
	InputTokens   int64   `parquet:"name=input_tokens, type=INT64"`
	OutputTokens  int64   `parquet:"name=output_tokens, type=INT64"`
	TotalTokens   int64   `parquet:"name=total_tokens, type=INT64"`
	CostCents     int64   `parquet:"name=cost_cents, type=INT64"`
	CostUSDMicros int64   `parquet:"name=cost_usd_micros, type=INT64"`
	CostUSD       float64 `parquet:"name=cost_usd, type=DOUBLE"`
}

func NewService(queries *db.Queries, files *filesvc.Service, loc *time.Location) *Service {
	return &Service{
		queries:   queries,
		files:     files,
		location:  timeutil.EnsureLocation(loc),
		maxWindow: time.Duration(maxExportRangeDays) * 24 * time.Hour,
	}
}

func (s *Service) Create(ctx context.Context, params CreateParams) (Export, error) {
	if s == nil || s.queries == nil {
		return Export{}, fmt.Errorf("usage exports unavailable")
	}
	if params.RequestedBy == uuid.Nil || params.FileTenantID == uuid.Nil {
		return Export{}, ErrInvalidExport
	}
	scope := strings.TrimSpace(params.Scope)
	if scope == "" {
		scope = "user"
	}
	format, err := normalizeFormat(params.Format)
	if err != nil {
		return Export{}, err
	}
	granularity, err := normalizeGranularity(params.Granularity)
	if err != nil {
		return Export{}, err
	}
	window, err := s.resolveWindow(params.Period, params.Start, params.End, params.Timezone)
	if err != nil {
		return Export{}, err
	}
	if window.Duration() > s.maxWindow {
		return Export{}, fmt.Errorf("export range exceeds %d days", maxExportRangeDays)
	}
	tenantIDs := dedupUUIDs(params.TenantIDs)
	record, err := s.queries.CreateUsageExport(ctx, db.CreateUsageExportParams{
		Scope:        scope,
		Status:       StatusQueued,
		Format:       format,
		Granularity:  granularity,
		Timezone:     window.Timezone(),
		PeriodStart:  toPgTime(window.Start()),
		PeriodEnd:    toPgTime(window.End()),
		TenantIds:    toPgUUIDArrayNullable(tenantIDs),
		RequestedBy:  toPgUUID(params.RequestedBy),
		FileTenantID: toPgUUID(params.FileTenantID),
	})
	if err != nil {
		return Export{}, err
	}
	return exportFromDB(record)
}

func (s *Service) ListAdmin(ctx context.Context, limit, offset int32) ([]Export, error) {
	if s == nil || s.queries == nil {
		return nil, fmt.Errorf("usage exports unavailable")
	}
	if limit <= 0 {
		limit = 25
	}
	rows, err := s.queries.ListUsageExportsAdmin(ctx, db.ListUsageExportsAdminParams{Limit: limit, Offset: offset})
	if err != nil {
		return nil, err
	}
	return exportsFromDB(rows)
}

func (s *Service) ListByRequester(ctx context.Context, userID uuid.UUID, limit, offset int32) ([]Export, error) {
	if s == nil || s.queries == nil {
		return nil, fmt.Errorf("usage exports unavailable")
	}
	if limit <= 0 {
		limit = 25
	}
	rows, err := s.queries.ListUsageExportsByRequester(ctx, db.ListUsageExportsByRequesterParams{
		RequestedBy: toPgUUID(userID),
		Limit:       limit,
		Offset:      offset,
	})
	if err != nil {
		return nil, err
	}
	return exportsFromDB(rows)
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (Export, error) {
	if s == nil || s.queries == nil {
		return Export{}, fmt.Errorf("usage exports unavailable")
	}
	row, err := s.queries.GetUsageExport(ctx, toPgUUID(id))
	if err != nil {
		return Export{}, err
	}
	return exportFromDB(row)
}

func (s *Service) ClaimNext(ctx context.Context) (db.UsageExport, error) {
	if s == nil || s.queries == nil {
		return db.UsageExport{}, fmt.Errorf("usage exports unavailable")
	}
	return s.queries.ClaimUsageExport(ctx)
}

func (s *Service) Process(ctx context.Context, record db.UsageExport) (Export, error) {
	if s == nil || s.queries == nil {
		return Export{}, fmt.Errorf("usage exports unavailable")
	}
	export, err := exportFromDB(record)
	if err != nil {
		return Export{}, err
	}
	if s.files == nil {
		return s.failExport(ctx, export.ID, "files service unavailable")
	}
	rows, err := s.loadRows(ctx, export)
	if err != nil {
		return s.failExport(ctx, export.ID, err.Error())
	}
	fileRec, err := s.writeExportFile(ctx, export, rows)
	if err != nil {
		return s.failExport(ctx, export.ID, err.Error())
	}
	updated, err := s.queries.CompleteUsageExport(ctx, db.CompleteUsageExportParams{
		ID:       toPgUUID(export.ID),
		FileID:   toPgUUID(fileRec.ID),
		RowCount: toPgInt32(int32(len(rows))),
	})
	if err != nil {
		return Export{}, err
	}
	return exportFromDB(updated)
}

func (s *Service) loadRows(ctx context.Context, export Export) ([]exportRow, error) {
	if export.PeriodEnd.Before(export.PeriodStart) {
		return nil, ErrInvalidExport
	}
	trunc := granularityToTrunc(export.Granularity)
	rows, err := s.queries.ExportUsageRows(ctx, db.ExportUsageRowsParams{
		Ts:        toPgTime(export.PeriodStart),
		Ts_2:      toPgTime(export.PeriodEnd),
		Column3:   toPgUUIDArrayNullable(export.TenantIDs),
		Column4:   trunc,
		Column5:   export.Timezone,
	})
	if err != nil {
		return nil, err
	}
	result := make([]exportRow, 0, len(rows))
	for _, row := range rows {
		bucket, err := timeFromPg(row.Bucket)
		if err != nil {
			return nil, err
		}
		tenantID, err := uuidFromPg(row.TenantID)
		if err != nil {
			return nil, err
		}
		result = append(result, exportRow{
			Bucket:       bucket,
			TenantID:     tenantID.String(),
			TenantName:   row.TenantName,
			ModelAlias:   row.ModelAlias,
			Provider:     row.Provider,
			Requests:     row.Requests,
			InputTokens:  row.InputTokens,
			OutputTokens: row.OutputTokens,
			TotalTokens:  row.InputTokens + row.OutputTokens,
			CostCents:    row.CostCents,
			CostUSDMicros: row.CostUsdMicros,
		})
	}
	return result, nil
}

func (s *Service) writeExportFile(ctx context.Context, export Export, rows []exportRow) (filesvc.FileRecord, error) {
	var tenantID uuid.UUID
	if export.FileTenantID != nil {
		tenantID = *export.FileTenantID
	}
	if tenantID == uuid.Nil {
		return filesvc.FileRecord{}, ErrInvalidExport
	}
	filename := fmt.Sprintf("usage_export_%s.%s", export.ID.String(), export.Format)
	switch export.Format {
	case FormatCSV:
		payload, err := encodeCSV(rows)
		if err != nil {
			return filesvc.FileRecord{}, err
		}
		return s.files.Upload(ctx, filesvc.UploadParams{
			TenantID:    tenantID,
			Filename:    filename,
			Purpose:     filesvc.PurposeUsageExport,
			ContentType: "text/csv",
			ContentLen:  int64(len(payload)),
			Reader:      bytes.NewReader(payload),
		})
	case FormatParquet:
		path, size, err := encodeParquet(rows)
		if err != nil {
			return filesvc.FileRecord{}, err
		}
		defer os.Remove(path)
		file, err := os.Open(path)
		if err != nil {
			return filesvc.FileRecord{}, err
		}
		defer file.Close()
		return s.files.Upload(ctx, filesvc.UploadParams{
			TenantID:    tenantID,
			Filename:    filename,
			Purpose:     filesvc.PurposeUsageExport,
			ContentType: "application/x-parquet",
			ContentLen:  size,
			Reader:      file,
		})
	default:
		return filesvc.FileRecord{}, ErrInvalidExport
	}
}

func (s *Service) failExport(ctx context.Context, id uuid.UUID, reason string) (Export, error) {
	updated, err := s.queries.FailUsageExport(ctx, db.FailUsageExportParams{
		ID:    toPgUUID(id),
		Error: toPgText(reason),
	})
	if err != nil {
		return Export{}, err
	}
	return exportFromDB(updated)
}

func (s *Service) resolveWindow(period string, start, end *time.Time, tz string) (timeutil.Window, error) {
	loc := s.location
	if strings.TrimSpace(tz) != "" {
		loaded, err := time.LoadLocation(tz)
		if err != nil {
			return timeutil.Window{}, fmt.Errorf("invalid timezone")
		}
		loc = loaded
	}
	if start != nil || end != nil {
		if start == nil || end == nil {
			return timeutil.Window{}, ErrInvalidExport
		}
		return timeutil.NewWindowFromRange(*start, *end, loc, "custom")
	}
	period = strings.TrimSpace(period)
	if period == "" {
		period = defaultExportPeriod
	}
	return timeutil.NewWindow(period, time.Now().In(loc), loc)
}

func normalizeFormat(format string) (string, error) {
	clean := strings.ToLower(strings.TrimSpace(format))
	if clean == "" {
		return FormatCSV, nil
	}
	switch clean {
	case FormatCSV, FormatParquet:
		return clean, nil
	default:
		return "", ErrInvalidExport
	}
}

func normalizeGranularity(granularity string) (string, error) {
	clean := strings.ToLower(strings.TrimSpace(granularity))
	if clean == "" {
		return GranularityDaily, nil
	}
	switch clean {
	case GranularityDaily, GranularityMonthly:
		return clean, nil
	default:
		return "", ErrInvalidExport
	}
}

func granularityToTrunc(granularity string) string {
	if granularity == GranularityMonthly {
		return "month"
	}
	return "day"
}

func encodeCSV(rows []exportRow) ([]byte, error) {
	buf := &bytes.Buffer{}
	writer := csv.NewWriter(buf)
	if err := writer.Write([]string{
		"bucket_start",
		"tenant_id",
		"tenant_name",
		"model_alias",
		"provider",
		"requests",
		"input_tokens",
		"output_tokens",
		"total_tokens",
		"cost_cents",
		"cost_usd_micros",
		"cost_usd",
	}); err != nil {
		return nil, err
	}
	for _, row := range rows {
		if err := writer.Write([]string{
			row.Bucket.Format(time.RFC3339),
			row.TenantID,
			row.TenantName,
			row.ModelAlias,
			row.Provider,
			strconv.FormatInt(row.Requests, 10),
			strconv.FormatInt(row.InputTokens, 10),
			strconv.FormatInt(row.OutputTokens, 10),
			strconv.FormatInt(row.TotalTokens, 10),
			strconv.FormatInt(row.CostCents, 10),
			strconv.FormatInt(row.CostUSDMicros, 10),
			formatUSD(row.CostUSDMicros),
		}); err != nil {
			return nil, err
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func encodeParquet(rows []exportRow) (string, int64, error) {
	file, err := os.CreateTemp("", "usage-export-*.parquet")
	if err != nil {
		return "", 0, err
	}
	path := file.Name()
	if err := file.Close(); err != nil {
		return "", 0, err
	}
	fw, err := local.NewLocalFileWriter(path)
	if err != nil {
		return "", 0, err
	}
	pw, err := writer.NewParquetWriter(fw, new(parquetExportRow), 4)
	if err != nil {
		_ = fw.Close()
		return "", 0, err
	}
	pw.CompressionType = parquet.CompressionCodec_SNAPPY
	for _, row := range rows {
		if err := pw.Write(parquetExportRow{
			Bucket:        row.Bucket.Format(time.RFC3339),
			TenantID:      row.TenantID,
			TenantName:    row.TenantName,
			ModelAlias:    row.ModelAlias,
			Provider:      row.Provider,
			Requests:      row.Requests,
			InputTokens:   row.InputTokens,
			OutputTokens:  row.OutputTokens,
			TotalTokens:   row.TotalTokens,
			CostCents:     row.CostCents,
			CostUSDMicros: row.CostUSDMicros,
			CostUSD:       microsToUSD(row.CostUSDMicros),
		}); err != nil {
			_ = pw.WriteStop()
			_ = fw.Close()
			return "", 0, err
		}
	}
	if err := pw.WriteStop(); err != nil {
		_ = fw.Close()
		return "", 0, err
	}
	if err := fw.Close(); err != nil {
		return "", 0, err
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", 0, err
	}
	return path, info.Size(), nil
}

func exportsFromDB(rows []db.UsageExport) ([]Export, error) {
	result := make([]Export, 0, len(rows))
	for _, row := range rows {
		export, err := exportFromDB(row)
		if err != nil {
			return nil, err
		}
		result = append(result, export)
	}
	return result, nil
}

func exportFromDB(row db.UsageExport) (Export, error) {
	id, err := uuidFromPg(row.ID)
	if err != nil {
		return Export{}, err
	}
	periodStart, err := timeFromPg(row.PeriodStart)
	if err != nil {
		return Export{}, err
	}
	periodEnd, err := timeFromPg(row.PeriodEnd)
	if err != nil {
		return Export{}, err
	}
	createdAt, err := timeFromPg(row.CreatedAt)
	if err != nil {
		return Export{}, err
	}
	updatedAt, err := timeFromPg(row.UpdatedAt)
	if err != nil {
		return Export{}, err
	}
	return Export{
		ID:           id,
		Scope:        row.Scope,
		Status:       row.Status,
		Format:       row.Format,
		Granularity:  row.Granularity,
		Timezone:     row.Timezone,
		PeriodStart:  periodStart,
		PeriodEnd:    periodEnd,
		TenantIDs:    uuidSliceFromPg(row.TenantIds),
		RequestedBy:  uuidPtrFromPg(row.RequestedBy),
		FileID:       uuidPtrFromPg(row.FileID),
		FileTenantID: uuidPtrFromPg(row.FileTenantID),
		RowCount:     int32PtrFromPg(row.RowCount),
		Error:        textFromPg(row.Error),
		CreatedAt:    createdAt,
		UpdatedAt:    updatedAt,
		StartedAt:    timePtrFromPg(row.StartedAt),
		CompletedAt:  timePtrFromPg(row.CompletedAt),
	}, nil
}

func uuidFromPg(id pgtype.UUID) (uuid.UUID, error) {
	if !id.Valid {
		return uuid.Nil, errors.New("invalid uuid")
	}
	return id.Bytes, nil
}

func uuidPtrFromPg(id pgtype.UUID) *uuid.UUID {
	if !id.Valid {
		return nil
	}
	val := uuid.UUID(id.Bytes)
	return &val
}

func uuidSliceFromPg(ids []pgtype.UUID) []uuid.UUID {
	result := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		if !id.Valid {
			continue
		}
		result = append(result, id.Bytes)
	}
	return result
}

func toPgUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: true}
}

func toPgUUIDArrayNullable(ids []uuid.UUID) []pgtype.UUID {
	if len(ids) == 0 {
		return nil
	}
	arr := make([]pgtype.UUID, 0, len(ids))
	for _, id := range ids {
		if id == uuid.Nil {
			continue
		}
		arr = append(arr, toPgUUID(id))
	}
	if len(arr) == 0 {
		return nil
	}
	return arr
}

func toPgTime(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}

func toPgText(value string) pgtype.Text {
	clean := strings.TrimSpace(value)
	if clean == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: clean, Valid: true}
}

func toPgInt32(value int32) pgtype.Int4 {
	return pgtype.Int4{Int32: value, Valid: true}
}

func timeFromPg(ts pgtype.Timestamptz) (time.Time, error) {
	if !ts.Valid {
		return time.Time{}, errors.New("invalid timestamp")
	}
	return ts.Time, nil
}

func timePtrFromPg(ts pgtype.Timestamptz) *time.Time {
	if !ts.Valid {
		return nil
	}
	val := ts.Time
	return &val
}

func int32PtrFromPg(value pgtype.Int4) *int32 {
	if !value.Valid {
		return nil
	}
	val := value.Int32
	return &val
}

func textFromPg(value pgtype.Text) string {
	if !value.Valid {
		return ""
	}
	return value.String
}

func microsToUSD(micros int64) float64 {
	return float64(micros) / 1_000_000
}

func formatUSD(micros int64) string {
	return strconv.FormatFloat(microsToUSD(micros), 'f', 6, 64)
}

func dedupUUIDs(ids []uuid.UUID) []uuid.UUID {
	if len(ids) == 0 {
		return nil
	}
	seen := make(map[uuid.UUID]struct{}, len(ids))
	result := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		if id == uuid.Nil {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	return result
}

func (s *Service) ProcessNext(ctx context.Context) (Export, error) {
	record, err := s.ClaimNext(ctx)
	if err != nil {
		return Export{}, err
	}
	return s.Process(ctx, record)
}

func IsNotFound(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}
