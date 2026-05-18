package usecase

import (
	"context"

	"learning-video-recommendation-system/internal/catalog/application/dto"
)

type SetVideoLikeUsecase interface {
	Execute(ctx context.Context, request dto.SetVideoLikeRequest) (dto.VideoLikeResponse, error)
}

type SetVideoFavoriteUsecase interface {
	Execute(ctx context.Context, request dto.SetVideoFavoriteRequest) (dto.VideoFavoriteResponse, error)
}
