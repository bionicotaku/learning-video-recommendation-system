package repository

import (
	"context"

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
	results := make([]model.RawEventWriteResult, 0, len(events))
	for _, event := range events {
		userID, err := mapper.StringToUUID(event.UserID)
		if err != nil {
			return nil, err
		}
		videoID, err := mapper.StringToUUID(event.VideoID)
		if err != nil {
			return nil, err
		}
		watchSessionID, err := mapper.StringToUUID(event.WatchSessionID)
		if err != nil {
			return nil, err
		}
		recommendationRunID, err := mapper.StringToUUID(event.RecommendationRunID)
		if err != nil {
			return nil, err
		}
		relatedQuizEventID, err := mapper.StringToUUID(event.RelatedQuizEventID)
		if err != nil {
			return nil, err
		}

		row, err := queries.InsertLearningInteractionEvent(ctx, analyticssqlc.InsertLearningInteractionEventParams{
			ClientEventID:                  event.ClientEventID,
			UserID:                         userID,
			ClientContext:                  defaultJSONObject(event.ClientContext),
			EventType:                      event.EventType,
			SourceSurface:                  event.SourceSurface,
			VideoID:                        videoID,
			WatchSessionID:                 watchSessionID,
			RecommendationRunID:            recommendationRunID,
			RelatedQuizEventID:             relatedQuizEventID,
			CoarseUnitID:                   mapper.Int64PointerToPG(event.CoarseUnitID),
			TokenText:                      mapper.StringToText(event.TokenText),
			SentenceIndex:                  mapper.Int32PointerToPG(event.SentenceIndex),
			SpanIndex:                      mapper.Int32PointerToPG(event.SpanIndex),
			OccurredAt:                     mapper.TimePointerToPG(&event.OccurredAt),
			ExposureStartMs:                mapper.Int32PointerToPG(event.ExposureStartMS),
			ExposureEndMs:                  mapper.Int32PointerToPG(event.ExposureEndMS),
			ExposureCount:                  mapper.Int32PointerToPG(event.ExposureCount),
			LookupVisibleMs:                mapper.Int32PointerToPG(event.LookupVisibleMS),
			LookupSentenceAudioReplayCount: event.LookupSentenceAudioReplayCount,
			LookupWordAudioPlayCount:       event.LookupWordAudioPlayCount,
			LookupPracticeNowClicked:       event.LookupPracticeNowClicked,
			EventPayload:                   defaultJSONObject(event.EventPayload),
		})
		if err != nil {
			return nil, err
		}
		results = append(results, model.RawEventWriteResult{ClientEventID: event.ClientEventID, EventID: mapper.UUIDToString(row.EventID), Inserted: row.Inserted})
	}
	return results, nil
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
