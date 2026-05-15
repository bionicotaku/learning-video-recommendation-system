//go:build integration

package application_test

import (
	"context"
	"testing"
	"time"

	learningservice "learning-video-recommendation-system/internal/learningengine/application/service"
	learningenum "learning-video-recommendation-system/internal/learningengine/domain/enum"
	learningtx "learning-video-recommendation-system/internal/learningengine/infrastructure/persistence/tx"
	"learning-video-recommendation-system/internal/learningengine/normalizer/application/dto"
	normalizerservice "learning-video-recommendation-system/internal/learningengine/normalizer/application/service"
	normalizerrepo "learning-video-recommendation-system/internal/learningengine/normalizer/infrastructure/persistence/repository"
	"learning-video-recommendation-system/internal/learningengine/normalizer/test/fixture"
)

func TestNormalizePendingEventsQuizUpdatesLearningState(t *testing.T) {
	t.Parallel()

	db := testDB(t)
	userID := "11111111-1111-1111-1111-111111111111"
	questionID := "22222222-2222-2222-2222-222222222222"
	db.SeedUser(t, userID)
	db.SeedCoarseUnit(t, 101)
	db.SeedQuestion(t, questionID)
	seedQuizEvent(t, db, "33333333-3333-3333-3333-333333333333", userID, questionID, 101, true, 5000, time.Date(2026, 5, 15, 10, 0, 0, 0, time.UTC))

	usecase := newNormalizerUsecase(db)
	response, err := usecase.Execute(context.Background(), dto.NormalizePendingEventsRequest{SourceKind: dto.SourceKindQuiz})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if response.RecordedEventCount != 1 {
		t.Fatalf("RecordedEventCount = %d, want 1", response.RecordedEventCount)
	}

	state := readState(t, db, userID, 101)
	if state.status != "learning" {
		t.Fatalf("status = %q, want learning", state.status)
	}
	if state.progressEventCount != 1 || state.lastProgressQuality == nil || *state.lastProgressQuality != 5 {
		t.Fatalf("progress count/quality = %d/%v, want 1/5", state.progressEventCount, state.lastProgressQuality)
	}
	if state.observationCount != 1 {
		t.Fatalf("observation_count = %d, want 1", state.observationCount)
	}
}

func TestNormalizePendingEventsSelfMarkSetsTerminalMastered(t *testing.T) {
	t.Parallel()

	db := testDB(t)
	userID := "11111111-1111-1111-1111-111111111111"
	db.SeedUser(t, userID)
	db.SeedCoarseUnit(t, 101)
	seedLearningInteraction(t, db, "44444444-4444-4444-4444-444444444444", userID, 101, learningenum.EventSelfMarkMastered, time.Date(2026, 5, 15, 10, 0, 0, 0, time.UTC))

	usecase := newNormalizerUsecase(db)
	response, err := usecase.Execute(context.Background(), dto.NormalizePendingEventsRequest{SourceKind: dto.SourceKindLearningInteraction})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if response.RecordedEventCount != 1 {
		t.Fatalf("RecordedEventCount = %d, want 1", response.RecordedEventCount)
	}

	state := readState(t, db, userID, 101)
	if state.status != "mastered" {
		t.Fatalf("status = %q, want mastered", state.status)
	}
	if state.isTarget {
		t.Fatalf("is_target = true, want false")
	}
	if state.progressPercent != 100 || state.masteryScore != 1 {
		t.Fatalf("progress/mastery = %v/%v, want 100/1", state.progressPercent, state.masteryScore)
	}
	if !state.nextReviewIsNull {
		t.Fatalf("next_review_at is not null, want null")
	}
}

func TestNormalizePendingEventsLookupAndExposureOnlyUpdateObservation(t *testing.T) {
	t.Parallel()

	db := testDB(t)
	userID := "11111111-1111-1111-1111-111111111111"
	db.SeedUser(t, userID)
	db.SeedCoarseUnit(t, 101)
	seedLearningInteraction(t, db, "55555555-5555-5555-5555-555555555555", userID, 101, learningenum.EventExposure, time.Date(2026, 5, 15, 10, 0, 0, 0, time.UTC))
	seedLearningInteraction(t, db, "66666666-6666-6666-6666-666666666666", userID, 101, learningenum.EventLookup, time.Date(2026, 5, 15, 10, 1, 0, 0, time.UTC))

	usecase := newNormalizerUsecase(db)
	response, err := usecase.Execute(context.Background(), dto.NormalizePendingEventsRequest{SourceKind: dto.SourceKindLearningInteraction})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if response.RecordedEventCount != 2 {
		t.Fatalf("RecordedEventCount = %d, want 2", response.RecordedEventCount)
	}

	state := readState(t, db, userID, 101)
	if state.status != "new" {
		t.Fatalf("status = %q, want new", state.status)
	}
	if state.observationCount != 2 || state.progressEventCount != 0 {
		t.Fatalf("observation/progress count = %d/%d, want 2/0", state.observationCount, state.progressEventCount)
	}
	if state.lastProgressQuality != nil {
		t.Fatalf("last_progress_quality = %v, want nil", state.lastProgressQuality)
	}
}

