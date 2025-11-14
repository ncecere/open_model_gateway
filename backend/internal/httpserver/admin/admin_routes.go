package admin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/redis/go-redis/v9"

	"github.com/ncecere/open_model_gateway/backend/internal/app"
	"github.com/ncecere/open_model_gateway/backend/internal/auth"
	"github.com/ncecere/open_model_gateway/backend/internal/config"
	"github.com/ncecere/open_model_gateway/backend/internal/db"
	"github.com/ncecere/open_model_gateway/backend/internal/httpserver/httputil"
)

const (
	oidcStatePrefix       = "oidc:state:"
	oidcStateTTL          = 10 * time.Minute
	defaultOIDCReturnPath = "/admin/ui/auth/oidc/callback"
)

type oidcStateData struct {
	Nonce    string `json:"nonce"`
	ReturnTo string `json:"return_to"`
}

func registerAdminAuthRoutes(router fiber.Router, container *app.Container) {
	handler := &adminAuthHandler{
		authService: container.AdminAuth,
		redis:       container.Redis,
		cfg:         container.Config.Admin,
		queries:     container.Queries,
	}

	router.Get("/methods", handler.listMethods)
	router.Post("/login", handler.loginLocal)
	router.Post("/refresh", handler.refresh)
	router.Post("/logout", handler.logout)
	router.Get("/oidc/start", handler.oidcStart)
	router.Get("/oidc/callback", handler.oidcCallback)
}

