package middleware

import (
	"net/http"

	"learning-video-recommendation-system/internal/api/infrastructure/http/response"
)

func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				response.WriteError(w, RequestIDFromContext(r.Context()), response.InternalError())
			}
		}()
		next.ServeHTTP(w, r)
	})
}
