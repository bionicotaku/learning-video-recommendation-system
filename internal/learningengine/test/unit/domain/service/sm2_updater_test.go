package service_test

import (
	"math"
	"testing"
	"time"

	"learning-video-recommendation-system/internal/learningengine/domain/model"
	"learning-video-recommendation-system/internal/learningengine/domain/policy"
	servicepkg "learning-video-recommendation-system/internal/learningengine/domain/service"
)

func TestSM2UpdaterSuccessBranches(t *testing.T) {
	updater := servicepkg.NewSM2Updater()
	schedulerPolicy := policy.DefaultLearningPolicy()
	occurredAt := time.Date(2026, 4, 5, 9, 0, 0, 0, time.UTC)

	tests := []struct {
		name             string
		state            model.UserUnitState
		quality          int
		wantRepetition   int
		wantIntervalDays float64
		wantEaseFactor   float64
	}{
		{
			name:             "first success uses 1-day interval",
			state:            model.UserUnitState{Repetition: 0, IntervalDays: 0, EaseFactor: 2.5},
			quality:          5,
			wantRepetition:   1,
			wantIntervalDays: 1,
			wantEaseFactor:   2.6,
		},
		{
			name:             "second success uses 3-day interval",
			state:            model.UserUnitState{Repetition: 1, IntervalDays: 1, EaseFactor: 2.5},
			quality:          4,
			wantRepetition:   2,
			wantIntervalDays: 3,
			wantEaseFactor:   2.5,
		},
		{
			name:             "third success uses 6-day interval",
			state:            model.UserUnitState{Repetition: 2, IntervalDays: 3, EaseFactor: 2.5},
			quality:          3,
			wantRepetition:   3,
			wantIntervalDays: 6,
			wantEaseFactor:   2.36,
		},
		{
			name:             "fourth success multiplies previous interval by ease factor",
			state:            model.UserUnitState{Repetition: 3, IntervalDays: 6, EaseFactor: 2.5},
			quality:          5,
			wantRepetition:   4,
			wantIntervalDays: 15,
			wantEaseFactor:   2.6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := tt.state
			if err := updater.Apply(&state, tt.quality, occurredAt, schedulerPolicy); err != nil {
				t.Fatalf("Apply() error = %v", err)
			}

			if state.Repetition != tt.wantRepetition {
				t.Fatalf("Repetition = %d, want %d", state.Repetition, tt.wantRepetition)
			}
			if state.IntervalDays != tt.wantIntervalDays {
				t.Fatalf("IntervalDays = %v, want %v", state.IntervalDays, tt.wantIntervalDays)
			}
			if math.Abs(state.EaseFactor-tt.wantEaseFactor) > 1e-9 {
				t.Fatalf("EaseFactor = %v, want %v", state.EaseFactor, tt.wantEaseFactor)
			}
			if state.NextReviewAt == nil {
				t.Fatal("NextReviewAt = nil, want value")
			}
			wantNext := occurredAt.Add(time.Duration(tt.wantIntervalDays * float64(24*time.Hour)))
			if !state.NextReviewAt.Equal(wantNext) {
				t.Fatalf("NextReviewAt = %v, want %v", state.NextReviewAt, wantNext)
			}
		})
	}
}

func TestSM2UpdaterFailureBranchResetsIntervalWithoutChangingEaseFactor(t *testing.T) {
	updater := servicepkg.NewSM2Updater()
	schedulerPolicy := policy.DefaultLearningPolicy()
	occurredAt := time.Date(2026, 4, 5, 9, 0, 0, 0, time.UTC)
	state := model.UserUnitState{
		Repetition:   3,
		IntervalDays: 6,
		EaseFactor:   2.1,
	}

	if err := updater.Apply(&state, 2, occurredAt, schedulerPolicy); err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	if state.Repetition != 0 {
		t.Fatalf("Repetition = %d, want 0", state.Repetition)
	}
	if state.IntervalDays != 1 {
		t.Fatalf("IntervalDays = %v, want 1", state.IntervalDays)
	}
	if state.EaseFactor != 2.1 {
		t.Fatalf("EaseFactor = %v, want 2.1", state.EaseFactor)
	}
	if state.NextReviewAt == nil {
		t.Fatal("NextReviewAt = nil, want value")
	}
	wantNext := occurredAt.Add(24 * time.Hour)
	if !state.NextReviewAt.Equal(wantNext) {
		t.Fatalf("NextReviewAt = %v, want %v", state.NextReviewAt, wantNext)
	}
}

func TestSM2UpdaterAppliesMinEaseFactorFloor(t *testing.T) {
	updater := servicepkg.NewSM2Updater()
	schedulerPolicy := policy.DefaultLearningPolicy()
	occurredAt := time.Date(2026, 4, 5, 9, 0, 0, 0, time.UTC)
	state := model.UserUnitState{
		Repetition:   0,
		IntervalDays: 0,
		EaseFactor:   1.35,
	}

	if err := updater.Apply(&state, 3, occurredAt, schedulerPolicy); err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	if state.EaseFactor != schedulerPolicy.MinEaseFactor {
		t.Fatalf("EaseFactor = %v, want %v", state.EaseFactor, schedulerPolicy.MinEaseFactor)
	}
}
