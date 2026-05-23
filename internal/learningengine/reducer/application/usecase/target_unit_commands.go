package usecase

import (
	"context"

	"learning-video-recommendation-system/internal/learningengine/reducer/application/dto"
)

type EnsureTargetUnitsUsecase interface {
	Execute(ctx context.Context, request dto.EnsureTargetUnitsRequest) (dto.EnsureTargetUnitsResponse, error)
}

type ActivateUnitCollectionTargetUsecase interface {
	Execute(ctx context.Context, request dto.ActivateUnitCollectionTargetRequest) (dto.ActivateUnitCollectionTargetResponse, error)
}

type GetActiveUnitCollectionUsecase interface {
	Execute(ctx context.Context, request dto.GetActiveUnitCollectionRequest) (dto.GetActiveUnitCollectionResponse, error)
}

type GetActiveLearningTargetCoarseUnitIDsUsecase interface {
	Execute(ctx context.Context, request dto.GetActiveLearningTargetCoarseUnitIDsRequest) (dto.GetActiveLearningTargetCoarseUnitIDsResponse, error)
}

type SetTargetInactiveUsecase interface {
	Execute(ctx context.Context, request dto.SetTargetInactiveRequest) (dto.SetTargetInactiveResponse, error)
}

type ResetUserUnitProgressUsecase interface {
	Execute(ctx context.Context, request dto.ResetUserUnitProgressRequest) (dto.ResetUserUnitProgressResponse, error)
}
