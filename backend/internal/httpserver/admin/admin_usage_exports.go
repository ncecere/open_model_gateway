package admin

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/ncecere/open_model_gateway/backend/internal/app"
	"github.com/ncecere/open_model_gateway/backend/internal/db"
	"github.com/ncecere/open_model_gateway/backend/internal/httpserver/httputil"
	exportsvc "github.com/ncecere/open_model_gateway/backend/internal/services/exports"
)

type usageExportHandler struct {
	container *app.Container
	exports   *exportsvc.Service
}

type usageExportCreateRequest struct {
	TenantIDs   []string `json:"tenant_ids"`
	TenantID    string   `json:"tenant_id"`
	Period      string   `json:"period"`
	Start       string   `json:"start"`
	End         string   `json:"end"`
	Granularity string   `json:"granularity"`
	Format      string   `json:"format"`
	Timezone    string   `json:"timezone"`
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

func registerAdminUsageExportRoutes(router fiber.Router, container *app.Container) {
	handler := &usageExportHandler{container: container, exports: container.Exports}
	group := router.Group("/usage-exports")
	group.Post("/", handler.create)
	group.Get("/", handler.list)
	group.Get("/:id", handler.get)
	group.Get("/:id/content", handler.download)
}

func (h *usageExportHandler) create(c *fiber.Ctx) error {
	if err := requireAnyRole(c, h.container, db.MembershipRoleViewer); err != nil {
		return err
	}
	if h.exports == nil {
		return httputil.WriteError(c, fiber.StatusNotImplemented, "usage exports unavailable")
	}
	var req usageExportCreateRequest
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
	adminUser, ok := adminUserFromContext(c.UserContext())
	if !ok {
		return httputil.WriteError(c, fiber.StatusUnauthorized, "missing admin context")
	}
	adminID, ok := adminUserIDFromContext(c.UserContext())
	if !ok {
		return httputil.WriteError(c, fiber.StatusUnauthorized, "missing admin context")
	}
	personalTenantID, err := fromPgUUID(adminUser.PersonalTenantID)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "admin personal tenant missing")
	}

	ids := append([]string{}, req.TenantIDs...)
	if strings.TrimSpace(req.TenantID) != "" {
		ids = append(ids, req.TenantID)
	}
	parsedIDs := make([]uuid.UUID, 0, len(ids))
	for _, raw := range ids {
		id, err := uuid.Parse(strings.TrimSpace(raw))
		if err != nil {
			return httputil.WriteError(c, fiber.StatusBadRequest, "invalid tenant_id")
		}
		if err := requireTenantRole(c, h.container, id, db.MembershipRoleViewer); err != nil {
			return err
		}
		parsedIDs = append(parsedIDs, id)
	}
	if len(parsedIDs) == 0 && !adminUser.IsSuperAdmin {
		return httputil.WriteError(c, fiber.StatusBadRequest, "tenant_ids required for non-super admins")
	}

	export, err := h.exports.Create(c.Context(), exportsvc.CreateParams{
		Scope:        "admin",
		RequestedBy:  adminID,
		FileTenantID: personalTenantID,
		TenantIDs:    parsedIDs,
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
	return c.JSON(serializeUsageExport(export, "/admin"))
}

func (h *usageExportHandler) list(c *fiber.Ctx) error {
	if err := requireAnyRole(c, h.container, db.MembershipRoleViewer); err != nil {
		return err
	}
	if h.exports == nil {
		return httputil.WriteError(c, fiber.StatusNotImplemented, "usage exports unavailable")
	}
	limit := int32(parsePositiveInt(c.Query("limit"), 25))
	offset := int32(parsePositiveInt(c.Query("offset"), 0))
	exports, err := h.exports.ListAdmin(c.Context(), limit, offset)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	items := make([]usageExportResponse, 0, len(exports))
	for _, export := range exports {
		items = append(items, serializeUsageExport(export, "/admin"))
	}
	return c.JSON(fiber.Map{"exports": items})
}

func (h *usageExportHandler) get(c *fiber.Ctx) error {
	if err := requireAnyRole(c, h.container, db.MembershipRoleViewer); err != nil {
		return err
	}
	if h.exports == nil {
		return httputil.WriteError(c, fiber.StatusNotImplemented, "usage exports unavailable")
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid export id")
	}
	export, err := h.exports.Get(c.Context(), id)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusNotFound, "export not found")
	}
	return c.JSON(serializeUsageExport(export, "/admin"))
}

func (h *usageExportHandler) download(c *fiber.Ctx) error {
	if err := requireAnyRole(c, h.container, db.MembershipRoleViewer); err != nil {
		return err
	}
	if h.exports == nil || h.container.Files == nil {
		return httputil.WriteError(c, fiber.StatusNotImplemented, "usage exports unavailable")
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid export id")
	}
	export, err := h.exports.Get(c.Context(), id)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusNotFound, "export not found")
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

func serializeUsageExport(export exportsvc.Export, basePath string) usageExportResponse {
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
		url := fmt.Sprintf("%s/usage-exports/%s/content", basePath, resp.ID)
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
