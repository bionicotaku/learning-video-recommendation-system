package service_test

import (
	"context"
	"testing"
	"time"

	"learning-video-recommendation-system/internal/analytics/application/dto"
	"learning-video-recommendation-system/internal/analytics/application/service"
	"learning-video-recommendation-system/internal/analytics/domain/model"
)

func TestRecordLearningInteractionsBatchWritesInteractions(t *testing.T) {
	now := time.Date(2026, 5, 15, 10, 0, 0, 0, time.UTC)
	unitID := int64(101)
	writer := &fakeRawEventWriter{
		interactionResults: []model.RawEventWriteResult{
			{ClientEventID: "interaction-1", EventID: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", Inserted: true},
			{ClientEventID: "interaction-2", EventID: "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb", Inserted: false},
		},
	}
	usecase := service.NewRecordLearningInteractionsBatchUsecase(writer)

	response, err := usecase.Execute(context.Background(), dto.RecordLearningInteractionsBatchRequest{
		UserID:        "11111111-1111-1111-1111-111111111111",
		ClientContext: []byte(`{"platform":"ios"}`),
		Events: []dto.LearningInteractionEventInput{
			{
				ClientEventID: "interaction-1",
				EventType:     "lookup",
				SourceSurface: "video_subtitle",
				CoarseUnitID:  &unitID,
				TokenText:     "test",
				OccurredAt:    now,
				EventPayload:  []byte(`{"lookup_visible_ms":5000}`),
			},
			{
				ClientEventID: "interaction-2",
				EventType:     "self_mark_mastered",
				SourceSurface: "word_detail",
				CoarseUnitID:  &unitID,
				OccurredAt:    now.Add(time.Second),
			},
		},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if response.AcceptedCount != 2 || response.InsertedCount != 1 || response.DuplicateCount != 1 {
		t.Fatalf("response = %+v, want accepted=2 inserted=1 duplicate=1", response)
	}
	if len(response.AcceptedEvents) != 2 {
		t.Fatalf("accepted events = %d, want 2", len(response.AcceptedEvents))
	}
	if len(writer.interactions) != 2 {
		t.Fatalf("writer interactions = %d, want 2", len(writer.interactions))
	}
	for _, event := range writer.interactions {
		if event.UserID != "11111111-1111-1111-1111-111111111111" {
			t.Fatalf("writer user id = %q", event.UserID)
		}
	}
}

func TestRecordLearningInteractionsBatchRejectsInvalidBatchBeforeWrite(t *testing.T) {
	now := time.Date(2026, 5, 15, 10, 0, 0, 0, time.UTC)
	writer := &fakeRawEventWriter{}
	usecase := service.NewRecordLearningInteractionsBatchUsecase(writer)

	_, err := usecase.Execute(context.Background(), dto.RecordLearningInteractionsBatchRequest{
		UserID: "11111111-1111-1111-1111-111111111111",
		Events: []dto.LearningInteractionEventInput{
			{
				ClientEventID: "valid-looking",
				EventType:     "lookup",
				SourceSurface: "video_subtitle",
				TokenText:     "test",
				OccurredAt:    now,
			},
			{
				ClientEventID: "invalid",
				EventType:     "exposure",
				SourceSurface: "video_subtitle",
				OccurredAt:    now,
			},
		},
	})
	if err == nil {
		t.Fatalf("Execute() error = nil, want validation error")
	}
	if len(writer.interactions) != 0 || len(writer.quizEvents) != 0 {
		t.Fatalf("writer was called interactions=%d quiz=%d, want no writes", len(writer.interactions), len(writer.quizEvents))
	}
}
