package repository

import (
	"context"
	"time"

	"learning-video-recommendation-system/internal/recommendation/scheduler/domain/model"

	"github.com/google/uuid"
)

type UnitLearningEventRepository interface {
	Append(ctx context.Context, events []model.LearningEvent) error
	FindForReplay(ctx context.Context, userID uuid.UUID, coarseUnitID *int64, from *time.Time) ([]model.LearningEvent, error)
}
