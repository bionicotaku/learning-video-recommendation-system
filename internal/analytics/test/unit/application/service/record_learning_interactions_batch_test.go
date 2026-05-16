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
	sentenceIndex := int32(12)
	spanIndex := int32(4)
	writer := &fakeRawEventWriter{
		interactionResults: []model.RawEventWriteResult{
			{ClientEventID: "interaction-1", EventID: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", Inserted: true},
			{ClientEventID: "interaction-2", EventID: "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb", Inserted: false},
		},
	}
	usecase := service.NewRecordLearningInteractionsBatchUsecase(writer)

	response, err := usecase.Execute(context.Background(), dto.RecordLearningInteractionsBatchRequest{
		UserID:              "11111111-1111-1111-1111-111111111111",
		ClientContext:       []byte(`{"platform":"ios"}`),
		VideoID:             "22222222-2222-2222-2222-222222222222",
		WatchSessionID:      "33333333-3333-3333-3333-333333333333",
		RecommendationRunID: "44444444-4444-4444-4444-444444444444",
		Events: []dto.LearningInteractionEventInput{
			{
				ClientEventID: "interaction-1",
				EventType:     "lookup",
				SourceSurface: "video_subtitle",
				CoarseUnitID:  &unitID,
				TokenText:     "test",
				SentenceIndex: &sentenceIndex,
				SpanIndex:     &spanIndex,
				OccurredAt:    now,
				EventPayload:  []byte(`{"lookup_visible_ms":5000}`),
			},
			{
				ClientEventID: "interaction-2",
				EventType:     "exposure",
				SourceSurface: "video_subtitle",
				CoarseUnitID:  &unitID,
				SentenceIndex: &sentenceIndex,
				SpanIndex:     &spanIndex,
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
		if event.VideoID != "22222222-2222-2222-2222-222222222222" || event.WatchSessionID != "33333333-3333-3333-3333-333333333333" || event.RecommendationRunID != "44444444-4444-4444-4444-444444444444" {
			t.Fatalf("writer context = video:%q watch:%q recommendation:%q", event.VideoID, event.WatchSessionID, event.RecommendationRunID)
		}
	}
}

func TestRecordLearningInteractionsBatchRejectsSelfMarkMastered(t *testing.T) {
	now := time.Date(2026, 5, 15, 10, 0, 0, 0, time.UTC)
	unitID := int64(101)
	writer := &fakeRawEventWriter{}
	usecase := service.NewRecordLearningInteractionsBatchUsecase(writer)

	_, err := usecase.Execute(context.Background(), dto.RecordLearningInteractionsBatchRequest{
		UserID:         "11111111-1111-1111-1111-111111111111",
		ClientContext:  []byte(`{"platform":"ios"}`),
		VideoID:        "22222222-2222-2222-2222-222222222222",
		WatchSessionID: "33333333-3333-3333-3333-333333333333",
		Events: []dto.LearningInteractionEventInput{
			{
				ClientEventID: "self-mark-1",
				EventType:     "self_mark_mastered",
				SourceSurface: "word_detail",
				CoarseUnitID:  &unitID,
				OccurredAt:    now,
			},
		},
	})
	if err == nil {
		t.Fatalf("Execute() error = nil, want validation error")
	}
	if !service.IsValidationError(err) {
		t.Fatalf("Execute() error = %v, want typed validation error", err)
	}
	if len(writer.interactions) != 0 {
		t.Fatalf("writer interactions = %d, want 0", len(writer.interactions))
	}
}

func TestRecordLearningInteractionsBatchRejectsMissingVideoContext(t *testing.T) {
	now := time.Date(2026, 5, 15, 10, 0, 0, 0, time.UTC)
	unitID := int64(101)
	sentenceIndex := int32(12)
	spanIndex := int32(4)
	cases := []struct {
		name           string
		videoID        string
		watchSessionID string
	}{
		{name: "video", watchSessionID: "33333333-3333-3333-3333-333333333333"},
		{name: "watch session", videoID: "22222222-2222-2222-2222-222222222222"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			writer := &fakeRawEventWriter{}
			usecase := service.NewRecordLearningInteractionsBatchUsecase(writer)

			_, err := usecase.Execute(context.Background(), dto.RecordLearningInteractionsBatchRequest{
				UserID:         "11111111-1111-1111-1111-111111111111",
				VideoID:        tc.videoID,
				WatchSessionID: tc.watchSessionID,
				Events: []dto.LearningInteractionEventInput{
					{
						ClientEventID: "lookup-1",
						EventType:     "lookup",
						SourceSurface: "video_subtitle",
						CoarseUnitID:  &unitID,
						TokenText:     "test",
						SentenceIndex: &sentenceIndex,
						SpanIndex:     &spanIndex,
						OccurredAt:    now,
					},
				},
			})
			if err == nil {
				t.Fatalf("Execute() error = nil, want validation error")
			}
			if !service.IsValidationError(err) {
				t.Fatalf("Execute() error = %v, want typed validation error", err)
			}
			if len(writer.interactions) != 0 {
				t.Fatalf("writer interactions = %d, want 0", len(writer.interactions))
			}
		})
	}
}

func TestRecordLearningInteractionsBatchNormalizesOccurredAtToUTC(t *testing.T) {
	localTime := time.Date(2026, 5, 15, 10, 0, 0, 0, time.FixedZone("PDT", -7*60*60))
	unitID := int64(101)
	sentenceIndex := int32(12)
	spanIndex := int32(4)
	writer := &fakeRawEventWriter{}
	usecase := service.NewRecordLearningInteractionsBatchUsecase(writer)

	_, err := usecase.Execute(context.Background(), dto.RecordLearningInteractionsBatchRequest{
		UserID:         "11111111-1111-1111-1111-111111111111",
		ClientContext:  []byte(`{"platform":"ios","app_version":"1.3.0","os_version":"18.5","device_model":"iPhone16,2"}`),
		VideoID:        "22222222-2222-2222-2222-222222222222",
		WatchSessionID: "33333333-3333-3333-3333-333333333333",
		Events: []dto.LearningInteractionEventInput{
			{
				ClientEventID: "interaction-utc",
				EventType:     "lookup",
				SourceSurface: "video_subtitle",
				CoarseUnitID:  &unitID,
				TokenText:     "test",
				SentenceIndex: &sentenceIndex,
				SpanIndex:     &spanIndex,
				OccurredAt:    localTime,
				EventPayload:  []byte(`{"lookup_visible_ms":5000}`),
			},
		},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(writer.interactions) != 1 {
		t.Fatalf("writer interactions = %d, want 1", len(writer.interactions))
	}
	got := writer.interactions[0].OccurredAt
	if got.Location() != time.UTC {
		t.Fatalf("OccurredAt location = %v, want UTC", got.Location())
	}
	if !got.Equal(localTime) {
		t.Fatalf("OccurredAt = %v, want same instant as %v", got, localTime)
	}
}

func TestRecordLearningInteractionsBatchAcceptsLooseClientContextObject(t *testing.T) {
	now := time.Date(2026, 5, 15, 10, 0, 0, 0, time.UTC)
	unitID := int64(101)
	sentenceIndex := int32(12)
	spanIndex := int32(4)
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
			usecase := service.NewRecordLearningInteractionsBatchUsecase(writer)

			_, err := usecase.Execute(context.Background(), dto.RecordLearningInteractionsBatchRequest{
				UserID:         "11111111-1111-1111-1111-111111111111",
				ClientContext:  tc.clientContext,
				VideoID:        "22222222-2222-2222-2222-222222222222",
				WatchSessionID: "33333333-3333-3333-3333-333333333333",
				Events: []dto.LearningInteractionEventInput{
					{
						ClientEventID: "interaction-" + tc.idSuffix,
						EventType:     "lookup",
						SourceSurface: "video_subtitle",
						CoarseUnitID:  &unitID,
						TokenText:     "test",
						SentenceIndex: &sentenceIndex,
						SpanIndex:     &spanIndex,
						OccurredAt:    now,
						EventPayload:  []byte(`{"lookup_visible_ms":5000}`),
					},
				},
			})
			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			if len(writer.interactions) != 1 {
				t.Fatalf("writer interactions = %d, want 1", len(writer.interactions))
			}
		})
	}
}

func TestRecordLearningInteractionsBatchRejectsNonObjectClientContext(t *testing.T) {
	now := time.Date(2026, 5, 15, 10, 0, 0, 0, time.UTC)
	unitID := int64(101)
	sentenceIndex := int32(12)
	spanIndex := int32(4)
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
			usecase := service.NewRecordLearningInteractionsBatchUsecase(writer)

			_, err := usecase.Execute(context.Background(), dto.RecordLearningInteractionsBatchRequest{
				UserID:         "11111111-1111-1111-1111-111111111111",
				ClientContext:  tc.clientContext,
				VideoID:        "22222222-2222-2222-2222-222222222222",
				WatchSessionID: "33333333-3333-3333-3333-333333333333",
				Events: []dto.LearningInteractionEventInput{
					{
						ClientEventID: "interaction-" + tc.name,
						EventType:     "lookup",
						SourceSurface: "video_subtitle",
						CoarseUnitID:  &unitID,
						TokenText:     "test",
						SentenceIndex: &sentenceIndex,
						SpanIndex:     &spanIndex,
						OccurredAt:    now,
						EventPayload:  []byte(`{"lookup_visible_ms":5000}`),
					},
				},
			})
			if err == nil {
				t.Fatalf("Execute() error = nil, want validation error")
			}
			if !service.IsValidationError(err) {
				t.Fatalf("Execute() error = %v, want typed validation error", err)
			}
			if len(writer.interactions) != 0 {
				t.Fatalf("writer interactions = %d, want 0", len(writer.interactions))
			}
		})
	}
}

