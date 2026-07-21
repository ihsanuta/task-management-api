package apperror

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/go-playground/validator/v10"
)

type AppError struct {
	HTTPStatus int    `json:"-"`
	Code       string `json:"code"`
	Message    string `json:"message"`
	Err        error  `json:"-"`
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

func (e *AppError) Unwrap() error { return e.Err }

func New(httpStatus int, code, message string) *AppError {
	return &AppError{HTTPStatus: httpStatus, Code: code, Message: message}
}

func Wrap(httpStatus int, code, message string, err error) *AppError {
	return &AppError{HTTPStatus: httpStatus, Code: code, Message: message, Err: err}
}

// Common, reusable constructors keep error codes consistent across the codebase.

func NewValidation(message string) *AppError {
	return New(http.StatusBadRequest, "VALIDATION_ERROR", message)
}

func NewUnauthorized(message string) *AppError {
	return New(http.StatusUnauthorized, "UNAUTHORIZED", message)
}

func NewForbidden(message string) *AppError {
	return New(http.StatusForbidden, "FORBIDDEN", message)
}

func NewNotFound(message string) *AppError {
	return New(http.StatusNotFound, "NOT_FOUND", message)
}

func NewConflict(message string) *AppError {
	return New(http.StatusConflict, "CONFLICT", message)
}

func NewInternal(err error) *AppError {
	return Wrap(http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error", nil)
}

var (
	ErrEmailAlreadyRegistered = New(http.StatusConflict, "EMAIL_ALREADY_REGISTERED", "email is already registered")
	ErrInvalidCredentials     = New(http.StatusUnauthorized, "INVALID_CREDENTIALS", "email or password is incorrect")
	ErrInvalidToken           = New(http.StatusUnauthorized, "INVALID_TOKEN", "token is missing, malformed, or expired")
	ErrTaskNotFound           = New(http.StatusNotFound, "TASK_NOT_FOUND", "task not found")
	ErrTaskForbidden          = New(http.StatusForbidden, "TASK_FORBIDDEN", "you do not have access to this task")
	ErrAssigneeNotInTeam      = New(http.StatusUnprocessableEntity, "ASSIGNEE_NOT_IN_TEAM", "assignee must belong to the same team")
	ErrAssigneeNotFound       = New(http.StatusNotFound, "ASSIGNEE_NOT_FOUND", "assignee user not found")
	ErrIdempotencyInProgress  = New(http.StatusConflict, "IDEMPOTENCY_IN_PROGRESS", "a request with this idempotency key is already being processed")
	ErrIdempotencyKeyReused   = New(http.StatusUnprocessableEntity, "IDEMPOTENCY_KEY_REUSED", "idempotency key was already used with a different request payload")
	ErrInvalidIdempotencyKey  = New(http.StatusBadRequest, "INVALID_IDEMPOTENCY_KEY", "Idempotency-Key header must be a valid UUID")
)

func NewValidationError(err error) *AppError {
	var msg string

	var validationErrors validator.ValidationErrors

	if errors.As(err, &validationErrors) {
		for _, e := range validationErrors {
			switch e.Tag() {
			case "required":
				msg = fmt.Sprintf("%s is required", e.Field())

			case "email":
				msg = "invalid email"

			case "min":
				msg = fmt.Sprintf("%s must be at least %s characters", e.Field(), e.Param())

			case "oneof":
				msg = fmt.Sprintf("%s must be one of %s", e.Field(), e.Param())
			}
		}
	}

	return &AppError{HTTPStatus: http.StatusBadRequest, Code: "VALIDATION_ERROR", Message: msg}
}
