package usecase

import (
	"context"

	"learning-video-recommendation-system/internal/learningengine/reducer/application/dto"
)

type ListUserUnitProgressUsecase interface {
	Execute(ctx context.Context, request dto.ListUserUnitProgressRequest) (dto.ListUserUnitProgressResponse, error)
}
