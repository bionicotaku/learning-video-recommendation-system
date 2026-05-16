package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"learning-video-recommendation-system/internal/api/infrastructure/http/auth"
)

func Logging(logger *slog.Logger) func(http.Handler) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			startedAt := time.Now()
			recorder := NewStatusRecorder(w)
			next.ServeHTTP(recorder, r)
			principal, _ := auth.PrincipalFromContext(r.Context())
			logger.InfoContext(
				r.Context(),
				"http request",
				"request_id", RequestIDFromContext(r.Context()),
				"method", r.Method,
				"path", r.URL.Path,
				"status_code", recorder.StatusCode,
				"duration_ms", time.Since(startedAt).Milliseconds(),
				"user_id", principal.UserID,
			)
		})
	}
}
