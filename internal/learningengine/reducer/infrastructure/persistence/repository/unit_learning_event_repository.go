package repository

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

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
			InputIndex:                len(payload),
			UserID:                    event.UserID,
			CoarseUnitID:              event.CoarseUnitID,
			VideoID:                   event.VideoID,
			EventType:                 event.EventType,
			ReducerEffect:             event.ReducerEffect,
			ProgressQuality:           event.ProgressQuality,
			SourceType:                event.SourceType,
			SourceRefID:               event.SourceRefID,
			IsCorrect:                 event.IsCorrect,
			CountsTowardSuccessStreak: event.CountsTowardSuccessStreak,
			ConsumedWatchSessionIDs:   append([]string(nil), event.ConsumedWatchSessionIDs...),
			Metadata:                  metadata,
			OccurredAt:                event.OccurredAt.UTC(),
			ResetBoundaryAt:           event.ResetBoundaryAt,
		})
	}

	encoded, err := json.Marshal(payload)
	if err != nil {
		return apprepo.AppendLearningEventsResult{}, err
	}

	rows, err := r.queries.AppendLearningEvents(ctx, encoded)
	if err != nil {
		if isResetClientEventDuplicate(err) {
			return apprepo.AppendLearningEventsResult{}, apprepo.ErrDuplicateResetClientEvent
		}
		return apprepo.AppendLearningEventsResult{}, err
	}

	for _, row := range rows {
		result.InsertedEvents = append(result.InsertedEvents, mapper.ToLearningEventFromAppendRow(row))
	}
	result.DuplicateCount = len(events) - len(result.InsertedEvents)

	return result, nil
}

func isResetClientEventDuplicate(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) &&
		pgErr.Code == "23505" &&
		pgErr.ConstraintName == "uq_unit_learning_events_reset_client_event"
}

type appendLearningEventPayload struct {
	InputIndex                int             `json:"input_index"`
	UserID                    string          `json:"user_id"`
	CoarseUnitID              int64           `json:"coarse_unit_id"`
	VideoID                   string          `json:"video_id,omitempty"`
	EventType                 string          `json:"event_type"`
	ReducerEffect             string          `json:"reducer_effect"`
	ProgressQuality           *int16          `json:"progress_quality"`
	SourceType                string          `json:"source_type"`
	SourceRefID               string          `json:"source_ref_id"`
	IsCorrect                 *bool           `json:"is_correct"`
	CountsTowardSuccessStreak bool            `json:"counts_toward_success_streak"`
	ConsumedWatchSessionIDs   []string        `json:"consumed_watch_session_ids"`
	Metadata                  json.RawMessage `json:"metadata"`
	OccurredAt                time.Time       `json:"occurred_at"`
	ResetBoundaryAt           *time.Time      `json:"reset_boundary_at,omitempty"`
}

func (r *UnitLearningEventRepository) GetByUserSourceRef(ctx context.Context, userID string, sourceType string, sourceRefID string) (*model.LearningEvent, error) {
	pgUserID, err := mapper.StringToUUID(userID)
	if err != nil {
		return nil, err
	}

	row, err := r.queries.GetLearningEventByUserSourceRef(ctx, learningenginesqlc.GetLearningEventByUserSourceRefParams{
		UserID:      pgUserID,
		SourceType:  sourceType,
		SourceRefID: sourceRefID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	event := mapper.ToLearningEvent(row)
	return &event, nil
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

func (r *UnitLearningEventRepository) ListWatermarksByUserUnits(ctx context.Context, userID string, coarseUnitIDs []int64) (map[int64]model.UnitLearningEventWatermark, error) {
	result := make(map[int64]model.UnitLearningEventWatermark, len(coarseUnitIDs))
	if len(coarseUnitIDs) == 0 {
		return result, nil
	}

	pgUserID, err := mapper.StringToUUID(userID)
	if err != nil {
		return nil, err
	}
	encodedUnitIDs, err := json.Marshal(coarseUnitIDs)
	if err != nil {
		return nil, err
	}

	rows, err := r.queries.ListLearningEventWatermarksByUserUnits(ctx, learningenginesqlc.ListLearningEventWatermarksByUserUnitsParams{
		UserID:        pgUserID,
		CoarseUnitIds: encodedUnitIDs,
	})
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		result[row.CoarseUnitID] = model.UnitLearningEventWatermark{
			CoarseUnitID:       row.CoarseUnitID,
			MaxOccurredAt:      mapper.TimePointerFromPG(row.MaxOccurredAt),
			MaxResetBoundaryAt: mapper.TimePointerFromPG(row.MaxResetBoundaryAt),
		}
	}
	return result, nil
}
