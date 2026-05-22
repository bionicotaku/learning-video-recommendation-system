package usecase

import (
	"context"

	"learning-video-recommendation-system/internal/catalog/application/dto"
)

type ListVideoFavoritesUsecase interface {
	Execute(ctx context.Context, request dto.ListVideoFavoritesRequest) (dto.ListVideoFavoritesResponse, error)
}

type ListVideoHistoryUsecase interface {
	Execute(ctx context.Context, request dto.ListVideoHistoryRequest) (dto.ListVideoHistoryResponse, error)
}
