package learningevents

import (
	"encoding/json"
	"net/http"

	apvdto "learning-video-recommendation-system/internal/api/application/dto"
	apiservice "learning-video-recommendation-system/internal/api/application/service"
	"learning-video-recommendation-system/internal/api/infrastructure/http/request"
	"learning-video-recommendation-system/internal/api/infrastructure/http/response"
)

type learningInteractionsBatchBody struct {
	ClientContext       json.RawMessage                `json:"client_context"`
	VideoID             string                         `json:"video_id"`
	WatchSessionID      string                         `json:"watch_session_id"`
	RecommendationRunID string                         `json:"recommendation_run_id"`
	Events              []learningInteractionEventBody `json:"events"`
}

type learningInteractionEventBody struct {
	ClientEventID string `json:"client_event_id"`

	EventType     string `json:"event_type"`
	SourceSurface string `json:"source_surface"`
	CoarseUnitID  *int64 `json:"coarse_unit_id"`
	TokenText     string `json:"token_text"`
	SentenceIndex *int32 `json:"sentence_index"`
	SpanIndex     *int32 `json:"span_index"`
	OccurredAt    string `json:"occurred_at"`

	ExposureStartMS *int32 `json:"exposure_start_ms"`
	ExposureEndMS   *int32 `json:"exposure_end_ms"`
	ExposureCount   *int32 `json:"exposure_count"`

	LookupVisibleMS                *int32 `json:"lookup_visible_ms"`
	LookupSentenceAudioReplayCount int32  `json:"lookup_sentence_audio_replay_count"`
	LookupWordAudioPlayCount       int32  `json:"lookup_word_audio_play_count"`
	LookupPracticeNowClicked       bool   `json:"lookup_practice_now_clicked"`

	EventPayload json.RawMessage `json:"event_payload"`
}

