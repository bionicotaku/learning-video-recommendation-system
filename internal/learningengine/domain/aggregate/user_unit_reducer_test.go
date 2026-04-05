package aggregate

import (
	"math"
	"testing"
	"time"

	"learning-video-recommendation-system/internal/learningengine/domain/enum"
	"learning-video-recommendation-system/internal/learningengine/domain/model"
	"learning-video-recommendation-system/internal/learningengine/domain/policy"

	"github.com/google/uuid"
)

func TestUserUnitReducerWeakEventsDoNotAdvanceScheduling(t *testing.T) {
	reducer := NewUserUnitReducer()
	userID := uuid.New()
	now := time.Date(2026, 4, 6, 10, 0, 0, 0, time.UTC)
	nextReviewAt := now.Add(24 * time.Hour)

	for _, eventType := range []enum.EventType{enum.EventTypeExposure, enum.EventTypeLookup} {
		t.Run(string(eventType), func(t *testing.T) {
			current := &model.UserUnitState{
				UserID:          userID,
				CoarseUnitID:    1,
				Status:          enum.UnitStatusNew,
				SeenCount:       2,
				IntervalDays:    6,
				EaseFactor:      2.5,
				NextReviewAt:    &nextReviewAt,
				ProgressPercent: 12,
				MasteryScore:    0.22,
			}

			next, err := reducer.Reduce(current, model.LearningEvent{
				UserID:       userID,
				CoarseUnitID: 1,
				EventType:    eventType,
				OccurredAt:   now,
				CreatedAt:    now,
			}, policy.DefaultLearningPolicy())
			if err != nil {
				t.Fatalf("Reduce() error = %v", err)
			}

			if next.IntervalDays != current.IntervalDays {
				t.Fatalf("IntervalDays = %v, want %v", next.IntervalDays, current.IntervalDays)
			}
			if next.Status != current.Status {
				t.Fatalf("Status = %q, want %q", next.Status, current.Status)
			}
			if next.EaseFactor != current.EaseFactor {
				t.Fatalf("EaseFactor = %v, want %v", next.EaseFactor, current.EaseFactor)
			}
			if !sameTimePtr(next.NextReviewAt, current.NextReviewAt) {
				t.Fatalf("NextReviewAt = %v, want %v", next.NextReviewAt, current.NextReviewAt)
			}
		})
	}
}

func TestUserUnitReducerFirstStrongEventMovesNewToLearning(t *testing.T) {
	reducer := NewUserUnitReducer()
	now := time.Date(2026, 4, 6, 10, 0, 0, 0, time.UTC)
	quality := 4
	correct := true

	next, err := reducer.Reduce(&model.UserUnitState{
		UserID:       uuid.New(),
		CoarseUnitID: 1,
		Status:       enum.UnitStatusNew,
		EaseFactor:   2.5,
	}, model.LearningEvent{
		EventType:  enum.EventTypeNewLearn,
		IsCorrect:  &correct,
		Quality:    &quality,
		OccurredAt: now,
		CreatedAt:  now,
	}, policy.DefaultLearningPolicy())
	if err != nil {
		t.Fatalf("Reduce() error = %v", err)
	}

	if next.Status != enum.UnitStatusLearning {
		t.Fatalf("Status = %q, want %q", next.Status, enum.UnitStatusLearning)
	}
	if len(next.RecentQualityWindow) != 1 || next.RecentQualityWindow[0] != 4 {
		t.Fatalf("RecentQualityWindow = %v, want [4]", next.RecentQualityWindow)
	}
	if len(next.RecentCorrectnessWindow) != 1 || !next.RecentCorrectnessWindow[0] {
		t.Fatalf("RecentCorrectnessWindow = %v, want [true]", next.RecentCorrectnessWindow)
	}
}

func TestUserUnitReducerTwoPassingStrongEventsMoveLearningToReviewing(t *testing.T) {
	reducer := NewUserUnitReducer()
	now := time.Date(2026, 4, 6, 10, 0, 0, 0, time.UTC)
	quality := 4
	correct := true

	next, err := reducer.Reduce(&model.UserUnitState{
		UserID:                  uuid.New(),
		CoarseUnitID:            1,
		Status:                  enum.UnitStatusLearning,
		StrongEventCount:        1,
		CorrectCount:            1,
		ConsecutiveCorrect:      1,
		EaseFactor:              2.5,
		IntervalDays:            1,
		RecentQualityWindow:     []int{4},
		RecentCorrectnessWindow: []bool{true},
	}, model.LearningEvent{
		EventType:  enum.EventTypeReview,
		IsCorrect:  &correct,
		Quality:    &quality,
		OccurredAt: now,
		CreatedAt:  now,
	}, policy.DefaultLearningPolicy())
	if err != nil {
		t.Fatalf("Reduce() error = %v", err)
	}

	if next.Status != enum.UnitStatusReviewing {
		t.Fatalf("Status = %q, want %q", next.Status, enum.UnitStatusReviewing)
	}
}

