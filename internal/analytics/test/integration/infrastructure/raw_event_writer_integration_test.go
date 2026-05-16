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

func int32Pointer(value int32) *int32 {
	return &value
}
