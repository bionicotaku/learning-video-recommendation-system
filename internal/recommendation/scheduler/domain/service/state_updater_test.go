package service

import (
	"math"
	"testing"
	"time"

	"learning-video-recommendation-system/internal/recommendation/scheduler/domain/enum"
	"learning-video-recommendation-system/internal/recommendation/scheduler/domain/model"
	"learning-video-recommendation-system/internal/recommendation/scheduler/domain/policy"

	"github.com/google/uuid"
)

func TestStateUpdaterWeakEventsDoNotAdvanceScheduling(t *testing.T) {
	updater := NewStateUpdater()
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

			next, result, err := updater.Apply(current, model.LearningEvent{
				UserID:       userID,
				CoarseUnitID: 1,
				EventType:    eventType,
				OccurredAt:   now,
			}, UpdateContext{SchedulerPolicy: policy.DefaultSchedulerPolicy(), Now: now})
			if err != nil {
				t.Fatalf("Apply() error = %v", err)
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
			if result.StatusChanged {
				t.Fatal("StatusChanged = true, want false")
			}
		})
	}
}

func TestStateUpdaterFirstStrongEventMovesNewToLearning(t *testing.T) {
	updater := NewStateUpdater()
	now := time.Date(2026, 4, 6, 10, 0, 0, 0, time.UTC)
	quality := 4
	correct := true

	next, result, err := updater.Apply(&model.UserUnitState{
		UserID:       uuid.New(),
		CoarseUnitID: 1,
		Status:       enum.UnitStatusNew,
		EaseFactor:   2.5,
	}, model.LearningEvent{
		EventType:  enum.EventTypeNewLearn,
		IsCorrect:  &correct,
		Quality:    &quality,
		OccurredAt: now,
	}, UpdateContext{SchedulerPolicy: policy.DefaultSchedulerPolicy(), Now: now})
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	if next.Status != enum.UnitStatusLearning {
		t.Fatalf("Status = %q, want %q", next.Status, enum.UnitStatusLearning)
	}
	if !result.StatusChanged {
		t.Fatal("StatusChanged = false, want true")
	}
}

func TestStateUpdaterTwoPassingStrongEventsMoveLearningToReviewing(t *testing.T) {
	updater := NewStateUpdater()
	now := time.Date(2026, 4, 6, 10, 0, 0, 0, time.UTC)
	quality := 4
	correct := true

	next, _, err := updater.Apply(&model.UserUnitState{
		UserID:             uuid.New(),
		CoarseUnitID:       1,
		Status:             enum.UnitStatusLearning,
		StrongEventCount:   1,
		CorrectCount:       1,
		ConsecutiveCorrect: 1,
		EaseFactor:         2.5,
		IntervalDays:       1,
	}, model.LearningEvent{
		EventType:  enum.EventTypeReview,
		IsCorrect:  &correct,
		Quality:    &quality,
		OccurredAt: now,
	}, UpdateContext{
		SchedulerPolicy:  policy.DefaultSchedulerPolicy(),
		RecentQualities:  []int{4},
		RecentCorrectness: []bool{true},
		Now:              now,
	})
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	if next.Status != enum.UnitStatusReviewing {
		t.Fatalf("Status = %q, want %q", next.Status, enum.UnitStatusReviewing)
	}
}

func TestStateUpdaterStrongSuccessAppliesSM2AndScores(t *testing.T) {
	updater := NewStateUpdater()
	now := time.Date(2026, 4, 6, 10, 0, 0, 0, time.UTC)
	quality := 5
	correct := true

	next, result, err := updater.Apply(&model.UserUnitState{
		UserID:          uuid.New(),
		CoarseUnitID:    1,
		Status:          enum.UnitStatusReviewing,
		StrongEventCount: 3,
		CorrectCount:    3,
		EaseFactor:      2.5,
		Repetition:      3,
		IntervalDays:    6,
		ProgressPercent: 10,
		MasteryScore:    0.1,
	}, model.LearningEvent{
		EventType:  enum.EventTypeReview,
		IsCorrect:  &correct,
		Quality:    &quality,
		OccurredAt: now,
	}, UpdateContext{
		SchedulerPolicy:  policy.DefaultSchedulerPolicy(),
		RecentQualities:  []int{4, 4},
		RecentCorrectness: []bool{true, true, true, true},
		Now:              now,
	})
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
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
	if !result.ProgressChanged {
		t.Fatal("ProgressChanged = false, want true")
	}
	if !result.MasteryScoreChanged {
		t.Fatal("MasteryScoreChanged = false, want true")
	}
	if !result.NextReviewChanged {
		t.Fatal("NextReviewChanged = false, want true")
	}
}

