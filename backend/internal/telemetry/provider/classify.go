package provider

import (
	"context"
	"errors"

	"github.com/gofiber/fiber/v2"

	"github.com/ncecere/open_model_gateway/backend/internal/limits"
	"github.com/ncecere/open_model_gateway/backend/internal/models"
)

// ClassifyError maps provider errors into coarse classes for telemetry.
func ClassifyError(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "timeout"
	}
	if errors.Is(err, context.Canceled) {
		return "canceled"
	}
	if errors.Is(err, limits.ErrLimitExceeded) {
		return "rate_limit"
	}
	if errors.Is(err, models.ErrImageOperationUnsupported) {
		return "unsupported"
	}
	var apiErr interface{ Status() int }
	if errors.As(err, &apiErr) {
		code := apiErr.Status()
		if code >= 500 {
			return "5xx"
		}
		return "4xx"
	}
	var fiberErr *fiber.Error
	if errors.As(err, &fiberErr) {
		if fiberErr.Code >= 500 {
			return "5xx"
		}
		return "4xx"
	}
	return "transport"
}
