package service

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"learning-video-recommendation-system/internal/learningengine/reducer/application/dto"
)

type unitProgressCursorPayload struct {
	Bucket          string   `json:"bucket"`
	LabelKey        string   `json:"label_key"`
	Label           string   `json:"label"`
	CoarseUnitID    int64    `json:"coarse_unit_id"`
	ProgressPercent *float64 `json:"progress_percent,omitempty"`
}

func encodeUnitProgressCursor(bucket string, item dto.UnitProgressItem) (string, error) {
	labelKey := strings.TrimSpace(item.LabelKey)
	if labelKey == "" {
		labelKey = strings.ToLower(item.Label)
	}
	payload := unitProgressCursorPayload{
		Bucket:       bucket,
		LabelKey:     labelKey,
		Label:        item.Label,
		CoarseUnitID: item.CoarseUnitID,
	}
	if bucket == dto.UnitProgressBucketUnmastered {
		progressPercent := item.ProgressPercent
		payload.ProgressPercent = &progressPercent
	}

	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(encoded), nil
}

func decodeUnitProgressCursor(value string) (*dto.UnitProgressCursor, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		if decoded, paddedErr := base64.URLEncoding.DecodeString(value); paddedErr == nil {
			payloadBytes = decoded
		} else {
			return nil, validationError("cursor is invalid")
		}
	}

	var payload unitProgressCursorPayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return nil, validationError("cursor is invalid")
	}
	if payload.Bucket != dto.UnitProgressBucketMastered && payload.Bucket != dto.UnitProgressBucketUnmastered {
		return nil, validationError("cursor bucket is invalid")
	}
	if strings.TrimSpace(payload.LabelKey) == "" {
		return nil, validationError("cursor label_key is required")
	}
	if strings.TrimSpace(payload.Label) == "" {
		return nil, validationError("cursor label is required")
	}
	if payload.CoarseUnitID <= 0 {
		return nil, validationError("cursor coarse_unit_id is required")
	}
	if payload.Bucket == dto.UnitProgressBucketUnmastered && payload.ProgressPercent == nil {
		return nil, validationError("cursor progress_percent is required")
	}
	if payload.Bucket == dto.UnitProgressBucketMastered && payload.ProgressPercent != nil {
		return nil, validationError("cursor progress_percent is not allowed for mastered")
	}

	cursor := &dto.UnitProgressCursor{
		Bucket:       payload.Bucket,
		LabelKey:     payload.LabelKey,
		Label:        payload.Label,
		CoarseUnitID: payload.CoarseUnitID,
	}
	if payload.ProgressPercent != nil {
		cursor.ProgressPercent = *payload.ProgressPercent
		cursor.HasProgressPercent = true
	}
	return cursor, nil
}

func validateUnitProgressCursorBucket(cursor *dto.UnitProgressCursor, bucket string) error {
	if cursor == nil {
		return nil
	}
	if cursor.Bucket != bucket {
		return validationError("cursor bucket does not match endpoint")
	}
	if bucket == dto.UnitProgressBucketUnmastered && !cursor.HasProgressPercent {
		return validationError("cursor progress_percent is required")
	}
	if bucket == dto.UnitProgressBucketMastered && cursor.HasProgressPercent {
		return validationError("cursor progress_percent is not allowed for mastered")
	}
	return nil
}

func invalidUnitProgressBucket(bucket string) error {
	return validationError("bucket must be %s or %s", dto.UnitProgressBucketMastered, dto.UnitProgressBucketUnmastered)
}

func wrapUnitProgressCursorEncodeError(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("encode unit progress cursor: %w", err)
}
