package service

import "encoding/json"

func normalizeJSONObject(value []byte) ([]byte, error) {
	if len(value) == 0 {
		return []byte(`{}`), nil
	}

	var decoded any
	if err := json.Unmarshal(value, &decoded); err != nil {
		return nil, err
	}
	if _, ok := decoded.(map[string]any); !ok {
		return nil, validationError("json value must be an object")
	}
	return value, nil
}
