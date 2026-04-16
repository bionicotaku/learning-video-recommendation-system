package repository

import (
	"context"

	"learning-video-recommendation-system/internal/recommendation/domain/model"
)

type VideoUserStateReader interface {
	ListByUserAndVideoIDs(ctx context.Context, userID string, videoIDs []string) ([]model.VideoUserState, error)
}
