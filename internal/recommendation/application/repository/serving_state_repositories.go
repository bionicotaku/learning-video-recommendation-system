package repository

import (
	"context"

	"learning-video-recommendation-system/internal/recommendation/domain/model"
)

type UnitServingStateRepository interface {
	ListByUserAndUnitIDs(ctx context.Context, userID string, coarseUnitIDs []int64) ([]model.UserUnitServingState, error)
	Upsert(ctx context.Context, state model.UserUnitServingState) error
}

type VideoServingStateRepository interface {
	ListByUserAndVideoIDs(ctx context.Context, userID string, videoIDs []string) ([]model.UserVideoServingState, error)
	Upsert(ctx context.Context, state model.UserVideoServingState) error
}
