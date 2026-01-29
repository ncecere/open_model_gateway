package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/ncecere/open_model_gateway/backend/internal/httpserver/httputil"
)

// CSRFOriginCheck returns a Fiber middleware that rejects state-changing requests
// (POST, PUT, PATCH, DELETE) that carry a session cookie but have a mismatched
// or missing Origin header. This prevents cross-origin form/fetch attacks on
// cookie-authenticated endpoints.
//
// Safe methods (GET, HEAD, OPTIONS) are always allowed through.
// Requests using only Bearer token auth (no cookie) are exempt.
func CSRFOriginCheck(cookieName string, trustedOrigins []string) fiber.Handler {
	originSet := make(map[string]struct{}, len(trustedOrigins))
	for _, o := range trustedOrigins {
		originSet[strings.TrimRight(strings.ToLower(o), "/")] = struct{}{}
	}

	return func(c *fiber.Ctx) error {
		method := c.Method()
		if method == fiber.MethodGet || method == fiber.MethodHead || method == fiber.MethodOptions {
			return c.Next()
		}

		// Only enforce when a session cookie is present (cookie-based auth).
		cookie := c.Cookies(cookieName)
		if cookie == "" {
			return c.Next()
		}

		origin := strings.TrimRight(strings.ToLower(c.Get("Origin")), "/")
		if origin == "" {
			// Some clients (curl, server-to-server) omit Origin; fall back to Referer.
			referer := c.Get("Referer")
			if referer != "" {
				// Extract origin portion of referer.
				if idx := strings.Index(referer, "://"); idx >= 0 {
					rest := referer[idx+3:]
					if slashIdx := strings.Index(rest, "/"); slashIdx >= 0 {
						origin = strings.ToLower(referer[:idx+3+slashIdx])
					} else {
						origin = strings.ToLower(referer)
					}
				}
			}
		}

		if origin == "" {
			// No origin information at all with a cookie-based request is suspicious.
			return httputil.WriteError(c, fiber.StatusForbidden, "origin header required for cookie-authenticated requests")
		}

		if _, ok := originSet[origin]; ok {
			return c.Next()
		}

		// Allow same-origin requests (Host matches Origin host).
		host := strings.ToLower(c.Hostname())
		originHost := origin
		if idx := strings.Index(origin, "://"); idx >= 0 {
			originHost = origin[idx+3:]
		}
		if originHost == host {
			return c.Next()
		}

		return httputil.WriteError(c, fiber.StatusForbidden, "cross-origin request blocked")
	}
}
