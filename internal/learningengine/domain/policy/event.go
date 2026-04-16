package policy

import (
	"fmt"

	"learning-video-recommendation-system/internal/learningengine/domain/enum"
	"learning-video-recommendation-system/internal/learningengine/domain/model"
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
	if event.OccurredAt.IsZero() {
		return fmt.Errorf("occurred_at is required")
	}
	if !IsSupportedEventType(event.EventType) {
		return fmt.Errorf("unsupported event_type: %s", event.EventType)
	}
	if IsStrongEventType(event.EventType) && event.Quality == nil {
		return fmt.Errorf("quality is required for strong events")
	}
	if event.Quality != nil && (*event.Quality < 0 || *event.Quality > 5) {
		return fmt.Errorf("quality must be between 0 and 5")
	}

	return nil
}

func IsSupportedEventType(eventType string) bool {
	switch eventType {
	case enum.EventExposure, enum.EventLookup, enum.EventNewLearn, enum.EventReview, enum.EventQuiz:
		return true
	default:
		return false
	}
}

func IsWeakEventType(eventType string) bool {
	return eventType == enum.EventExposure || eventType == enum.EventLookup
}

func IsStrongEventType(eventType string) bool {
	return eventType == enum.EventNewLearn || eventType == enum.EventReview || eventType == enum.EventQuiz
}

func IsPassingQuality(quality int16) bool {
	return quality >= 3
}
