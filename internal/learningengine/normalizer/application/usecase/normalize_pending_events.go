package usecase

import (
	"context"

	"learning-video-recommendation-system/internal/learningengine/normalizer/application/dto"
)

type NormalizePendingEventsUsecase interface {
	Execute(ctx context.Context, request dto.NormalizePendingEventsRequest) (dto.NormalizePendingEventsResponse, error)
}
