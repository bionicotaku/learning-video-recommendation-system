package repository

import (
	"context"

	"learning-video-recommendation-system/internal/recommendation/domain/model"
)

type UnitServingStateRepository interface {
	Upsert(ctx context.Context, state model.UserUnitServingState) error
}

type VideoServingStateRepository interface {
	Upsert(ctx context.Context, state model.UserVideoServingState) error
}
