package repository

import (
	"context"

	"learning-video-recommendation-system/internal/semantic/domain/model"
)

type UnitCollectionReader interface {
	ListActiveUnitCollections(ctx context.Context) ([]model.UnitCollection, error)
}
