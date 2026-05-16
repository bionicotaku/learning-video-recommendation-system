package middleware

import "net/http"

type StatusRecorder struct {
	http.ResponseWriter
	StatusCode int
}

func NewStatusRecorder(w http.ResponseWriter) *StatusRecorder {
	return &StatusRecorder{ResponseWriter: w, StatusCode: http.StatusOK}
}

func (r *StatusRecorder) WriteHeader(statusCode int) {
	r.StatusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}
