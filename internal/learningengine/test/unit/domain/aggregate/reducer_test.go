package aggregate_test

import (
	"errors"
	"testing"
	"time"

	"learning-video-recommendation-system/internal/learningengine/domain/aggregate"
	"learning-video-recommendation-system/internal/learningengine/domain/enum"
	"learning-video-recommendation-system/internal/learningengine/domain/model"
)

func TestReduce_WeakEventOnlyUpdatesSeenFields(t *testing.T) {
	eventTime := time.Date(2026, 4, 16, 10, 0, 0, 0, time.UTC)

	state, err := aggregate.Reduce(nil, learningEvent("exposure", nil, eventTime))
	if err != nil {
		t.Fatalf("Reduce() error = %v", err)
	}

	if state.Status != enum.StatusNew {
		t.Fatalf("status = %q, want %q", state.Status, enum.StatusNew)
	}
	if state.SeenCount != 1 {
		t.Fatalf("seen_count = %d, want 1", state.SeenCount)
	}
	if state.StrongEventCount != 0 {
		t.Fatalf("strong_event_count = %d, want 0", state.StrongEventCount)
	}
	if state.FirstSeenAt == nil || !state.FirstSeenAt.Equal(eventTime) {
		t.Fatalf("first_seen_at = %v, want %v", state.FirstSeenAt, eventTime)
	}
	if state.LastSeenAt == nil || !state.LastSeenAt.Equal(eventTime) {
		t.Fatalf("last_seen_at = %v, want %v", state.LastSeenAt, eventTime)
	}
	if state.NextReviewAt != nil {
		t.Fatalf("next_review_at = %v, want nil", state.NextReviewAt)
	}
}

func TestReduce_FirstStrongEventTransitionsToLearning(t *testing.T) {
	eventTime := time.Date(2026, 4, 16, 10, 0, 0, 0, time.UTC)
	quality := int16(4)

	state, err := aggregate.Reduce(nil, learningEvent("new_learn", &quality, eventTime))
	if err != nil {
		t.Fatalf("Reduce() error = %v", err)
	}

	if state.Status != enum.StatusLearning {
		t.Fatalf("status = %q, want %q", state.Status, enum.StatusLearning)
	}
	if state.StrongEventCount != 1 {
		t.Fatalf("strong_event_count = %d, want 1", state.StrongEventCount)
	}
	if state.Repetition != 1 {
		t.Fatalf("repetition = %d, want 1", state.Repetition)
	}
	if state.IntervalDays != 1 {
		t.Fatalf("interval_days = %v, want 1", state.IntervalDays)
	}
	if state.NextReviewAt == nil || !state.NextReviewAt.Equal(eventTime.Add(24*time.Hour)) {
		t.Fatalf("next_review_at = %v, want %v", state.NextReviewAt, eventTime.Add(24*time.Hour))
	}
}

