package request_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"learning-video-recommendation-system/internal/api/infrastructure/http/request"
)

type sampleBody struct {
	Name      string          `json:"name"`
	When      string          `json:"when"`
	Metadata  json.RawMessage `json:"metadata"`
	ElapsedMS *int32          `json:"elapsed_ms"`
}

func TestDecodeJSONObjectRejectsNonObjectBody(t *testing.T) {
	var body sampleBody

	err := request.DecodeJSONObject(strings.NewReader(`[]`), &body)

	if err == nil {
		t.Fatalf("expected non-object body to be rejected")
	}
}

func TestDecodeJSONObjectRejectsUnknownFields(t *testing.T) {
	var body sampleBody

	err := request.DecodeJSONObject(strings.NewReader(`{"name":"x","unknown":true}`), &body)

	if err == nil {
		t.Fatalf("expected unknown field to be rejected")
	}
}

func TestParseRequiredTimeRequiresExplicitOffsetAndReturnsUTC(t *testing.T) {
	_, err := request.ParseRequiredTime("shown_at", "2026-05-15T10:00:01")
	if err == nil {
		t.Fatalf("expected timestamp without offset to be rejected")
	}

	parsed, err := request.ParseRequiredTime("shown_at", "2026-05-15T10:00:01-07:00")
	if err != nil {
		t.Fatalf("expected timestamp with offset to parse: %v", err)
	}
	if parsed.Location() != time.UTC {
		t.Fatalf("expected UTC location, got %v", parsed.Location())
	}
	if got := parsed.Format(time.RFC3339Nano); got != "2026-05-15T17:00:01Z" {
		t.Fatalf("unexpected UTC timestamp: %s", got)
	}
}

func TestValidateJSONObjectRejectsNonObjectRawMessage(t *testing.T) {
	err := request.ValidateJSONObject("client_context", json.RawMessage(`[]`))

	if err == nil {
		t.Fatalf("expected array client_context to be rejected")
	}
}

func TestValidateNonNegativeInt32RejectsNegativeValue(t *testing.T) {
	value := int32(-1)

	err := request.ValidateNonNegativeInt32("elapsed_ms", &value)

	if err == nil {
		t.Fatalf("expected negative value to be rejected")
	}
}
