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

func TestRecordQuizAttemptNormalizesTimesToUTC(t *testing.T) {
	shownAt := time.Date(2026, 5, 15, 9, 59, 58, 0, time.FixedZone("PDT", -7*60*60))
	completedAt := time.Date(2026, 5, 15, 10, 0, 0, 0, time.FixedZone("PDT", -7*60*60))
	writer := &fakeRawEventWriter{}
	usecase := service.NewRecordQuizAttemptUsecase(writer)

	_, err := usecase.Execute(context.Background(), dto.RecordQuizAttemptRequest{
		UserID:              "11111111-1111-1111-1111-111111111111",
		ClientContext:       []byte(`{"platform":"ios","app_version":"1.3.0","os_version":"18.5","device_model":"iPhone16,2"}`),
		ClientEventID:       "quiz-utc",
		QuestionID:          "33333333-3333-3333-3333-333333333333",
		CoarseUnitID:        101,
		TriggerType:         "manual",
		SelectedOptionIDs:   []string{"correct"},
		SelectionIntervalMS: []int32{1000},
		IsFirstTryCorrect:   true,
		TotalElapsedMS:      1000,
		ShownAt:             shownAt,
		CompletedAt:         completedAt,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(writer.quizEvents) != 1 {
		t.Fatalf("writer quiz events = %d, want 1", len(writer.quizEvents))
	}
	got := writer.quizEvents[0]
	if got.ShownAt.Location() != time.UTC {
		t.Fatalf("ShownAt location = %v, want UTC", got.ShownAt.Location())
	}
	if got.CompletedAt.Location() != time.UTC {
		t.Fatalf("CompletedAt location = %v, want UTC", got.CompletedAt.Location())
	}
	if !got.ShownAt.Equal(shownAt) {
		t.Fatalf("ShownAt = %v, want same instant as %v", got.ShownAt, shownAt)
	}
	if !got.CompletedAt.Equal(completedAt) {
		t.Fatalf("CompletedAt = %v, want same instant as %v", got.CompletedAt, completedAt)
	}
}

func TestRecordQuizAttemptAcceptsLooseClientContextObject(t *testing.T) {
	now := time.Date(2026, 5, 15, 10, 0, 0, 0, time.UTC)
	cases := []struct {
		name          string
		idSuffix      string
		clientContext []byte
	}{
		{name: "empty", idSuffix: "empty", clientContext: []byte(`{}`)},
		{name: "single field", idSuffix: "single-field", clientContext: []byte(`{"platform":"ios"}`)},
		{name: "recommended fields", idSuffix: "recommended-fields", clientContext: []byte(`{"platform":"ios","app_version":"1.3.0","os_version":"18.5","device_model":"iPhone16,2"}`)},
		{name: "extra fields", idSuffix: "extra-fields", clientContext: []byte(`{"platform":"ios","app_version":"1.3.0","locale":"en-US","timezone":"America/Los_Angeles"}`)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			writer := &fakeRawEventWriter{}
			usecase := service.NewRecordQuizAttemptUsecase(writer)

			_, err := usecase.Execute(context.Background(), dto.RecordQuizAttemptRequest{
				UserID:              "11111111-1111-1111-1111-111111111111",
				ClientContext:       tc.clientContext,
				ClientEventID:       "quiz-" + tc.idSuffix,
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
			if len(writer.quizEvents) != 1 {
				t.Fatalf("writer quiz events = %d, want 1", len(writer.quizEvents))
			}
		})
	}
}

func TestRecordQuizAttemptRejectsNonObjectClientContext(t *testing.T) {
	now := time.Date(2026, 5, 15, 10, 0, 0, 0, time.UTC)
	cases := []struct {
		name          string
		clientContext []byte
	}{
		{name: "array", clientContext: []byte(`[]`)},
		{name: "string", clientContext: []byte(`"ios"`)},
		{name: "number", clientContext: []byte(`123`)},
		{name: "null", clientContext: []byte(`null`)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			writer := &fakeRawEventWriter{}
			usecase := service.NewRecordQuizAttemptUsecase(writer)

			_, err := usecase.Execute(context.Background(), dto.RecordQuizAttemptRequest{
				UserID:              "11111111-1111-1111-1111-111111111111",
				ClientContext:       tc.clientContext,
				ClientEventID:       "quiz-" + tc.name,
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
			if err == nil {
				t.Fatalf("Execute() error = nil, want validation error")
			}
			if !service.IsValidationError(err) {
				t.Fatalf("Execute() error = %v, want typed validation error", err)
			}
			if len(writer.quizEvents) != 0 {
				t.Fatalf("writer quiz events = %d, want 0", len(writer.quizEvents))
			}
		})
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
	if !service.IsValidationError(err) {
		t.Fatalf("Execute() error = %v, want typed validation error", err)
	}
	if len(writer.quizEvents) != 0 || len(writer.interactions) != 0 {
		t.Fatalf("writer was called quiz=%d interactions=%d, want no writes", len(writer.quizEvents), len(writer.interactions))
	}
}

func TestRecordQuizAttemptValidatesTriggerType(t *testing.T) {
	now := time.Date(2026, 5, 15, 10, 0, 0, 0, time.UTC)
	validTriggers := []string{"video_end", "lookup_practice", "feed_review", "mid_video", "manual"}
	for _, triggerType := range validTriggers {
		t.Run("valid "+triggerType, func(t *testing.T) {
			writer := &fakeRawEventWriter{}
			usecase := service.NewRecordQuizAttemptUsecase(writer)

			_, err := usecase.Execute(context.Background(), dto.RecordQuizAttemptRequest{
				UserID:              "11111111-1111-1111-1111-111111111111",
				ClientEventID:       "quiz-" + triggerType,
				QuestionID:          "33333333-3333-3333-3333-333333333333",
				CoarseUnitID:        101,
				TriggerType:         triggerType,
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
			if len(writer.quizEvents) != 1 {
				t.Fatalf("writer quiz events = %d, want 1", len(writer.quizEvents))
			}
		})
	}

	for _, triggerType := range []string{"practice_now", "scheduled_review"} {
		t.Run("invalid "+triggerType, func(t *testing.T) {
			writer := &fakeRawEventWriter{}
			usecase := service.NewRecordQuizAttemptUsecase(writer)

			_, err := usecase.Execute(context.Background(), dto.RecordQuizAttemptRequest{
				UserID:              "11111111-1111-1111-1111-111111111111",
				ClientEventID:       "quiz-" + triggerType,
				QuestionID:          "33333333-3333-3333-3333-333333333333",
				CoarseUnitID:        101,
				TriggerType:         triggerType,
				SelectedOptionIDs:   []string{"correct"},
				SelectionIntervalMS: []int32{1000},
				IsFirstTryCorrect:   true,
				TotalElapsedMS:      1000,
				ShownAt:             now.Add(-time.Second),
				CompletedAt:         now,
			})
			if err == nil {
				t.Fatalf("Execute() error = nil, want validation error")
			}
			if !service.IsValidationError(err) {
				t.Fatalf("Execute() error = %v, want typed validation error", err)
			}
			if len(writer.quizEvents) != 0 {
				t.Fatalf("writer quiz events = %d, want 0", len(writer.quizEvents))
			}
		})
	}
}
