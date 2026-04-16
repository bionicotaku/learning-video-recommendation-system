package repository

import (
	"context"

	"learning-video-recommendation-system/internal/learningengine/domain/model"
)

type UserUnitStateRepository interface {
	GetByUserAndUnitForUpdate(ctx context.Context, userID string, coarseUnitID int64) (*model.UserUnitState, error)
	Upsert(ctx context.Context, state *model.UserUnitState) (*model.UserUnitState, error)
	BatchUpsert(ctx context.Context, states []*model.UserUnitState) ([]*model.UserUnitState, error)
	ListByUser(ctx context.Context, userID string) ([]model.UserUnitState, error)
	ListRecommendationStates(ctx context.Context, userID string) ([]model.UserUnitState, error)
}
