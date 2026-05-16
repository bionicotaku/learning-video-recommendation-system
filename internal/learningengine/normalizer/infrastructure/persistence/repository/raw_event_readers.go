package repository

import (
	"context"

	apprepo "learning-video-recommendation-system/internal/learningengine/normalizer/application/repository"
	"learning-video-recommendation-system/internal/learningengine/normalizer/domain/model"
	"learning-video-recommendation-system/internal/learningengine/normalizer/infrastructure/persistence/mapper"
	normalizersqlc "learning-video-recommendation-system/internal/learningengine/normalizer/infrastructure/persistence/sqlcgen"
)

type RawQuizEventReader struct {
	queries *normalizersqlc.Queries
}

var _ apprepo.RawQuizEventReader = (*RawQuizEventReader)(nil)

func NewRawQuizEventReader(db normalizersqlc.DBTX) *RawQuizEventReader {
	return &RawQuizEventReader{queries: normalizersqlc.New(db)}
}

func (r *RawQuizEventReader) ListPendingQuizEvents(ctx context.Context, filter apprepo.PendingRawEventFilter) ([]model.RawQuizEvent, error) {
	userID, err := mapper.StringToUUID(filter.UserID)
	if err != nil {
		return nil, err
	}

	rows, err := r.queries.ListPendingQuizEvents(ctx, normalizersqlc.ListPendingQuizEventsParams{
		UserID:         userID,
		OccurredBefore: mapper.TimePointerToPG(filter.OccurredBefore),
		LimitCount:     int32(filter.Limit),
	})
	if err != nil {
		return nil, err
	}

	result := make([]model.RawQuizEvent, 0, len(rows))
	for _, row := range rows {
		result = append(result, mapper.ToRawQuizEvent(row))
	}
	return result, nil
}

func (r *RawQuizEventReader) ListQuizEventsByIDs(ctx context.Context, userID string, eventIDs []string) ([]model.RawQuizEvent, error) {
	pgUserID, err := mapper.StringToUUID(userID)
	if err != nil {
		return nil, err
	}
	pgEventIDs, err := mapper.StringsToUUIDs(eventIDs)
	if err != nil {
		return nil, err
	}

	rows, err := r.queries.ListQuizEventsByIDs(ctx, normalizersqlc.ListQuizEventsByIDsParams{
		UserID:   pgUserID,
		EventIds: pgEventIDs,
	})
	if err != nil {
		return nil, err
	}

	result := make([]model.RawQuizEvent, 0, len(rows))
	for _, row := range rows {
		result = append(result, mapper.ToRawQuizEventByID(row))
	}
	return result, nil
}

type RawLearningInteractionReader struct {
	queries *normalizersqlc.Queries
}

var _ apprepo.RawLearningInteractionReader = (*RawLearningInteractionReader)(nil)

func NewRawLearningInteractionReader(db normalizersqlc.DBTX) *RawLearningInteractionReader {
	return &RawLearningInteractionReader{queries: normalizersqlc.New(db)}
}

func (r *RawLearningInteractionReader) ListPendingLearningInteractions(ctx context.Context, filter apprepo.PendingRawEventFilter) ([]model.RawLearningInteraction, error) {
	userID, err := mapper.StringToUUID(filter.UserID)
	if err != nil {
		return nil, err
	}

	rows, err := r.queries.ListPendingLearningInteractions(ctx, normalizersqlc.ListPendingLearningInteractionsParams{
		UserID:         userID,
		OccurredBefore: mapper.TimePointerToPG(filter.OccurredBefore),
		LimitCount:     int32(filter.Limit),
	})
	if err != nil {
		return nil, err
	}

	result := make([]model.RawLearningInteraction, 0, len(rows))
	for _, row := range rows {
		result = append(result, mapper.ToRawLearningInteraction(row))
	}
	return result, nil
}

func (r *RawLearningInteractionReader) ListLearningInteractionsByIDs(ctx context.Context, userID string, eventIDs []string) ([]model.RawLearningInteraction, error) {
	pgUserID, err := mapper.StringToUUID(userID)
	if err != nil {
		return nil, err
	}
	pgEventIDs, err := mapper.StringsToUUIDs(eventIDs)
	if err != nil {
		return nil, err
	}

	rows, err := r.queries.ListLearningInteractionsByIDs(ctx, normalizersqlc.ListLearningInteractionsByIDsParams{
		UserID:   pgUserID,
		EventIds: pgEventIDs,
	})
	if err != nil {
		return nil, err
	}

	result := make([]model.RawLearningInteraction, 0, len(rows))
	for _, row := range rows {
		result = append(result, mapper.ToRawLearningInteractionByID(row))
	}
	return result, nil
}
