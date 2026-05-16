package service_test

import (
	"context"
	"testing"
	"time"

	"learning-video-recommendation-system/internal/catalog/application/dto"
	"learning-video-recommendation-system/internal/catalog/application/service"
	"learning-video-recommendation-system/internal/catalog/domain/model"
)

func TestRecordVideoWatchProgressRejectsInvalidInput(t *testing.T) {
	now := time.Date(2026, 5, 16, 12, 0, 0, 0, time.UTC)
	recorder := &fakeWatchProgressRecorder{}
	usecase := service.NewRecordVideoWatchProgressUsecase(recorder, service.WithNow(func() time.Time { return now }))

	_, err := usecase.Execute(context.Background(), dto.RecordVideoWatchProgressRequest{
		UserID:         "11111111-1111-1111-1111-111111111111",
		VideoID:        "22222222-2222-2222-2222-222222222222",
		WatchSessionID: "33333333-3333-3333-3333-333333333333",
		PositionMS:     -1,
		ActiveWatchMS:  100,
		OccurredAt:     now,
		ClientContext:  []byte(`{}`),
		Metadata:       []byte(`{}`),
		SourceSurface:  "fullscreen",
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !service.IsValidationError(err) {
		t.Fatalf("expected validation error, got %T %v", err, err)
	}
	if recorder.called {
		t.Fatal("recorder should not be called for invalid request")
	}
}

func TestRecordVideoWatchProgressDefaultsAndNormalizesInput(t *testing.T) {
	location := time.FixedZone("UTC-7", -7*60*60)
	occurredAt := time.Date(2026, 5, 16, 5, 0, 0, 0, location)
	recorder := &fakeWatchProgressRecorder{}
	usecase := service.NewRecordVideoWatchProgressUsecase(recorder, service.WithNow(func() time.Time { return occurredAt.UTC() }))

	response, err := usecase.Execute(context.Background(), dto.RecordVideoWatchProgressRequest{
		UserID:         "11111111-1111-1111-1111-111111111111",
		VideoID:        "22222222-2222-2222-2222-222222222222",
		WatchSessionID: "33333333-3333-3333-3333-333333333333",
		PositionMS:     5000,
		ActiveWatchMS:  7000,
		OccurredAt:     occurredAt,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if !response.Accepted {
		t.Fatal("expected accepted response")
	}
	if !recorder.called {
		t.Fatal("expected recorder call")
	}
	if recorder.request.OccurredAt.Location() != time.UTC {
		t.Fatalf("expected UTC occurred_at, got %s", recorder.request.OccurredAt.Location())
	}
	if !recorder.request.OccurredAt.Equal(occurredAt) {
		t.Fatalf("expected same instant, got %s want %s", recorder.request.OccurredAt, occurredAt)
	}
	if string(recorder.request.ClientContext) != "{}" {
		t.Fatalf("expected default client_context {}, got %s", recorder.request.ClientContext)
	}
	if string(recorder.request.Metadata) != "{}" {
		t.Fatalf("expected default metadata {}, got %s", recorder.request.Metadata)
	}
}

func TestRecordVideoWatchProgressMapsRepositoryErrors(t *testing.T) {
	recorder := &fakeWatchProgressRecorder{err: service.NotFoundError("video not found")}
	usecase := service.NewRecordVideoWatchProgressUsecase(recorder)

	_, err := usecase.Execute(context.Background(), validRequest())
	if err == nil {
		t.Fatal("expected error")
	}
	if !service.IsNotFoundError(err) {
		t.Fatalf("expected not found error, got %T %v", err, err)
	}
}

func TestRecordVideoWatchProgressRejectsMalformedJSONObjects(t *testing.T) {
	usecase := service.NewRecordVideoWatchProgressUsecase(&fakeWatchProgressRecorder{})

	_, err := usecase.Execute(context.Background(), dto.RecordVideoWatchProgressRequest{
		UserID:         "11111111-1111-1111-1111-111111111111",
		VideoID:        "22222222-2222-2222-2222-222222222222",
		WatchSessionID: "33333333-3333-3333-3333-333333333333",
		PositionMS:     1,
		ActiveWatchMS:  1,
		OccurredAt:     time.Now().UTC(),
		ClientContext:  []byte(`[]`),
		Metadata:       []byte(`{}`),
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !service.IsValidationError(err) {
		t.Fatalf("expected validation error, got %T %v", err, err)
	}
}

func validRequest() dto.RecordVideoWatchProgressRequest {
	return dto.RecordVideoWatchProgressRequest{
		UserID:         "11111111-1111-1111-1111-111111111111",
		VideoID:        "22222222-2222-2222-2222-222222222222",
		WatchSessionID: "33333333-3333-3333-3333-333333333333",
		PositionMS:     1000,
		ActiveWatchMS:  2000,
		OccurredAt:     time.Now().UTC(),
		ClientContext:  []byte(`{}`),
		Metadata:       []byte(`{}`),
	}
}

type fakeWatchProgressRecorder struct {
	called  bool
	request model.VideoWatchProgress
	err     error
}

func (f *fakeWatchProgressRecorder) RecordVideoWatchProgress(ctx context.Context, request model.VideoWatchProgress) (model.VideoWatchProgressResult, error) {
	f.called = true
	f.request = request
	if f.err != nil {
		return model.VideoWatchProgressResult{}, f.err
	}
	return model.VideoWatchProgressResult{Accepted: true}, nil
}
