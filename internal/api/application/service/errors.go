package service

import "errors"

type ErrorCode string

const (
	ErrorCodeInvalidRequest     ErrorCode = "invalid_request"
	ErrorCodeServiceUnavailable ErrorCode = "service_unavailable"
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

func InvalidRequestError(message string) error {
	return &Error{Code: ErrorCodeInvalidRequest, Message: message}
}

func IsInvalidRequest(err error) bool {
	var appErr *Error
	return errors.As(err, &appErr) && appErr.Code == ErrorCodeInvalidRequest
}

func ServiceUnavailableError(message string) error {
	return &Error{Code: ErrorCodeServiceUnavailable, Message: message}
}

func IsServiceUnavailable(err error) bool {
	var appErr *Error
	return errors.As(err, &appErr) && appErr.Code == ErrorCodeServiceUnavailable
}
