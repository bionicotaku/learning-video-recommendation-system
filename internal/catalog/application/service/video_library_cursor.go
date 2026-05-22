package service

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"

	"learning-video-recommendation-system/internal/catalog/application/dto"
)

type videoLibraryCursorPayload struct {
	Kind    string `json:"kind"`
	At      string `json:"at"`
	VideoID string `json:"video_id"`
}

func encodeVideoLibraryCursor(kind string, at time.Time, videoID string) (string, error) {
	payload := videoLibraryCursorPayload{
		Kind:    kind,
		At:      at.UTC().Format(time.RFC3339Nano),
		VideoID: videoID,
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(encoded), nil
}

func decodeVideoLibraryCursor(value string, expectedKind string) (*dto.VideoLibraryCursor, error) {
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

	var payload videoLibraryCursorPayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return nil, validationError("cursor is invalid")
	}
	if payload.Kind != dto.VideoLibraryCursorKindFavorites && payload.Kind != dto.VideoLibraryCursorKindHistory {
		return nil, validationError("cursor kind is invalid")
	}
	if payload.Kind != expectedKind {
		return nil, validationError("cursor kind does not match endpoint")
	}
	at, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(payload.At))
	if err != nil {
		return nil, validationError("cursor at is invalid")
	}
	if !isUUID(payload.VideoID) {
		return nil, validationError("cursor video_id must be a uuid")
	}
	return &dto.VideoLibraryCursor{
		Kind:    payload.Kind,
		SortAt:  at,
		VideoID: payload.VideoID,
	}, nil
}

func isUUID(value string) bool {
	if len(value) != 36 {
		return false
	}
	for index, char := range value {
		switch index {
		case 8, 13, 18, 23:
			if char != '-' {
				return false
			}
		default:
			if !strings.ContainsRune("0123456789abcdefABCDEF", char) {
				return false
			}
		}
	}
	return true
}