func TestReduce_TwoPassingStrongEventsTransitionToReviewing(t *testing.T) {
	t1 := time.Date(2026, 4, 16, 10, 0, 0, 0, time.UTC)
	t2 := t1.Add(24 * time.Hour)
	q := int16(4)

	state, err := aggregate.Reduce(nil, learningEvent("new_learn", &q, t1))
	if err != nil {
		t.Fatalf("first Reduce() error = %v", err)
	}

	state, err = aggregate.Reduce(state, learningEvent("review", &q, t2))
	if err != nil {
		t.Fatalf("second Reduce() error = %v", err)
	}

	if state.Status != enum.StatusReviewing {
		t.Fatalf("status = %q, want %q", state.Status, enum.StatusReviewing)
	}
	if state.ReviewCount != 1 {
		t.Fatalf("review_count = %d, want 1", state.ReviewCount)
	}
	if state.Repetition != 2 {
		t.Fatalf("repetition = %d, want 2", state.Repetition)
	}
	if state.IntervalDays != 3 {
		t.Fatalf("interval_days = %v, want 3", state.IntervalDays)
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

	state, err := aggregate.Reduce(nil, learningEvent("new_learn", &q4, t1))
	if err != nil {
		t.Fatalf("Reduce() error = %v", err)
	}
	state, err = aggregate.Reduce(state, learningEvent("review", &q4, t2))
	if err != nil {
		t.Fatalf("Reduce() error = %v", err)
	}
	state, err = aggregate.Reduce(state, learningEvent("review", &q5, t3))
	if err != nil {
		t.Fatalf("Reduce() error = %v", err)
	}
	state, err = aggregate.Reduce(state, learningEvent("review", &q5, t4))
	if err != nil {
		t.Fatalf("Reduce() error = %v", err)
	}
	state, err = aggregate.Reduce(state, learningEvent("review", &q5, t5))
	if err != nil {
		t.Fatalf("Reduce() error = %v", err)
	}

	if state.Status != enum.StatusMastered {
		t.Fatalf("status = %q, want %q", state.Status, enum.StatusMastered)
	}
	if state.IntervalDays < 21 {
		t.Fatalf("interval_days = %v, want >= 21", state.IntervalDays)
	}
	if state.ProgressPercent != 100 {
		t.Fatalf("progress_percent = %v, want 100", state.ProgressPercent)
	}
	if state.MasteryScore < 0.8 {
		t.Fatalf("mastery_score = %v, want >= 0.8", state.MasteryScore)
	}
}

func TestReduce_FailureAfterMasteredFallsBackToReviewing(t *testing.T) {
	state := masteredState()
	eventTime := state.LastReviewedAt.Add(24 * time.Hour)
	q := int16(1)

	next, err := aggregate.Reduce(&state, learningEvent("review", &q, eventTime))
	if err != nil {
		t.Fatalf("Reduce() error = %v", err)
	}

	if next.Status != enum.StatusReviewing {
		t.Fatalf("status = %q, want %q", next.Status, enum.StatusReviewing)
	}
	if next.Repetition != 0 {
		t.Fatalf("repetition = %d, want 0", next.Repetition)
	}
	if next.IntervalDays != 1 {
		t.Fatalf("interval_days = %v, want 1", next.IntervalDays)
	}
	if next.MasteryScore >= state.MasteryScore {
		t.Fatalf("mastery_score = %v, want lower than %v", next.MasteryScore, state.MasteryScore)
	}
}

func TestReduce_TruncatesRecentWindowsToFive(t *testing.T) {
	state := emptyState()
	t0 := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
	qualities := []int16{5, 4, 3, 5, 4, 2}

	var err error
	current := &state
	for idx, q := range qualities {
		current, err = aggregate.Reduce(current, learningEvent("review", &q, t0.Add(time.Duration(idx)*24*time.Hour)))
		if err != nil {
			t.Fatalf("Reduce() error = %v", err)
		}
	}

	wantQualities := []int16{4, 3, 5, 4, 2}
	if len(current.RecentQualityWindow) != len(wantQualities) {
		t.Fatalf("recent_quality_window len = %d, want %d", len(current.RecentQualityWindow), len(wantQualities))
	}
	for idx, want := range wantQualities {
		if current.RecentQualityWindow[idx] != want {
			t.Fatalf("recent_quality_window[%d] = %d, want %d", idx, current.RecentQualityWindow[idx], want)
		}
	}
}

func TestReduce_SuspendedControlOverlaysFinalStatus(t *testing.T) {
	state := emptyState()
	state.Status = enum.StatusSuspended
	state.SuspendedReason = "manual_pause"

	eventTime := time.Date(2026, 4, 16, 10, 0, 0, 0, time.UTC)
	q := int16(4)

	next, err := aggregate.Reduce(&state, learningEvent("review", &q, eventTime))
	if err != nil {
		t.Fatalf("Reduce() error = %v", err)
	}

	if next.Status != enum.StatusSuspended {
		t.Fatalf("status = %q, want %q", next.Status, enum.StatusSuspended)
	}
	if next.StrongEventCount != 1 {
		t.Fatalf("strong_event_count = %d, want 1", next.StrongEventCount)
	}
}

func TestReduce_RejectsLateStrongEvent(t *testing.T) {
	state := emptyState()
	lastReviewedAt := time.Date(2026, 4, 16, 10, 0, 0, 0, time.UTC)
	state.LastReviewedAt = &lastReviewedAt

	q := int16(4)
	_, err := aggregate.Reduce(&state, learningEvent("review", &q, lastReviewedAt.Add(-time.Hour)))
	if !errors.Is(err, aggregate.ErrLateStrongEvent) {
		t.Fatalf("Reduce() error = %v, want ErrLateStrongEvent", err)
	}
}

func learningEvent(eventType string, quality *int16, occurredAt time.Time) model.LearningEvent {
	return model.LearningEvent{
		UserID:       "11111111-1111-1111-1111-111111111111",
		CoarseUnitID: 101,
		EventType:    eventType,
		SourceType:   "quiz_session",
		Quality:      quality,
		OccurredAt:   occurredAt,
	}
}

func emptyState() model.UserUnitState {
	return model.UserUnitState{
		UserID:         "11111111-1111-1111-1111-111111111111",
		CoarseUnitID:   101,
		IsTarget:       true,
		Status:         enum.StatusNew,
		TargetPriority: 0.5,
		EaseFactor:     2.5,
	}
}

func masteredState() model.UserUnitState {
	lastReviewedAt := time.Date(2026, 4, 16, 10, 0, 0, 0, time.UTC)
	lastQuality := int16(5)
	lastSeenAt := lastReviewedAt
	firstSeenAt := lastReviewedAt.Add(-14 * 24 * time.Hour)
	nextReviewAt := lastReviewedAt.Add(21 * 24 * time.Hour)
	return model.UserUnitState{
		UserID:                  "11111111-1111-1111-1111-111111111111",
		CoarseUnitID:            101,
		IsTarget:                true,
		Status:                  enum.StatusMastered,
		TargetPriority:          0.5,
		StrongEventCount:        4,
		ReviewCount:             3,
		CorrectCount:            4,
		ConsecutiveCorrect:      4,
		LastQuality:             &lastQuality,
		RecentQualityWindow:     []int16{4, 4, 5, 5},
		RecentCorrectnessWindow: []bool{true, true, true, true},
		Repetition:              4,
		IntervalDays:            21,
		EaseFactor:              2.6,
		ProgressPercent:         100,
		MasteryScore:            0.9,
		LastReviewedAt:          &lastReviewedAt,
		LastSeenAt:              &lastSeenAt,
		FirstSeenAt:             &firstSeenAt,
		NextReviewAt:            &nextReviewAt,
	}
}
