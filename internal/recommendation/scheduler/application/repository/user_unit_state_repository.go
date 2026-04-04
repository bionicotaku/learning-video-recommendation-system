package repository

import (
	"context"
	"time"

	"learning-video-recommendation-system/internal/recommendation/scheduler/application/query"
	"learning-video-recommendation-system/internal/recommendation/scheduler/domain/model"

	"github.com/google/uuid"
)

type UserUnitStateRepository interface {
	GetByUserAndUnit(ctx context.Context, userID uuid.UUID, coarseUnitID int64) (*model.UserUnitState, error)
	Upsert(ctx context.Context, state *model.UserUnitState) error
	BatchUpsert(ctx context.Context, states []*model.UserUnitState) error
	DeleteByUser(ctx context.Context, userID uuid.UUID) error
	FindDueReviewCandidates(ctx context.Context, userID uuid.UUID, now time.Time) ([]query.ReviewCandidate, error)
	FindNewCandidates(ctx context.Context, userID uuid.UUID) ([]query.NewCandidate, error)
}
