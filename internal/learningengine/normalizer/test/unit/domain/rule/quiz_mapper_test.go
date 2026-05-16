package rule_test

import (
	"testing"
	"time"

	"learning-video-recommendation-system/internal/learningengine/normalizer/domain/model"
	"learning-video-recommendation-system/internal/learningengine/normalizer/domain/rule"
	learningenum "learning-video-recommendation-system/internal/learningengine/reducer/domain/enum"
)

func TestMapQuizEventQualityUsesFiveSecondBoundary(t *testing.T) {
	tests := []struct {
		name      string
		correct   bool
		elapsedMS int32
		want      int16
	}{
		{name: "correct fast at threshold", correct: true, elapsedMS: 5000, want: 5},
		{name: "correct slow over threshold", correct: true, elapsedMS: 5001, want: 4},
		{name: "wrong fast at threshold", correct: false, elapsedMS: 5000, want: 2},
		{name: "wrong slow over threshold", correct: false, elapsedMS: 5001, want: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw := validQuizEvent()
			raw.IsFirstTryCorrect = tt.correct
			raw.TotalElapsedMS = tt.elapsedMS
			if tt.correct {
				raw.SelectedOptionIDs = []string{"correct"}
			} else {
				raw.SelectedOptionIDs = []string{"wrong_1", "wrong_2", "correct"}
			}

			result, err := rule.MapQuizEvent(raw)
			if err != nil {
				t.Fatalf("MapQuizEvent() error = %v", err)
			}
			if result.Event == nil || result.Event.ProgressQuality == nil {
				t.Fatalf("ProgressQuality = nil, want %d", tt.want)
			}
			if *result.Event.ProgressQuality != tt.want {
				t.Fatalf("ProgressQuality = %d, want %d", *result.Event.ProgressQuality, tt.want)
			}
		})
	}
}

func TestMapQuizEventWrongAttemptCountDoesNotChangeQuality(t *testing.T) {
	raw := validQuizEvent()
	raw.IsFirstTryCorrect = false
	raw.TotalElapsedMS = 4000
	raw.SelectedOptionIDs = []string{"wrong_1", "wrong_2", "wrong_3", "correct"}

	result, err := rule.MapQuizEvent(raw)
	if err != nil {
		t.Fatalf("MapQuizEvent() error = %v", err)
	}
	if got := *result.Event.ProgressQuality; got != 2 {
		t.Fatalf("ProgressQuality = %d, want 2", got)
	}
}

func TestMapQuizEventMapsLearningEventShape(t *testing.T) {
	raw := validQuizEvent()

	result, err := rule.MapQuizEvent(raw)
	if err != nil {
		t.Fatalf("MapQuizEvent() error = %v", err)
	}
	event := result.Event
	if event == nil {
		t.Fatal("Event = nil, want normalized event")
	}
	if event.EventType != learningenum.EventQuiz {
		t.Fatalf("EventType = %q, want %q", event.EventType, learningenum.EventQuiz)
	}
	if event.ReducerEffect != learningenum.ReducerEffectAffectsProgress {
		t.Fatalf("ReducerEffect = %q, want %q", event.ReducerEffect, learningenum.ReducerEffectAffectsProgress)
	}
	if event.SourceType != rule.SourceTypeQuizEvent || event.SourceRefID != raw.EventID {
		t.Fatalf("source = %s/%s, want %s/%s", event.SourceType, event.SourceRefID, rule.SourceTypeQuizEvent, raw.EventID)
	}
	if event.IsCorrect == nil || !*event.IsCorrect {
		t.Fatalf("IsCorrect = %v, want true", event.IsCorrect)
	}
}

func TestMapQuizEventSkipsValidationFailures(t *testing.T) {
	raw := validQuizEvent()
	raw.TotalElapsedMS = -1

	result, err := rule.MapQuizEvent(raw)
	if err != nil {
		t.Fatalf("MapQuizEvent() error = %v", err)
	}
	if !result.Skipped {
		t.Fatal("Skipped = false, want true")
	}
}

func validQuizEvent() model.RawQuizEvent {
	now := time.Date(2026, 5, 15, 10, 0, 0, 0, time.UTC)
	return model.RawQuizEvent{
		EventID:             "11111111-1111-1111-1111-111111111111",
		UserID:              "22222222-2222-2222-2222-222222222222",
		QuestionID:          "33333333-3333-3333-3333-333333333333",
		CoarseUnitID:        101,
		VideoID:             "44444444-4444-4444-4444-444444444444",
		RecommendationRunID: "55555555-5555-5555-5555-555555555555",
		TriggerType:         "lookup_practice",
		SelectedOptionIDs:   []string{"correct"},
		SelectionIntervalMS: []int32{1200},
		IsFirstTryCorrect:   true,
		TotalElapsedMS:      5000,
		ShownAt:             now.Add(-5 * time.Second),
		CompletedAt:         now,
	}
}
