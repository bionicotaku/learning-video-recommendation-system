package rule

import (
	"encoding/json"
)

func marshalMetadata(values map[string]any) ([]byte, error) {
	metadata, err := json.Marshal(values)
	if err != nil {
		return nil, err
	}
	if !isJSONObject(metadata) {
		return nil, errInvalidMetadata
	}
	return metadata, nil
}

func isJSONObject(raw []byte) bool {
	var value map[string]any
	if err := json.Unmarshal(raw, &value); err != nil {
		return false
	}
	return value != nil
}
