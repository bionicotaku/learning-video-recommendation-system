//go:build integration

package infrastructure_test

import (
	"context"
	"testing"
	"time"

	"learning-video-recommendation-system/internal/analytics/domain/model"
	analyticsrepo "learning-video-recommendation-system/internal/analytics/infrastructure/persistence/repository"
)

func TestRawEventWriterReturnsExistingIDsForDuplicates(t *testing.T) {
	t.Parallel()

	db := testDB(t)
	userID := "11111111-1111-1111-1111-111111111111"
	questionID := "22222222-2222-2222-2222-222222222222"
	unitID := int64(101)
	now := time.Date(2026, 5, 15, 10, 0, 0, 0, time.UTC)
	db.SeedUser(t, userID)
	db.SeedCoarseUnit(t, unitID)
	db.SeedQuestion(t, questionID)

	writer := analyticsrepo.NewRawEventWriter(db.Pool)
	quiz := model.RawQuizEvent{
		ClientEventID:       "client-quiz-1",
		UserID:              userID,
		ClientContext:       []byte(`{"platform":"ios"}`),
		QuestionID:          questionID,
		CoarseUnitID:        unitID,
		TriggerType:         "manual",
		SelectedOptionIDs:   []string{"correct"},
		SelectionIntervalMS: []int32{1000},
		IsFirstTryCorrect:   true,
		TotalElapsedMS:      1000,
		ShownAt:             now.Add(-time.Second),
		CompletedAt:         now,
	}
	firstQuiz, err := writer.UpsertQuizEvent(context.Background(), quiz)
	if err != nil {
		t.Fatalf("first UpsertQuizEvent() error = %v", err)
	}
	secondQuiz, err := writer.UpsertQuizEvent(context.Background(), quiz)
	if err != nil {
		t.Fatalf("second UpsertQuizEvent() error = %v", err)
	}
	if !firstQuiz.Inserted || secondQuiz.Inserted {
		t.Fatalf("quiz results first=%+v second=%+v", firstQuiz, secondQuiz)
	}
	if firstQuiz.EventID == "" || firstQuiz.EventID != secondQuiz.EventID {
		t.Fatalf("quiz event ids first=%q second=%q, want same non-empty", firstQuiz.EventID, secondQuiz.EventID)
	}

	interaction := []model.RawLearningInteractionEvent{
		{
			ClientEventID:   "client-interaction-1",
			UserID:          userID,
			ClientContext:   []byte(`{"platform":"ios"}`),
			EventType:       "lookup",
			SourceSurface:   "video_subtitle",
			CoarseUnitID:    &unitID,
			TokenText:       "example",
			OccurredAt:      now,
			LookupVisibleMS: int32Pointer(5000),
			EventPayload:    []byte(`{}`),
		},
	}
	firstInteraction, err := writer.UpsertLearningInteractions(context.Background(), interaction)
	if err != nil {
		t.Fatalf("first UpsertLearningInteractions() error = %v", err)
	}
	secondInteraction, err := writer.UpsertLearningInteractions(context.Background(), interaction)
	if err != nil {
		t.Fatalf("second UpsertLearningInteractions() error = %v", err)
	}
	if len(firstInteraction) != 1 || len(secondInteraction) != 1 || !firstInteraction[0].Inserted || secondInteraction[0].Inserted {
		t.Fatalf("interaction results first=%+v second=%+v", firstInteraction, secondInteraction)
	}
	if firstInteraction[0].EventID == "" || firstInteraction[0].EventID != secondInteraction[0].EventID {
		t.Fatalf("interaction event ids first=%q second=%q, want same non-empty", firstInteraction[0].EventID, secondInteraction[0].EventID)
	}
}

