package service

import (
	"errors"
	"fmt"
)

type ErrorCode string

const (
	ErrorCodeValidation ErrorCode = "validation"
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

var (
	ErrUserUnitStateNotFound     = errors.New("user unit state not found")
	ErrUserUnitStateNotSuspended = errors.New("user unit state is not suspended")
	ErrLateProgressEvent         = errors.New("late progress event")
	ErrUnitCollectionNotFound    = errors.New("unit collection not found")
)

func validationError(format string, args ...any) error {
	return &Error{Code: ErrorCodeValidation, Message: fmt.Sprintf(format, args...)}
}

func IsValidationError(err error) bool {
	var appErr *Error
	return errors.As(err, &appErr) && appErr.Code == ErrorCodeValidation
}
