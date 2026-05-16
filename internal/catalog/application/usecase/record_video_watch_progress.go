package usecase

import (
	"context"

	"learning-video-recommendation-system/internal/catalog/application/dto"
)

type RecordVideoWatchProgressUsecase interface {
	Execute(ctx context.Context, request dto.RecordVideoWatchProgressRequest) (dto.RecordVideoWatchProgressResponse, error)
}