func TestRawEventWriterUpsertsLearningInteractionBatchInInputOrder(t *testing.T) {
	t.Parallel()

	db := testDB(t)
	userID := "11111111-1111-1111-1111-111111111112"
	unitID := int64(102)
	now := time.Date(2026, 5, 15, 10, 0, 0, 0, time.UTC)
	db.SeedUser(t, userID)
	db.SeedCoarseUnit(t, unitID)

	writer := analyticsrepo.NewRawEventWriter(db.Pool)
	firstBatch := []model.RawLearningInteractionEvent{
		learningInteraction(userID, unitID, "client-interaction-batch-1", now),
	}
	firstResults, err := writer.UpsertLearningInteractions(context.Background(), firstBatch)
	if err != nil {
		t.Fatalf("first UpsertLearningInteractions() error = %v", err)
	}
	if len(firstResults) != 1 || !firstResults[0].Inserted {
		t.Fatalf("first results = %+v, want one inserted", firstResults)
	}

	secondBatch := []model.RawLearningInteractionEvent{
		learningInteraction(userID, unitID, "client-interaction-batch-2", now.Add(time.Second)),
		learningInteraction(userID, unitID, "client-interaction-batch-1", now),
	}
	secondResults, err := writer.UpsertLearningInteractions(context.Background(), secondBatch)
	if err != nil {
		t.Fatalf("second UpsertLearningInteractions() error = %v", err)
	}
	if len(secondResults) != 2 {
		t.Fatalf("second results = %d, want 2", len(secondResults))
	}
	if secondResults[0].ClientEventID != "client-interaction-batch-2" || !secondResults[0].Inserted {
		t.Fatalf("first returned batch row = %+v, want new inserted event in input order", secondResults[0])
	}
	if secondResults[1].ClientEventID != "client-interaction-batch-1" || secondResults[1].Inserted {
		t.Fatalf("second returned batch row = %+v, want duplicate event in input order", secondResults[1])
	}
	if secondResults[1].EventID != firstResults[0].EventID {
		t.Fatalf("duplicate event id = %q, want existing %q", secondResults[1].EventID, firstResults[0].EventID)
	}

	var count int
	if err := db.Pool.QueryRow(context.Background(), `select count(*) from analytics.learning_interaction_events where user_id = $1`, userID).Scan(&count); err != nil {
		t.Fatalf("count learning interaction events: %v", err)
	}
	if count != 2 {
		t.Fatalf("event count = %d, want 2", count)
	}
}

func TestRawEventWriterAllowsLearningInteractionWeakContextIDsWithoutParentRows(t *testing.T) {
	t.Parallel()

	db := testDB(t)
	userID := "11111111-1111-1111-1111-111111111118"
	unitID := int64(108)
	now := time.Date(2026, 5, 15, 10, 0, 0, 0, time.UTC)
	db.SeedUser(t, userID)
	db.SeedCoarseUnit(t, unitID)

	writer := analyticsrepo.NewRawEventWriter(db.Pool)
	events := []model.RawLearningInteractionEvent{
		{
			ClientEventID:      "client-interaction-weak-context",
			UserID:             userID,
			ClientContext:      []byte(`{"platform":"ios"}`),
			EventType:          "self_mark_mastered",
			SourceSurface:      "quiz_result",
			WatchSessionID:     "44444444-4444-4444-4444-444444444448",
			RelatedQuizEventID: "55555555-5555-5555-5555-555555555558",
			CoarseUnitID:       &unitID,
			TokenText:          "example",
			OccurredAt:         now,
			EventPayload:       []byte(`{}`),
		},
	}

	results, err := writer.UpsertLearningInteractions(context.Background(), events)
	if err != nil {
		t.Fatalf("UpsertLearningInteractions() error = %v", err)
	}
	if len(results) != 1 || !results[0].Inserted || results[0].EventID == "" {
		t.Fatalf("results = %+v, want one inserted event", results)
	}
}