func TestRecordLearningInteractionsBatchRejectsInvalidBatchBeforeWrite(t *testing.T) {
	now := time.Date(2026, 5, 15, 10, 0, 0, 0, time.UTC)
	sentenceIndex := int32(12)
	spanIndex := int32(4)
	writer := &fakeRawEventWriter{}
	usecase := service.NewRecordLearningInteractionsBatchUsecase(writer)

	_, err := usecase.Execute(context.Background(), dto.RecordLearningInteractionsBatchRequest{
		UserID:         "11111111-1111-1111-1111-111111111111",
		VideoID:        "22222222-2222-2222-2222-222222222222",
		WatchSessionID: "33333333-3333-3333-3333-333333333333",
		Events: []dto.LearningInteractionEventInput{
			{
				ClientEventID: "valid-looking",
				EventType:     "lookup",
				SourceSurface: "video_subtitle",
				TokenText:     "test",
				SentenceIndex: &sentenceIndex,
				SpanIndex:     &spanIndex,
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
	if !service.IsValidationError(err) {
		t.Fatalf("Execute() error = %v, want typed validation error", err)
	}
	if len(writer.interactions) != 0 || len(writer.quizEvents) != 0 {
		t.Fatalf("writer was called interactions=%d quiz=%d, want no writes", len(writer.interactions), len(writer.quizEvents))
	}
}

func TestRecordLearningInteractionsBatchRequiresSentenceAndSpanIndexesForCurrentEventTypes(t *testing.T) {
	now := time.Date(2026, 5, 15, 10, 0, 0, 0, time.UTC)
	unitID := int64(101)
	sentenceIndex := int32(12)
	spanIndex := int32(4)
	cases := []struct {
		name  string
		event dto.LearningInteractionEventInput
	}{
		{
			name: "lookup sentence_index",
			event: dto.LearningInteractionEventInput{
				ClientEventID: "lookup-1",
				EventType:     "lookup",
				SourceSurface: "video_subtitle",
				TokenText:     "test",
				SpanIndex:     &spanIndex,
				OccurredAt:    now,
			},
		},
		{
			name: "lookup span_index",
			event: dto.LearningInteractionEventInput{
				ClientEventID: "lookup-1",
				EventType:     "lookup",
				SourceSurface: "video_subtitle",
				TokenText:     "test",
				SentenceIndex: &sentenceIndex,
				OccurredAt:    now,
			},
		},
		{
			name: "exposure sentence_index",
			event: dto.LearningInteractionEventInput{
				ClientEventID: "exposure-1",
				EventType:     "exposure",
				SourceSurface: "video_subtitle",
				CoarseUnitID:  &unitID,
				SpanIndex:     &spanIndex,
				OccurredAt:    now,
			},
		},
		{
			name: "exposure span_index",
			event: dto.LearningInteractionEventInput{
				ClientEventID: "exposure-1",
				EventType:     "exposure",
				SourceSurface: "video_subtitle",
				CoarseUnitID:  &unitID,
				SentenceIndex: &sentenceIndex,
				OccurredAt:    now,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			writer := &fakeRawEventWriter{}
			usecase := service.NewRecordLearningInteractionsBatchUsecase(writer)

			_, err := usecase.Execute(context.Background(), dto.RecordLearningInteractionsBatchRequest{
				UserID:         "11111111-1111-1111-1111-111111111111",
				ClientContext:  []byte(`{"platform":"ios"}`),
				VideoID:        "22222222-2222-2222-2222-222222222222",
				WatchSessionID: "33333333-3333-3333-3333-333333333333",
				Events:         []dto.LearningInteractionEventInput{tc.event},
			})
			if err == nil {
				t.Fatalf("Execute() error = nil, want validation error")
			}
			if !service.IsValidationError(err) {
				t.Fatalf("Execute() error = %v, want typed validation error", err)
			}
			if len(writer.interactions) != 0 {
				t.Fatalf("writer interactions = %d, want 0", len(writer.interactions))
			}
		})
	}
}
