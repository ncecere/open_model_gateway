// Package apperror provides standardized error handling for the Open Model Gateway.
// It defines sentinel errors, error wrapping utilities, and HTTP status mapping.
package apperror

import (
	"errors"
	"fmt"
	"net/http"
	"time"
)

// Sentinel errors for common application errors.
// These can be checked with errors.Is() for error categorization.
var (
	// ErrNotFound indicates the requested resource does not exist.
	ErrNotFound = errors.New("not found")

	// ErrUnauthorized indicates missing or invalid authentication.
	ErrUnauthorized = errors.New("unauthorized")

	// ErrForbidden indicates the user lacks permission for the operation.
	ErrForbidden = errors.New("forbidden")

	// ErrRateLimited indicates the request was rate limited.
	ErrRateLimited = errors.New("rate limited")

	// ErrBudgetExceeded indicates the tenant's budget has been exceeded.
	ErrBudgetExceeded = errors.New("budget exceeded")

	// ErrBadRequest indicates invalid input or request format.
	ErrBadRequest = errors.New("bad request")

	// ErrConflict indicates a resource conflict (e.g., duplicate key).
	ErrConflict = errors.New("conflict")

	// ErrInternal indicates an unexpected internal error.
	ErrInternal = errors.New("internal error")

	// ErrServiceUnavailable indicates a service is temporarily unavailable.
	ErrServiceUnavailable = errors.New("service unavailable")

	// ErrTimeout indicates the operation timed out.
	ErrTimeout = errors.New("timeout")

	// ErrValidation indicates input validation failed.
	ErrValidation = errors.New("validation error")
)

// Code represents an application-specific error code.
type Code string

// Common error codes for categorization and client handling.
const (
	CodeNotFound           Code = "NOT_FOUND"
	CodeUnauthorized       Code = "UNAUTHORIZED"
	CodeForbidden          Code = "FORBIDDEN"
	CodeRateLimited        Code = "RATE_LIMITED"
	CodeBudgetExceeded     Code = "BUDGET_EXCEEDED"
	CodeBadRequest         Code = "BAD_REQUEST"
	CodeConflict           Code = "CONFLICT"
	CodeInternal           Code = "INTERNAL_ERROR"
	CodeServiceUnavailable Code = "SERVICE_UNAVAILABLE"
	CodeTimeout            Code = "TIMEOUT"
	CodeValidation         Code = "VALIDATION_ERROR"
)

// Error represents a structured application error with context.
type Error struct {
	// Op is the operation being performed (e.g., "tenant.Create").
	Op string

	// Code is the application-specific error code.
	Code Code

	// Message is a human-readable error message.
	Message string

	// Err is the underlying error, if any.
	Err error

	// RetryAfter is the duration the caller should wait before retrying.
	// Only meaningful for rate-limited or overloaded errors.
	RetryAfter time.Duration
}

