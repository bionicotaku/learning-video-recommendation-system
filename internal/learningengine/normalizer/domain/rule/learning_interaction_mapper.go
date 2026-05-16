package rule

import (
	"encoding/json"
	"fmt"

	"learning-video-recommendation-system/internal/learningengine/normalizer/domain/model"
	learningenum "learning-video-recommendation-system/internal/learningengine/reducer/domain/enum"
)

const SourceTypeLearningInteractionEvent = "learning_interaction_event"

func MapLearningInteraction(raw model.RawLearningInteraction) (model.NormalizationResult, error) {
	if raw.EventID == "" {
		return model.Skipped("event_id is required"), nil
	}
	if raw.UserID == "" {
		return model.Skipped("user_id is required"), nil
	}
	if raw.CoarseUnitID == 0 {
		return model.Skipped("coarse_unit_id is required"), nil
	}
	if raw.OccurredAt.IsZero() {
		return model.Skipped("occurred_at is required"), nil
	}

	eventType := ""
	reducerEffect := ""
	switch raw.EventType {
	case learningenum.EventExposure:
		eventType = learningenum.EventExposure
		reducerEffect = learningenum.ReducerEffectObserveOnly
	case learningenum.EventLookup:
		eventType = learningenum.EventLookup
		reducerEffect = learningenum.ReducerEffectObserveOnly
	case learningenum.EventSelfMarkMastered:
		eventType = learningenum.EventSelfMarkMastered
		reducerEffect = learningenum.ReducerEffectSetMastered
	default:
		return model.Skipped("unsupported event_type"), nil
	}

	metadata, err := buildInteractionMetadata(raw)
	if err != nil {
		if err == errInvalidMetadata {
			return model.Skipped("event_payload must be a json object"), nil
		}
		return model.NormalizationResult{}, err
	}

	return model.Normalized(model.NormalizedLearningEvent{
		UserID:        raw.UserID,
		CoarseUnitID:  raw.CoarseUnitID,
		VideoID:       raw.VideoID,
		EventType:     eventType,
		ReducerEffect: reducerEffect,
		SourceType:    SourceTypeLearningInteractionEvent,
		SourceRefID:   raw.EventID,
		Metadata:      metadata,
		OccurredAt:    raw.OccurredAt,
	}), nil
}

func buildInteractionMetadata(raw model.RawLearningInteraction) ([]byte, error) {
	values := map[string]any{
		"source_surface":                     raw.SourceSurface,
		"video_id":                           raw.VideoID,
		"watch_session_id":                   raw.WatchSessionID,
		"recommendation_run_id":              raw.RecommendationRunID,
		"related_quiz_event_id":              raw.RelatedQuizEventID,
		"token_text":                         raw.TokenText,
		"sentence_index":                     raw.SentenceIndex,
		"span_index":                         raw.SpanIndex,
		"exposure_start_ms":                  raw.ExposureStartMS,
		"exposure_end_ms":                    raw.ExposureEndMS,
		"exposure_count":                     raw.ExposureCount,
		"lookup_visible_ms":                  raw.LookupVisibleMS,
		"lookup_sentence_audio_replay_count": raw.LookupSentenceAudioReplayCount,
		"lookup_word_audio_play_count":       raw.LookupWordAudioPlayCount,
		"lookup_practice_now_clicked":        raw.LookupPracticeNowClicked,
	}
	if len(raw.EventPayload) > 0 {
		var payload map[string]any
		if err := json.Unmarshal(raw.EventPayload, &payload); err != nil {
			return nil, errInvalidMetadata
		}
		if payload == nil {
			return nil, errInvalidMetadata
		}
		values["event_payload"] = payload
	}

	metadata, err := marshalMetadata(values)
	if err != nil {
		return nil, fmt.Errorf("build learning interaction metadata: %w", err)
	}
	return metadata, nil
}
