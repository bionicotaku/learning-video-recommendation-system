package middleware_test

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"learning-video-recommendation-system/internal/api/infrastructure/http/middleware"
)

func TestBodyLimitByPathUsesFeedbackOverride(t *testing.T) {
	handler := middleware.BodyLimitByPath(10, map[string]int64{
		"/api/feedback": 20,
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := io.ReadAll(r.Body); err != nil {
			var maxBytesError *http.MaxBytesError
			if errors.As(err, &maxBytesError) {
				w.WriteHeader(http.StatusRequestEntityTooLarge)
				return
			}
			t.Fatalf("read body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))

	feedback := httptest.NewRecorder()
	handler.ServeHTTP(feedback, httptest.NewRequest(http.MethodPost, "/api/feedback", strings.NewReader(strings.Repeat("x", 15))))
	if feedback.Code != http.StatusOK {
		t.Fatalf("feedback status = %d, want 200", feedback.Code)
	}

	regular := httptest.NewRecorder()
	handler.ServeHTTP(regular, httptest.NewRequest(http.MethodPost, "/api/feed", strings.NewReader(strings.Repeat("x", 15))))
	if regular.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("regular status = %d, want 413", regular.Code)
	}
}
