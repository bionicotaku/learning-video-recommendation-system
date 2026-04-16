package usecase

import (
	"context"

	"learning-video-recommendation-system/internal/learningengine/application/dto"
)

type RecordLearningEventsUsecase interface {
	Execute(ctx context.Context, request dto.RecordLearningEventsRequest) (dto.RecordLearningEventsResponse, error)
}
