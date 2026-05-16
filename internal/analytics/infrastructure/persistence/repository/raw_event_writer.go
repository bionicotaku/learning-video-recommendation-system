package repository

import (
	"context"
	"encoding/json"

	apprepo "learning-video-recommendation-system/internal/analytics/application/repository"
	"learning-video-recommendation-system/internal/analytics/domain/model"
	"learning-video-recommendation-system/internal/analytics/infrastructure/persistence/mapper"
	analyticssqlc "learning-video-recommendation-system/internal/analytics/infrastructure/persistence/sqlcgen"
)

type RawEventWriter struct {
	db      analyticssqlc.DBTX
	queries *analyticssqlc.Queries
}

var _ apprepo.RawEventWriter = (*RawEventWriter)(nil)

func NewRawEventWriter(db analyticssqlc.DBTX) *RawEventWriter {
	return &RawEventWriter{
		db:      db,
		queries: analyticssqlc.New(db),
	}
}

func (w *RawEventWriter) UpsertLearningInteractions(ctx context.Context, events []model.RawLearningInteractionEvent) ([]model.RawEventWriteResult, error) {
	return upsertLearningInteractions(ctx, w.queries, events)
}

func (w *RawEventWriter) UpsertQuizEvent(ctx context.Context, event model.RawQuizEvent) (model.RawEventWriteResult, error) {
	return upsertQuizEvent(ctx, w.queries, event)
}

func upsertLearningInteractions(ctx context.Context, queries *analyticssqlc.Queries, events []model.RawLearningInteractionEvent) ([]model.RawEventWriteResult, error) {
	if len(events) == 0 {
		return nil, nil
	}

	eventsJSON, err := learningInteractionEventsJSON(events)
	if err != nil {
		return nil, err
	}
	rows, err := queries.InsertLearningInteractionEvents(ctx, eventsJSON)
	if err != nil {
		return nil, err
	}
	results := make([]model.RawEventWriteResult, 0, len(events))
	for _, row := range rows {
		results = append(results, model.RawEventWriteResult{ClientEventID: row.ClientEventID, EventID: mapper.UUIDToString(row.EventID), Inserted: row.Inserted})
	}
	return results, nil
}

type learningInteractionEventJSON struct {
	ClientEventID                  string          `json:"client_event_id"`
	UserID                         string          `json:"user_id"`
	ClientContext                  json.RawMessage `json:"client_context"`
	EventType                      string          `json:"event_type"`
	SourceSurface                  string          `json:"source_surface"`
	VideoID                        string          `json:"video_id,omitempty"`
	WatchSessionID                 string          `json:"watch_session_id,omitempty"`
	RecommendationRunID            string          `json:"recommendation_run_id,omitempty"`
	RelatedQuizEventID             string          `json:"related_quiz_event_id,omitempty"`
	CoarseUnitID                   *int64          `json:"coarse_unit_id,omitempty"`
	TokenText                      string          `json:"token_text,omitempty"`
	SentenceIndex                  *int32          `json:"sentence_index,omitempty"`
	SpanIndex                      *int32          `json:"span_index,omitempty"`
	OccurredAt                     string          `json:"occurred_at"`
	ExposureStartMS                *int32          `json:"exposure_start_ms,omitempty"`
	ExposureEndMS                  *int32          `json:"exposure_end_ms,omitempty"`
	ExposureCount                  *int32          `json:"exposure_count,omitempty"`
	LookupVisibleMS                *int32          `json:"lookup_visible_ms,omitempty"`
	LookupSentenceAudioReplayCount int32           `json:"lookup_sentence_audio_replay_count,omitempty"`
	LookupWordAudioPlayCount       int32           `json:"lookup_word_audio_play_count,omitempty"`
	LookupPracticeNowClicked       bool            `json:"lookup_practice_now_clicked,omitempty"`
	EventPayload                   json.RawMessage `json:"event_payload"`
}

func learningInteractionEventsJSON(events []model.RawLearningInteractionEvent) ([]byte, error) {
	rows := make([]learningInteractionEventJSON, 0, len(events))
	for _, event := range events {
		occurredAt := event.OccurredAt.UTC()
		rows = append(rows, learningInteractionEventJSON{
			ClientEventID:                  event.ClientEventID,
			UserID:                         event.UserID,
			ClientContext:                  json.RawMessage(defaultJSONObject(event.ClientContext)),
			EventType:                      event.EventType,
			SourceSurface:                  event.SourceSurface,
			VideoID:                        event.VideoID,
			WatchSessionID:                 event.WatchSessionID,
			RecommendationRunID:            event.RecommendationRunID,
			RelatedQuizEventID:             event.RelatedQuizEventID,
			CoarseUnitID:                   event.CoarseUnitID,
			TokenText:                      event.TokenText,
			SentenceIndex:                  event.SentenceIndex,
			SpanIndex:                      event.SpanIndex,
			OccurredAt:                     occurredAt.Format("2006-01-02T15:04:05.999999999Z07:00"),
			ExposureStartMS:                event.ExposureStartMS,
			ExposureEndMS:                  event.ExposureEndMS,
			ExposureCount:                  event.ExposureCount,
			LookupVisibleMS:                event.LookupVisibleMS,
			LookupSentenceAudioReplayCount: event.LookupSentenceAudioReplayCount,
			LookupWordAudioPlayCount:       event.LookupWordAudioPlayCount,
			LookupPracticeNowClicked:       event.LookupPracticeNowClicked,
			EventPayload:                   json.RawMessage(defaultJSONObject(event.EventPayload)),
		})
	}
	return json.Marshal(rows)
}

func upsertQuizEvent(ctx context.Context, queries *analyticssqlc.Queries, event model.RawQuizEvent) (model.RawEventWriteResult, error) {
	userID, err := mapper.StringToUUID(event.UserID)
	if err != nil {
		return model.RawEventWriteResult{}, err
	}
	questionID, err := mapper.StringToUUID(event.QuestionID)
	if err != nil {
		return model.RawEventWriteResult{}, err
	}
	videoID, err := mapper.StringToUUID(event.VideoID)
	if err != nil {
		return model.RawEventWriteResult{}, err
	}
	recommendationRunID, err := mapper.StringToUUID(event.RecommendationRunID)
	if err != nil {
		return model.RawEventWriteResult{}, err
	}

	row, err := queries.InsertQuizEvent(ctx, analyticssqlc.InsertQuizEventParams{
		ClientEventID:       event.ClientEventID,
		UserID:              userID,
		ClientContext:       defaultJSONObject(event.ClientContext),
		QuestionID:          questionID,
		CoarseUnitID:        event.CoarseUnitID,
		VideoID:             videoID,
		RecommendationRunID: recommendationRunID,
		TriggerType:         event.TriggerType,
		SelectedOptionIds:   event.SelectedOptionIDs,
		SelectionIntervalMs: event.SelectionIntervalMS,
		IsFirstTryCorrect:   event.IsFirstTryCorrect,
		TotalElapsedMs:      event.TotalElapsedMS,
		ShownAt:             mapper.TimePointerToPG(&event.ShownAt),
		CompletedAt:         mapper.TimePointerToPG(&event.CompletedAt),
	})
	if err != nil {
		return model.RawEventWriteResult{}, err
	}
	return model.RawEventWriteResult{ClientEventID: event.ClientEventID, EventID: mapper.UUIDToString(row.EventID), Inserted: row.Inserted}, nil
}

func defaultJSONObject(value []byte) []byte {
	if len(value) == 0 {
		return []byte("{}")
	}
	return value
}
