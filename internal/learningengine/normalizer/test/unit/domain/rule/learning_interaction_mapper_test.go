package rule_test

import (
	"testing"
	"time"

	learningenum "learning-video-recommendation-system/internal/learningengine/domain/enum"
	"learning-video-recommendation-system/internal/learningengine/normalizer/domain/model"
	"learning-video-recommendation-system/internal/learningengine/normalizer/domain/rule"
)

func TestMapLearningInteractionMapsSupportedEvents(t *testing.T) {
	tests := []struct {
		eventType     string
		reducerEffect string
	}{
		{eventType: learningenum.EventExposure, reducerEffect: learningenum.ReducerEffectObserveOnly},
		{eventType: learningenum.EventLookup, reducerEffect: learningenum.ReducerEffectObserveOnly},
		{eventType: learningenum.EventSelfMarkMastered, reducerEffect: learningenum.ReducerEffectSetMastered},
	}

	for _, tt := range tests {
		t.Run(tt.eventType, func(t *testing.T) {
			raw := validLearningInteraction(tt.eventType)
			raw.LookupVisibleMS = int32Ptr(9000)
			raw.LookupWordAudioPlayCount = 1
			raw.LookupSentenceAudioReplayCount = 2
			raw.LookupPracticeNowClicked = true

			result, err := rule.MapLearningInteraction(raw)
			if err != nil {
				t.Fatalf("MapLearningInteraction() error = %v", err)
			}
			event := result.Event
			if event == nil {
				t.Fatal("Event = nil, want normalized event")
			}
			if event.EventType != tt.eventType {
				t.Fatalf("EventType = %q, want %q", event.EventType, tt.eventType)
			}
			if event.ReducerEffect != tt.reducerEffect {
				t.Fatalf("ReducerEffect = %q, want %q", event.ReducerEffect, tt.reducerEffect)
			}
			if event.ProgressQuality != nil {
				t.Fatalf("ProgressQuality = %v, want nil", event.ProgressQuality)
			}
			if event.SourceType != rule.SourceTypeLearningInteractionEvent || event.SourceRefID != raw.EventID {
				t.Fatalf("source = %s/%s, want %s/%s", event.SourceType, event.SourceRefID, rule.SourceTypeLearningInteractionEvent, raw.EventID)
			}
		})
	}
}

func TestMapLearningInteractionSkipsValidationFailures(t *testing.T) {
	raw := validLearningInteraction(learningenum.EventLookup)
	raw.CoarseUnitID = 0

	result, err := rule.MapLearningInteraction(raw)
	if err != nil {
		t.Fatalf("MapLearningInteraction() error = %v", err)
	}
	if !result.Skipped {
		t.Fatal("Skipped = false, want true")
	}
}

func TestMapLearningInteractionSkipsInvalidEventPayload(t *testing.T) {
	raw := validLearningInteraction(learningenum.EventLookup)
	raw.EventPayload = []byte(`[]`)

	result, err := rule.MapLearningInteraction(raw)
	if err != nil {
		t.Fatalf("MapLearningInteraction() error = %v", err)
	}
	if !result.Skipped {
		t.Fatal("Skipped = false, want true")
	}
}

func validLearningInteraction(eventType string) model.RawLearningInteraction {
	return model.RawLearningInteraction{
		EventID:       "11111111-1111-1111-1111-111111111111",
		UserID:        "22222222-2222-2222-2222-222222222222",
		EventType:     eventType,
		SourceSurface: "video_subtitle",
		VideoID:       "33333333-3333-3333-3333-333333333333",
		CoarseUnitID:  101,
		TokenText:     "example",
		OccurredAt:    time.Date(2026, 5, 15, 10, 0, 0, 0, time.UTC),
		EventPayload:  []byte(`{"client":"test"}`),
	}
}

func int32Ptr(value int32) *int32 {
	return &value
}
