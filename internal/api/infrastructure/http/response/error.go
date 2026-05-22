package response

import (
	"encoding/json"
	"net/http"
)

type FieldError struct {
	Field  string `json:"field"`
	Reason string `json:"reason"`
}

type Error struct {
	StatusCode int
	Code       string
	Message    string
	Details    []FieldError
}

func (e *Error) Error() string {
	return e.Message
}

func InvalidRequest(message string) *Error {
	return &Error{StatusCode: http.StatusBadRequest, Code: "invalid_request", Message: message}
}

func Unauthorized(message string) *Error {
	return &Error{StatusCode: http.StatusUnauthorized, Code: "unauthorized", Message: message}
}

func NotFound(message string) *Error {
	return &Error{StatusCode: http.StatusNotFound, Code: "not_found", Message: message}
}

func Conflict(message string) *Error {
	return &Error{StatusCode: http.StatusConflict, Code: "conflict", Message: message}
}

func UnprocessableEntity(message string) *Error {
	return &Error{StatusCode: http.StatusUnprocessableEntity, Code: "unprocessable_entity", Message: message}
}

func PayloadTooLarge(message string) *Error {
	return &Error{StatusCode: http.StatusRequestEntityTooLarge, Code: "payload_too_large", Message: message}
}

func ServiceUnavailable(message string) *Error {
	return &Error{StatusCode: http.StatusServiceUnavailable, Code: "service_unavailable", Message: message}
}

func InternalError() *Error {
	return &Error{StatusCode: http.StatusInternalServerError, Code: "internal_error", Message: "internal error"}
}

func WriteError(w http.ResponseWriter, requestID string, err *Error) {
	if err == nil {
		err = InternalError()
	}
	details := err.Details
	if details == nil {
		details = []FieldError{}
	}
	WriteJSON(w, err.StatusCode, map[string]any{
		"error": map[string]any{
			"code":       err.Code,
			"message":    err.Message,
			"details":    details,
			"request_id": requestID,
		},
	})
}

func WriteJSON(w http.ResponseWriter, statusCode int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(value)
}
