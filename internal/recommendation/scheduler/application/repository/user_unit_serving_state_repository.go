package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type UserUnitServingStateRepository interface {
	TouchRecommendedAt(ctx context.Context, userID uuid.UUID, runID uuid.UUID, coarseUnitIDs []int64, recommendedAt time.Time) error
}
