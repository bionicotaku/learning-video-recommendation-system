package repository

import (
	"context"

	"learning-video-recommendation-system/internal/catalog/domain/model"
)

type VideoInteractionWriter interface {
	SetVideoLike(ctx context.Context, command model.VideoLikeCommand) (model.VideoLikeResult, error)
	SetVideoFavorite(ctx context.Context, command model.VideoFavoriteCommand) (model.VideoFavoriteResult, error)
}
