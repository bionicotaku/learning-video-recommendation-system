package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	learningdto "learning-video-recommendation-system/internal/learningengine/application/dto"
	learningenum "learning-video-recommendation-system/internal/learningengine/domain/enum"
	"learning-video-recommendation-system/internal/learningengine/normalizer/application/dto"
	normalizerrepo "learning-video-recommendation-system/internal/learningengine/normalizer/application/repository"
	"learning-video-recommendation-system/internal/learningengine/normalizer/application/service"
	"learning-video-recommendation-system/internal/learningengine/normalizer/domain/model"
)

func TestNormalizePendingEventsDefaultsToAllAndGroupsByUser(t *testing.T) {
	t1 := time.Date(2026, 5, 15, 10, 0, 0, 0, time.UTC)
	quizReader := &fakeQuizReader{events: []model.RawQuizEvent{
		validRawQuiz("11111111-1111-1111-1111-111111111111", "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", 101, t1),
	}}
	interactionReader := &fakeInteractionReader{events: []model.RawLearningInteraction{
		validRawInteraction("22222222-2222-2222-2222-222222222222", "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb", 102, learningenum.EventLookup, t1.Add(time.Second)),
	}}
	recorder := &fakeRecorder{}
	usecase := service.NewNormalizePendingEventsUsecase(quizReader, interactionReader, recorder)

	response, err := usecase.Execute(context.Background(), dto.NormalizePendingEventsRequest{})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !quizReader.called || !interactionReader.called {
		t.Fatalf("readers called quiz=%v interaction=%v, want both true", quizReader.called, interactionReader.called)
	}
	if quizReader.lastFilter.Limit != dto.DefaultNormalizeLimit || interactionReader.lastFilter.Limit != dto.DefaultNormalizeLimit {
		t.Fatalf("limit = %d/%d, want default %d", quizReader.lastFilter.Limit, interactionReader.lastFilter.Limit, dto.DefaultNormalizeLimit)
	}
	if response.ReadRawCount != 2 || response.NormalizedEventCount != 2 || response.RecordedEventCount != 2 {
		t.Fatalf("response = %+v, want read=2 normalized=2 recorded=2", response)
	}
	if response.RecordedUserBatchCount != 2 {
		t.Fatalf("RecordedUserBatchCount = %d, want 2", response.RecordedUserBatchCount)
	}
	if len(recorder.requests) != 2 {
		t.Fatalf("recorder requests = %d, want 2", len(recorder.requests))
	}
}

func TestNormalizePendingEventsSkipsInvalidRawFactsWithoutRecording(t *testing.T) {
	quizReader := &fakeQuizReader{events: []model.RawQuizEvent{{EventID: "missing-required-fields"}}}
	interactionReader := &fakeInteractionReader{}
	recorder := &fakeRecorder{}
	usecase := service.NewNormalizePendingEventsUsecase(quizReader, interactionReader, recorder)

	response, err := usecase.Execute(context.Background(), dto.NormalizePendingEventsRequest{SourceKind: dto.SourceKindQuiz})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if response.SkippedCount != 1 || response.RecordedEventCount != 0 {
		t.Fatalf("response = %+v, want skipped=1 recorded=0", response)
	}
	if len(recorder.requests) != 0 {
		t.Fatalf("recorder requests = %d, want 0", len(recorder.requests))
	}
}

func TestNormalizePendingEventsFailFastOnRecorderError(t *testing.T) {
	wantErr := errors.New("record failed")
	now := time.Date(2026, 5, 15, 10, 0, 0, 0, time.UTC)
	quizReader := &fakeQuizReader{events: []model.RawQuizEvent{
		validRawQuiz("11111111-1111-1111-1111-111111111111", "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", 101, now),
	}}
	interactionReader := &fakeInteractionReader{}
	recorder := &fakeRecorder{err: wantErr}
	usecase := service.NewNormalizePendingEventsUsecase(quizReader, interactionReader, recorder)

	response, err := usecase.Execute(context.Background(), dto.NormalizePendingEventsRequest{SourceKind: dto.SourceKindQuiz})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Execute() error = %v, want %v", err, wantErr)
	}
	if response.ErrorCount != 1 || response.NormalizedEventCount != 1 || response.RecordedEventCount != 0 {
		t.Fatalf("response = %+v, want error=1 normalized=1 recorded=0", response)
	}
}

type fakeQuizReader struct {
	events     []model.RawQuizEvent
	called     bool
	lastFilter normalizerrepo.PendingRawEventFilter
}

func (r *fakeQuizReader) ListPendingQuizEvents(_ context.Context, filter normalizerrepo.PendingRawEventFilter) ([]model.RawQuizEvent, error) {
	r.called = true
	r.lastFilter = filter
	return r.events, nil
}

type fakeInteractionReader struct {
	events     []model.RawLearningInteraction
	called     bool
	lastFilter normalizerrepo.PendingRawEventFilter
}

func (r *fakeInteractionReader) ListPendingLearningInteractions(_ context.Context, filter normalizerrepo.PendingRawEventFilter) ([]model.RawLearningInteraction, error) {
	r.called = true
	r.lastFilter = filter
	return r.events, nil
}

type fakeRecorder struct {
	requests []learningdto.RecordLearningEventsRequest
	err      error
}

func (r *fakeRecorder) Execute(_ context.Context, request learningdto.RecordLearningEventsRequest) (learningdto.RecordLearningEventsResponse, error) {
	r.requests = append(r.requests, request)
	if r.err != nil {
		return learningdto.RecordLearningEventsResponse{}, r.err
	}
	return learningdto.RecordLearningEventsResponse{RecordedCount: len(request.Events)}, nil
}

func validRawQuiz(userID, eventID string, unitID int64, completedAt time.Time) model.RawQuizEvent {
	return model.RawQuizEvent{
		EventID:             eventID,
		UserID:              userID,
		QuestionID:          "33333333-3333-3333-3333-333333333333",
		CoarseUnitID:        unitID,
		TriggerType:         "manual",
		SelectedOptionIDs:   []string{"correct"},
		SelectionIntervalMS: []int32{1000},
		IsFirstTryCorrect:   true,
		TotalElapsedMS:      1000,
		ShownAt:             completedAt.Add(-time.Second),
		CompletedAt:         completedAt,
	}
}

func validRawInteraction(userID, eventID string, unitID int64, eventType string, occurredAt time.Time) model.RawLearningInteraction {
	return model.RawLearningInteraction{
		EventID:       eventID,
		UserID:        userID,
		EventType:     eventType,
		SourceSurface: "video_subtitle",
		CoarseUnitID:  unitID,
		OccurredAt:    occurredAt,
		EventPayload:  []byte(`{}`),
	}
}
