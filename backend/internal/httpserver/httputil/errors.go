package httputil

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
)

// WriteError standardizes JSON error responses for both admin and public APIs.
func WriteError(c *fiber.Ctx, status int, msg string) error {
	if msg == "" {
		msg = http.StatusText(status)
		if msg == "" {
			msg = "unknown error"
		}
	}
	return c.Status(status).JSON(fiber.Map{
		"error": msg,
	})
}
