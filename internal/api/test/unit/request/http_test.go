package request_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"learning-video-recommendation-system/internal/api/infrastructure/http/request"
)

func TestRequireJSONContentTypeAcceptsJSONWithParameters(t *testing.T) {
	httpRequest := httptest.NewRequest(http.MethodPost, "/api/feed", strings.NewReader(`{}`))
	httpRequest.Header.Set("Content-Type", "application/json; charset=utf-8")

	if err := request.RequireJSONContentType(httpRequest); err != nil {
		t.Fatalf("RequireJSONContentType() error = %v", err)
	}
}

func TestRequireJSONContentTypeRejectsMissingOrNonJSON(t *testing.T) {
	for _, contentType := range []string{"", "text/plain"} {
		httpRequest := httptest.NewRequest(http.MethodPost, "/api/feed", strings.NewReader(`{}`))
		if contentType != "" {
			httpRequest.Header.Set("Content-Type", contentType)
		}

		err := request.RequireJSONContentType(httpRequest)
		if err == nil || err.Error() != "content-type must be application/json" {
			t.Fatalf("content-type %q error = %v, want json content-type error", contentType, err)
		}
	}
}

func TestParseOptionalLimitReturnsZeroWhenMissingAndRejectsOutOfRange(t *testing.T) {
	missing := httptest.NewRequest(http.MethodGet, "/api/video-history", nil)
	limit, err := request.ParseOptionalLimit(missing, 1, 100)
	if err != nil || limit != 0 {
		t.Fatalf("missing limit = %d error=%v, want 0 nil", limit, err)
	}

	for _, path := range []string{"/api/video-history?limit=abc", "/api/video-history?limit=0", "/api/video-history?limit=101"} {
		httpRequest := httptest.NewRequest(http.MethodGet, path, nil)
		if _, err := request.ParseOptionalLimit(httpRequest, 1, 100); err == nil {
			t.Fatalf("ParseOptionalLimit(%s) error = nil, want error", path)
		}
	}
}

func TestParseOptionalLimitTrimsValue(t *testing.T) {
	httpRequest := httptest.NewRequest(http.MethodGet, "/api/video-history?limit=%2020%20", nil)

	limit, err := request.ParseOptionalLimit(httpRequest, 1, 100)
	if err != nil || limit != 20 {
		t.Fatalf("limit = %d error=%v, want 20 nil", limit, err)
	}
}

func TestParseCursorTrimsValue(t *testing.T) {
	httpRequest := httptest.NewRequest(http.MethodGet, "/api/video-history?cursor=%20abc%20", nil)

	if cursor := request.ParseCursor(httpRequest); cursor != "abc" {
		t.Fatalf("cursor = %q, want abc", cursor)
	}
}

func TestPathRequiredUUIDValidatesServeMuxPathValue(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/videos/{video_id}", func(w http.ResponseWriter, r *http.Request) {
		videoID, err := request.PathRequiredUUID(r, "video_id")
		if err != nil {
			t.Fatalf("PathRequiredUUID() error = %v", err)
		}
		if videoID != "11111111-1111-1111-1111-111111111111" {
			t.Fatalf("videoID = %q", videoID)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/videos/11111111-1111-1111-1111-111111111111", nil))
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", recorder.Code)
	}
}
