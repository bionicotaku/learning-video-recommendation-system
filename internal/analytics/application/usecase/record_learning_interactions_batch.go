package usecase

import (
	"context"

	"learning-video-recommendation-system/internal/analytics/application/dto"
)

type RecordLearningInteractionsBatchUsecase interface {
	Execute(ctx context.Context, request dto.RecordLearningInteractionsBatchRequest) (dto.RecordLearningInteractionsBatchResponse, error)
}