// Error returns a formatted error message.
func (e *Error) Error() string {
	if e.Op != "" {
		if e.Message != "" {
			return fmt.Sprintf("%s: %s", e.Op, e.Message)
		}
		if e.Err != nil {
			return fmt.Sprintf("%s: %v", e.Op, e.Err)
		}
		return e.Op
	}
	if e.Message != "" {
		return e.Message
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return string(e.Code)
}

// Unwrap returns the underlying error for error chain traversal.
func (e *Error) Unwrap() error {
	return e.Err
}

// Is reports whether target matches this error.
func (e *Error) Is(target error) bool {
	if e.Err != nil && errors.Is(e.Err, target) {
		return true
	}
	// Check by code
	if t, ok := target.(*Error); ok {
		return e.Code == t.Code
	}
	return false
}

// New creates a new Error with the given parameters.
func New(op string, code Code, message string) *Error {
	return &Error{
		Op:      op,
		Code:    code,
		Message: message,
	}
}

// Wrap wraps an existing error with additional context.
func Wrap(op string, err error) *Error {
	if err == nil {
		return nil
	}

	// If already an Error, preserve the code but add operation context
	if e, ok := err.(*Error); ok {
		return &Error{
			Op:      op,
			Code:    e.Code,
			Message: e.Message,
			Err:     err,
		}
	}

	// Determine code from sentinel errors
	code := codeFromError(err)
	return &Error{
		Op:   op,
		Code: code,
		Err:  err,
	}
}

// WrapWithMessage wraps an error with additional context and a custom message.
func WrapWithMessage(op string, err error, message string) *Error {
	if err == nil {
		return nil
	}

	code := codeFromError(err)
	if e, ok := err.(*Error); ok {
		code = e.Code
	}

	return &Error{
		Op:      op,
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// codeFromError determines the error code from sentinel errors.
func codeFromError(err error) Code {
	switch {
	case errors.Is(err, ErrNotFound):
		return CodeNotFound
	case errors.Is(err, ErrUnauthorized):
		return CodeUnauthorized
	case errors.Is(err, ErrForbidden):
		return CodeForbidden
	case errors.Is(err, ErrRateLimited):
		return CodeRateLimited
	case errors.Is(err, ErrBudgetExceeded):
		return CodeBudgetExceeded
	case errors.Is(err, ErrBadRequest):
		return CodeBadRequest
	case errors.Is(err, ErrConflict):
		return CodeConflict
	case errors.Is(err, ErrServiceUnavailable):
		return CodeServiceUnavailable
	case errors.Is(err, ErrTimeout):
		return CodeTimeout
	case errors.Is(err, ErrValidation):
		return CodeValidation
	default:
		return CodeInternal
	}
}

// NotFound creates a not found error.
func NotFound(op, message string) *Error {
	return &Error{
		Op:      op,
		Code:    CodeNotFound,
		Message: message,
		Err:     ErrNotFound,
	}
}

// Unauthorized creates an unauthorized error.
func Unauthorized(op, message string) *Error {
	return &Error{
		Op:      op,
		Code:    CodeUnauthorized,
		Message: message,
		Err:     ErrUnauthorized,
	}
}

// Forbidden creates a forbidden error.
func Forbidden(op, message string) *Error {
	return &Error{
		Op:      op,
		Code:    CodeForbidden,
		Message: message,
		Err:     ErrForbidden,
	}
}

// RateLimited creates a rate limited error.
func RateLimited(op, message string) *Error {
	return &Error{
		Op:      op,
		Code:    CodeRateLimited,
		Message: message,
		Err:     ErrRateLimited,
	}
}

// RateLimitedWithRetryAfter creates a rate limited error with a retry-after hint.
func RateLimitedWithRetryAfter(op, message string, retryAfter time.Duration) *Error {
	return &Error{
		Op:         op,
		Code:       CodeRateLimited,
		Message:    message,
		Err:        ErrRateLimited,
		RetryAfter: retryAfter,
	}
}

// GetRetryAfter extracts the RetryAfter duration from an error chain.
// Returns 0 if the error does not carry retry-after information.
func GetRetryAfter(err error) time.Duration {
	var e *Error
	if errors.As(err, &e) && e.RetryAfter > 0 {
		return e.RetryAfter
	}
	return 0
}

// BudgetExceeded creates a budget exceeded error.
func BudgetExceeded(op, message string) *Error {
	return &Error{
		Op:      op,
		Code:    CodeBudgetExceeded,
		Message: message,
		Err:     ErrBudgetExceeded,
	}
}

// BadRequest creates a bad request error.
func BadRequest(op, message string) *Error {
	return &Error{
		Op:      op,
		Code:    CodeBadRequest,
		Message: message,
		Err:     ErrBadRequest,
	}
}

// Conflict creates a conflict error.
func Conflict(op, message string) *Error {
	return &Error{
		Op:      op,
		Code:    CodeConflict,
		Message: message,
		Err:     ErrConflict,
	}
}

// Internal creates an internal error.
func Internal(op, message string) *Error {
	return &Error{
		Op:      op,
		Code:    CodeInternal,
		Message: message,
		Err:     ErrInternal,
	}
}

// ServiceUnavailable creates a service unavailable error.
func ServiceUnavailable(op, message string) *Error {
	return &Error{
		Op:      op,
		Code:    CodeServiceUnavailable,
		Message: message,
		Err:     ErrServiceUnavailable,
	}
}

// Validation creates a validation error.
func Validation(op, message string) *Error {
	return &Error{
		Op:      op,
		Code:    CodeValidation,
		Message: message,
		Err:     ErrValidation,
	}
}

// StatusCode returns the appropriate HTTP status code for the error.
func StatusCode(err error) int {
	if err == nil {
		return http.StatusOK
	}

	// Check for Error type first
	var e *Error
	if errors.As(err, &e) {
		return statusFromCode(e.Code)
	}

	// Check sentinel errors
	switch {
	case errors.Is(err, ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, ErrUnauthorized):
		return http.StatusUnauthorized
	case errors.Is(err, ErrForbidden):
		return http.StatusForbidden
	case errors.Is(err, ErrRateLimited):
		return http.StatusTooManyRequests
	case errors.Is(err, ErrBudgetExceeded):
		return http.StatusPaymentRequired
	case errors.Is(err, ErrBadRequest):
		return http.StatusBadRequest
	case errors.Is(err, ErrConflict):
		return http.StatusConflict
	case errors.Is(err, ErrServiceUnavailable):
		return http.StatusServiceUnavailable
	case errors.Is(err, ErrTimeout):
		return http.StatusGatewayTimeout
	case errors.Is(err, ErrValidation):
		return http.StatusUnprocessableEntity
	default:
		return http.StatusInternalServerError
	}
}

// statusFromCode maps error codes to HTTP status codes.
func statusFromCode(code Code) int {
	switch code {
	case CodeNotFound:
		return http.StatusNotFound
	case CodeUnauthorized:
		return http.StatusUnauthorized
	case CodeForbidden:
		return http.StatusForbidden
	case CodeRateLimited:
		return http.StatusTooManyRequests
	case CodeBudgetExceeded:
		return http.StatusPaymentRequired
	case CodeBadRequest:
		return http.StatusBadRequest
	case CodeConflict:
		return http.StatusConflict
	case CodeServiceUnavailable:
		return http.StatusServiceUnavailable
	case CodeTimeout:
		return http.StatusGatewayTimeout
	case CodeValidation:
		return http.StatusUnprocessableEntity
	default:
		return http.StatusInternalServerError
	}
}

// GetCode extracts the error code from an error.
// Returns CodeInternal if the error is not an Error type.
func GetCode(err error) Code {
	if err == nil {
		return ""
	}
	var e *Error
	if errors.As(err, &e) {
		return e.Code
	}
	return codeFromError(err)
}

// GetMessage extracts a user-friendly message from an error.
func GetMessage(err error) string {
	if err == nil {
		return ""
	}
	var e *Error
	if errors.As(err, &e) && e.Message != "" {
		return e.Message
	}
	return err.Error()
}
