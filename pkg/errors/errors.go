package apperrors

import (
	"errors"
	"fmt"
	"net/http"
)

type Code string

const (
	CodeValidation     Code = "VALIDATION_ERROR"
	CodeUnauthorized   Code = "UNAUTHORIZED"
	CodeForbidden      Code = "FORBIDDEN"
	CodeNotFound       Code = "NOT_FOUND"
	CodeConflict       Code = "CONFLICT"
	CodeRateLimited    Code = "RATE_LIMITED"
	CodeInternal       Code = "INTERNAL_ERROR"
	CodeInvalidCreds   Code = "INVALID_CREDENTIALS"
	CodeTokenExpired   Code = "TOKEN_EXPIRED"
	CodeTokenInvalid   Code = "TOKEN_INVALID"
	CodeEmailTaken     Code = "EMAIL_TAKEN"
	CodeUsernameTaken  Code = "USERNAME_TAKEN"
)

type AppError struct {
	Code       Code           `json:"code"`
	Message    string         `json:"message"`
	HTTPStatus int            `json:"-"`
	Details    map[string]any `json:"details,omitempty"`
	Err        error          `json:"-"`
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.Err
}

func New(code Code, message string, httpStatus int) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: httpStatus,
		Details:    map[string]any{},
	}
}

func Wrap(code Code, message string, httpStatus int, err error) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: httpStatus,
		Err:        err,
		Details:    map[string]any{},
	}
}

func Validation(message string, details map[string]any) *AppError {
	e := New(CodeValidation, message, http.StatusBadRequest)
	e.Details = details
	return e
}

func Unauthorized(message string) *AppError {
	return New(CodeUnauthorized, message, http.StatusUnauthorized)
}

func Forbidden(message string) *AppError {
	return New(CodeForbidden, message, http.StatusForbidden)
}

func NotFound(message string) *AppError {
	return New(CodeNotFound, message, http.StatusNotFound)
}

func Conflict(code Code, message string) *AppError {
	return New(code, message, http.StatusConflict)
}

func Internal(err error) *AppError {
	return Wrap(CodeInternal, "internal server error", http.StatusInternalServerError, err)
}

func RateLimited() *AppError {
	return New(CodeRateLimited, "too many requests", http.StatusTooManyRequests)
}

func IsAppError(err error) (*AppError, bool) {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr, true
	}
	return nil, false
}
