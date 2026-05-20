package aggregate_test

import (
	"errors"
	"testing"
	"time"

	"learning-video-recommendation-system/internal/learningengine/reducer/domain/aggregate"
	"learning-video-recommendation-system/internal/learningengine/reducer/domain/enum"
	"learning-video-recommendation-system/internal/learningengine/reducer/domain/model"
)

func TestReduce_ObserveOnlyEventOnlyUpdatesObservationFields(t *testing.T) {
	eventTime := time.Date(2026, 4, 16, 10, 0, 0, 0, time.UTC)

	state, err := aggregate.Reduce(nil, learningEvent(enum.EventExposure, enum.ReducerEffectObserveOnly, nil, eventTime))
	if err != nil {
		t.Fatalf("Reduce() error = %v", err)
	}

	if state.Status != enum.StatusNew {
		t.Fatalf("status = %q, want %q", state.Status, enum.StatusNew)
	}
	if state.ObservationCount != 1 {
		t.Fatalf("observation_count = %d, want 1", state.ObservationCount)
	}
	if state.ProgressEventCount != 0 {
		t.Fatalf("progress_event_count = %d, want 0", state.ProgressEventCount)
	}
	if state.FirstObservedAt == nil || !state.FirstObservedAt.Equal(eventTime) {
		t.Fatalf("first_observed_at = %v, want %v", state.FirstObservedAt, eventTime)
	}
	if state.LastObservedAt == nil || !state.LastObservedAt.Equal(eventTime) {
		t.Fatalf("last_observed_at = %v, want %v", state.LastObservedAt, eventTime)
	}
	if state.NextReviewAt != nil {
		t.Fatalf("next_review_at = %v, want nil", state.NextReviewAt)
	}
}

func TestReduce_ObserveOnlyRejectsProgressQuality(t *testing.T) {
	eventTime := time.Date(2026, 4, 16, 10, 0, 0, 0, time.UTC)
	quality := int16(4)

	_, err := aggregate.Reduce(nil, learningEvent(enum.EventLookup, enum.ReducerEffectObserveOnly, &quality, eventTime))
	if err == nil {
		t.Fatal("Reduce() error = nil, want validation error")
	}
}

func TestReduce_AffectsProgressRequiresProgressQuality(t *testing.T) {
	eventTime := time.Date(2026, 4, 16, 10, 0, 0, 0, time.UTC)

	_, err := aggregate.Reduce(nil, learningEvent(enum.EventQuiz, enum.ReducerEffectAffectsProgress, nil, eventTime))
	if err == nil {
		t.Fatal("Reduce() error = nil, want validation error")
	}
}

func TestReduce_FirstProgressEventTransitionsToLearning(t *testing.T) {
	eventTime := time.Date(2026, 4, 16, 10, 0, 0, 0, time.UTC)
	quality := int16(4)

	state, err := aggregate.Reduce(nil, learningEvent(enum.EventQuiz, enum.ReducerEffectAffectsProgress, &quality, eventTime))
	if err != nil {
		t.Fatalf("Reduce() error = %v", err)
	}

	if state.Status != enum.StatusLearning {
		t.Fatalf("status = %q, want %q", state.Status, enum.StatusLearning)
	}
	if state.ProgressEventCount != 1 {
		t.Fatalf("progress_event_count = %d, want 1", state.ProgressEventCount)
	}
	if state.ScheduleRepetition != 1 {
		t.Fatalf("schedule_repetition = %d, want 1", state.ScheduleRepetition)
	}
	if state.ScheduleIntervalDays != 1 {
		t.Fatalf("schedule_interval_days = %v, want 1", state.ScheduleIntervalDays)
	}
	if state.NextReviewAt == nil || !state.NextReviewAt.Equal(eventTime.Add(24*time.Hour)) {
		t.Fatalf("next_review_at = %v, want %v", state.NextReviewAt, eventTime.Add(24*time.Hour))
	}
}

