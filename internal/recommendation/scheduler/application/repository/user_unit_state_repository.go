package repository

import (
	"context"
	"time"

	"learning-video-recommendation-system/internal/recommendation/scheduler/application/query"
	"learning-video-recommendation-system/internal/recommendation/scheduler/domain/model"
	"learning-video-recommendation-system/internal/recommendation/scheduler/infrastructure/persistence/sqlcgen"

	"github.com/google/uuid"
)

type UserUnitStateRepository interface {
	GetByUserAndUnit(ctx context.Context, q sqlcgen.Querier, userID uuid.UUID, coarseUnitID int64) (*model.UserUnitState, error)
	Upsert(ctx context.Context, q sqlcgen.Querier, state *model.UserUnitState) error
	BatchUpsert(ctx context.Context, q sqlcgen.Querier, states []*model.UserUnitState) error
	FindDueReviewCandidates(ctx context.Context, q sqlcgen.Querier, userID uuid.UUID, now time.Time) ([]query.ReviewCandidate, error)
	FindNewCandidates(ctx context.Context, q sqlcgen.Querier, userID uuid.UUID) ([]query.NewCandidate, error)
}
