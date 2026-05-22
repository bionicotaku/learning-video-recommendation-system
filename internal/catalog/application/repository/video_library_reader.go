package repository

import (
	"context"

	"learning-video-recommendation-system/internal/catalog/application/dto"
	"learning-video-recommendation-system/internal/catalog/domain/model"
)

type VideoLibraryReader interface {
	ListVideoFavorites(ctx context.Context, query dto.ListVideoFavoritesQuery) ([]model.VideoFavoriteListItem, error)
	ListVideoHistory(ctx context.Context, query dto.ListVideoHistoryQuery) ([]model.VideoHistoryListItem, error)
}