func TestUserUnitReducerStrongSuccessAppliesSM2AndScores(t *testing.T) {
	reducer := NewUserUnitReducer()
	now := time.Date(2026, 4, 6, 10, 0, 0, 0, time.UTC)
	quality := 5
	correct := true

	next, err := reducer.Reduce(&model.UserUnitState{
		UserID:                  uuid.New(),
		CoarseUnitID:            1,
		Status:                  enum.UnitStatusReviewing,
		StrongEventCount:        3,
		CorrectCount:            3,
		EaseFactor:              2.5,
		Repetition:              3,
		IntervalDays:            6,
		ProgressPercent:         10,
		MasteryScore:            0.1,
		RecentQualityWindow:     []int{4, 4},
		RecentCorrectnessWindow: []bool{true, true, true, true},
	}, model.LearningEvent{
		EventType:  enum.EventTypeReview,
		IsCorrect:  &correct,
		Quality:    &quality,
		OccurredAt: now,
		CreatedAt:  now,
	}, policy.DefaultLearningPolicy())
	if err != nil {
		t.Fatalf("Reduce() error = %v", err)
	}

	if next.Repetition != 4 {
		t.Fatalf("Repetition = %d, want 4", next.Repetition)
	}
	if next.IntervalDays != 15 {
		t.Fatalf("IntervalDays = %v, want 15", next.IntervalDays)
	}
	if math.Abs(next.EaseFactor-2.6) > 1e-9 {
		t.Fatalf("EaseFactor = %v, want 2.6", next.EaseFactor)
	}
	if next.ProgressPercent <= 10 {
		t.Fatalf("ProgressPercent = %v, want > 10", next.ProgressPercent)
	}
	if next.MasteryScore <= 0.1 {
		t.Fatalf("MasteryScore = %v, want > 0.1", next.MasteryScore)
	}
}

func TestUserUnitReducerFailureResetsIntervalWithoutChangingEF(t *testing.T) {
	reducer := NewUserUnitReducer()
	now := time.Date(2026, 4, 6, 10, 0, 0, 0, time.UTC)
	quality := 2
	wrong := false

	next, err := reducer.Reduce(&model.UserUnitState{
		UserID:                  uuid.New(),
		CoarseUnitID:            1,
		Status:                  enum.UnitStatusReviewing,
		EaseFactor:              2.1,
		Repetition:              3,
		IntervalDays:            6,
		ConsecutiveCorrect:      2,
		RecentQualityWindow:     []int{4, 4},
		RecentCorrectnessWindow: []bool{true, true, false},
	}, model.LearningEvent{
		EventType:  enum.EventTypeReview,
		IsCorrect:  &wrong,
		Quality:    &quality,
		OccurredAt: now,
		CreatedAt:  now,
	}, policy.DefaultLearningPolicy())
	if err != nil {
		t.Fatalf("Reduce() error = %v", err)
	}

	if next.Repetition != 0 {
		t.Fatalf("Repetition = %d, want 0", next.Repetition)
	}
	if next.IntervalDays != 1 {
		t.Fatalf("IntervalDays = %v, want 1", next.IntervalDays)
	}
	if next.EaseFactor != 2.1 {
		t.Fatalf("EaseFactor = %v, want 2.1", next.EaseFactor)
	}
}

func TestUserUnitReducerStableLongIntervalMovesToMastered(t *testing.T) {
	reducer := NewUserUnitReducer()
	now := time.Date(2026, 4, 6, 10, 0, 0, 0, time.UTC)
	quality := 5
	correct := true
	schedulerPolicy := policy.DefaultLearningPolicy()

	next, err := reducer.Reduce(&model.UserUnitState{
		UserID:                  uuid.New(),
		CoarseUnitID:            1,
		Status:                  enum.UnitStatusReviewing,
		StrongEventCount:        4,
		CorrectCount:            4,
		ConsecutiveCorrect:      4,
		ConsecutiveWrong:        0,
		EaseFactor:              3.6,
		Repetition:              3,
		IntervalDays:            6,
		RecentQualityWindow:     []int{4, 5},
		RecentCorrectnessWindow: []bool{true, true, true, true},
	}, model.LearningEvent{
		EventType:  enum.EventTypeReview,
		IsCorrect:  &correct,
		Quality:    &quality,
		OccurredAt: now,
		CreatedAt:  now,
	}, schedulerPolicy)
	if err != nil {
		t.Fatalf("Reduce() error = %v", err)
	}

	if next.Status != enum.UnitStatusMastered {
		t.Fatalf("Status = %q, want %q", next.Status, enum.UnitStatusMastered)
	}
}

func TestUserUnitReducerMasteredFailureDropsBackToReviewing(t *testing.T) {
	reducer := NewUserUnitReducer()
	now := time.Date(2026, 4, 6, 10, 0, 0, 0, time.UTC)
	quality := 2
	wrong := false

	next, err := reducer.Reduce(&model.UserUnitState{
		UserID:                  uuid.New(),
		CoarseUnitID:            1,
		Status:                  enum.UnitStatusMastered,
		EaseFactor:              2.2,
		Repetition:              5,
		IntervalDays:            21,
		ConsecutiveWrong:        0,
		RecentQualityWindow:     []int{5, 5},
		RecentCorrectnessWindow: []bool{true, true, false},
	}, model.LearningEvent{
		EventType:  enum.EventTypeReview,
		IsCorrect:  &wrong,
		Quality:    &quality,
		OccurredAt: now,
		CreatedAt:  now,
	}, policy.DefaultLearningPolicy())
	if err != nil {
		t.Fatalf("Reduce() error = %v", err)
	}

	if next.Status != enum.UnitStatusReviewing {
		t.Fatalf("Status = %q, want %q", next.Status, enum.UnitStatusReviewing)
	}
}

func sameTimePtr(left, right *time.Time) bool {
	switch {
	case left == nil && right == nil:
		return true
	case left == nil || right == nil:
		return false
	default:
		return left.Equal(*right)
	}
}