func TestReduce_TwoPassingProgressEventsTransitionToReviewing(t *testing.T) {
	t1 := time.Date(2026, 4, 16, 10, 0, 0, 0, time.UTC)
	t2 := t1.Add(24 * time.Hour)
	q := int16(4)

	state, err := aggregate.Reduce(nil, learningEvent(enum.EventQuiz, enum.ReducerEffectAffectsProgress, &q, t1))
	if err != nil {
		t.Fatalf("first Reduce() error = %v", err)
	}

	state, err = aggregate.Reduce(state, learningEvent(enum.EventQuiz, enum.ReducerEffectAffectsProgress, &q, t2))
	if err != nil {
		t.Fatalf("second Reduce() error = %v", err)
	}

	if state.Status != enum.StatusReviewing {
		t.Fatalf("status = %q, want %q", state.Status, enum.StatusReviewing)
	}
	if state.ProgressEventCount != 2 {
		t.Fatalf("progress_event_count = %d, want 2", state.ProgressEventCount)
	}
	if state.ScheduleRepetition != 2 {
		t.Fatalf("schedule_repetition = %d, want 2", state.ScheduleRepetition)
	}
	if state.ScheduleIntervalDays != 3 {
		t.Fatalf("schedule_interval_days = %v, want 3", state.ScheduleIntervalDays)
	}
}

func TestReduce_MasteredWhenStableAndIntervalLargeEnough(t *testing.T) {
	t1 := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
	t2 := t1.Add(24 * time.Hour)
	t3 := t2.Add(72 * time.Hour)
	t4 := t3.Add(6 * 24 * time.Hour)
	t5 := t4.Add(16 * 24 * time.Hour)
	q4 := int16(4)
	q5 := int16(5)

	state, err := aggregate.Reduce(nil, learningEvent(enum.EventQuiz, enum.ReducerEffectAffectsProgress, &q4, t1))
	if err != nil {
		t.Fatalf("Reduce() error = %v", err)
	}
	state, err = aggregate.Reduce(state, learningEvent(enum.EventQuiz, enum.ReducerEffectAffectsProgress, &q4, t2))
	if err != nil {
		t.Fatalf("Reduce() error = %v", err)
	}
	state, err = aggregate.Reduce(state, learningEvent(enum.EventQuiz, enum.ReducerEffectAffectsProgress, &q5, t3))
	if err != nil {
		t.Fatalf("Reduce() error = %v", err)
	}
	state, err = aggregate.Reduce(state, learningEvent(enum.EventQuiz, enum.ReducerEffectAffectsProgress, &q5, t4))
	if err != nil {
		t.Fatalf("Reduce() error = %v", err)
	}
	state, err = aggregate.Reduce(state, learningEvent(enum.EventQuiz, enum.ReducerEffectAffectsProgress, &q5, t5))
	if err != nil {
		t.Fatalf("Reduce() error = %v", err)
	}

	if state.Status != enum.StatusMastered {
		t.Fatalf("status = %q, want %q", state.Status, enum.StatusMastered)
	}
	if state.IsTarget {
		t.Fatalf("is_target = true, want false")
	}
	if state.ProgressPercent != 100 {
		t.Fatalf("progress_percent = %v, want 100", state.ProgressPercent)
	}
	if state.MasteryScore != 1 {
		t.Fatalf("mastery_score = %v, want 1", state.MasteryScore)
	}
	if state.NextReviewAt != nil {
		t.Fatalf("next_review_at = %v, want nil", state.NextReviewAt)
	}
}

func TestReduce_SetMasteredMarksCompletedAndInactive(t *testing.T) {
	state := emptyState()
	state.Status = enum.StatusSuspended
	state.SuspendedReason = "manual_pause"
	nextReviewAt := time.Date(2026, 4, 20, 10, 0, 0, 0, time.UTC)
	state.NextReviewAt = &nextReviewAt
	eventTime := time.Date(2026, 4, 16, 10, 0, 0, 0, time.UTC)

	next, err := aggregate.Reduce(&state, learningEvent(enum.EventSelfMarkMastered, enum.ReducerEffectSetMastered, nil, eventTime))
	if err != nil {
		t.Fatalf("Reduce() error = %v", err)
	}

	if next.Status != enum.StatusMastered {
		t.Fatalf("status = %q, want %q", next.Status, enum.StatusMastered)
	}
	if next.IsTarget {
		t.Fatalf("is_target = true, want false")
	}
	if next.ProgressPercent != 100 {
		t.Fatalf("progress_percent = %v, want 100", next.ProgressPercent)
	}
	if next.MasteryScore != 1 {
		t.Fatalf("mastery_score = %v, want 1", next.MasteryScore)
	}
	if next.NextReviewAt != nil {
		t.Fatalf("next_review_at = %v, want nil", next.NextReviewAt)
	}
	if next.SuspendedReason != "" {
		t.Fatalf("suspended_reason = %q, want empty", next.SuspendedReason)
	}
	if next.ProgressEventCount != 0 {
		t.Fatalf("progress_event_count = %d, want 0", next.ProgressEventCount)
	}
	if next.LastProgressAt == nil || !next.LastProgressAt.Equal(eventTime) {
		t.Fatalf("last_progress_at = %v, want %v", next.LastProgressAt, eventTime)
	}
	if next.ObservationCount != state.ObservationCount+1 {
		t.Fatalf("observation_count = %d, want %d", next.ObservationCount, state.ObservationCount+1)
	}
}

