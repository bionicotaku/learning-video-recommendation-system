//go:build integration

package application_test

import (
	"context"
	"testing"
	"time"

	"learning-video-recommendation-system/internal/learningengine/normalizer/application/dto"
	normalizerservice "learning-video-recommendation-system/internal/learningengine/normalizer/application/service"
	normalizerrepo "learning-video-recommendation-system/internal/learningengine/normalizer/infrastructure/persistence/repository"
	"learning-video-recommendation-system/internal/learningengine/normalizer/test/fixture"
	learningservice "learning-video-recommendation-system/internal/learningengine/reducer/application/service"
	learningenum "learning-video-recommendation-system/internal/learningengine/reducer/domain/enum"
	learningrepo "learning-video-recommendation-system/internal/learningengine/reducer/infrastructure/persistence/repository"
	learningtx "learning-video-recommendation-system/internal/learningengine/reducer/infrastructure/persistence/tx"
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

func TestNormalizeByIDsOnlyProcessesRequestedUserRows(t *testing.T) {
	t.Parallel()

	db := testDB(t)
	userID := "11111111-1111-1111-1111-111111111111"
	otherUserID := "99999999-9999-9999-9999-999999999999"
	questionID := "22222222-2222-2222-2222-222222222222"
	db.SeedUser(t, userID)
	db.SeedUser(t, otherUserID)
	db.SeedCoarseUnit(t, 101)
	db.SeedCoarseUnit(t, 102)
	db.SeedQuestion(t, questionID)

	quizEventID := "33333333-3333-3333-3333-333333333333"
	lookupEventID := "44444444-4444-4444-4444-444444444444"
	otherUserEventID := "55555555-5555-5555-5555-555555555555"
	occurredAt := time.Date(2026, 5, 15, 10, 0, 0, 0, time.UTC)
	seedQuizEvent(t, db, quizEventID, userID, questionID, 101, true, 5000, occurredAt)
	seedLearningInteraction(t, db, lookupEventID, userID, 102, learningenum.EventLookup, occurredAt.Add(time.Second))
	seedLearningInteraction(t, db, otherUserEventID, otherUserID, 102, learningenum.EventLookup, occurredAt.Add(2*time.Second))

	quizUsecase := newNormalizeQuizAttemptByIDUsecase(db)
	quizResponse, err := quizUsecase.Execute(context.Background(), dto.NormalizeQuizAttemptByIDRequest{
		UserID:      userID,
		QuizEventID: quizEventID,
	})
	if err != nil {
		t.Fatalf("quiz Execute() error = %v", err)
	}
	if quizResponse.ReadRawCount != 1 || quizResponse.RecordedEventCount != 1 {
		t.Fatalf("quiz response = %+v, want read=1 recorded=1", quizResponse)
	}

	interactionUsecase := newNormalizeLearningInteractionsByIDsUsecase(db)
	interactionResponse, err := interactionUsecase.Execute(context.Background(), dto.NormalizeLearningInteractionsByIDsRequest{
		UserID:                      userID,
		LearningInteractionEventIDs: []string{lookupEventID, otherUserEventID},
	})
	if err != nil {
		t.Fatalf("interaction Execute() error = %v", err)
	}
	if interactionResponse.ReadRawCount != 1 || interactionResponse.RecordedEventCount != 1 {
		t.Fatalf("interaction response = %+v, want read=1 recorded=1", interactionResponse)
	}

	quizState := readState(t, db, userID, 101)
	if quizState.progressEventCount != 1 || quizState.lastProgressQuality == nil || *quizState.lastProgressQuality != 5 {
		t.Fatalf("quiz state = %+v, want one quality=5 progress event", quizState)
	}
	lookupState := readState(t, db, userID, 102)
	if lookupState.observationCount != 1 || lookupState.progressEventCount != 0 {
		t.Fatalf("lookup state = %+v, want observation only", lookupState)
	}

	var otherUserStateCount int
	if err := db.Pool.QueryRow(context.Background(), `select count(*) from learning.user_unit_states where user_id = $1`, otherUserID).Scan(&otherUserStateCount); err != nil {
		t.Fatalf("count other user states: %v", err)
	}
	if otherUserStateCount != 0 {
		t.Fatalf("other user states = %d, want 0", otherUserStateCount)
	}
}

func TestNormalizeSelfMarkMasteredByIDSetsTerminalMastered(t *testing.T) {
	t.Parallel()

	db := testDB(t)
	userID := "11111111-1111-1111-1111-111111111111"
	eventID := "44444444-4444-4444-4444-444444444444"
	db.SeedUser(t, userID)
	db.SeedCoarseUnit(t, 101)
	seedLearningInteraction(t, db, eventID, userID, 101, learningenum.EventSelfMarkMastered, time.Date(2026, 5, 15, 10, 0, 0, 0, time.UTC))

	usecase := newNormalizeSelfMarkMasteredByIDUsecase(db)
	response, err := usecase.Execute(context.Background(), dto.NormalizeSelfMarkMasteredByIDRequest{
		UserID:                     userID,
		LearningInteractionEventID: eventID,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if response.ReadRawCount != 1 || response.RecordedEventCount != 1 {
		t.Fatalf("response = %+v, want read=1 recorded=1", response)
	}

	state := readState(t, db, userID, 101)
	if state.status != "mastered" || state.isTarget || state.progressPercent != 100 || state.masteryScore != 1 || !state.nextReviewIsNull {
		t.Fatalf("state = %+v, want terminal mastered", state)
	}
}

func TestNormalizeSelfMarkMasteredByIDRejectsNonSelfMarkRawRow(t *testing.T) {
	t.Parallel()

	db := testDB(t)
	userID := "11111111-1111-1111-1111-111111111111"
	eventID := "44444444-4444-4444-4444-444444444444"
	db.SeedUser(t, userID)
	db.SeedCoarseUnit(t, 101)
	seedLearningInteraction(t, db, eventID, userID, 101, learningenum.EventLookup, time.Date(2026, 5, 15, 10, 0, 0, 0, time.UTC))

	usecase := newNormalizeSelfMarkMasteredByIDUsecase(db)
	response, err := usecase.Execute(context.Background(), dto.NormalizeSelfMarkMasteredByIDRequest{
		UserID:                     userID,
		LearningInteractionEventID: eventID,
	})
	if err == nil {
		t.Fatalf("Execute() error = nil, want event type error")
	}
	if response.ReadRawCount != 1 || response.ErrorCount != 1 {
		t.Fatalf("response = %+v, want read=1 error=1", response)
	}

	var eventCount int
	if err := db.Pool.QueryRow(context.Background(), `select count(*) from learning.unit_learning_events`).Scan(&eventCount); err != nil {
		t.Fatalf("count learning events: %v", err)
	}
	if eventCount != 0 {
		t.Fatalf("learning events = %d, want 0", eventCount)
	}
}

func TestNormalizeByIDWritesLearningEventOccurredAtAsUTCInstant(t *testing.T) {
	t.Parallel()

	db := testDB(t)
	userID := "11111111-1111-1111-1111-111111111111"
	questionID := "22222222-2222-2222-2222-222222222222"
	quizEventID := "33333333-3333-3333-3333-333333333333"
	completedAt := time.Date(2026, 5, 15, 10, 0, 0, 0, time.FixedZone("PDT", -7*60*60))
	db.SeedUser(t, userID)
	db.SeedCoarseUnit(t, 101)
	db.SeedQuestion(t, questionID)
	seedQuizEvent(t, db, quizEventID, userID, questionID, 101, true, 5000, completedAt)

	usecase := newNormalizeQuizAttemptByIDUsecase(db)
	response, err := usecase.Execute(context.Background(), dto.NormalizeQuizAttemptByIDRequest{
		UserID:      userID,
		QuizEventID: quizEventID,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if response.RecordedEventCount != 1 {
		t.Fatalf("RecordedEventCount = %d, want 1", response.RecordedEventCount)
	}

	events, err := learningrepo.NewUnitLearningEventRepository(db.Pool).ListByUserOrdered(context.Background(), userID)
	if err != nil {
		t.Fatalf("ListByUserOrdered() error = %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("events = %d, want 1", len(events))
	}
	occurredAt := events[0].OccurredAt
	if occurredAt.Location() != time.UTC {
		t.Fatalf("occurred_at location = %v, want UTC", occurredAt.Location())
	}
	if !occurredAt.Equal(completedAt) {
		t.Fatalf("occurred_at = %v, want same instant as %v", occurredAt, completedAt)
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

func newNormalizeLearningInteractionsByIDsUsecase(db *fixture.TestDatabase) *normalizerservice.NormalizeLearningInteractionsByIDsUsecase {
	recordUsecase := learningservice.NewRecordLearningEventsUsecase(learningtx.NewManager(db.Pool))
	return normalizerservice.NewNormalizeLearningInteractionsByIDsUsecase(
		normalizerrepo.NewRawLearningInteractionReader(db.Pool),
		recordUsecase,
	)
}

func newNormalizeQuizAttemptByIDUsecase(db *fixture.TestDatabase) *normalizerservice.NormalizeQuizAttemptByIDUsecase {
	recordUsecase := learningservice.NewRecordLearningEventsUsecase(learningtx.NewManager(db.Pool))
	return normalizerservice.NewNormalizeQuizAttemptByIDUsecase(
		normalizerrepo.NewRawQuizEventReader(db.Pool),
		recordUsecase,
	)
}

func newNormalizeSelfMarkMasteredByIDUsecase(db *fixture.TestDatabase) *normalizerservice.NormalizeSelfMarkMasteredByIDUsecase {
	recordUsecase := learningservice.NewRecordLearningEventsUsecase(learningtx.NewManager(db.Pool))
	return normalizerservice.NewNormalizeSelfMarkMasteredByIDUsecase(
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
