//go:build integration

package infrastructure_test

import (
	"context"
	"testing"
	"time"

	apprepo "learning-video-recommendation-system/internal/learningengine/normalizer/application/repository"
	normalizerrepo "learning-video-recommendation-system/internal/learningengine/normalizer/infrastructure/persistence/repository"
	"learning-video-recommendation-system/internal/learningengine/normalizer/test/fixture"
)

func TestRawQuizEventReaderExcludesAlreadyRecordedEvents(t *testing.T) {
	t.Parallel()

	db := testDB(t)
	userID := "11111111-1111-1111-1111-111111111111"
	questionID := "22222222-2222-2222-2222-222222222222"
	eventID := "33333333-3333-3333-3333-333333333333"
	db.SeedUser(t, userID)
	db.SeedCoarseUnit(t, 101)
	db.SeedQuestion(t, questionID)
	seedQuizEvent(t, db, eventID, userID, questionID, 101, time.Date(2026, 5, 15, 10, 0, 0, 0, time.UTC))

	reader := normalizerrepo.NewRawQuizEventReader(db.Pool)
	rows, err := reader.ListPendingQuizEvents(context.Background(), apprepo.PendingRawEventFilter{Limit: 100})
	if err != nil {
		t.Fatalf("ListPendingQuizEvents() error = %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("pending rows = %d, want 1", len(rows))
	}

	seedLearningEvent(t, db, userID, 101, "quiz", "affects_progress", "quiz_event", eventID, "4")

	rows, err = reader.ListPendingQuizEvents(context.Background(), apprepo.PendingRawEventFilter{Limit: 100})
	if err != nil {
		t.Fatalf("ListPendingQuizEvents() after recorded error = %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("pending rows after recorded = %d, want 0", len(rows))
	}
}

func TestRawLearningInteractionReaderExcludesAlreadyRecordedEvents(t *testing.T) {
	t.Parallel()

	db := testDB(t)
	userID := "11111111-1111-1111-1111-111111111111"
	eventID := "44444444-4444-4444-4444-444444444444"
	db.SeedUser(t, userID)
	db.SeedCoarseUnit(t, 101)
	seedLearningInteraction(t, db, eventID, userID, 101, "lookup", time.Date(2026, 5, 15, 10, 0, 0, 0, time.UTC))

	reader := normalizerrepo.NewRawLearningInteractionReader(db.Pool)
	rows, err := reader.ListPendingLearningInteractions(context.Background(), apprepo.PendingRawEventFilter{Limit: 100})
	if err != nil {
		t.Fatalf("ListPendingLearningInteractions() error = %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("pending rows = %d, want 1", len(rows))
	}

	seedLearningEvent(t, db, userID, 101, "lookup", "observe_only", "learning_interaction_event", eventID, "null")

	rows, err = reader.ListPendingLearningInteractions(context.Background(), apprepo.PendingRawEventFilter{Limit: 100})
	if err != nil {
		t.Fatalf("ListPendingLearningInteractions() after recorded error = %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("pending rows after recorded = %d, want 0", len(rows))
	}
}

func TestRawReadersByIDsFilterByUserAndSelectedIDs(t *testing.T) {
	t.Parallel()

	db := testDB(t)
	userID := "11111111-1111-1111-1111-111111111111"
	otherUserID := "99999999-9999-9999-9999-999999999999"
	questionID := "22222222-2222-2222-2222-222222222222"
	quizEventID := "33333333-3333-3333-3333-333333333333"
	otherQuizEventID := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	interactionEventID := "44444444-4444-4444-4444-444444444444"
	otherInteractionEventID := "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
	db.SeedUser(t, userID)
	db.SeedUser(t, otherUserID)
	db.SeedCoarseUnit(t, 101)
	db.SeedQuestion(t, questionID)
	now := time.Date(2026, 5, 15, 10, 0, 0, 0, time.UTC)
	seedQuizEvent(t, db, quizEventID, userID, questionID, 101, now)
	seedQuizEvent(t, db, otherQuizEventID, otherUserID, questionID, 101, now.Add(time.Second))
	seedLearningInteraction(t, db, interactionEventID, userID, 101, "lookup", now)
	seedLearningInteraction(t, db, otherInteractionEventID, otherUserID, 101, "lookup", now.Add(time.Second))

	quizReader := normalizerrepo.NewRawQuizEventReader(db.Pool)
	quizRows, err := quizReader.ListQuizEventsByIDs(context.Background(), userID, []string{quizEventID, otherQuizEventID})
	if err != nil {
		t.Fatalf("ListQuizEventsByIDs() error = %v", err)
	}
	if len(quizRows) != 1 || quizRows[0].EventID != quizEventID {
		t.Fatalf("quiz rows = %+v, want only requested user row", quizRows)
	}

	interactionReader := normalizerrepo.NewRawLearningInteractionReader(db.Pool)
	interactionRows, err := interactionReader.ListLearningInteractionsByIDs(context.Background(), userID, []string{interactionEventID, otherInteractionEventID})
	if err != nil {
		t.Fatalf("ListLearningInteractionsByIDs() error = %v", err)
	}
	if len(interactionRows) != 1 || interactionRows[0].EventID != interactionEventID {
		t.Fatalf("interaction rows = %+v, want only requested user row", interactionRows)
	}
}

func TestRawReadersReturnTimesInUTC(t *testing.T) {
	t.Parallel()

	db := testDB(t)
	userID := "11111111-1111-1111-1111-111111111111"
	questionID := "22222222-2222-2222-2222-222222222222"
	quizEventID := "33333333-3333-3333-3333-333333333333"
	interactionEventID := "44444444-4444-4444-4444-444444444444"
	completedAt := time.Date(2026, 5, 15, 10, 0, 0, 0, time.FixedZone("PDT", -7*60*60))
	occurredAt := completedAt.Add(2 * time.Second)
	db.SeedUser(t, userID)
	db.SeedCoarseUnit(t, 101)
	db.SeedQuestion(t, questionID)
	seedQuizEvent(t, db, quizEventID, userID, questionID, 101, completedAt)
	seedLearningInteraction(t, db, interactionEventID, userID, 101, "lookup", occurredAt)

	quizReader := normalizerrepo.NewRawQuizEventReader(db.Pool)
	quizRows, err := quizReader.ListQuizEventsByIDs(context.Background(), userID, []string{quizEventID})
	if err != nil {
		t.Fatalf("ListQuizEventsByIDs() error = %v", err)
	}
	if len(quizRows) != 1 {
		t.Fatalf("quiz rows = %d, want 1", len(quizRows))
	}
	if quizRows[0].CompletedAt.Location() != time.UTC {
		t.Fatalf("CompletedAt location = %v, want UTC", quizRows[0].CompletedAt.Location())
	}
	if !quizRows[0].CompletedAt.Equal(completedAt) {
		t.Fatalf("CompletedAt = %v, want same instant as %v", quizRows[0].CompletedAt, completedAt)
	}
	if quizRows[0].ShownAt.Location() != time.UTC {
		t.Fatalf("ShownAt location = %v, want UTC", quizRows[0].ShownAt.Location())
	}

	interactionReader := normalizerrepo.NewRawLearningInteractionReader(db.Pool)
	interactionRows, err := interactionReader.ListLearningInteractionsByIDs(context.Background(), userID, []string{interactionEventID})
	if err != nil {
		t.Fatalf("ListLearningInteractionsByIDs() error = %v", err)
	}
	if len(interactionRows) != 1 {
		t.Fatalf("interaction rows = %d, want 1", len(interactionRows))
	}
	if interactionRows[0].OccurredAt.Location() != time.UTC {
		t.Fatalf("OccurredAt location = %v, want UTC", interactionRows[0].OccurredAt.Location())
	}
	if !interactionRows[0].OccurredAt.Equal(occurredAt) {
		t.Fatalf("OccurredAt = %v, want same instant as %v", interactionRows[0].OccurredAt, occurredAt)
	}
}

func seedQuizEvent(t *testing.T, db *fixture.TestDatabase, eventID, userID, questionID string, unitID int64, completedAt time.Time) {
	t.Helper()
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
			array['correct']::text[],
			array[1000]::integer[],
			true,
			1000,
			$5::timestamptz - interval '1 second',
			$5
		)`, eventID, userID, questionID, unitID, completedAt); err != nil {
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
			9000,
			'{}'::jsonb
		)`, eventID, userID, eventType, unitID, occurredAt); err != nil {
		t.Fatalf("seed analytics.learning_interaction_events: %v", err)
	}
}

func seedLearningEvent(t *testing.T, db *fixture.TestDatabase, userID string, unitID int64, eventType, reducerEffect, sourceType, sourceRefID, qualitySQL string) {
	t.Helper()
	if _, err := db.Pool.Exec(context.Background(), `
		insert into learning.unit_learning_events (
			user_id,
			coarse_unit_id,
			event_type,
			reducer_effect,
			progress_quality,
			source_type,
			source_ref_id,
			metadata,
			occurred_at
		) values (
			$1::uuid,
			$2,
			$3,
			$4,
			case when $5 = 'null' then null else $5::smallint end,
			$6,
			$7,
			'{}'::jsonb,
			'2026-05-15 10:00:00+00'::timestamptz
		)`, userID, unitID, eventType, reducerEffect, qualitySQL, sourceType, sourceRefID); err != nil {
		t.Fatalf("seed learning.unit_learning_events: %v", err)
	}
}
