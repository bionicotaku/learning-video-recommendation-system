package service_test

import (
	"context"
	"testing"
	"time"

	"learning-video-recommendation-system/internal/analytics/application/dto"
	"learning-video-recommendation-system/internal/analytics/application/service"
	"learning-video-recommendation-system/internal/analytics/domain/model"
)

func TestRecordSelfMarkMasteredWritesSingleRawInteraction(t *testing.T) {
	now := time.Date(2026, 5, 15, 10, 0, 0, 0, time.FixedZone("PDT", -7*60*60))
	writer := &fakeRawEventWriter{
		interactionResults: []model.RawEventWriteResult{
			{ClientEventID: "self-mark-1", EventID: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", Inserted: true},
		},
	}
	usecase := service.NewRecordSelfMarkMasteredUsecase(writer)

	response, err := usecase.Execute(context.Background(), dto.RecordSelfMarkMasteredRequest{
		UserID:        "11111111-1111-1111-1111-111111111111",
		ClientContext: []byte(`{"platform":"ios"}`),
		ClientEventID: "self-mark-1",
		CoarseUnitID:  101,
		SourceSurface: "word_detail",
		VideoID:       "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
		TokenText:     "trivial",
		OccurredAt:    now,
		EventPayload:  []byte(`{"entry":"lookup_sheet"}`),
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !response.Accepted || !response.Inserted || response.LearningInteractionEventID != "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa" {
		t.Fatalf("response = %+v, want accepted inserted self mark id", response)
	}
	if len(writer.interactions) != 1 {
		t.Fatalf("writer interactions = %d, want 1", len(writer.interactions))
	}
	event := writer.interactions[0]
	if event.EventType != "self_mark_mastered" {
		t.Fatalf("event_type = %q, want self_mark_mastered", event.EventType)
	}
	if event.CoarseUnitID == nil || *event.CoarseUnitID != 101 {
		t.Fatalf("coarse_unit_id = %v, want 101", event.CoarseUnitID)
	}
	if event.OccurredAt.Location() != time.UTC || !event.OccurredAt.Equal(now) {
		t.Fatalf("occurred_at = %v, want UTC same instant as %v", event.OccurredAt, now)
	}
}

func TestRecordSelfMarkMasteredReturnsDuplicateExistingID(t *testing.T) {
	writer := &fakeRawEventWriter{
		interactionResults: []model.RawEventWriteResult{
			{ClientEventID: "self-mark-duplicate", EventID: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", Inserted: false},
		},
	}
	usecase := service.NewRecordSelfMarkMasteredUsecase(writer)

	response, err := usecase.Execute(context.Background(), dto.RecordSelfMarkMasteredRequest{
		UserID:        "11111111-1111-1111-1111-111111111111",
		ClientEventID: "self-mark-duplicate",
		CoarseUnitID:  101,
		SourceSurface: "word_detail",
		OccurredAt:    time.Date(2026, 5, 15, 10, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !response.Accepted || response.Inserted || response.LearningInteractionEventID != "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa" {
		t.Fatalf("response = %+v, want accepted duplicate existing id", response)
	}
}

func TestRecordSelfMarkMasteredRejectsMissingRequiredFields(t *testing.T) {
	now := time.Date(2026, 5, 15, 10, 0, 0, 0, time.UTC)
	cases := []struct {
		name    string
		request dto.RecordSelfMarkMasteredRequest
	}{
		{name: "client event", request: dto.RecordSelfMarkMasteredRequest{UserID: "11111111-1111-1111-1111-111111111111", CoarseUnitID: 101, SourceSurface: "word_detail", OccurredAt: now}},
		{name: "coarse unit", request: dto.RecordSelfMarkMasteredRequest{UserID: "11111111-1111-1111-1111-111111111111", ClientEventID: "self-mark", SourceSurface: "word_detail", OccurredAt: now}},
		{name: "source surface", request: dto.RecordSelfMarkMasteredRequest{UserID: "11111111-1111-1111-1111-111111111111", ClientEventID: "self-mark", CoarseUnitID: 101, OccurredAt: now}},
		{name: "occurred at", request: dto.RecordSelfMarkMasteredRequest{UserID: "11111111-1111-1111-1111-111111111111", ClientEventID: "self-mark", CoarseUnitID: 101, SourceSurface: "word_detail"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			writer := &fakeRawEventWriter{}
			usecase := service.NewRecordSelfMarkMasteredUsecase(writer)

			_, err := usecase.Execute(context.Background(), tc.request)
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

func TestRecordSelfMarkMasteredAcceptsLooseClientContextObject(t *testing.T) {
	cases := []struct {
		name          string
		clientContext []byte
	}{
		{name: "empty", clientContext: []byte(`{}`)},
		{name: "single field", clientContext: []byte(`{"platform":"ios"}`)},
		{name: "recommended fields", clientContext: []byte(`{"platform":"ios","app_version":"1.3.0","os_version":"18.5","device_model":"iPhone16,2"}`)},
		{name: "extra fields", clientContext: []byte(`{"platform":"ios","locale":"en-US"}`)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			writer := &fakeRawEventWriter{}
			usecase := service.NewRecordSelfMarkMasteredUsecase(writer)

			_, err := usecase.Execute(context.Background(), dto.RecordSelfMarkMasteredRequest{
				UserID:        "11111111-1111-1111-1111-111111111111",
				ClientContext: tc.clientContext,
				ClientEventID: "self-mark-" + tc.name,
				CoarseUnitID:  101,
				SourceSurface: "word_detail",
				OccurredAt:    time.Date(2026, 5, 15, 10, 0, 0, 0, time.UTC),
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
