package repository

import (
	"context"

	"learning-video-recommendation-system/internal/learningengine/reducer/domain/model"
)

type UserUnitStateRepository interface {
	GetByUserAndUnit(ctx context.Context, userID string, coarseUnitID int64) (*model.UserUnitState, error)
	GetByUserAndUnitForUpdate(ctx context.Context, userID string, coarseUnitID int64) (*model.UserUnitState, error)
	ListByUserAndUnitIDsForUpdate(ctx context.Context, userID string, coarseUnitIDs []int64) (map[int64]*model.UserUnitState, error)
	Upsert(ctx context.Context, state *model.UserUnitState) (*model.UserUnitState, error)
	BatchUpsert(ctx context.Context, states []*model.UserUnitState) ([]*model.UserUnitState, error)
	DeleteByUser(ctx context.Context, userID string) error
	ListByUser(ctx context.Context, userID string, filter model.UserUnitStateFilter) ([]model.UserUnitState, error)
}
