//go:build integration

package application_test

import (
	"context"
	"testing"
	"time"

	"learning-video-recommendation-system/internal/analytics/application/dto"
	analyticsservice "learning-video-recommendation-system/internal/analytics/application/service"
	analyticsrepo "learning-video-recommendation-system/internal/analytics/infrastructure/persistence/repository"
)

func TestRecordQuizAttemptWritesDBTriggerType(t *testing.T) {
	t.Parallel()

	db := testDB(t)
	userID := "11111111-1111-1111-1111-111111111111"
	questionID := "44444444-4444-4444-4444-444444444444"
	db.SeedUser(t, userID)
	db.SeedCoarseUnit(t, 101)
	db.SeedQuestion(t, questionID)

	usecase := analyticsservice.NewRecordQuizAttemptUsecase(analyticsrepo.NewRawEventWriter(db.Pool))
	response, err := usecase.Execute(context.Background(), dto.RecordQuizAttemptRequest{
		UserID:              userID,
		ClientContext:       []byte(`{"platform":"ios"}`),
		ClientEventID:       "quiz-lookup-practice",
		QuestionID:          questionID,
		CoarseUnitID:        101,
		TriggerType:         "lookup_practice",
		SelectedOptionIDs:   []string{"correct"},
		SelectionIntervalMS: []int32{1200},
		IsFirstTryCorrect:   true,
		TotalElapsedMS:      1200,
		ShownAt:             time.Date(2026, 5, 15, 10, 0, 0, 0, time.UTC),
		CompletedAt:         time.Date(2026, 5, 15, 10, 0, 2, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !response.Accepted || !response.Inserted || response.QuizEventID == "" {
		t.Fatalf("response = %+v, want accepted inserted quiz event id", response)
	}

	var triggerType string
	if err := db.Pool.QueryRow(context.Background(), `
		select trigger_type
		from analytics.quiz_events
		where event_id = $1::uuid
	`, response.QuizEventID).Scan(&triggerType); err != nil {
		t.Fatalf("read quiz event: %v", err)
	}
	if triggerType != "lookup_practice" {
		t.Fatalf("trigger_type = %q, want lookup_practice", triggerType)
	}
}

func TestRecordQuizAttemptRejectsOldTriggerTypeBeforeDB(t *testing.T) {
	t.Parallel()

	db := testDB(t)
	userID := "11111111-1111-1111-1111-111111111111"
	questionID := "44444444-4444-4444-4444-444444444444"
	db.SeedUser(t, userID)
	db.SeedCoarseUnit(t, 101)
	db.SeedQuestion(t, questionID)

	usecase := analyticsservice.NewRecordQuizAttemptUsecase(analyticsrepo.NewRawEventWriter(db.Pool))
	_, err := usecase.Execute(context.Background(), dto.RecordQuizAttemptRequest{
		UserID:              userID,
		ClientContext:       []byte(`{"platform":"ios"}`),
		ClientEventID:       "quiz-practice-now",
		QuestionID:          questionID,
		CoarseUnitID:        101,
		TriggerType:         "practice_now",
		SelectedOptionIDs:   []string{"correct"},
		SelectionIntervalMS: []int32{1200},
		IsFirstTryCorrect:   true,
		TotalElapsedMS:      1200,
		ShownAt:             time.Date(2026, 5, 15, 10, 0, 0, 0, time.UTC),
		CompletedAt:         time.Date(2026, 5, 15, 10, 0, 2, 0, time.UTC),
	})
	if err == nil {
		t.Fatalf("Execute() error = nil, want validation error")
	}
	if !analyticsservice.IsValidationError(err) {
		t.Fatalf("Execute() error = %v, want typed validation error", err)
	}

	var count int
	if err := db.Pool.QueryRow(context.Background(), `select count(*) from analytics.quiz_events`).Scan(&count); err != nil {
		t.Fatalf("count quiz events: %v", err)
	}
	if count != 0 {
		t.Fatalf("quiz rows = %d, want 0", count)
	}
}
