package usecase

import (
	"context"

	"learning-video-recommendation-system/internal/semantic/application/dto"
)

type ListUnitCollectionsUsecase interface {
	Execute(ctx context.Context) (dto.ListUnitCollectionsResponse, error)
}