func newNormalizerUsecase(db *fixture.TestDatabase) *normalizerservice.NormalizePendingEventsUsecase {
	recordUsecase := learningservice.NewRecordLearningEventsUsecase(learningtx.NewManager(db.Pool))
	return normalizerservice.NewNormalizePendingEventsUsecase(
		normalizerrepo.NewRawQuizEventReader(db.Pool),
		normalizerrepo.NewRawLearningInteractionReader(db.Pool),
		recordUsecase,
	)
}

type stateRow struct {
	status              string
	isTarget            bool
	progressPercent     float64
	masteryScore        float64
	observationCount    int32
	progressEventCount  int32
	lastProgressQuality *int16
	nextReviewIsNull    bool
}

func readState(t *testing.T, db *fixture.TestDatabase, userID string, unitID int64) stateRow {
	t.Helper()

	var row stateRow
	if err := db.Pool.QueryRow(context.Background(), `
		select
			status,
			is_target,
			progress_percent::float8,
			mastery_score::float8,
			observation_count,
			progress_event_count,
			last_progress_quality,
			next_review_at is null
		from learning.user_unit_states
		where user_id = $1 and coarse_unit_id = $2
	`, userID, unitID).Scan(
		&row.status,
		&row.isTarget,
		&row.progressPercent,
		&row.masteryScore,
		&row.observationCount,
		&row.progressEventCount,
		&row.lastProgressQuality,
		&row.nextReviewIsNull,
	); err != nil {
		t.Fatalf("read learning.user_unit_states: %v", err)
	}
	return row
}

func seedQuizEvent(t *testing.T, db *fixture.TestDatabase, eventID, userID, questionID string, unitID int64, correct bool, elapsedMS int32, completedAt time.Time) {
	t.Helper()
	selectedOptionIDs := []string{"correct"}
	selectionIntervalMS := []int32{elapsedMS}
	if !correct {
		selectedOptionIDs = []string{"wrong", "correct"}
		selectionIntervalMS = []int32{1000, elapsedMS - 1000}
	}
	if _, err := db.Pool.Exec(context.Background(), `
		insert into analytics.quiz_events (
			event_id,
			client_event_id,
			user_id,
			question_id,
			coarse_unit_id,
			trigger_type,
			selected_option_ids,
			selection_interval_ms,
			is_first_try_correct,
			total_elapsed_ms,
			shown_at,
			completed_at
		) values (
			$1::uuid,
			'client-' || $1::text,
			$2::uuid,
			$3::uuid,
			$4,
			'manual',
			$5,
			$6,
			$7,
			$8,
			$9::timestamptz - interval '1 second',
			$9
		)`, eventID, userID, questionID, unitID, selectedOptionIDs, selectionIntervalMS, correct, elapsedMS, completedAt); err != nil {
		t.Fatalf("seed analytics.quiz_events: %v", err)
	}
}

func seedLearningInteraction(t *testing.T, db *fixture.TestDatabase, eventID, userID string, unitID int64, eventType string, occurredAt time.Time) {
	t.Helper()
	if _, err := db.Pool.Exec(context.Background(), `
		insert into analytics.learning_interaction_events (
			event_id,
			client_event_id,
			user_id,
			event_type,
			source_surface,
			coarse_unit_id,
			token_text,
			occurred_at,
			exposure_start_ms,
			exposure_end_ms,
			exposure_count,
			lookup_visible_ms,
			event_payload
		) values (
			$1::uuid,
			'client-' || $1::text,
			$2::uuid,
			$3,
			'video_subtitle',
			$4,
			'example',
			$5,
			100,
			1200,
			1,
			9000,
			'{}'::jsonb
		)`, eventID, userID, eventType, unitID, occurredAt); err != nil {
		t.Fatalf("seed analytics.learning_interaction_events: %v", err)
	}
}
