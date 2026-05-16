package service_test

import (
	"context"
	"testing"
	"time"

	"learning-video-recommendation-system/internal/analytics/application/dto"
	"learning-video-recommendation-system/internal/analytics/application/service"
	"learning-video-recommendation-system/internal/analytics/domain/model"
)

func TestRecordQuizAttemptWritesSingleAttempt(t *testing.T) {
	now := time.Date(2026, 5, 15, 10, 0, 0, 0, time.UTC)
	writer := &fakeRawEventWriter{
		quizResult: model.RawEventWriteResult{ClientEventID: "quiz-1", EventID: "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb", Inserted: false},
	}
	usecase := service.NewRecordQuizAttemptUsecase(writer)

	response, err := usecase.Execute(context.Background(), dto.RecordQuizAttemptRequest{
		UserID:              "11111111-1111-1111-1111-111111111111",
		ClientContext:       []byte(`{"platform":"ios"}`),
		ClientEventID:       "quiz-1",
		QuestionID:          "33333333-3333-3333-3333-333333333333",
		CoarseUnitID:        101,
		TriggerType:         "manual",
		SelectedOptionIDs:   []string{"correct"},
		SelectionIntervalMS: []int32{1000},
		IsFirstTryCorrect:   true,
		TotalElapsedMS:      1000,
		ShownAt:             now.Add(-time.Second),
		CompletedAt:         now,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !response.Accepted || response.Inserted || response.QuizEventID != "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb" {
		t.Fatalf("response = %+v, want accepted duplicate quiz id", response)
	}
	if len(writer.quizEvents) != 1 {
		t.Fatalf("writer quiz events = %d, want 1", len(writer.quizEvents))
	}
	if writer.quizEvents[0].UserID != "11111111-1111-1111-1111-111111111111" {
		t.Fatalf("writer user id = %q", writer.quizEvents[0].UserID)
	}
}

func TestRecordQuizAttemptRejectsInvalidAttemptBeforeWrite(t *testing.T) {
	now := time.Date(2026, 5, 15, 10, 0, 0, 0, time.UTC)
	writer := &fakeRawEventWriter{}
	usecase := service.NewRecordQuizAttemptUsecase(writer)

	_, err := usecase.Execute(context.Background(), dto.RecordQuizAttemptRequest{
		UserID:              "11111111-1111-1111-1111-111111111111",
		ClientEventID:       "quiz-1",
		QuestionID:          "33333333-3333-3333-3333-333333333333",
		CoarseUnitID:        101,
		TriggerType:         "manual",
		SelectedOptionIDs:   []string{"wrong"},
		SelectionIntervalMS: []int32{1000},
		IsFirstTryCorrect:   false,
		TotalElapsedMS:      1000,
		ShownAt:             now.Add(-time.Second),
		CompletedAt:         now,
	})
	if err == nil {
		t.Fatalf("Execute() error = nil, want validation error")
	}
	if len(writer.quizEvents) != 0 || len(writer.interactions) != 0 {
		t.Fatalf("writer was called quiz=%d interactions=%d, want no writes", len(writer.quizEvents), len(writer.interactions))
	}
}
