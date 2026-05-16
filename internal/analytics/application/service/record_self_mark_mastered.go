package service

import (
	"context"
	"fmt"

	"learning-video-recommendation-system/internal/analytics/application/dto"
	apprepo "learning-video-recommendation-system/internal/analytics/application/repository"
	appusecase "learning-video-recommendation-system/internal/analytics/application/usecase"
	"learning-video-recommendation-system/internal/analytics/domain/model"
)

type RecordSelfMarkMasteredUsecase struct {
	writer apprepo.RawEventWriter
}

var _ appusecase.RecordSelfMarkMasteredUsecase = (*RecordSelfMarkMasteredUsecase)(nil)

func NewRecordSelfMarkMasteredUsecase(writer apprepo.RawEventWriter) *RecordSelfMarkMasteredUsecase {
	return &RecordSelfMarkMasteredUsecase{writer: writer}
}

func (u *RecordSelfMarkMasteredUsecase) Execute(ctx context.Context, request dto.RecordSelfMarkMasteredRequest) (dto.RecordSelfMarkMasteredResponse, error) {
	if u.writer == nil {
		return dto.RecordSelfMarkMasteredResponse{}, fmt.Errorf("raw event writer is required")
	}
	event, err := mapSelfMarkMasteredRequest(request)
	if err != nil {
		return dto.RecordSelfMarkMasteredResponse{}, err
	}

	results, err := u.writer.UpsertLearningInteractions(ctx, []model.RawLearningInteractionEvent{event})
	if err != nil {
		return dto.RecordSelfMarkMasteredResponse{}, err
	}
	if len(results) != 1 {
		return dto.RecordSelfMarkMasteredResponse{}, fmt.Errorf("expected one self mark write result, got %d", len(results))
	}
	return dto.RecordSelfMarkMasteredResponse{
		Accepted:                   true,
		LearningInteractionEventID: results[0].EventID,
		Inserted:                   results[0].Inserted,
	}, nil
}

func mapSelfMarkMasteredRequest(request dto.RecordSelfMarkMasteredRequest) (model.RawLearningInteractionEvent, error) {
	if request.UserID == "" {
		return model.RawLearningInteractionEvent{}, validationError("user_id is required")
	}
	if request.ClientEventID == "" {
		return model.RawLearningInteractionEvent{}, validationError("client_event_id is required")
	}
	if request.CoarseUnitID <= 0 {
		return model.RawLearningInteractionEvent{}, validationError("coarse_unit_id is required")
	}
	if request.SourceSurface == "" {
		return model.RawLearningInteractionEvent{}, validationError("source_surface is required")
	}
	if request.OccurredAt.IsZero() {
		return model.RawLearningInteractionEvent{}, validationError("occurred_at is required")
	}

	clientContext, err := normalizeJSONObject(request.ClientContext, "client_context")
	if err != nil {
		return model.RawLearningInteractionEvent{}, err
	}
	eventPayload, err := normalizeJSONObject(request.EventPayload, "event_payload")
	if err != nil {
		return model.RawLearningInteractionEvent{}, err
	}

	return model.RawLearningInteractionEvent{
		ClientEventID:       request.ClientEventID,
		UserID:              request.UserID,
		ClientContext:       clientContext,
		EventType:           "self_mark_mastered",
		SourceSurface:       request.SourceSurface,
		VideoID:             request.VideoID,
		WatchSessionID:      request.WatchSessionID,
		RecommendationRunID: request.RecommendationRunID,
		RelatedQuizEventID:  request.RelatedQuizEventID,
		CoarseUnitID:        &request.CoarseUnitID,
		TokenText:           request.TokenText,
		SentenceIndex:       request.SentenceIndex,
		SpanIndex:           request.SpanIndex,
		OccurredAt:          request.OccurredAt.UTC(),
		EventPayload:        eventPayload,
	}, nil
}
