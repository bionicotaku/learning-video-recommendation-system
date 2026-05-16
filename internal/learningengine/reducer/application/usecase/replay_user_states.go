package usecase

import (
	"context"

	"learning-video-recommendation-system/internal/learningengine/reducer/application/dto"
)

type ReplayUserStatesUsecase interface {
	Execute(ctx context.Context, request dto.ReplayUserStatesRequest) (dto.ReplayUserStatesResponse, error)
}
