package repository

import (
	"context"
	"time"

	"learning-video-recommendation-system/internal/learningengine/normalizer/domain/model"
)

type PendingRawEventFilter struct {
	UserID         string
	Limit          int
	OccurredBefore *time.Time
}

type RawQuizEventReader interface {
	ListPendingQuizEvents(ctx context.Context, filter PendingRawEventFilter) ([]model.RawQuizEvent, error)
}

type RawLearningInteractionReader interface {
	ListPendingLearningInteractions(ctx context.Context, filter PendingRawEventFilter) ([]model.RawLearningInteraction, error)
}
