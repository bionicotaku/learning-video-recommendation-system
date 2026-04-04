package repository

import (
	"context"

	"learning-video-recommendation-system/internal/recommendation/scheduler/domain/model"

	"github.com/google/uuid"
)

type UnitLearningEventRepository interface {
	Append(ctx context.Context, events []model.LearningEvent) error
	ListByUserOrdered(ctx context.Context, userID uuid.UUID) ([]model.LearningEvent, error)
}