func TestStateUpdaterFailureResetsIntervalWithoutChangingEF(t *testing.T) {
	updater := NewStateUpdater()
	now := time.Date(2026, 4, 6, 10, 0, 0, 0, time.UTC)
	quality := 2
	wrong := false

	next, _, err := updater.Apply(&model.UserUnitState{
		UserID:          uuid.New(),
		CoarseUnitID:    1,
		Status:          enum.UnitStatusReviewing,
		EaseFactor:      2.1,
		Repetition:      3,
		IntervalDays:    6,
		ConsecutiveCorrect: 2,
	}, model.LearningEvent{
		EventType:  enum.EventTypeReview,
		IsCorrect:  &wrong,
		Quality:    &quality,
		OccurredAt: now,
	}, UpdateContext{
		SchedulerPolicy:  policy.DefaultSchedulerPolicy(),
		RecentQualities:  []int{4, 4},
		RecentCorrectness: []bool{true, true, false},
		Now:              now,
	})
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
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

func TestStateUpdaterStableLongIntervalMovesToMastered(t *testing.T) {
	updater := NewStateUpdater()
	now := time.Date(2026, 4, 6, 10, 0, 0, 0, time.UTC)
	quality := 5
	correct := true
	schedulerPolicy := policy.DefaultSchedulerPolicy()

	next, result, err := updater.Apply(&model.UserUnitState{
		UserID:             uuid.New(),
		CoarseUnitID:       1,
		Status:             enum.UnitStatusReviewing,
		StrongEventCount:   4,
		CorrectCount:       4,
		ConsecutiveCorrect: 4,
		ConsecutiveWrong:   0,
		EaseFactor:         3.6,
		Repetition:         3,
		IntervalDays:       6,
	}, model.LearningEvent{
		EventType:  enum.EventTypeReview,
		IsCorrect:  &correct,
		Quality:    &quality,
		OccurredAt: now,
	}, UpdateContext{
		SchedulerPolicy:  schedulerPolicy,
		RecentQualities:  []int{4, 5},
		RecentCorrectness: []bool{true, true, true, true},
		Now:              now,
	})
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	if next.Status != enum.UnitStatusMastered {
		t.Fatalf("Status = %q, want %q", next.Status, enum.UnitStatusMastered)
	}
	if !result.StatusChanged {
		t.Fatal("StatusChanged = false, want true")
	}
}

func TestStateUpdaterMasteredFailureDropsBackToReviewing(t *testing.T) {
	updater := NewStateUpdater()
	now := time.Date(2026, 4, 6, 10, 0, 0, 0, time.UTC)
	quality := 2
	wrong := false

	next, result, err := updater.Apply(&model.UserUnitState{
		UserID:          uuid.New(),
		CoarseUnitID:    1,
		Status:          enum.UnitStatusMastered,
		EaseFactor:      2.2,
		Repetition:      5,
		IntervalDays:    21,
		ConsecutiveWrong: 0,
	}, model.LearningEvent{
		EventType:  enum.EventTypeReview,
		IsCorrect:  &wrong,
		Quality:    &quality,
		OccurredAt: now,
	}, UpdateContext{
		SchedulerPolicy:  policy.DefaultSchedulerPolicy(),
		RecentQualities:  []int{5, 5},
		RecentCorrectness: []bool{true, true, false},
		Now:              now,
	})
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	if next.Status != enum.UnitStatusReviewing {
		t.Fatalf("Status = %q, want %q", next.Status, enum.UnitStatusReviewing)
	}
	if !result.StatusChanged {
		t.Fatal("StatusChanged = false, want true")
	}
}
