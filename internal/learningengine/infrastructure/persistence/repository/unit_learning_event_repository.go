package repository

import (
	"context"

	apprepo "learning-video-recommendation-system/internal/learningengine/application/repository"
	"learning-video-recommendation-system/internal/learningengine/domain/model"
	"learning-video-recommendation-system/internal/learningengine/infrastructure/persistence/mapper"
	learningenginesqlc "learning-video-recommendation-system/internal/learningengine/infrastructure/persistence/sqlcgen"
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

func (r *UnitLearningEventRepository) Append(ctx context.Context, events []model.LearningEvent) error {
	for _, event := range events {
		userID, err := mapper.StringToUUID(event.UserID)
		if err != nil {
			return err
		}
		videoID, err := mapper.StringToUUID(event.VideoID)
		if err != nil {
			return err
		}

		if _, err := r.queries.AppendLearningEvent(ctx, learningenginesqlc.AppendLearningEventParams{
			UserID:         userID,
			CoarseUnitID:   event.CoarseUnitID,
			VideoID:        videoID,
			EventType:      event.EventType,
			SourceType:     event.SourceType,
			SourceRefID:    mapper.StringToText(event.SourceRefID),
			IsCorrect:      mapper.BoolPointerToPG(event.IsCorrect),
			Quality:        mapper.Int16PointerToPG(event.Quality),
			ResponseTimeMs: mapper.Int32PointerToPG(event.ResponseTimeMs),
			Metadata:       event.Metadata,
			OccurredAt:     mapper.TimePointerToPG(&event.OccurredAt),
		}); err != nil {
			return err
		}
	}

	return nil
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