type adminAuthHandler struct {
	authService *auth.AdminAuthService
	redis       *redis.Client
	cfg         config.AdminConfig
	queries     *db.Queries
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type tokenResponse struct {
	AccessToken      string       `json:"access_token"`
	AccessExpiresAt  time.Time    `json:"access_expires_at"`
	RefreshExpiresAt time.Time    `json:"refresh_expires_at"`
	Method           string       `json:"method"`
	User             userResponse `json:"user"`
	RefreshToken     string       `json:"refresh_token,omitempty"`
}

type userResponse struct {
	ID           string     `json:"id"`
	Email        string     `json:"email"`
	Name         string     `json:"name"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	LastLoginAt  *time.Time `json:"last_login_at,omitempty"`
	IsSuperAdmin bool       `json:"is_super_admin"`
}

type oidcStartResponse struct {
	AuthURL string `json:"auth_url"`
	State   string `json:"state"`
}

func (h *adminAuthHandler) listMethods(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"methods": h.authService.AllowedAuthMethods(),
	})
}

func (h *adminAuthHandler) loginLocal(c *fiber.Ctx) error {
	if !h.cfg.Local.Enabled {
		return httputil.WriteError(c, fiber.StatusNotFound, "local authentication disabled")
	}

	var req loginRequest
	if err := c.BodyParser(&req); err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid request body")
	}
	req.Email = strings.TrimSpace(req.Email)

	if req.Email == "" || req.Password == "" {
		return httputil.WriteError(c, fiber.StatusBadRequest, "email and password required")
	}

	ctx := userContext(c)
	pair, user, err := h.authService.AuthenticateLocal(ctx, req.Email, req.Password)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidCredentials) {
			return httputil.WriteError(c, fiber.StatusUnauthorized, "invalid credentials")
		}
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}

	h.setRefreshCookie(c, pair)

	resp, err := buildTokenResponse(pair, user, auth.ProviderLocal)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	resp.RefreshToken = pair.RefreshToken

	return c.JSON(resp)
}

func (h *adminAuthHandler) refresh(c *fiber.Ctx) error {
	var req refreshRequest
	if len(c.Body()) > 0 {
		if err := c.BodyParser(&req); err != nil {
			return httputil.WriteError(c, fiber.StatusBadRequest, "invalid request body")
		}
	}

	token := strings.TrimSpace(req.RefreshToken)
	if token == "" {
		token = strings.TrimSpace(c.Cookies(h.cfg.Session.CookieName))
	}
	if token == "" {
		return httputil.WriteError(c, fiber.StatusBadRequest, "refresh token required")
	}

	ctx := userContext(c)
	userID, err := h.authService.ValidateRefreshToken(token)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusUnauthorized, "invalid refresh token")
	}

	user, err := h.queries.GetUserByID(ctx, toPgUUID(userID))
	if err != nil {
		return httputil.WriteError(c, fiber.StatusUnauthorized, "user not found")
	}

	pair, err := h.authService.IssueTokenPair(user)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}

	h.setRefreshCookie(c, pair)
	resp, err := buildTokenResponse(pair, user, "refresh")
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	resp.RefreshToken = pair.RefreshToken

	return c.JSON(resp)
}

func (h *adminAuthHandler) logout(c *fiber.Ctx) error {
	h.clearRefreshCookie(c)
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *adminAuthHandler) oidcStart(c *fiber.Ctx) error {
	if !h.cfg.OIDC.Enabled {
		return httputil.WriteError(c, fiber.StatusNotFound, "oidc disabled")
	}

	returnTo := sanitizeReturnPath(c.Query("return_to"))
	if returnTo == "" {
		returnTo = defaultOIDCReturnPath
	}

	state, err := auth.GenerateState(32)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	nonce, err := auth.GenerateState(32)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}

	authURL, err := h.authService.StartOIDCAuth(state, nonce)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}

	ctx := userContext(c)
	payload, err := json.Marshal(oidcStateData{
		Nonce:    nonce,
		ReturnTo: returnTo,
	})
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "failed to encode oidc state")
	}
	if err := h.redis.Set(ctx, oidcStateKey(state), payload, oidcStateTTL).Err(); err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "failed to persist oidc state")
	}

	return c.JSON(oidcStartResponse{
		AuthURL: authURL,
		State:   state,
	})
}

func (h *adminAuthHandler) oidcCallback(c *fiber.Ctx) error {
	if !h.cfg.OIDC.Enabled {
		return httputil.WriteError(c, fiber.StatusNotFound, "oidc disabled")
	}

	state := c.Query("state")
	code := c.Query("code")
	if state == "" || code == "" {
		return httputil.WriteError(c, fiber.StatusBadRequest, "state and code required")
	}

	ctx := userContext(c)
	key := oidcStateKey(state)
	rawState, err := h.redis.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return redirectOIDC(c, "", fmt.Errorf("invalid or expired state"))
		}
		return redirectOIDC(c, "", fmt.Errorf("failed to validate oidc state: %w", err))
	}
	_ = h.redis.Del(ctx, key).Err()

	stateData := oidcStateData{ReturnTo: defaultOIDCReturnPath}
	if err := json.Unmarshal(rawState, &stateData); err != nil {
		return redirectOIDC(c, "", fmt.Errorf("invalid oidc state payload"))
	}

	pair, _, err := h.authService.CompleteOIDCAuth(ctx, code, stateData.Nonce)
	if err != nil {
		return redirectOIDC(c, stateData.ReturnTo, err)
	}

	h.setRefreshCookie(c, pair)

	return redirectOIDC(c, stateData.ReturnTo, nil)
}

func (h *adminAuthHandler) setRefreshCookie(c *fiber.Ctx, pair *auth.TokenPair) {
	secure := strings.EqualFold(c.Protocol(), "https")

	c.Cookie(&fiber.Cookie{
		Name:        h.cfg.Session.CookieName,
		Value:       pair.RefreshToken,
		HTTPOnly:    true,
		Secure:      secure,
		Path:        "/",
		Expires:     pair.RefreshExpiresAt,
		SameSite:    fiber.CookieSameSiteLaxMode,
		SessionOnly: false,
	})
}

func (h *adminAuthHandler) clearRefreshCookie(c *fiber.Ctx) {
	secure := strings.EqualFold(c.Protocol(), "https")
	c.Cookie(&fiber.Cookie{
		Name:        h.cfg.Session.CookieName,
		Value:       "",
		Path:        "/",
		Expires:     time.Unix(0, 0),
		HTTPOnly:    true,
		Secure:      secure,
		SameSite:    fiber.CookieSameSiteLaxMode,
		SessionOnly: false,
	})
}

func buildTokenResponse(pair *auth.TokenPair, user db.User, method string) (tokenResponse, error) {
	userResp, err := toUserResponse(user)
	if err != nil {
		return tokenResponse{}, err
	}

	return tokenResponse{
		AccessToken:      pair.AccessToken,
		AccessExpiresAt:  pair.AccessExpiresAt,
		RefreshExpiresAt: pair.RefreshExpiresAt,
		Method:           method,
		User:             userResp,
	}, nil
}

func toUserResponse(u db.User) (userResponse, error) {
	id, err := uuidFromPg(u.ID)
	if err != nil {
		return userResponse{}, err
	}
	createdAt, err := timeFromPg(u.CreatedAt)
	if err != nil {
		return userResponse{}, err
	}
	updatedAt, err := timeFromPg(u.UpdatedAt)
	if err != nil {
		return userResponse{}, err
	}

	var lastLoginPtr *time.Time
	if u.LastLoginAt.Valid {
		t, err := timeFromPg(u.LastLoginAt)
		if err != nil {
			return userResponse{}, err
		}
		lastLoginPtr = &t
	}

	return userResponse{
		ID:           id,
		Email:        u.Email,
		Name:         u.Name,
		CreatedAt:    createdAt,
		UpdatedAt:    updatedAt,
		LastLoginAt:  lastLoginPtr,
		IsSuperAdmin: u.IsSuperAdmin,
	}, nil
}

func uuidFromPg(id pgtype.UUID) (string, error) {
	if !id.Valid {
		return "", errors.New("invalid uuid value")
	}
	u, err := uuid.FromBytes(id.Bytes[:])
	if err != nil {
		return "", err
	}
	return u.String(), nil
}

func timeFromPg(ts pgtype.Timestamptz) (time.Time, error) {
	if !ts.Valid {
		return time.Time{}, errors.New("invalid timestamp")
	}
	return ts.Time, nil
}

func oidcStateKey(state string) string {
	return fmt.Sprintf("%s%s", oidcStatePrefix, state)
}

func sanitizeReturnPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return defaultOIDCReturnPath
	}
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return defaultOIDCReturnPath
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	allowed := []string{"/admin", "/user", "/auth"}
	for _, prefix := range allowed {
		if strings.HasPrefix(path, prefix) {
			return path
		}
	}
	return defaultOIDCReturnPath
}

func redirectOIDC(c *fiber.Ctx, path string, err error) error {
	target := sanitizeReturnPath(path)
	if err != nil {
		target = appendQueryParam(target, "error", err.Error())
	} else {
		target = appendQueryParam(target, "status", "success")
	}
	return c.Redirect(target, fiber.StatusTemporaryRedirect)
}

func appendQueryParam(path string, key, value string) string {
	if key == "" || value == "" {
		return path
	}
	escaped := url.QueryEscape(value)
	sep := "?"
	if strings.Contains(path, "?") {
		sep = "&"
	}
	return fmt.Sprintf("%s%s%s=%s", path, sep, key, escaped)
}

func userContext(c *fiber.Ctx) context.Context {
	if uc := c.UserContext(); uc != nil {
		return uc
	}
	return context.Background()
}

func toPgUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: true}
}

func fromPgUUID(id pgtype.UUID) (uuid.UUID, error) {
	if !id.Valid {
		return uuid.Nil, errors.New("invalid uuid")
	}
	return id.Bytes, nil
}
