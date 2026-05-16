package usecase

import (
	"context"

	"learning-video-recommendation-system/internal/learningengine/normalizer/application/dto"
)

type NormalizePendingEventsUsecase interface {
	Execute(ctx context.Context, request dto.NormalizePendingEventsRequest) (dto.NormalizePendingEventsResponse, error)
}

type NormalizeLearningInteractionsByIDsUsecase interface {
	Execute(ctx context.Context, request dto.NormalizeLearningInteractionsByIDsRequest) (dto.NormalizeLearningInteractionsByIDsResponse, error)
}

type NormalizeQuizAttemptByIDUsecase interface {
	Execute(ctx context.Context, request dto.NormalizeQuizAttemptByIDRequest) (dto.NormalizeQuizAttemptByIDResponse, error)
}
