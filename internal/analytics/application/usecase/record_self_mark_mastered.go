package usecase

import (
	"context"

	"learning-video-recommendation-system/internal/analytics/application/dto"
)

type RecordSelfMarkMasteredUsecase interface {
	Execute(ctx context.Context, request dto.RecordSelfMarkMasteredRequest) (dto.RecordSelfMarkMasteredResponse, error)
}
