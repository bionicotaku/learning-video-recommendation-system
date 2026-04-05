package rule_test

import (
	"testing"
	"time"

	"learning-video-recommendation-system/internal/learningengine/domain/enum"
	"learning-video-recommendation-system/internal/learningengine/domain/model"
	rulepkg "learning-video-recommendation-system/internal/learningengine/domain/rule"

	"github.com/google/uuid"
)

func TestStrongEventHandlerUpdatesCoreCounters(t *testing.T) {
	handler := rulepkg.NewStrongEventHandler()
	userID := uuid.New()
	occurredAt := time.Date(2026, 4, 4, 14, 30, 0, 0, time.UTC)
	quality := 4
	correct := true
	wrong := false

	tests := []struct {
		name                   string
		eventType              enum.EventType
		isCorrect              *bool
		quality                *int
		base                   *model.UserUnitState
		wantReviewCount        int
		wantLastReviewed       bool
		wantCorrectCount       int
		wantWrongCount         int
		wantConsecutiveCorrect int
		wantConsecutiveWrong   int
		wantLastQuality        *int
	}{
		{
			name:      "new learn increments strong stats but not review stats",
			eventType: enum.EventTypeNewLearn,
			isCorrect: &correct,
			quality:   &quality,
			base: &model.UserUnitState{
				UserID:             userID,
				CoarseUnitID:       7,
				Status:             enum.UnitStatusNew,
				SeenCount:          1,
				StrongEventCount:   0,
				ReviewCount:        0,
				CorrectCount:       0,
				WrongCount:         0,
				ConsecutiveCorrect: 0,
				ConsecutiveWrong:   0,
			},
			wantReviewCount:        0,
			wantLastReviewed:       false,
			wantCorrectCount:       1,
			wantWrongCount:         0,
			wantConsecutiveCorrect: 1,
			wantConsecutiveWrong:   0,
			wantLastQuality:        &quality,
		},
		{
			name:      "review increments review stats and resets wrong streak",
			eventType: enum.EventTypeReview,
			isCorrect: &correct,
			quality:   &quality,
			base: &model.UserUnitState{
				UserID:             userID,
				CoarseUnitID:       7,
				Status:             enum.UnitStatusLearning,
				SeenCount:          2,
				StrongEventCount:   1,
				ReviewCount:        1,
				CorrectCount:       1,
				WrongCount:         2,
				ConsecutiveCorrect: 0,
				ConsecutiveWrong:   2,
			},
			wantReviewCount:        2,
			wantLastReviewed:       true,
			wantCorrectCount:       2,
			wantWrongCount:         2,
			wantConsecutiveCorrect: 1,
			wantConsecutiveWrong:   0,
			wantLastQuality:        &quality,
		},
		{
			name:      "quiz wrong answer increments wrong stats and keeps nil quality",
			eventType: enum.EventTypeQuiz,
			isCorrect: &wrong,
			quality:   nil,
			base: &model.UserUnitState{
				UserID:             userID,
				CoarseUnitID:       7,
				Status:             enum.UnitStatusReviewing,
				SeenCount:          3,
				StrongEventCount:   2,
				ReviewCount:        1,
				CorrectCount:       3,
				WrongCount:         1,
				ConsecutiveCorrect: 2,
				ConsecutiveWrong:   0,
			},
			wantReviewCount:        2,
			wantLastReviewed:       true,
			wantCorrectCount:       3,
			wantWrongCount:         2,
			wantConsecutiveCorrect: 0,
			wantConsecutiveWrong:   1,
			wantLastQuality:        nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := model.LearningEvent{
				UserID:       userID,
				CoarseUnitID: 7,
				EventType:    tt.eventType,
				IsCorrect:    tt.isCorrect,
				Quality:      tt.quality,
				OccurredAt:   occurredAt,
			}

			got, err := handler.Apply(tt.base, event)
			if err != nil {
				t.Fatalf("Apply() error = %v", err)
			}

			if got.SeenCount != tt.base.SeenCount+1 {
				t.Fatalf("SeenCount = %d, want %d", got.SeenCount, tt.base.SeenCount+1)
			}
			if got.StrongEventCount != tt.base.StrongEventCount+1 {
				t.Fatalf("StrongEventCount = %d, want %d", got.StrongEventCount, tt.base.StrongEventCount+1)
			}
			if got.LastSeenAt == nil || !got.LastSeenAt.Equal(occurredAt) {
				t.Fatalf("LastSeenAt = %v, want %v", got.LastSeenAt, occurredAt)
			}
			if got.ReviewCount != tt.wantReviewCount {
				t.Fatalf("ReviewCount = %d, want %d", got.ReviewCount, tt.wantReviewCount)
			}
			if tt.wantLastReviewed {
				if got.LastReviewedAt == nil || !got.LastReviewedAt.Equal(occurredAt) {
					t.Fatalf("LastReviewedAt = %v, want %v", got.LastReviewedAt, occurredAt)
				}
			} else if got.LastReviewedAt != nil {
				t.Fatalf("LastReviewedAt = %v, want nil", got.LastReviewedAt)
			}
			if got.CorrectCount != tt.wantCorrectCount {
				t.Fatalf("CorrectCount = %d, want %d", got.CorrectCount, tt.wantCorrectCount)
			}
			if got.WrongCount != tt.wantWrongCount {
				t.Fatalf("WrongCount = %d, want %d", got.WrongCount, tt.wantWrongCount)
			}
			if got.ConsecutiveCorrect != tt.wantConsecutiveCorrect {
				t.Fatalf("ConsecutiveCorrect = %d, want %d", got.ConsecutiveCorrect, tt.wantConsecutiveCorrect)
			}
			if got.ConsecutiveWrong != tt.wantConsecutiveWrong {
				t.Fatalf("ConsecutiveWrong = %d, want %d", got.ConsecutiveWrong, tt.wantConsecutiveWrong)
			}
			switch {
			case tt.wantLastQuality == nil && got.LastQuality != nil:
				t.Fatalf("LastQuality = %v, want nil", got.LastQuality)
			case tt.wantLastQuality != nil:
				if got.LastQuality == nil || *got.LastQuality != *tt.wantLastQuality {
					t.Fatalf("LastQuality = %v, want %v", got.LastQuality, *tt.wantLastQuality)
				}
			}
		})
	}
}

func TestStrongEventHandlerRejectsWeakEvents(t *testing.T) {
	handler := rulepkg.NewStrongEventHandler()

	_, err := handler.Apply(nil, model.LearningEvent{EventType: enum.EventTypeExposure})
	if err == nil {
		t.Fatal("Apply() error = nil, want unsupported event error")
	}
}
