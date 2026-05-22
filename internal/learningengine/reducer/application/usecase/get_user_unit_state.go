package usecase

import (
	"context"

	"learning-video-recommendation-system/internal/learningengine/reducer/application/dto"
)

type GetUserUnitStateUsecase interface {
	Execute(ctx context.Context, request dto.GetUserUnitStateRequest) (dto.GetUserUnitStateResponse, error)
}
