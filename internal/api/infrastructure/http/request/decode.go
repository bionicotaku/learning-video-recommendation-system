package request

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"
)

var explicitOffsetPattern = regexp.MustCompile(`(?:Z|[+-]\d{2}:\d{2})$`)

func DecodeJSONObject(reader io.Reader, target any) error {
	body, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("read json body: %w", err)
	}
	if len(bytes.TrimSpace(body)) == 0 {
		return fmt.Errorf("request body is required")
	}
	if bytes.TrimSpace(body)[0] != '{' {
		return fmt.Errorf("request body must be a json object")
	}

	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("invalid json body: %w", err)
	}
	if decoder.More() {
		return fmt.Errorf("request body must contain a single json object")
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return fmt.Errorf("request body must contain a single json object")
	}
	return nil
}

func ParseRequiredTime(field string, value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, fmt.Errorf("%s is required", field)
	}
	if !explicitOffsetPattern.MatchString(value) {
		return time.Time{}, fmt.Errorf("%s must include explicit timezone offset", field)
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("%s must be RFC3339 datetime", field)
	}
	return parsed.UTC(), nil
}

func ValidateJSONObject(field string, raw json.RawMessage) error {
	if len(raw) == 0 {
		return nil
	}
	var value map[string]any
	if err := json.Unmarshal(raw, &value); err != nil {
		return fmt.Errorf("%s must be a json object", field)
	}
	if value == nil {
		return fmt.Errorf("%s must be a json object", field)
	}
	return nil
}

func ValidateNonNegativeInt32(field string, value *int32) error {
	if value != nil && *value < 0 {
		return fmt.Errorf("%s must be non-negative", field)
	}
	return nil
}

func ValidateOptionalUUID(field string, value string) error {
	if value == "" {
		return nil
	}
	return ValidateRequiredUUID(field, value)
}

func ValidateRequiredUUID(field string, value string) error {
	if value == "" {
		return fmt.Errorf("%s is required", field)
	}
	if len(value) != 36 {
		return fmt.Errorf("%s must be a uuid", field)
	}
	for index, char := range value {
		switch index {
		case 8, 13, 18, 23:
			if char != '-' {
				return fmt.Errorf("%s must be a uuid", field)
			}
		default:
			if !strings.ContainsRune("0123456789abcdefABCDEF", char) {
				return fmt.Errorf("%s must be a uuid", field)
			}
		}
	}
	return nil
}
