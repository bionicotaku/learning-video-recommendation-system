package service

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"

	"learning-video-recommendation-system/internal/catalog/application/dto"
	"learning-video-recommendation-system/internal/catalog/domain/model"
)

type wordFavoritesCursorPayload struct {
	Kind        string `json:"kind"`
	FavoritedAt string `json:"favorited_at"`
	FavoriteID  string `json:"favorite_id"`
}

func encodeWordFavoritesCursor(item model.WordFavoriteListItem) (string, error) {
	payload := wordFavoritesCursorPayload{
		Kind:        dto.WordFavoritesCursorKind,
		FavoritedAt: item.FavoritedAt.UTC().Format(time.RFC3339Nano),
		FavoriteID:  item.FavoriteID,
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(encoded), nil
}

func decodeWordFavoritesCursor(value string) (*dto.WordFavoritesCursor, error) {
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

	var payload wordFavoritesCursorPayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return nil, validationError("cursor is invalid")
	}
	if payload.Kind != dto.WordFavoritesCursorKind {
		return nil, validationError("cursor kind does not match endpoint")
	}
	favoritedAt, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(payload.FavoritedAt))
	if err != nil {
		return nil, validationError("cursor favorited_at is invalid")
	}
	if !isUUID(payload.FavoriteID) {
		return nil, validationError("cursor favorite_id must be a uuid")
	}
	return &dto.WordFavoritesCursor{
		Kind:        payload.Kind,
		FavoritedAt: favoritedAt,
		FavoriteID:  payload.FavoriteID,
	}, nil
}
