package httperror_test

import (
	"errors"
	"net/http"
	"testing"

	"learning-video-recommendation-system/internal/api/infrastructure/http/handler/httperror"
)

func TestInvalidRequestMapsMaxBytesErrorToPayloadTooLarge(t *testing.T) {
	mapped := httperror.InvalidRequest(&http.MaxBytesError{Limit: 1 << 20})

	if mapped.StatusCode != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want 413", mapped.StatusCode)
	}
	if mapped.Code != "payload_too_large" {
		t.Fatalf("code = %q, want payload_too_large", mapped.Code)
	}
}

func TestInvalidRequestMapsRegularErrorsToInvalidRequest(t *testing.T) {
	mapped := httperror.InvalidRequest(errors.New("invalid json body"))

	if mapped.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", mapped.StatusCode)
	}
	if mapped.Code != "invalid_request" {
		t.Fatalf("code = %q, want invalid_request", mapped.Code)
	}
	if mapped.Message != "invalid json body" {
		t.Fatalf("message = %q, want original message", mapped.Message)
	}
}