func TestReduce_SetMasteredRejectsProgressQuality(t *testing.T) {
	eventTime := time.Date(2026, 4, 16, 10, 0, 0, 0, time.UTC)
	quality := int16(5)

	_, err := aggregate.Reduce(nil, learningEvent(enum.EventSelfMarkMastered, enum.ReducerEffectSetMastered, &quality, eventTime))
	if err == nil {
		t.Fatal("Reduce() error = nil, want validation error")
	}
}

func TestReduce_SetMasteredRequiresSelfMarkEvent(t *testing.T) {
	eventTime := time.Date(2026, 4, 16, 10, 0, 0, 0, time.UTC)

	_, err := aggregate.Reduce(nil, learningEvent(enum.EventQuiz, enum.ReducerEffectSetMastered, nil, eventTime))
	if err == nil {
		t.Fatal("Reduce() error = nil, want validation error")
	}
}

func TestReduce_TerminalMasteredIgnoresLaterProgressEvent(t *testing.T) {
	state := masteredState()
	state.IsTarget = false
	state.NextReviewAt = nil
	state.MasteryScore = 1
	eventTime := state.LastProgressAt.Add(24 * time.Hour)
	q := int16(1)

	next, err := aggregate.Reduce(&state, learningEvent(enum.EventQuiz, enum.ReducerEffectAffectsProgress, &q, eventTime))
	if err != nil {
		t.Fatalf("Reduce() error = %v", err)
	}

	if next.Status != enum.StatusMastered {
		t.Fatalf("status = %q, want %q", next.Status, enum.StatusMastered)
	}
	if next.ProgressEventCount != state.ProgressEventCount {
		t.Fatalf("progress_event_count = %d, want %d", next.ProgressEventCount, state.ProgressEventCount)
	}
	if next.ObservationCount != state.ObservationCount {
		t.Fatalf("observation_count = %d, want %d", next.ObservationCount, state.ObservationCount)
	}
	if next.ScheduleRepetition != state.ScheduleRepetition {
		t.Fatalf("schedule_repetition = %d, want %d", next.ScheduleRepetition, state.ScheduleRepetition)
	}
}

func TestReduce_FailureAfterRetargetedMasteredFallsBackToReviewing(t *testing.T) {
	state := masteredState()
	eventTime := state.LastProgressAt.Add(24 * time.Hour)
	q := int16(1)

	next, err := aggregate.Reduce(&state, learningEvent(enum.EventQuiz, enum.ReducerEffectAffectsProgress, &q, eventTime))
	if err != nil {
		t.Fatalf("Reduce() error = %v", err)
	}

	if next.Status != enum.StatusReviewing {
		t.Fatalf("status = %q, want %q", next.Status, enum.StatusReviewing)
	}
	if next.ScheduleRepetition != 0 {
		t.Fatalf("schedule_repetition = %d, want 0", next.ScheduleRepetition)
	}
	if next.ScheduleIntervalDays != 1 {
		t.Fatalf("schedule_interval_days = %v, want 1", next.ScheduleIntervalDays)
	}
	if next.MasteryScore >= state.MasteryScore {
		t.Fatalf("mastery_score = %v, want lower than %v", next.MasteryScore, state.MasteryScore)
	}
}

func TestReduce_TruncatesRecentProgressWindowsToFive(t *testing.T) {
	state := emptyState()
	t0 := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
	qualities := []int16{4, 3, 2, 4, 2, 4}

	var err error
	current := &state
	for idx, q := range qualities {
		current, err = aggregate.Reduce(current, learningEvent(enum.EventQuiz, enum.ReducerEffectAffectsProgress, &q, t0.Add(time.Duration(idx)*24*time.Hour)))
		if err != nil {
			t.Fatalf("Reduce() error = %v", err)
		}
	}

	wantQualities := []int16{3, 2, 4, 2, 4}
	if len(current.RecentProgressQualities) != len(wantQualities) {
		t.Fatalf("recent_progress_qualities len = %d, want %d", len(current.RecentProgressQualities), len(wantQualities))
	}
	for idx, want := range wantQualities {
		if current.RecentProgressQualities[idx] != want {
			t.Fatalf("recent_progress_qualities[%d] = %d, want %d", idx, current.RecentProgressQualities[idx], want)
		}
	}
}

