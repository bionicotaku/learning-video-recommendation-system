package repository

import (
	"context"
	"errors"

	"learning-video-recommendation-system/internal/learningengine/reducer/domain/model"
)

var ErrDuplicateResetClientEvent = errors.New("duplicate reset client_event_id")

type AppendLearningEventsResult struct {
	InsertedEvents []model.LearningEvent
	DuplicateCount int
}

type UnitLearningEventRepository interface {
	Append(ctx context.Context, events []model.LearningEvent) (AppendLearningEventsResult, error)
	GetByUserSourceRef(ctx context.Context, userID string, sourceType string, sourceRefID string) (*model.LearningEvent, error)
	ListByUserOrdered(ctx context.Context, userID string) ([]model.LearningEvent, error)
	ListByUserAndUnitOrdered(ctx context.Context, userID string, coarseUnitID int64) ([]model.LearningEvent, error)
	ListWatermarksByUserUnits(ctx context.Context, userID string, coarseUnitIDs []int64) (map[int64]model.UnitLearningEventWatermark, error)
}
