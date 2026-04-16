package repository

import (
	"context"

	"learning-video-recommendation-system/internal/learningengine/domain/model"
)

type TargetStateCommandRepository interface {
	EnsureTargetUnits(ctx context.Context, userID string, targets []model.TargetUnitSpec) error
	SetTargetInactive(ctx context.Context, userID string, coarseUnitID int64) error
}
