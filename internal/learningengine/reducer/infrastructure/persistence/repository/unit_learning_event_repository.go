package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	apprepo "learning-video-recommendation-system/internal/learningengine/reducer/application/repository"
	"learning-video-recommendation-system/internal/learningengine/reducer/domain/model"
	"learning-video-recommendation-system/internal/learningengine/reducer/infrastructure/persistence/mapper"
	learningenginesqlc "learning-video-recommendation-system/internal/learningengine/reducer/infrastructure/persistence/sqlcgen"
)

type UnitLearningEventRepository struct {
	queries *learningenginesqlc.Queries
}

var _ apprepo.UnitLearningEventRepository = (*UnitLearningEventRepository)(nil)

func NewUnitLearningEventRepository(db learningenginesqlc.DBTX) *UnitLearningEventRepository {
	return &UnitLearningEventRepository{
		queries: learningenginesqlc.New(db),
	}
}

func (r *UnitLearningEventRepository) Append(ctx context.Context, events []model.LearningEvent) (apprepo.AppendLearningEventsResult, error) {
	result := apprepo.AppendLearningEventsResult{
		InsertedEvents: make([]model.LearningEvent, 0, len(events)),
	}
	for _, event := range events {
		userID, err := mapper.StringToUUID(event.UserID)
		if err != nil {
			return apprepo.AppendLearningEventsResult{}, err
		}
		videoID, err := mapper.StringToUUID(event.VideoID)
		if err != nil {
			return apprepo.AppendLearningEventsResult{}, err
		}

		metadata := event.Metadata
		if len(metadata) == 0 {
			metadata = []byte("{}")
		}

		row, err := r.queries.AppendLearningEvent(ctx, learningenginesqlc.AppendLearningEventParams{
			UserID:          userID,
			CoarseUnitID:    event.CoarseUnitID,
			VideoID:         videoID,
			EventType:       event.EventType,
			ReducerEffect:   event.ReducerEffect,
			ProgressQuality: mapper.Int16PointerToPG(event.ProgressQuality),
			SourceType:      event.SourceType,
			SourceRefID:     event.SourceRefID,
			IsCorrect:       mapper.BoolPointerToPG(event.IsCorrect),
			Metadata:        metadata,
			OccurredAt:      mapper.TimePointerToPG(&event.OccurredAt),
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				result.DuplicateCount++
				continue
			}
			return apprepo.AppendLearningEventsResult{}, err
		}
		result.InsertedEvents = append(result.InsertedEvents, mapper.ToLearningEvent(row))
	}

	return result, nil
}

func (r *UnitLearningEventRepository) ListByUserOrdered(ctx context.Context, userID string) ([]model.LearningEvent, error) {
	pgUserID, err := mapper.StringToUUID(userID)
	if err != nil {
		return nil, err
	}

	rows, err := r.queries.ListLearningEventsByUserOrdered(ctx, pgUserID)
	if err != nil {
		return nil, err
	}

	result := make([]model.LearningEvent, 0, len(rows))
	for _, row := range rows {
		result = append(result, mapper.ToLearningEvent(row))
	}
	return result, nil
}

func (r *UnitLearningEventRepository) ListByUserAndUnitOrdered(ctx context.Context, userID string, coarseUnitID int64) ([]model.LearningEvent, error) {
	pgUserID, err := mapper.StringToUUID(userID)
	if err != nil {
		return nil, err
	}

	rows, err := r.queries.ListLearningEventsByUserUnitOrdered(ctx, learningenginesqlc.ListLearningEventsByUserUnitOrderedParams{
		UserID:       pgUserID,
		CoarseUnitID: coarseUnitID,
	})
	if err != nil {
		return nil, err
	}

	result := make([]model.LearningEvent, 0, len(rows))
	for _, row := range rows {
		result = append(result, mapper.ToLearningEvent(row))
	}
	return result, nil
}
