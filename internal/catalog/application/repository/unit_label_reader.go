package repository

import (
	"context"

	"learning-video-recommendation-system/internal/catalog/domain/model"
)

type UnitLabelReader interface {
	ListUnitLabelsByIDs(ctx context.Context, coarseUnitIDs []int64) ([]model.UnitLabel, error)
}
