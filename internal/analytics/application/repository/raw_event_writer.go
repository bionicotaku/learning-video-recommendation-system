package repository

import (
	"context"

	"learning-video-recommendation-system/internal/analytics/domain/model"
)

type RawEventWriter interface {
	UpsertLearningInteractions(ctx context.Context, events []model.RawLearningInteractionEvent) ([]model.RawEventWriteResult, error)
	UpsertQuizEvent(ctx context.Context, event model.RawQuizEvent) (model.RawEventWriteResult, error)
}
