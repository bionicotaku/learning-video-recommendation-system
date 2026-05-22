package service_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	analyticsdto "learning-video-recommendation-system/internal/analytics/application/dto"
	analyticsservice "learning-video-recommendation-system/internal/analytics/application/service"
	apvdto "learning-video-recommendation-system/internal/api/application/dto"
	apiservice "learning-video-recommendation-system/internal/api/application/service"
	normalizerdto "learning-video-recommendation-system/internal/learningengine/normalizer/application/dto"
	learningdto "learning-video-recommendation-system/internal/learningengine/reducer/application/dto"
)

func TestRecordLearningInteractionsBatchReturnsRawAcceptedWhenNormalizerFails(t *testing.T) {
	rawWriter := &fakeInteractionRawWriter{
		response: analyticsdto.RecordLearningInteractionsBatchResponse{
			AcceptedCount:  2,
			InsertedCount:  1,
			DuplicateCount: 1,
			AcceptedEvents: []analyticsdto.AcceptedLearningInteractionEvent{
				{ClientEventID: "event-1", LearningInteractionEventID: "11111111-1111-1111-1111-111111111111", Inserted: true},
				{ClientEventID: "event-2", LearningInteractionEventID: "22222222-2222-2222-2222-222222222222", Inserted: false},
			},
		},
	}
	normalizer := &fakeInteractionNormalizer{err: errors.New("normalizer down")}
	service := apiservice.NewRecordLearningInteractionsBatchService(rawWriter, normalizer, discardLogger())

	response, err := service.Execute(context.Background(), apvdto.RecordLearningInteractionsBatchRequest{
		UserID: "user-1",
		Events: []apvdto.LearningInteractionEvent{
			{ClientEventID: "event-1"},
			{ClientEventID: "event-2"},
		},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if response.AcceptedCount != 2 || response.InsertedCount != 1 || response.DuplicateCount != 1 || len(response.Events) != 2 {
		t.Fatalf("response = %+v, want raw accepted response", response)
	}
	if normalizer.request.UserID != "user-1" || len(normalizer.request.LearningInteractionEventIDs) != 2 {
		t.Fatalf("normalizer request = %+v, want raw interaction ids", normalizer.request)
	}
}

func TestRecordQuizAttemptPassesRawIDToNormalizer(t *testing.T) {
	rawWriter := &fakeQuizRawWriter{
		response: analyticsdto.RecordQuizAttemptResponse{
			Accepted:    true,
			QuizEventID: "33333333-3333-3333-3333-333333333333",
			Inserted:    true,
		},
	}
	normalizer := &fakeQuizNormalizer{}
	service := apiservice.NewRecordQuizAttemptService(rawWriter, normalizer, discardLogger())

	response, err := service.Execute(context.Background(), apvdto.RecordQuizAttemptRequest{
		UserID:              "user-1",
		ClientEventID:       "quiz-1",
		QuestionID:          "44444444-4444-4444-4444-444444444444",
		CoarseUnitID:        101,
		TriggerType:         "lookup_practice",
		SelectedOptionIDs:   []string{"correct"},
		SelectionIntervalMS: []int32{1000},
		IsFirstTryCorrect:   true,
		TotalElapsedMS:      1000,
		ShownAt:             time.Date(2026, 5, 15, 10, 0, 0, 0, time.UTC),
		CompletedAt:         time.Date(2026, 5, 15, 10, 0, 1, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !response.Accepted || response.QuizEventID != "33333333-3333-3333-3333-333333333333" {
		t.Fatalf("response = %+v, want quiz raw response", response)
	}
	if normalizer.request.UserID != "user-1" || normalizer.request.QuizEventID != "33333333-3333-3333-3333-333333333333" {
		t.Fatalf("normalizer request = %+v, want quiz raw id", normalizer.request)
	}
}

func TestRecordSelfMarkMasteredPassesRawIDToNormalizer(t *testing.T) {
	rawWriter := &fakeSelfMarkRawWriter{
		response: analyticsdto.RecordSelfMarkMasteredResponse{
			Accepted:                   true,
			LearningInteractionEventID: "55555555-5555-5555-5555-555555555555",
			Inserted:                   true,
		},
	}
	normalizer := &fakeSelfMarkNormalizer{}
	stateReader := &fakeUserUnitStateReader{response: learningdto.GetUserUnitStateResponse{Found: true}}
	service := apiservice.NewRecordSelfMarkMasteredService(rawWriter, normalizer, stateReader, discardLogger())

	response, err := service.Execute(context.Background(), apvdto.RecordSelfMarkMasteredRequest{
		UserID:        "user-1",
		ClientEventID: "self-mark-1",
		CoarseUnitID:  101,
		SourceSurface: "word_detail",
		OccurredAt:    time.Date(2026, 5, 15, 10, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !response.Accepted || response.LearningInteractionEventID != "55555555-5555-5555-5555-555555555555" {
		t.Fatalf("response = %+v, want self mark raw response", response)
	}
	if normalizer.request.UserID != "user-1" || normalizer.request.LearningInteractionEventID != "55555555-5555-5555-5555-555555555555" {
		t.Fatalf("normalizer request = %+v, want self mark raw id", normalizer.request)
	}
}

func TestRecordSelfMarkMasteredRejectsMissingUserUnitStateBeforeRawWrite(t *testing.T) {
	rawWriter := &fakeSelfMarkRawWriter{
		response: analyticsdto.RecordSelfMarkMasteredResponse{
			Accepted:                   true,
			LearningInteractionEventID: "55555555-5555-5555-5555-555555555555",
			Inserted:                   true,
		},
	}
	stateReader := &fakeUserUnitStateReader{response: learningdto.GetUserUnitStateResponse{Found: false}}
	service := apiservice.NewRecordSelfMarkMasteredService(rawWriter, &fakeSelfMarkNormalizer{}, stateReader, discardLogger())

	_, err := service.Execute(context.Background(), apvdto.RecordSelfMarkMasteredRequest{
		UserID:        "user-1",
		ClientEventID: "self-mark-1",
		CoarseUnitID:  101,
		SourceSurface: "word_detail",
		OccurredAt:    time.Date(2026, 5, 15, 10, 0, 0, 0, time.UTC),
	})
	if !apiservice.IsInvalidRequest(err) {
		t.Fatalf("Execute() error = %v, want invalid request", err)
	}
	if rawWriter.called {
		t.Fatalf("raw writer was called for missing user unit state")
	}
	if stateReader.request.UserID != "user-1" || stateReader.request.CoarseUnitID != 101 {
		t.Fatalf("state reader request = %+v, want user/unit lookup", stateReader.request)
	}
}

func TestRecordQuizAttemptMapsAnalyticsValidationErrorToInvalidRequest(t *testing.T) {
	rawWriter := &fakeQuizRawWriter{err: &analyticsservice.ValidationError{Message: "trigger_type is unsupported"}}
	service := apiservice.NewRecordQuizAttemptService(rawWriter, nil, discardLogger())

	_, err := service.Execute(context.Background(), apvdto.RecordQuizAttemptRequest{UserID: "user-1"})
	if !apiservice.IsInvalidRequest(err) {
		t.Fatalf("Execute() error = %v, want invalid request", err)
	}
}

func TestRecordSelfMarkMasteredKeepsRawWriterInternalErrorInternal(t *testing.T) {
	rawWriter := &fakeSelfMarkRawWriter{err: errors.New("db down")}
	stateReader := &fakeUserUnitStateReader{response: learningdto.GetUserUnitStateResponse{Found: true}}
	service := apiservice.NewRecordSelfMarkMasteredService(rawWriter, nil, stateReader, discardLogger())

	_, err := service.Execute(context.Background(), apvdto.RecordSelfMarkMasteredRequest{UserID: "user-1"})
	if err == nil {
		t.Fatalf("Execute() error = nil, want raw writer error")
	}
	if apiservice.IsInvalidRequest(err) {
		t.Fatalf("Execute() error = %v, should not map internal writer error to invalid request", err)
	}
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

type fakeInteractionRawWriter struct {
	request  analyticsdto.RecordLearningInteractionsBatchRequest
	response analyticsdto.RecordLearningInteractionsBatchResponse
	err      error
}

func (f *fakeInteractionRawWriter) Execute(ctx context.Context, request analyticsdto.RecordLearningInteractionsBatchRequest) (analyticsdto.RecordLearningInteractionsBatchResponse, error) {
	f.request = request
	return f.response, f.err
}

type fakeQuizRawWriter struct {
	request  analyticsdto.RecordQuizAttemptRequest
	response analyticsdto.RecordQuizAttemptResponse
	err      error
}

func (f *fakeQuizRawWriter) Execute(ctx context.Context, request analyticsdto.RecordQuizAttemptRequest) (analyticsdto.RecordQuizAttemptResponse, error) {
	f.request = request
	return f.response, f.err
}

type fakeSelfMarkRawWriter struct {
	request  analyticsdto.RecordSelfMarkMasteredRequest
	response analyticsdto.RecordSelfMarkMasteredResponse
	err      error
	called   bool
}

func (f *fakeSelfMarkRawWriter) Execute(ctx context.Context, request analyticsdto.RecordSelfMarkMasteredRequest) (analyticsdto.RecordSelfMarkMasteredResponse, error) {
	f.called = true
	f.request = request
	return f.response, f.err
}

type fakeUserUnitStateReader struct {
	request  learningdto.GetUserUnitStateRequest
	response learningdto.GetUserUnitStateResponse
	err      error
}

func (f *fakeUserUnitStateReader) Execute(ctx context.Context, request learningdto.GetUserUnitStateRequest) (learningdto.GetUserUnitStateResponse, error) {
	f.request = request
	return f.response, f.err
}

type fakeInteractionNormalizer struct {
	request normalizerdto.NormalizeLearningInteractionsByIDsRequest
	err     error
}

func (f *fakeInteractionNormalizer) Execute(ctx context.Context, request normalizerdto.NormalizeLearningInteractionsByIDsRequest) (normalizerdto.NormalizeLearningInteractionsByIDsResponse, error) {
	f.request = request
	return normalizerdto.NormalizeLearningInteractionsByIDsResponse{}, f.err
}

type fakeQuizNormalizer struct {
	request normalizerdto.NormalizeQuizAttemptByIDRequest
	err     error
}

func (f *fakeQuizNormalizer) Execute(ctx context.Context, request normalizerdto.NormalizeQuizAttemptByIDRequest) (normalizerdto.NormalizeQuizAttemptByIDResponse, error) {
	f.request = request
	return normalizerdto.NormalizeQuizAttemptByIDResponse{}, f.err
}

type fakeSelfMarkNormalizer struct {
	request normalizerdto.NormalizeSelfMarkMasteredByIDRequest
	err     error
}

func (f *fakeSelfMarkNormalizer) Execute(ctx context.Context, request normalizerdto.NormalizeSelfMarkMasteredByIDRequest) (normalizerdto.NormalizeSelfMarkMasteredByIDResponse, error) {
	f.request = request
	return normalizerdto.NormalizeSelfMarkMasteredByIDResponse{}, f.err
}
