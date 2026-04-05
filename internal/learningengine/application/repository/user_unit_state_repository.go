package repository

import (
	"context"

	"learning-video-recommendation-system/internal/learningengine/domain/model"

	"github.com/google/uuid"
)

type UserUnitStateRepository interface {
	GetByUserAndUnit(ctx context.Context, userID uuid.UUID, coarseUnitID int64) (*model.UserUnitState, error)
	Upsert(ctx context.Context, state *model.UserUnitState) error
	BatchUpsert(ctx context.Context, states []*model.UserUnitState) error
	DeleteByUser(ctx context.Context, userID uuid.UUID) error
}
