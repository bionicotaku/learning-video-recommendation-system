package repository

import (
	"context"

	"learning-video-recommendation-system/internal/learningengine/domain/model"
)

type UnitLearningEventRepository interface {
	Append(ctx context.Context, events []model.LearningEvent) error
	ListByUserOrdered(ctx context.Context, userID string) ([]model.LearningEvent, error)
	ListByUserAndUnitOrdered(ctx context.Context, userID string, coarseUnitID int64) ([]model.LearningEvent, error)
}
