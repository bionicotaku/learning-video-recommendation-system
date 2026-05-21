package usecase

import (
	"context"

	"learning-video-recommendation-system/internal/user/application/dto"
)

type GetMeUsecase interface {
	Execute(ctx context.Context, request dto.MeRequest) (dto.MeResponse, error)
}

type GetActivityCalendarUsecase interface {
	Execute(ctx context.Context, request dto.ActivityCalendarRequest) (dto.ActivityCalendarResponse, error)
}

type UpdateOnboardingStatusUsecase interface {
	Execute(ctx context.Context, request dto.UpdateOnboardingStatusRequest) (dto.UpdateOnboardingStatusResponse, error)
}
