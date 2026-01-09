package httputil

import (
	"net/http"

	"github.com/gofiber/fiber/v2"

	"github.com/ncecere/open_model_gateway/backend/internal/apperror"
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

// WriteAppError writes an error response using the apperror package to determine
// the appropriate status code and message. This provides consistent error handling
// across the application.
func WriteAppError(c *fiber.Ctx, err error) error {
	if err == nil {
		return nil
	}
	status := apperror.StatusCode(err)
	msg := apperror.GetMessage(err)
	code := apperror.GetCode(err)

	return c.Status(status).JSON(fiber.Map{
		"error": msg,
		"code":  string(code),
	})
}

// WriteAppErrorWithMessage writes an error response using the apperror package
// but allows overriding the message.
func WriteAppErrorWithMessage(c *fiber.Ctx, err error, message string) error {
	if err == nil {
		return nil
	}
	status := apperror.StatusCode(err)
	code := apperror.GetCode(err)

	return c.Status(status).JSON(fiber.Map{
		"error": message,
		"code":  string(code),
	})
}
