package repository

import (
	"context"
	"encoding/json"
	"time"

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
	result := apprepo.AppendLearningEventsResult{InsertedEvents: make([]model.LearningEvent, 0, len(events))}
	if len(events) == 0 {
		return result, nil
	}

	payload := make([]appendLearningEventPayload, 0, len(events))
	for _, event := range events {
		metadata := event.Metadata
		if len(metadata) == 0 {
			metadata = []byte("{}")
		}

		payload = append(payload, appendLearningEventPayload{
			InputIndex:      len(payload),
			UserID:          event.UserID,
			CoarseUnitID:    event.CoarseUnitID,
			VideoID:         event.VideoID,
			EventType:       event.EventType,
			ReducerEffect:   event.ReducerEffect,
			ProgressQuality: event.ProgressQuality,
			SourceType:      event.SourceType,
			SourceRefID:     event.SourceRefID,
			IsCorrect:       event.IsCorrect,
			Metadata:        metadata,
			OccurredAt:      event.OccurredAt.UTC(),
		})
	}

	encoded, err := json.Marshal(payload)
	if err != nil {
		return apprepo.AppendLearningEventsResult{}, err
	}

	rows, err := r.queries.AppendLearningEvents(ctx, encoded)
	if err != nil {
		return apprepo.AppendLearningEventsResult{}, err
	}

	for _, row := range rows {
		result.InsertedEvents = append(result.InsertedEvents, mapper.ToLearningEventFromAppendRow(row))
	}
	result.DuplicateCount = len(events) - len(result.InsertedEvents)

	return result, nil
}

type appendLearningEventPayload struct {
	InputIndex      int             `json:"input_index"`
	UserID          string          `json:"user_id"`
	CoarseUnitID    int64           `json:"coarse_unit_id"`
	VideoID         string          `json:"video_id,omitempty"`
	EventType       string          `json:"event_type"`
	ReducerEffect   string          `json:"reducer_effect"`
	ProgressQuality *int16          `json:"progress_quality"`
	SourceType      string          `json:"source_type"`
	SourceRefID     string          `json:"source_ref_id"`
	IsCorrect       *bool           `json:"is_correct"`
	Metadata        json.RawMessage `json:"metadata"`
	OccurredAt      time.Time       `json:"occurred_at"`
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
