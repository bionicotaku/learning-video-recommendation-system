package response_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"learning-video-recommendation-system/internal/api/infrastructure/http/response"
)

func TestWriteErrorIncludesStableEnvelopeAndRequestID(t *testing.T) {
	recorder := httptest.NewRecorder()

	response.WriteError(recorder, "req_test", response.InvalidRequest("events must not be empty"))

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", recorder.Code)
	}

	var body struct {
		Error struct {
			Code      string `json:"code"`
			Message   string `json:"message"`
			RequestID string `json:"request_id"`
		} `json:"error"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Error.Code != "invalid_request" {
		t.Fatalf("unexpected error code: %s", body.Error.Code)
	}
	if body.Error.RequestID != "req_test" {
		t.Fatalf("unexpected request id: %s", body.Error.RequestID)
	}
}

func TestPayloadTooLargeUsesStableCode(t *testing.T) {
	recorder := httptest.NewRecorder()

	response.WriteError(recorder, "req_big", response.PayloadTooLarge("request body must not exceed 5 MiB"))

	if recorder.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected status 413, got %d", recorder.Code)
	}

	var body struct {
		Error struct {
			Code      string `json:"code"`
			Message   string `json:"message"`
			RequestID string `json:"request_id"`
		} `json:"error"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Error.Code != "payload_too_large" {
		t.Fatalf("unexpected error code: %s", body.Error.Code)
	}
	if body.Error.RequestID != "req_big" {
		t.Fatalf("unexpected request id: %s", body.Error.RequestID)
	}
}
