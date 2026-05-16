package service

import (
	"context"
	"fmt"

	"learning-video-recommendation-system/internal/analytics/application/dto"
	apprepo "learning-video-recommendation-system/internal/analytics/application/repository"
	appusecase "learning-video-recommendation-system/internal/analytics/application/usecase"
	"learning-video-recommendation-system/internal/analytics/domain/model"
)

type RecordLearningInteractionsBatchUsecase struct {
	writer apprepo.RawEventWriter
}

var _ appusecase.RecordLearningInteractionsBatchUsecase = (*RecordLearningInteractionsBatchUsecase)(nil)

func NewRecordLearningInteractionsBatchUsecase(writer apprepo.RawEventWriter) *RecordLearningInteractionsBatchUsecase {
	return &RecordLearningInteractionsBatchUsecase{writer: writer}
}

func (u *RecordLearningInteractionsBatchUsecase) Execute(ctx context.Context, request dto.RecordLearningInteractionsBatchRequest) (dto.RecordLearningInteractionsBatchResponse, error) {
	if u.writer == nil {
		return dto.RecordLearningInteractionsBatchResponse{}, fmt.Errorf("raw event writer is required")
	}
	if request.UserID == "" {
		return dto.RecordLearningInteractionsBatchResponse{}, validationError("user_id is required")
	}
	if request.VideoID == "" {
		return dto.RecordLearningInteractionsBatchResponse{}, validationError("video_id is required")
	}
	if request.WatchSessionID == "" {
		return dto.RecordLearningInteractionsBatchResponse{}, validationError("watch_session_id is required")
	}
	if len(request.Events) == 0 {
		return dto.RecordLearningInteractionsBatchResponse{}, validationError("events are required")
	}

	clientContext, err := normalizeJSONObject(request.ClientContext, "client_context")
	if err != nil {
		return dto.RecordLearningInteractionsBatchResponse{}, err
	}

	events := make([]model.RawLearningInteractionEvent, 0, len(request.Events))
	for index, input := range request.Events {
		event, err := mapLearningInteractionInput(request, clientContext, input, index)
		if err != nil {
			return dto.RecordLearningInteractionsBatchResponse{}, err
		}
		events = append(events, event)
	}

	results, err := u.writer.UpsertLearningInteractions(ctx, events)
	if err != nil {
		return dto.RecordLearningInteractionsBatchResponse{}, err
	}

	response := dto.RecordLearningInteractionsBatchResponse{AcceptedCount: len(events)}
	for _, result := range results {
		if result.Inserted {
			response.InsertedCount++
		} else {
			response.DuplicateCount++
		}
		response.AcceptedEvents = append(response.AcceptedEvents, dto.AcceptedLearningInteractionEvent{
			ClientEventID:              result.ClientEventID,
			LearningInteractionEventID: result.EventID,
			Inserted:                   result.Inserted,
		})
	}
	return response, nil
}

