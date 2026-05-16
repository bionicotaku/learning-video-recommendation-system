package policy

import (
	"encoding/json"
	"fmt"

	"learning-video-recommendation-system/internal/learningengine/reducer/domain/enum"
	"learning-video-recommendation-system/internal/learningengine/reducer/domain/model"
)

func ValidateEvent(event model.LearningEvent) error {
	if event.UserID == "" {
		return fmt.Errorf("user_id is required")
	}
	if event.CoarseUnitID == 0 {
		return fmt.Errorf("coarse_unit_id is required")
	}
	if event.SourceType == "" {
		return fmt.Errorf("source_type is required")
	}
	if event.SourceRefID == "" {
		return fmt.Errorf("source_ref_id is required")
	}
	if event.OccurredAt.IsZero() {
		return fmt.Errorf("occurred_at is required")
	}
	if !IsSupportedEventType(event.EventType) {
		return fmt.Errorf("unsupported event_type: %s", event.EventType)
	}
	if !IsSupportedReducerEffect(event.ReducerEffect) {
		return fmt.Errorf("unsupported reducer_effect: %s", event.ReducerEffect)
	}
	if IsAffectsProgressEffect(event.ReducerEffect) && event.ProgressQuality == nil {
		return fmt.Errorf("progress_quality is required for affects_progress events")
	}
	if IsObserveOnlyEffect(event.ReducerEffect) && event.ProgressQuality != nil {
		return fmt.Errorf("progress_quality must be empty for observe_only events")
	}
	if IsSetMasteredEffect(event.ReducerEffect) {
		if event.EventType != enum.EventSelfMarkMastered {
			return fmt.Errorf("set_mastered reducer_effect requires self_mark_mastered event_type")
		}
		if event.ProgressQuality != nil {
			return fmt.Errorf("progress_quality must be empty for set_mastered events")
		}
	}
	if event.ProgressQuality != nil && (*event.ProgressQuality < 0 || *event.ProgressQuality > 5) {
		return fmt.Errorf("progress_quality must be between 0 and 5")
	}
	if len(event.Metadata) > 0 && !isJSONObject(event.Metadata) {
		return fmt.Errorf("metadata must be a json object")
	}

	return nil
}

func IsSupportedEventType(eventType string) bool {
	switch eventType {
	case enum.EventExposure, enum.EventLookup, enum.EventQuiz, enum.EventSelfMarkMastered:
		return true
	default:
		return false
	}
}

func IsSupportedReducerEffect(reducerEffect string) bool {
	switch reducerEffect {
	case enum.ReducerEffectObserveOnly, enum.ReducerEffectAffectsProgress, enum.ReducerEffectSetMastered:
		return true
	default:
		return false
	}
}

func IsObserveOnlyEffect(reducerEffect string) bool {
	return reducerEffect == enum.ReducerEffectObserveOnly
}

func IsAffectsProgressEffect(reducerEffect string) bool {
	return reducerEffect == enum.ReducerEffectAffectsProgress
}

func IsSetMasteredEffect(reducerEffect string) bool {
	return reducerEffect == enum.ReducerEffectSetMastered
}

func IsPassingQuality(quality int16) bool {
	return quality >= 3
}

func isJSONObject(raw []byte) bool {
	var value map[string]any
	if err := json.Unmarshal(raw, &value); err != nil {
		return false
	}
	return value != nil
}
