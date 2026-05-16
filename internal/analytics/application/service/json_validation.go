package service

import (
	"encoding/json"
)

func normalizeJSONObject(raw []byte, field string) ([]byte, error) {
	if len(raw) == 0 {
		return []byte("{}"), nil
	}
	var value map[string]any
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, validationError("%s must be a json object", field)
	}
	if value == nil {
		return nil, validationError("%s must be a json object", field)
	}
	return raw, nil
}

func validateNonNegativePointer(value *int32, field string) error {
	if value != nil && *value < 0 {
		return validationError("%s must be non-negative", field)
	}
	return nil
}
