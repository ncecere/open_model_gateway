package admin

import (
	"encoding/json"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/ncecere/open_model_gateway/backend/internal/db"
	"github.com/ncecere/open_model_gateway/backend/internal/httpserver/httputil"
)

func (h *tenantHandler) listAPIKeys(c *fiber.Ctx) error {
	id, err := uuid.Parse(strings.TrimSpace(c.Params("tenantID")))
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid tenant id")
	}

	if err := requireTenantRole(c, h.container, id, db.MembershipRoleViewer); err != nil {
		return err
	}

	if h.service == nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "tenant service unavailable")
	}
	keys, err := h.service.ListAPIKeys(c.Context(), id)
	if err != nil {
		return writeTenantServiceError(c, err)
	}

	tenantName := h.lookupTenantName(c.Context(), id)
	responses := make([]apiKeyResponse, 0, len(keys))
	for _, key := range keys {
		issuer := resolveAPIKeyIssuer(key, tenantName, "", "")
		resp, err := buildAPIKeyResponse(c.Context(), h.container, key, id, tenantName, issuer)
		if err != nil {
			return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
		}
		responses = append(responses, resp)
	}

	if err := recordAudit(c, h.container, "api_key.list", "tenant", id.String(), fiber.Map{
		"count": len(responses),
	}); err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(fiber.Map{
		"api_keys": responses,
	})
}

func (h *tenantHandler) createAPIKey(c *fiber.Ctx) error {
	tenantID, err := uuid.Parse(strings.TrimSpace(c.Params("tenantID")))
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid tenant id")
	}

	if err := requireTenantRole(c, h.container, tenantID, db.MembershipRoleAdmin); err != nil {
		return err
	}

	var req createAPIKeyRequest
	if err := c.BodyParser(&req); err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid request body")
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		return httputil.WriteError(c, fiber.StatusBadRequest, "name is required")
	}

	var scopesJSON []byte
	if len(req.Scopes) > 0 {
		var err error
		scopesJSON, err = json.Marshal(req.Scopes)
		if err != nil {
			return httputil.WriteError(c, fiber.StatusBadRequest, "invalid scopes")
		}
	} else {
		scopesJSON = []byte("[]")
	}

	var quotaJSON []byte
	if req.Quota != nil {
		var err error
		quotaJSON, err = json.Marshal(req.Quota)
		if err != nil {
			return httputil.WriteError(c, fiber.StatusBadRequest, "invalid quota")
		}
	} else {
		quotaJSON = []byte("{}")
	}

	budgetLimit, _, err := h.container.EffectiveTenantBudget(c.Context(), tenantID)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	if err := validateQuotaAgainstBudget(req.Quota, budgetLimit); err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, err.Error())
	}

	rateLimitCfg, err := validateAPIKeyRateLimitRequest(h.container, tenantID, req.RateLimits)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, err.Error())
	}

	if h.service == nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "tenant service unavailable")
	}
	result, err := h.service.CreateAPIKey(c.Context(), tenantID, req.Name, scopesJSON, quotaJSON, rateLimitCfg)
	if err != nil {
		return writeTenantServiceError(c, err)
	}
	tenantName := h.lookupTenantName(c.Context(), tenantID)
	issuer := resolveAPIKeyIssuer(result.Key, tenantName, "", "")
	response, err := buildAPIKeyResponse(c.Context(), h.container, result.Key, tenantID, tenantName, issuer)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}

	if err := recordAudit(c, h.container, "api_key.create", "api_key", response.ID, fiber.Map{
		"tenant_id": tenantID.String(),
		"prefix":    response.Prefix,
	}); err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}

	return c.Status(fiber.StatusCreated).JSON(createAPIKeyResponse{
		APIKeyResponse: response,
		Secret:         result.Secret,
		Token:          result.Token,
	})
}

func (h *tenantHandler) revokeAPIKey(c *fiber.Ctx) error {
	tenantID, err := uuid.Parse(strings.TrimSpace(c.Params("tenantID")))
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid tenant id")
	}
	apiKeyID, err := uuid.Parse(strings.TrimSpace(c.Params("apiKeyID")))
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid api key id")
	}

	if err := requireTenantRole(c, h.container, tenantID, db.MembershipRoleAdmin); err != nil {
		return err
	}

	if h.service == nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "tenant service unavailable")
	}

	record, err := h.service.RevokeAPIKey(c.Context(), tenantID, apiKeyID)
	if err != nil {
		return writeTenantServiceError(c, err)
	}

	tenantName := h.lookupTenantName(c.Context(), tenantID)
	issuer := resolveAPIKeyIssuer(record, tenantName, "", "")
	response, err := buildAPIKeyResponse(c.Context(), h.container, record, tenantID, tenantName, issuer)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	response.Revoked = true

	if err := recordAudit(c, h.container, "api_key.revoke", "api_key", response.ID, fiber.Map{
		"tenant_id": tenantID.String(),
	}); err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(response)
}