func (h *Handler) recordLearningInteractionsBatch(w http.ResponseWriter, r *http.Request) {
	principal, err := requiredPrincipal(r)
	if err != nil {
		writeHandlerError(w, r, err)
		return
	}
	if err := request.RequireJSONContentType(r); err != nil {
		writeHandlerError(w, r, invalidRequest(err))
		return
	}

	var body learningInteractionsBatchBody
	if err := request.DecodeJSONObject(r.Body, &body); err != nil {
		writeHandlerError(w, r, invalidRequest(err))
		return
	}

	command, err := mapLearningInteractionsBatchBody(principal.UserID, body)
	if err != nil {
		writeHandlerError(w, r, err)
		return
	}

	result, err := h.learningInteractions.Execute(r.Context(), command)
	if err != nil {
		writeHandlerError(w, r, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, result)
}

func mapLearningInteractionsBatchBody(userID string, body learningInteractionsBatchBody) (apvdto.RecordLearningInteractionsBatchRequest, error) {
	if len(body.Events) == 0 {
		return apvdto.RecordLearningInteractionsBatchRequest{}, apiservice.InvalidRequestError("events must not be empty")
	}
	if len(body.ClientContext) == 0 {
		body.ClientContext = json.RawMessage(`{}`)
	}
	if err := request.ValidateJSONObject("client_context", body.ClientContext); err != nil {
		return apvdto.RecordLearningInteractionsBatchRequest{}, invalidRequest(err)
	}
	if err := request.ValidateRequiredUUID("video_id", body.VideoID); err != nil {
		return apvdto.RecordLearningInteractionsBatchRequest{}, invalidRequest(err)
	}
	if err := request.ValidateRequiredUUID("watch_session_id", body.WatchSessionID); err != nil {
		return apvdto.RecordLearningInteractionsBatchRequest{}, invalidRequest(err)
	}
	if err := request.ValidateOptionalUUID("recommendation_run_id", body.RecommendationRunID); err != nil {
		return apvdto.RecordLearningInteractionsBatchRequest{}, invalidRequest(err)
	}

	events := make([]apvdto.LearningInteractionEvent, 0, len(body.Events))
	for index, input := range body.Events {
		event, err := mapLearningInteractionEventBody(index, input)
		if err != nil {
			return apvdto.RecordLearningInteractionsBatchRequest{}, err
		}
		events = append(events, event)
	}

	return apvdto.RecordLearningInteractionsBatchRequest{
		UserID:              userID,
		ClientContext:       body.ClientContext,
		VideoID:             body.VideoID,
		WatchSessionID:      body.WatchSessionID,
		RecommendationRunID: body.RecommendationRunID,
		Events:              events,
	}, nil
}

func mapLearningInteractionEventBody(index int, input learningInteractionEventBody) (apvdto.LearningInteractionEvent, error) {
	prefix := func(field string) string {
		return "events[" + itoa(index) + "]." + field
	}
	if input.ClientEventID == "" {
		return apvdto.LearningInteractionEvent{}, apiservice.InvalidRequestError(prefix("client_event_id") + " is required")
	}
	if input.EventType == "" {
		return apvdto.LearningInteractionEvent{}, apiservice.InvalidRequestError(prefix("event_type") + " is required")
	}
	switch input.EventType {
	case "exposure", "lookup":
	case "self_mark_mastered":
		return apvdto.LearningInteractionEvent{}, apiservice.InvalidRequestError(prefix("event_type") + " must use /api/learning-units:mark-mastered")
	default:
		return apvdto.LearningInteractionEvent{}, apiservice.InvalidRequestError(prefix("event_type") + " is unsupported")
	}
	if input.SourceSurface == "" {
		return apvdto.LearningInteractionEvent{}, apiservice.InvalidRequestError(prefix("source_surface") + " is required")
	}
	if input.EventType == "lookup" && input.TokenText == "" {
		return apvdto.LearningInteractionEvent{}, apiservice.InvalidRequestError(prefix("token_text") + " is required for lookup")
	}
	if input.EventType == "exposure" && (input.CoarseUnitID == nil || *input.CoarseUnitID <= 0) {
		return apvdto.LearningInteractionEvent{}, apiservice.InvalidRequestError(prefix("coarse_unit_id") + " is required for " + input.EventType)
	}
	if input.EventType == "lookup" && input.CoarseUnitID != nil && *input.CoarseUnitID <= 0 {
		return apvdto.LearningInteractionEvent{}, apiservice.InvalidRequestError(prefix("coarse_unit_id") + " must be positive when provided")
	}
	if learningInteractionEventRequiresSubtitleIndexes(input.EventType) {
		if input.SentenceIndex == nil {
			return apvdto.LearningInteractionEvent{}, apiservice.InvalidRequestError(prefix("sentence_index") + " is required for " + input.EventType)
		}
		if input.SpanIndex == nil {
			return apvdto.LearningInteractionEvent{}, apiservice.InvalidRequestError(prefix("span_index") + " is required for " + input.EventType)
		}
	}
	occurredAt, err := request.ParseRequiredTime(prefix("occurred_at"), input.OccurredAt)
	if err != nil {
		return apvdto.LearningInteractionEvent{}, invalidRequest(err)
	}
	if err := request.ValidateNonNegativeInt32(prefix("exposure_start_ms"), input.ExposureStartMS); err != nil {
		return apvdto.LearningInteractionEvent{}, invalidRequest(err)
	}
	if err := request.ValidateNonNegativeInt32(prefix("exposure_end_ms"), input.ExposureEndMS); err != nil {
		return apvdto.LearningInteractionEvent{}, invalidRequest(err)
	}
	if input.ExposureStartMS != nil && input.ExposureEndMS != nil && *input.ExposureEndMS < *input.ExposureStartMS {
		return apvdto.LearningInteractionEvent{}, apiservice.InvalidRequestError(prefix("exposure_end_ms") + " must be >= exposure_start_ms")
	}
	if input.ExposureCount != nil && *input.ExposureCount < 1 {
		return apvdto.LearningInteractionEvent{}, apiservice.InvalidRequestError(prefix("exposure_count") + " must be >= 1")
	}
	if err := request.ValidateNonNegativeInt32(prefix("lookup_visible_ms"), input.LookupVisibleMS); err != nil {
		return apvdto.LearningInteractionEvent{}, invalidRequest(err)
	}
	if input.LookupSentenceAudioReplayCount < 0 {
		return apvdto.LearningInteractionEvent{}, apiservice.InvalidRequestError(prefix("lookup_sentence_audio_replay_count") + " must be non-negative")
	}
	if input.LookupWordAudioPlayCount < 0 {
		return apvdto.LearningInteractionEvent{}, apiservice.InvalidRequestError(prefix("lookup_word_audio_play_count") + " must be non-negative")
	}
	if len(input.EventPayload) == 0 {
		input.EventPayload = json.RawMessage(`{}`)
	}
	if err := request.ValidateJSONObject(prefix("event_payload"), input.EventPayload); err != nil {
		return apvdto.LearningInteractionEvent{}, invalidRequest(err)
	}

	return apvdto.LearningInteractionEvent{
		ClientEventID:                  input.ClientEventID,
		EventType:                      input.EventType,
		SourceSurface:                  input.SourceSurface,
		CoarseUnitID:                   input.CoarseUnitID,
		TokenText:                      input.TokenText,
		SentenceIndex:                  input.SentenceIndex,
		SpanIndex:                      input.SpanIndex,
		OccurredAt:                     occurredAt,
		ExposureStartMS:                input.ExposureStartMS,
		ExposureEndMS:                  input.ExposureEndMS,
		ExposureCount:                  input.ExposureCount,
		LookupVisibleMS:                input.LookupVisibleMS,
		LookupSentenceAudioReplayCount: input.LookupSentenceAudioReplayCount,
		LookupWordAudioPlayCount:       input.LookupWordAudioPlayCount,
		LookupPracticeNowClicked:       input.LookupPracticeNowClicked,
		EventPayload:                   input.EventPayload,
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
