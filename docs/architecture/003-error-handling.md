# ADR 003: Standardized Error Handling

## Status
Accepted

## Context
Error handling was inconsistent across the codebase:
- Different HTTP status codes for similar errors
- Inconsistent error messages
- No structured error context
- Difficult to trace errors through the stack

## Decision
Create a centralized error handling package (`internal/apperror`) with:

1. **Sentinel errors** for common error conditions
2. **Structured Error type** with operation, code, and message
3. **HTTP status mapping** for consistent API responses
4. **Error wrapping utilities** for context preservation

## Error Types
```go
// Sentinel errors
var (
    ErrNotFound      = errors.New("not found")
    ErrUnauthorized  = errors.New("unauthorized")
    ErrForbidden     = errors.New("forbidden")
    ErrRateLimited   = errors.New("rate limited")
    ErrBudgetExceeded = errors.New("budget exceeded")
    ErrValidation    = errors.New("validation error")
    ErrInternal      = errors.New("internal error")
)

// Structured error with context
type Error struct {
    Op      string  // Operation being performed
    Code    string  // Machine-readable error code
    Message string  // Human-readable message
    Err     error   // Underlying error
}
```

## HTTP Status Mapping
```go
func StatusCode(err error) int {
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
    case errors.Is(err, ErrValidation):
        return http.StatusBadRequest
    default:
        return http.StatusInternalServerError
    }
}
```

## Usage Pattern
```go
// In service layer
func (s *Service) GetTenant(ctx context.Context, id uuid.UUID) (*Tenant, error) {
    tenant, err := s.repo.GetTenantByID(ctx, id)
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, apperror.Wrap(apperror.ErrNotFound, "GetTenant", err)
        }
        return nil, apperror.Wrap(apperror.ErrInternal, "GetTenant", err)
    }
    return tenant, nil
}

// In HTTP handler
func (h *Handler) GetTenant(c *fiber.Ctx) error {
    tenant, err := h.service.GetTenant(ctx, id)
    if err != nil {
        return httputil.WriteAppError(c, err)
    }
    return c.JSON(tenant)
}
```

## Consequences

### Positive
- Consistent error responses across API
- Better error context for debugging
- Clear mapping between errors and HTTP status codes
- Errors can be checked with `errors.Is()`
- Stack trace preserved through wrapping

### Negative
- Requires updating existing error handling code
- Additional package to learn
- Some overhead in error creation

## Implementation
The refactoring was done in Phase 2.1 of the refactoring effort. The `apperror` package was created and HTTP handlers were updated to use `WriteAppError`.
