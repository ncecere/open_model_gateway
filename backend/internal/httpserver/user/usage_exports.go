package user

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/ncecere/open_model_gateway/backend/internal/httpserver/httputil"
	exportsvc "github.com/ncecere/open_model_gateway/backend/internal/services/exports"
)

type usageExportRequest struct {
	TenantID    string `json:"tenant_id"`
	Period      string `json:"period"`
	Start       string `json:"start"`
	End         string `json:"end"`
	Granularity string `json:"granularity"`
	Format      string `json:"format"`
	Timezone    string `json:"timezone"`
}

type usageExportResponse struct {
	ID           string   `json:"id"`
	Scope        string   `json:"scope"`
	Status       string   `json:"status"`
	Format       string   `json:"format"`
	Granularity  string   `json:"granularity"`
	Timezone     string   `json:"timezone"`
	PeriodStart  string   `json:"period_start"`
	PeriodEnd    string   `json:"period_end"`
	TenantIDs    []string `json:"tenant_ids"`
	RequestedBy  *string  `json:"requested_by,omitempty"`
	FileID       *string  `json:"file_id,omitempty"`
	FileTenantID *string  `json:"file_tenant_id,omitempty"`
	RowCount     *int32   `json:"row_count,omitempty"`
	Error        string   `json:"error,omitempty"`
	CreatedAt    string   `json:"created_at"`
	UpdatedAt    string   `json:"updated_at"`
	StartedAt    *string  `json:"started_at,omitempty"`
	CompletedAt  *string  `json:"completed_at,omitempty"`
	DownloadURL  *string  `json:"download_url,omitempty"`
}

func (h *userHandler) registerUsageExportRoutes(group fiber.Router) {
	group.Post("/usage-exports", h.createUsageExport)
	group.Get("/usage-exports", h.listUsageExports)
	group.Get("/usage-exports/:id", h.getUsageExport)
	group.Get("/usage-exports/:id/content", h.downloadUsageExport)
}

func (h *userHandler) createUsageExport(c *fiber.Ctx) error {
	if h.container == nil || h.container.Exports == nil {
		return httputil.WriteError(c, fiber.StatusNotImplemented, "usage exports unavailable")
	}
	var req usageExportRequest
	if err := c.BodyParser(&req); err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid request body")
	}
	startPtr, endPtr, err := parseRangeParams(req.Start, req.End)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, err.Error())
	}
	period := strings.TrimSpace(req.Period)
	if period == "" {
		period = "30d"
	}
	user, ok := userFromContext(c.UserContext())
	if !ok {
		return httputil.WriteError(c, fiber.StatusUnauthorized, "authentication required")
	}
	userID, err := uuidFromPg(user.ID)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusUnauthorized, "invalid user context")
	}
	_, allowed, err := h.loadTenantScope(c.Context(), user)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	var tenantID uuid.UUID
	if raw := strings.TrimSpace(req.TenantID); raw != "" {
		parsed, err := uuid.Parse(raw)
		if err != nil {
			return httputil.WriteError(c, fiber.StatusBadRequest, "invalid tenant_id")
		}
		if _, ok := allowed[parsed]; !ok {
			return httputil.WriteError(c, fiber.StatusForbidden, "tenant access denied")
		}
		tenantID = parsed
	} else {
		personalID, err := uuidFromPg(user.PersonalTenantID)
		if err != nil {
			return httputil.WriteError(c, fiber.StatusBadRequest, "personal tenant required")
		}
		tenantID = personalID
	}

	export, err := h.container.Exports.Create(c.Context(), exportsvc.CreateParams{
		Scope:        "user",
		RequestedBy:  userID,
		FileTenantID: tenantID,
		TenantIDs:    []uuid.UUID{tenantID},
		Period:       period,
		Start:        startPtr,
		End:          endPtr,
		Granularity:  req.Granularity,
		Format:       req.Format,
		Timezone:     req.Timezone,
	})
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, err.Error())
	}
	return c.JSON(serializeUserUsageExport(export))
}

