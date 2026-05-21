package repository

import (
	"context"
	"errors"

	"learning-video-recommendation-system/internal/learningengine/reducer/domain/model"
)

var ErrUnitCollectionNotFound = errors.New("unit collection not found")

type TargetStateCommandRepository interface {
	EnsureTargetUnits(ctx context.Context, userID string, targets []model.TargetUnitSpec) error
	ActivateUnitCollectionTarget(ctx context.Context, userID string, collectionSlug string) (model.ActivatedUnitCollectionTarget, error)
	SetTargetInactive(ctx context.Context, userID string, coarseUnitID int64) error
}
