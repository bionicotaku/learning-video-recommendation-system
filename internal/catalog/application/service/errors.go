package service

import (
	"errors"
	"fmt"
)

type ErrorCode string

const (
	ErrorCodeValidation    ErrorCode = "validation"
	ErrorCodeNotFound      ErrorCode = "not_found"
	ErrorCodeConflict      ErrorCode = "conflict"
	ErrorCodeUnprocessable ErrorCode = "unprocessable"
)

type Error struct {
	Code    ErrorCode
	Message string
	Err     error
}

func (e *Error) Error() string {
	if e.Message != "" {
		return e.Message
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return string(e.Code)
}

func (e *Error) Unwrap() error {
	return e.Err
}

func validationError(format string, args ...any) error {
	return &Error{Code: ErrorCodeValidation, Message: fmt.Sprintf(format, args...)}
}

func NotFoundError(message string) error {
	return &Error{Code: ErrorCodeNotFound, Message: message}
}

func ConflictError(message string) error {
	return &Error{Code: ErrorCodeConflict, Message: message}
}

func UnprocessableError(message string) error {
	return &Error{Code: ErrorCodeUnprocessable, Message: message}
}

func IsValidationError(err error) bool {
	return hasCode(err, ErrorCodeValidation)
}

func IsNotFoundError(err error) bool {
	return hasCode(err, ErrorCodeNotFound)
}

func IsConflictError(err error) bool {
	return hasCode(err, ErrorCodeConflict)
}

func IsUnprocessableError(err error) bool {
	return hasCode(err, ErrorCodeUnprocessable)
}

func hasCode(err error, code ErrorCode) bool {
	var appErr *Error
	return errors.As(err, &appErr) && appErr.Code == code
}
