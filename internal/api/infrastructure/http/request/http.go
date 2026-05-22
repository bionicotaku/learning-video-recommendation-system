package request

import (
	"fmt"
	"mime"
	"net/http"
	"strconv"
	"strings"
)

func RequireJSONContentType(r *http.Request) error {
	contentType := r.Header.Get("Content-Type")
	if strings.TrimSpace(contentType) == "" {
		return fmt.Errorf("content-type must be application/json")
	}
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err == nil && mediaType == "application/json" {
		return nil
	}
	return fmt.Errorf("content-type must be application/json")
}

func ParseOptionalLimit(r *http.Request, min int, max int) (int, error) {
	rawLimit := strings.TrimSpace(r.URL.Query().Get("limit"))
	if rawLimit == "" {
		return 0, nil
	}
	limit, err := strconv.Atoi(rawLimit)
	if err != nil {
		return 0, fmt.Errorf("limit must be an integer")
	}
	if limit < min || limit > max {
		return 0, fmt.Errorf("limit must be between %d and %d", min, max)
	}
	return limit, nil
}

func ParseCursor(r *http.Request) string {
	return strings.TrimSpace(r.URL.Query().Get("cursor"))
}

func PathRequiredUUID(r *http.Request, name string) (string, error) {
	value := r.PathValue(name)
	if err := ValidateRequiredUUID(name, value); err != nil {
		return "", err
	}
	return value, nil
}
