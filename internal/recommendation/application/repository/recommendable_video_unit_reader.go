package repository

import (
	"context"

	"learning-video-recommendation-system/internal/recommendation/domain/model"
)

type RecommendableVideoUnitReader interface {
	ListByUnitIDs(ctx context.Context, coarseUnitIDs []int64) ([]model.RecommendableVideoUnit, error)
}
