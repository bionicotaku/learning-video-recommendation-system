package usecase

import (
	"context"

	"learning-video-recommendation-system/internal/analytics/application/dto"
)

type RecordQuizAttemptUsecase interface {
	Execute(ctx context.Context, request dto.RecordQuizAttemptRequest) (dto.RecordQuizAttemptResponse, error)
}