func mapLearningInteractionInput(request dto.RecordLearningInteractionsBatchRequest, clientContext []byte, input dto.LearningInteractionEventInput, index int) (model.RawLearningInteractionEvent, error) {
	if input.ClientEventID == "" {
		return model.RawLearningInteractionEvent{}, validationError("events[%d].client_event_id is required", index)
	}
	if input.EventType == "" {
		return model.RawLearningInteractionEvent{}, validationError("events[%d].event_type is required", index)
	}
	switch input.EventType {
	case "exposure", "lookup":
	default:
		return model.RawLearningInteractionEvent{}, validationError("events[%d].event_type is unsupported: %s", index, input.EventType)
	}
	if input.SourceSurface == "" {
		return model.RawLearningInteractionEvent{}, validationError("events[%d].source_surface is required", index)
	}
	if input.OccurredAt.IsZero() {
		return model.RawLearningInteractionEvent{}, validationError("events[%d].occurred_at is required", index)
	}
	if input.EventType == "lookup" && input.TokenText == "" {
		return model.RawLearningInteractionEvent{}, validationError("events[%d].token_text is required for lookup", index)
	}
	if input.EventType == "exposure" && (input.CoarseUnitID == nil || *input.CoarseUnitID <= 0) {
		return model.RawLearningInteractionEvent{}, validationError("events[%d].coarse_unit_id is required for exposure", index)
	}
	if input.EventType == "lookup" && input.CoarseUnitID != nil && *input.CoarseUnitID <= 0 {
		return model.RawLearningInteractionEvent{}, validationError("events[%d].coarse_unit_id must be positive when provided", index)
	}
	if learningInteractionEventRequiresSubtitleIndexes(input.EventType) {
		if input.SentenceIndex == nil {
			return model.RawLearningInteractionEvent{}, validationError("events[%d].sentence_index is required for %s", index, input.EventType)
		}
		if input.SpanIndex == nil {
			return model.RawLearningInteractionEvent{}, validationError("events[%d].span_index is required for %s", index, input.EventType)
		}
	}
	if err := validateNonNegativePointer(input.ExposureStartMS, fmt.Sprintf("events[%d].exposure_start_ms", index)); err != nil {
		return model.RawLearningInteractionEvent{}, err
	}
	if err := validateNonNegativePointer(input.ExposureEndMS, fmt.Sprintf("events[%d].exposure_end_ms", index)); err != nil {
		return model.RawLearningInteractionEvent{}, err
	}
	if input.ExposureStartMS != nil && input.ExposureEndMS != nil && *input.ExposureEndMS < *input.ExposureStartMS {
		return model.RawLearningInteractionEvent{}, validationError("events[%d].exposure_end_ms must be >= exposure_start_ms", index)
	}
	if input.ExposureCount != nil && *input.ExposureCount < 1 {
		return model.RawLearningInteractionEvent{}, validationError("events[%d].exposure_count must be >= 1", index)
	}
	if err := validateNonNegativePointer(input.LookupVisibleMS, fmt.Sprintf("events[%d].lookup_visible_ms", index)); err != nil {
		return model.RawLearningInteractionEvent{}, err
	}
	if input.LookupSentenceAudioReplayCount < 0 {
		return model.RawLearningInteractionEvent{}, validationError("events[%d].lookup_sentence_audio_replay_count must be non-negative", index)
	}
	if input.LookupWordAudioPlayCount < 0 {
		return model.RawLearningInteractionEvent{}, validationError("events[%d].lookup_word_audio_play_count must be non-negative", index)
	}
	eventPayload, err := normalizeJSONObject(input.EventPayload, fmt.Sprintf("events[%d].event_payload", index))
	if err != nil {
		return model.RawLearningInteractionEvent{}, err
	}

	return model.RawLearningInteractionEvent{
		ClientEventID:                  input.ClientEventID,
		UserID:                         request.UserID,
		ClientContext:                  clientContext,
		EventType:                      input.EventType,
		SourceSurface:                  input.SourceSurface,
		VideoID:                        request.VideoID,
		WatchSessionID:                 request.WatchSessionID,
		RecommendationRunID:            request.RecommendationRunID,
		CoarseUnitID:                   input.CoarseUnitID,
		TokenText:                      input.TokenText,
		SentenceIndex:                  input.SentenceIndex,
		SpanIndex:                      input.SpanIndex,
		OccurredAt:                     input.OccurredAt.UTC(),
		ExposureStartMS:                input.ExposureStartMS,
		ExposureEndMS:                  input.ExposureEndMS,
		ExposureCount:                  input.ExposureCount,
		LookupVisibleMS:                input.LookupVisibleMS,
		LookupSentenceAudioReplayCount: input.LookupSentenceAudioReplayCount,
		LookupWordAudioPlayCount:       input.LookupWordAudioPlayCount,
		LookupPracticeNowClicked:       input.LookupPracticeNowClicked,
		EventPayload:                   eventPayload,
	}, nil
}

func learningInteractionEventRequiresSubtitleIndexes(eventType string) bool {
	switch eventType {
	case "exposure", "lookup":
		return true
	default:
		return false
	}
}
