package usecase

import (
	"context"

	"learning-video-recommendation-system/internal/learningengine/application/dto"
)

type EnsureTargetUnitsUsecase interface {
	Execute(ctx context.Context, request dto.EnsureTargetUnitsRequest) (dto.EnsureTargetUnitsResponse, error)
}

type SetTargetInactiveUsecase interface {
	Execute(ctx context.Context, request dto.SetTargetInactiveRequest) (dto.SetTargetInactiveResponse, error)
}

type SuspendTargetUnitUsecase interface {
	Execute(ctx context.Context, request dto.SuspendTargetUnitRequest) (dto.SuspendTargetUnitResponse, error)
}

type ResumeTargetUnitUsecase interface {
	Execute(ctx context.Context, request dto.ResumeTargetUnitRequest) (dto.ResumeTargetUnitResponse, error)
}
