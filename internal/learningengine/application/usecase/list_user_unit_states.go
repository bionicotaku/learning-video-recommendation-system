package usecase

import (
	"context"

	"learning-video-recommendation-system/internal/learningengine/application/dto"
)

type ListUserUnitStatesUsecase interface {
	Execute(ctx context.Context, request dto.ListUserUnitStatesRequest) (dto.ListUserUnitStatesResponse, error)
}