func (h *userHandler) listUsageExports(c *fiber.Ctx) error {
	if h.container == nil || h.container.Exports == nil {
		return httputil.WriteError(c, fiber.StatusNotImplemented, "usage exports unavailable")
	}
	user, ok := userFromContext(c.UserContext())
	if !ok {
		return httputil.WriteError(c, fiber.StatusUnauthorized, "authentication required")
	}
	userID, err := uuidFromPg(user.ID)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusUnauthorized, "invalid user context")
	}
	limit := int32(parsePositiveInt(c.Query("limit"), 25))
	offset := int32(parsePositiveInt(c.Query("offset"), 0))
	exports, err := h.container.Exports.ListByRequester(c.Context(), userID, limit, offset)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	items := make([]usageExportResponse, 0, len(exports))
	for _, export := range exports {
		items = append(items, serializeUserUsageExport(export))
	}
	return c.JSON(fiber.Map{"exports": items})
}

func (h *userHandler) getUsageExport(c *fiber.Ctx) error {
	if h.container == nil || h.container.Exports == nil {
		return httputil.WriteError(c, fiber.StatusNotImplemented, "usage exports unavailable")
	}
	user, ok := userFromContext(c.UserContext())
	if !ok {
		return httputil.WriteError(c, fiber.StatusUnauthorized, "authentication required")
	}
	userID, err := uuidFromPg(user.ID)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusUnauthorized, "invalid user context")
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid export id")
	}
	export, err := h.container.Exports.Get(c.Context(), id)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusNotFound, "export not found")
	}
	if export.RequestedBy == nil || *export.RequestedBy != userID {
		return httputil.WriteError(c, fiber.StatusForbidden, "export access denied")
	}
	return c.JSON(serializeUserUsageExport(export))
}

func (h *userHandler) downloadUsageExport(c *fiber.Ctx) error {
	if h.container == nil || h.container.Exports == nil || h.container.Files == nil {
		return httputil.WriteError(c, fiber.StatusNotImplemented, "usage exports unavailable")
	}
	user, ok := userFromContext(c.UserContext())
	if !ok {
		return httputil.WriteError(c, fiber.StatusUnauthorized, "authentication required")
	}
	userID, err := uuidFromPg(user.ID)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusUnauthorized, "invalid user context")
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid export id")
	}
	export, err := h.container.Exports.Get(c.Context(), id)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusNotFound, "export not found")
	}
	if export.RequestedBy == nil || *export.RequestedBy != userID {
		return httputil.WriteError(c, fiber.StatusForbidden, "export access denied")
	}
	if export.Status != exportsvc.StatusReady || export.FileID == nil || export.FileTenantID == nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "export not ready")
	}
	reader, rec, err := h.container.Files.Open(c.Context(), *export.FileTenantID, *export.FileID)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusNotFound, "export file not found")
	}
	defer reader.Close()
	c.Set("Content-Type", rec.ContentType)
	c.Set("Content-Length", strconv.FormatInt(rec.Bytes, 10))
	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", rec.Filename))
	_, err = io.Copy(c, reader)
	return err
}

func serializeUserUsageExport(export exportsvc.Export) usageExportResponse {
	resp := usageExportResponse{
		ID:          export.ID.String(),
		Scope:       export.Scope,
		Status:      export.Status,
		Format:      export.Format,
		Granularity: export.Granularity,
		Timezone:    export.Timezone,
		PeriodStart: export.PeriodStart.Format(time.RFC3339),
		PeriodEnd:   export.PeriodEnd.Format(time.RFC3339),
		TenantIDs:   stringifyUUIDs(export.TenantIDs),
		Error:       export.Error,
		CreatedAt:   export.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   export.UpdatedAt.Format(time.RFC3339),
	}
	if export.RequestedBy != nil {
		id := export.RequestedBy.String()
		resp.RequestedBy = &id
	}
	if export.FileID != nil {
		id := export.FileID.String()
		resp.FileID = &id
	}
	if export.FileTenantID != nil {
		id := export.FileTenantID.String()
		resp.FileTenantID = &id
	}
	if export.RowCount != nil {
		resp.RowCount = export.RowCount
	}
	if export.StartedAt != nil {
		val := export.StartedAt.Format(time.RFC3339)
		resp.StartedAt = &val
	}
	if export.CompletedAt != nil {
		val := export.CompletedAt.Format(time.RFC3339)
		resp.CompletedAt = &val
	}
	if export.Status == exportsvc.StatusReady {
		url := fmt.Sprintf("/user/usage-exports/%s/content", resp.ID)
		resp.DownloadURL = &url
	}
	return resp
}

func stringifyUUIDs(ids []uuid.UUID) []string {
	result := make([]string, 0, len(ids))
	for _, id := range ids {
		result = append(result, id.String())
	}
	return result
}
