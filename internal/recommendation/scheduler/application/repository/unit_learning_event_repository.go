package repository

import (
	"context"
	"time"

	"learning-video-recommendation-system/internal/recommendation/scheduler/domain/model"
	"learning-video-recommendation-system/internal/recommendation/scheduler/infrastructure/persistence/sqlcgen"

	"github.com/google/uuid"
)

type UnitLearningEventRepository interface {
	Append(ctx context.Context, q sqlcgen.Querier, events []model.LearningEvent) error
	FindForReplay(ctx context.Context, q sqlcgen.Querier, userID uuid.UUID, coarseUnitID *int64, from *time.Time) ([]model.LearningEvent, error)
}
