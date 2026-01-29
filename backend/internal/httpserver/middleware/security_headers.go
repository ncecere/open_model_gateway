package middleware

import (
	"fmt"

	"github.com/gofiber/fiber/v2"

	"github.com/ncecere/open_model_gateway/backend/internal/config"
)

// SecurityHeaders returns a Fiber middleware that sets standard security headers
// based on the provided configuration.
func SecurityHeaders(cfg config.SecurityConfig) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if cfg.ContentTypeNosniff {
			c.Set("X-Content-Type-Options", "nosniff")
		}

		if cfg.FrameOptions != "" {
			c.Set("X-Frame-Options", cfg.FrameOptions)
		}

		if cfg.HSTS {
			maxAge := cfg.HSTSMaxAge
			if maxAge <= 0 {
				maxAge = 31536000
			}
			c.Set("Strict-Transport-Security", fmt.Sprintf("max-age=%d; includeSubDomains", maxAge))
		}

		if cfg.CSP != "" {
			c.Set("Content-Security-Policy", cfg.CSP)
		}

		// Always set referrer policy for defense-in-depth.
		c.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Set("X-XSS-Protection", "0") // Modern browsers: CSP replaces this; "0" avoids XSS auditor bugs.

		return c.Next()
	}
}