func TestRawEventWriterWithActivityStatsIncrementsOnlyInsertedEvents(t *testing.T) {
	t.Parallel()

	db := testDB(t)
	userID := "11111111-1111-1111-1111-111111111113"
	questionID := "22222222-2222-2222-2222-222222222223"
	unitID := int64(103)
	now := time.Date(2026, 5, 15, 10, 0, 0, 0, time.UTC)
	db.SeedUser(t, userID)
	db.SeedCoarseUnit(t, unitID)
	db.SeedQuestion(t, questionID)

	writer := analyticsrepo.NewRawEventWriterWithActivityStats(db.Pool)
	quiz := model.RawQuizEvent{
		ClientEventID:       "client-quiz-stats-1",
		UserID:              userID,
		ClientContext:       []byte(`{"platform":"ios"}`),
		QuestionID:          questionID,
		CoarseUnitID:        unitID,
		TriggerType:         "manual",
		SelectedOptionIDs:   []string{"correct"},
		SelectionIntervalMS: []int32{1000},
		IsFirstTryCorrect:   true,
		TotalElapsedMS:      1000,
		ShownAt:             now.Add(-time.Second),
		CompletedAt:         now,
	}
	if _, err := writer.UpsertQuizEvent(context.Background(), quiz); err != nil {
		t.Fatalf("first UpsertQuizEvent() error = %v", err)
	}
	if _, err := writer.UpsertQuizEvent(context.Background(), quiz); err != nil {
		t.Fatalf("duplicate UpsertQuizEvent() error = %v", err)
	}

	interaction := []model.RawLearningInteractionEvent{
		learningInteraction(userID, unitID, "client-interaction-stats-1", now),
	}
	if _, err := writer.UpsertLearningInteractions(context.Background(), interaction); err != nil {
		t.Fatalf("first UpsertLearningInteractions() error = %v", err)
	}
	if _, err := writer.UpsertLearningInteractions(context.Background(), interaction); err != nil {
		t.Fatalf("duplicate UpsertLearningInteractions() error = %v", err)
	}

	var quizAttemptCount int64
	if err := db.Pool.QueryRow(context.Background(), `
		select quiz_attempt_count
		from app_user.user_activity_stats
		where user_id = $1
	`, userID).Scan(&quizAttemptCount); err != nil {
		t.Fatalf("query user_activity_stats: %v", err)
	}
	if quizAttemptCount != 1 {
		t.Fatalf("quiz_attempt_count = %d, want 1", quizAttemptCount)
	}

	var dailyQuizCount int64
	var dailyInteractionCount int64
	if err := db.Pool.QueryRow(context.Background(), `
		select quiz_attempt_count, learning_interaction_count
		from app_user.user_daily_activity_stats
		where user_id = $1
		  and local_date = date '2026-05-15'
	`, userID).Scan(&dailyQuizCount, &dailyInteractionCount); err != nil {
		t.Fatalf("query user_daily_activity_stats: %v", err)
	}
	if dailyQuizCount != 1 || dailyInteractionCount != 2 {
		t.Fatalf("daily counts quiz=%d interaction=%d, want 1/2", dailyQuizCount, dailyInteractionCount)
	}
}

func TestNormalizerPendingIndexesExist(t *testing.T) {
	t.Parallel()

	db := testDB(t)
	ctx := context.Background()

	indexes := []string{
		"idx_quiz_events_completed_at_event_id",
		"idx_learning_interaction_events_pending_normalizer",
		"idx_learning_interaction_events_exposure_session",
		"idx_learning_interaction_events_lookup_unit_time",
	}
	for _, indexName := range indexes {
		var exists bool
		if err := db.Pool.QueryRow(ctx, `
			select exists (
				select 1
				from pg_indexes
				where schemaname = 'analytics'
				  and indexname = $1
			)
		`, indexName).Scan(&exists); err != nil {
			t.Fatalf("check index %s: %v", indexName, err)
		}
		if !exists {
			t.Fatalf("index %s does not exist", indexName)
		}
	}
}

func learningInteraction(userID string, unitID int64, clientEventID string, occurredAt time.Time) model.RawLearningInteractionEvent {
	return model.RawLearningInteractionEvent{
		ClientEventID:   clientEventID,
		UserID:          userID,
		ClientContext:   []byte(`{"platform":"ios"}`),
		EventType:       "lookup",
		SourceSurface:   "video_subtitle",
		CoarseUnitID:    &unitID,
		TokenText:       "example",
		OccurredAt:      occurredAt,
		LookupVisibleMS: int32Pointer(5000),
		EventPayload:    []byte(`{}`),
	}
}

func int32Pointer(value int32) *int32 {
	return &value
}
