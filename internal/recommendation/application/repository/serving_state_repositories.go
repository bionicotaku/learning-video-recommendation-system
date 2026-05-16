package repository

import (
	"context"
	"time"

	"learning-video-recommendation-system/internal/recommendation/domain/model"
)

type UnitServingStateRepository interface {
	ListByUserAndUnitIDs(ctx context.Context, userID string, coarseUnitIDs []int64) ([]model.UserUnitServingState, error)
	IncrementServedCounts(ctx context.Context, userID string, runID string, servedAt time.Time, coarseUnitIDs []int64) error
}

type VideoServingStateRepository interface {
	ListByUserAndVideoIDs(ctx context.Context, userID string, videoIDs []string) ([]model.UserVideoServingState, error)
	IncrementServedCounts(ctx context.Context, userID string, runID string, servedAt time.Time, videoIDs []string) error
}