func TestReduce_SuspendedControlOverlaysFinalStatus(t *testing.T) {
	state := emptyState()
	state.Status = enum.StatusSuspended
	state.SuspendedReason = "manual_pause"

	eventTime := time.Date(2026, 4, 16, 10, 0, 0, 0, time.UTC)
	q := int16(4)

	next, err := aggregate.Reduce(&state, learningEvent(enum.EventQuiz, enum.ReducerEffectAffectsProgress, &q, eventTime))
	if err != nil {
		t.Fatalf("Reduce() error = %v", err)
	}

	if next.Status != enum.StatusSuspended {
		t.Fatalf("status = %q, want %q", next.Status, enum.StatusSuspended)
	}
	if next.ProgressEventCount != 1 {
		t.Fatalf("progress_event_count = %d, want 1", next.ProgressEventCount)
	}
}

func TestReduce_RejectsLateProgressEvent(t *testing.T) {
	state := emptyState()
	lastProgressAt := time.Date(2026, 4, 16, 10, 0, 0, 0, time.UTC)
	state.LastProgressAt = &lastProgressAt

	q := int16(4)
	_, err := aggregate.Reduce(&state, learningEvent(enum.EventQuiz, enum.ReducerEffectAffectsProgress, &q, lastProgressAt.Add(-time.Hour)))
	if !errors.Is(err, aggregate.ErrLateProgressEvent) {
		t.Fatalf("Reduce() error = %v, want ErrLateProgressEvent", err)
	}
}

func learningEvent(eventType string, reducerEffect string, progressQuality *int16, occurredAt time.Time) model.LearningEvent {
	return model.LearningEvent{
		UserID:          "11111111-1111-1111-1111-111111111111",
		CoarseUnitID:    101,
		EventType:       eventType,
		ReducerEffect:   reducerEffect,
		SourceType:      "quiz_event",
		SourceRefID:     "event_1",
		ProgressQuality: progressQuality,
		Metadata:        []byte("{}"),
		OccurredAt:      occurredAt,
	}
}

func emptyState() model.UserUnitState {
	return model.UserUnitState{
		UserID:                  "11111111-1111-1111-1111-111111111111",
		CoarseUnitID:            101,
		IsTarget:                true,
		Status:                  enum.StatusNew,
		TargetPriority:          0.5,
		ScheduleEaseFactor:      2.5,
		ScheduleIntervalDays:    0,
		ScheduleRepetition:      0,
		ProgressEventCount:      0,
		ObservationCount:        0,
		RecentProgressPasses:    nil,
		RecentProgressQualities: nil,
	}
}

func masteredState() model.UserUnitState {
	lastProgressAt := time.Date(2026, 4, 16, 10, 0, 0, 0, time.UTC)
	lastQuality := int16(5)
	lastObservedAt := lastProgressAt
	firstObservedAt := lastProgressAt.Add(-14 * 24 * time.Hour)
	nextReviewAt := lastProgressAt.Add(21 * 24 * time.Hour)
	return model.UserUnitState{
		UserID:                  "11111111-1111-1111-1111-111111111111",
		CoarseUnitID:            101,
		IsTarget:                true,
		Status:                  enum.StatusMastered,
		TargetPriority:          0.5,
		ObservationCount:        4,
		ProgressEventCount:      4,
		ProgressSuccessCount:    4,
		ConsecutiveSuccessCount: 4,
		LastProgressQuality:     &lastQuality,
		RecentProgressQualities: []int16{4, 4, 5, 5},
		RecentProgressPasses:    []bool{true, true, true, true},
		ScheduleRepetition:      4,
		ScheduleIntervalDays:    21,
		ScheduleEaseFactor:      2.6,
		ProgressPercent:         100,
		MasteryScore:            0.9,
		LastProgressAt:          &lastProgressAt,
		LastObservedAt:          &lastObservedAt,
		FirstObservedAt:         &firstObservedAt,
		NextReviewAt:            &nextReviewAt,
	}
}
