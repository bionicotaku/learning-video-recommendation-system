package repository

import (
	"context"
	"time"

	"learning-video-recommendation-system/internal/learningengine/normalizer/domain/model"
	learningdto "learning-video-recommendation-system/internal/learningengine/reducer/application/dto"
)

type PendingRawEventFilter struct {
	UserID         string
	Limit          int
	OccurredBefore *time.Time
}

type RawQuizEventReader interface {
	ListPendingQuizEvents(ctx context.Context, filter PendingRawEventFilter) ([]model.RawQuizEvent, error)
	ListQuizEventsByIDs(ctx context.Context, userID string, eventIDs []string) ([]model.RawQuizEvent, error)
}

type RawLearningInteractionReader interface {
	ListPendingLearningInteractions(ctx context.Context, filter PendingRawEventFilter) ([]model.RawLearningInteraction, error)
	ListLearningInteractionsByIDs(ctx context.Context, userID string, eventIDs []string) ([]model.RawLearningInteraction, error)
}

type LearningEventRecorder interface {
	Execute(ctx context.Context, request learningdto.RecordLearningEventsRequest) (learningdto.RecordLearningEventsResponse, error)
}
