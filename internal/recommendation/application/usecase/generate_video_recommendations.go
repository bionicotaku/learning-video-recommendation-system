package usecase

import (
	"context"

	"learning-video-recommendation-system/internal/recommendation/application/dto"
)

type GenerateVideoRecommendationsUsecase interface {
	Execute(ctx context.Context, request dto.GenerateVideoRecommendationsRequest) (dto.GenerateVideoRecommendationsResponse, error)
}
