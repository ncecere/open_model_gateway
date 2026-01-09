package apperror

import (
	"errors"
	"net/http"
	"testing"
)

func TestError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *Error
		expected string
	}{
		{
			name: "with op and message",
			err: &Error{
				Op:      "tenant.Create",
				Code:    CodeBadRequest,
				Message: "invalid tenant name",
			},
			expected: "tenant.Create: invalid tenant name",
		},
		{
			name: "with op and underlying error",
			err: &Error{
				Op:   "tenant.Create",
				Code: CodeInternal,
				Err:  errors.New("database error"),
			},
			expected: "tenant.Create: database error",
		},
		{
			name: "with only message",
			err: &Error{
				Code:    CodeBadRequest,
				Message: "invalid input",
			},
			expected: "invalid input",
		},
		{
			name: "with only op",
			err: &Error{
				Op:   "tenant.Create",
				Code: CodeInternal,
			},
			expected: "tenant.Create",
		},
		{
			name: "with only code",
			err: &Error{
				Code: CodeNotFound,
			},
			expected: "NOT_FOUND",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.expected {
				t.Errorf("Error() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestError_Unwrap(t *testing.T) {
	underlying := errors.New("underlying error")
	err := &Error{
		Op:   "test",
		Code: CodeInternal,
		Err:  underlying,
	}

	if got := err.Unwrap(); got != underlying {
		t.Errorf("Unwrap() = %v, want %v", got, underlying)
	}
}

func TestError_Is(t *testing.T) {
	tests := []struct {
		name   string
		err    *Error
		target error
		want   bool
	}{
		{
			name: "matches underlying sentinel",
			err: &Error{
				Code: CodeNotFound,
				Err:  ErrNotFound,
			},
			target: ErrNotFound,
			want:   true,
		},
		{
			name: "matches by code",
			err: &Error{
				Code: CodeNotFound,
			},
			target: &Error{Code: CodeNotFound},
			want:   true,
		},
		{
			name: "does not match different sentinel",
			err: &Error{
				Code: CodeNotFound,
				Err:  ErrNotFound,
			},
			target: ErrForbidden,
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := errors.Is(tt.err, tt.target); got != tt.want {
				t.Errorf("Is() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNew(t *testing.T) {
	err := New("tenant.Create", CodeBadRequest, "invalid name")
	if err.Op != "tenant.Create" {
		t.Errorf("Op = %q, want %q", err.Op, "tenant.Create")
	}
	if err.Code != CodeBadRequest {
		t.Errorf("Code = %q, want %q", err.Code, CodeBadRequest)
	}
	if err.Message != "invalid name" {
		t.Errorf("Message = %q, want %q", err.Message, "invalid name")
	}
}

func TestWrap(t *testing.T) {
	t.Run("wraps nil", func(t *testing.T) {
		if got := Wrap("test", nil); got != nil {
			t.Errorf("Wrap(nil) = %v, want nil", got)
		}
	})

	t.Run("wraps regular error", func(t *testing.T) {
		underlying := errors.New("some error")
		err := Wrap("tenant.Get", underlying)
		if err.Op != "tenant.Get" {
			t.Errorf("Op = %q, want %q", err.Op, "tenant.Get")
		}
		if err.Err != underlying {
			t.Errorf("Err = %v, want %v", err.Err, underlying)
		}
	})

	t.Run("wraps Error preserving code", func(t *testing.T) {
		underlying := NotFound("inner", "resource not found")
		err := Wrap("outer", underlying)
		if err.Code != CodeNotFound {
			t.Errorf("Code = %q, want %q", err.Code, CodeNotFound)
		}
	})

	t.Run("wraps sentinel error", func(t *testing.T) {
		err := Wrap("test", ErrNotFound)
		if err.Code != CodeNotFound {
			t.Errorf("Code = %q, want %q", err.Code, CodeNotFound)
		}
	})
}

func TestWrapWithMessage(t *testing.T) {
	underlying := errors.New("some error")
	err := WrapWithMessage("tenant.Get", underlying, "failed to get tenant")
	if err.Message != "failed to get tenant" {
		t.Errorf("Message = %q, want %q", err.Message, "failed to get tenant")
	}
}

func TestHelperFunctions(t *testing.T) {
	tests := []struct {
		name     string
		fn       func(string, string) *Error
		wantCode Code
		wantErr  error
	}{
		{"NotFound", NotFound, CodeNotFound, ErrNotFound},
		{"Unauthorized", Unauthorized, CodeUnauthorized, ErrUnauthorized},
		{"Forbidden", Forbidden, CodeForbidden, ErrForbidden},
		{"RateLimited", RateLimited, CodeRateLimited, ErrRateLimited},
		{"BudgetExceeded", BudgetExceeded, CodeBudgetExceeded, ErrBudgetExceeded},
		{"BadRequest", BadRequest, CodeBadRequest, ErrBadRequest},
		{"Conflict", Conflict, CodeConflict, ErrConflict},
		{"Internal", Internal, CodeInternal, ErrInternal},
		{"ServiceUnavailable", ServiceUnavailable, CodeServiceUnavailable, ErrServiceUnavailable},
		{"Validation", Validation, CodeValidation, ErrValidation},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn("op", "message")
			if err.Code != tt.wantCode {
				t.Errorf("Code = %q, want %q", err.Code, tt.wantCode)
			}
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Is(%v) = false, want true", tt.wantErr)
			}
		})
	}
}

func TestStatusCode(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{"nil", nil, http.StatusOK},
		{"NotFound error", NotFound("test", "not found"), http.StatusNotFound},
		{"Unauthorized error", Unauthorized("test", "unauthorized"), http.StatusUnauthorized},
		{"Forbidden error", Forbidden("test", "forbidden"), http.StatusForbidden},
		{"RateLimited error", RateLimited("test", "rate limited"), http.StatusTooManyRequests},
		{"BudgetExceeded error", BudgetExceeded("test", "budget exceeded"), http.StatusPaymentRequired},
		{"BadRequest error", BadRequest("test", "bad request"), http.StatusBadRequest},
		{"Conflict error", Conflict("test", "conflict"), http.StatusConflict},
		{"Internal error", Internal("test", "internal"), http.StatusInternalServerError},
		{"ServiceUnavailable error", ServiceUnavailable("test", "unavailable"), http.StatusServiceUnavailable},
		{"Validation error", Validation("test", "invalid"), http.StatusUnprocessableEntity},
		{"sentinel ErrNotFound", ErrNotFound, http.StatusNotFound},
		{"sentinel ErrUnauthorized", ErrUnauthorized, http.StatusUnauthorized},
		{"sentinel ErrRateLimited", ErrRateLimited, http.StatusTooManyRequests},
		{"generic error", errors.New("generic"), http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := StatusCode(tt.err); got != tt.want {
				t.Errorf("StatusCode() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestGetCode(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want Code
	}{
		{"nil", nil, ""},
		{"Error type", NotFound("test", "not found"), CodeNotFound},
		{"sentinel", ErrForbidden, CodeForbidden},
		{"generic", errors.New("generic"), CodeInternal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetCode(tt.err); got != tt.want {
				t.Errorf("GetCode() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetMessage(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{"nil", nil, ""},
		{"Error with message", NotFound("test", "resource not found"), "resource not found"},
		{"Error without message", &Error{Code: CodeNotFound}, "NOT_FOUND"},
		{"generic error", errors.New("some error"), "some error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetMessage(tt.err); got != tt.want {
				t.Errorf("GetMessage() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestErrorChaining(t *testing.T) {
	// Test that error chains work correctly with errors.Is
	base := ErrNotFound
	wrapped := Wrap("layer1", base)
	doubleWrapped := Wrap("layer2", wrapped)

	if !errors.Is(doubleWrapped, ErrNotFound) {
		t.Error("errors.Is should find ErrNotFound in chain")
	}

	// Test with errors.As
	var appErr *Error
	if !errors.As(doubleWrapped, &appErr) {
		t.Error("errors.As should find *Error in chain")
	}
	if appErr.Op != "layer2" {
		t.Errorf("Op = %q, want %q", appErr.Op, "layer2")
	}
}
