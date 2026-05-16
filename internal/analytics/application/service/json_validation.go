package service

import (
	"encoding/json"
	"fmt"
)

func normalizeJSONObject(raw []byte, field string) ([]byte, error) {
	if len(raw) == 0 {
		return []byte("{}"), nil
	}
	var value map[string]any
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, fmt.Errorf("%s must be a json object", field)
	}
	if value == nil {
		return nil, fmt.Errorf("%s must be a json object", field)
	}
	return raw, nil
}

func validateNonNegativePointer(value *int32, field string) error {
	if value != nil && *value < 0 {
		return fmt.Errorf("%s must be non-negative", field)
	}
	return nil
}
