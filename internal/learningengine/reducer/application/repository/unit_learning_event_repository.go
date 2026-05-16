package repository

import (
	"context"

	"learning-video-recommendation-system/internal/learningengine/reducer/domain/model"
)

type AppendLearningEventsResult struct {
	InsertedEvents []model.LearningEvent
	DuplicateCount int
}

type UnitLearningEventRepository interface {
	Append(ctx context.Context, events []model.LearningEvent) (AppendLearningEventsResult, error)
	ListByUserOrdered(ctx context.Context, userID string) ([]model.LearningEvent, error)
	ListByUserAndUnitOrdered(ctx context.Context, userID string, coarseUnitID int64) ([]model.LearningEvent, error)
}
