package usecase

import (
	"context"

	"learning-video-recommendation-system/internal/catalog/application/dto"
)

type FeedVideoLookupUsecase interface {
	Execute(ctx context.Context, request dto.FeedVideoLookupRequest) (dto.FeedVideoLookupResponse, error)
}

type UnitLabelLookupUsecase interface {
	Execute(ctx context.Context, request dto.UnitLabelLookupRequest) (dto.UnitLabelLookupResponse, error)
}

type GetVideoDetailUsecase interface {
	Execute(ctx context.Context, request dto.GetVideoDetailRequest) (dto.VideoDetailResponse, error)
}
