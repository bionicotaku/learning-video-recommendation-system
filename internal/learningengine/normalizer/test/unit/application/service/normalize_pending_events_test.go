package service_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"

	"learning-video-recommendation-system/internal/learningengine/normalizer/application/dto"
	normalizerrepo "learning-video-recommendation-system/internal/learningengine/normalizer/application/repository"
	"learning-video-recommendation-system/internal/learningengine/normalizer/application/service"
	"learning-video-recommendation-system/internal/learningengine/normalizer/domain/model"
	learningdto "learning-video-recommendation-system/internal/learningengine/reducer/application/dto"
	learningenum "learning-video-recommendation-system/internal/learningengine/reducer/domain/enum"
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

func TestNormalizeLearningInteractionsByIDsReadsSelectedRowsAndRecords(t *testing.T) {
	now := time.Date(2026, 5, 15, 10, 0, 0, 0, time.UTC)
	interactionReader := &fakeInteractionReader{eventsByID: []model.RawLearningInteraction{
		validRawInteraction("11111111-1111-1111-1111-111111111111", "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb", 102, learningenum.EventLookup, now.Add(time.Second)),
	}}
	recorder := &fakeRecorder{}
	usecase := service.NewNormalizeLearningInteractionsByIDsUsecase(interactionReader, recorder)

	response, err := usecase.Execute(context.Background(), dto.NormalizeLearningInteractionsByIDsRequest{
		UserID:                      "11111111-1111-1111-1111-111111111111",
		LearningInteractionEventIDs: []string{"bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !interactionReader.byIDsCalled {
		t.Fatalf("interaction by IDs reader was not called")
	}
	if interactionReader.lastByIDsUserID != "11111111-1111-1111-1111-111111111111" {
		t.Fatalf("reader user id = %q", interactionReader.lastByIDsUserID)
	}
	if response.ReadRawCount != 1 || response.NormalizedEventCount != 1 || response.RecordedEventCount != 1 {
		t.Fatalf("response = %+v, want read=1 normalized=1 recorded=1", response)
	}
	if len(recorder.requests) != 1 {
		t.Fatalf("recorder requests = %d, want 1", len(recorder.requests))
	}
	if len(recorder.requests[0].Events) != 1 {
		t.Fatalf("recorded events = %d, want 1", len(recorder.requests[0].Events))
	}
}

func TestNormalizeLearningInteractionsByIDsAggregatesThreeExposureSessions(t *testing.T) {
	t1 := time.Date(2026, 5, 15, 10, 0, 0, 0, time.UTC)
	userID := "11111111-1111-1111-1111-111111111111"
	interactionReader := &fakeInteractionReader{
		eventsByID: []model.RawLearningInteraction{
			validRawInteraction(userID, "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb", 102, learningenum.EventExposure, t1),
		},
		windowsByIDs: []model.ExposureSession3Window{{
			UserID:          userID,
			CoarseUnitID:    102,
			OccurredAt:      t1.Add(2 * time.Hour),
			ThirdVideoID:    "33333333-3333-3333-3333-333333333333",
			WatchSessionIDs: []string{"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaa0001", "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaa0002", "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaa0003"},
			VideoIDs:        []string{"33333333-3333-3333-3333-333333333331", "33333333-3333-3333-3333-333333333332", "33333333-3333-3333-3333-333333333333"},
			RawEventCount:   5,
		}},
	}
	recorder := &fakeRecorder{}
	usecase := service.NewNormalizeLearningInteractionsByIDsUsecase(interactionReader, recorder)

	response, err := usecase.Execute(context.Background(), dto.NormalizeLearningInteractionsByIDsRequest{
		UserID:                      userID,
		LearningInteractionEventIDs: []string{"bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !interactionReader.windowsByIDsCalled {
		t.Fatalf("exposure session windows by IDs reader was not called")
	}
	if response.ReadRawCount != 1 || response.NormalizedEventCount != 1 || response.RecordedEventCount != 1 {
		t.Fatalf("response = %+v, want read=1 normalized=1 recorded=1", response)
	}
	if len(recorder.requests) != 1 || len(recorder.requests[0].Events) != 1 {
		t.Fatalf("recorder requests = %+v, want one request with one event", recorder.requests)
	}
	recorded := recorder.requests[0].Events[0]
	wantSourceRef := expectedExposureSession3SourceRef([]string{"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaa0001", "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaa0002", "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaa0003"})
	if recorded.SourceType != "exposure_session3_v1" || recorded.SourceRefID != wantSourceRef {
		t.Fatalf("source = %s/%s, want %s", recorded.SourceType, recorded.SourceRefID, wantSourceRef)
	}
	if recorded.ReducerEffect != learningenum.ReducerEffectAffectsProgress || recorded.ProgressQuality == nil || *recorded.ProgressQuality != 4 {
		t.Fatalf("recorded event = %+v, want q4 affects_progress", recorded)
	}
	if recorded.CountsTowardSuccessStreak {
		t.Fatalf("counts_toward_success_streak = true, want false")
	}
	wantConsumedSessions := []string{
		"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaa0001",
		"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaa0002",
		"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaa0003",
	}
	if strings.Join(recorded.ConsumedWatchSessionIDs, ",") != strings.Join(wantConsumedSessions, ",") {
		t.Fatalf("consumed_watch_session_ids = %v, want %v", recorded.ConsumedWatchSessionIDs, wantConsumedSessions)
	}
}

func TestNormalizeLearningInteractionsByIDsRejectsIncompleteExposureSessionWindow(t *testing.T) {
	t1 := time.Date(2026, 5, 15, 10, 0, 0, 0, time.UTC)
	userID := "11111111-1111-1111-1111-111111111111"
	interactionReader := &fakeInteractionReader{
		eventsByID: []model.RawLearningInteraction{
			validRawInteraction(userID, "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb", 102, learningenum.EventExposure, t1),
		},
		windowsByIDs: []model.ExposureSession3Window{{
			UserID:          userID,
			CoarseUnitID:    102,
			OccurredAt:      t1.Add(time.Hour),
			ThirdVideoID:    "33333333-3333-3333-3333-333333333332",
			WatchSessionIDs: []string{"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaa0001", "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaa0002"},
			VideoIDs:        []string{"33333333-3333-3333-3333-333333333331", "33333333-3333-3333-3333-333333333332"},
			RawEventCount:   2,
		}},
	}
	recorder := &fakeRecorder{}
	usecase := service.NewNormalizeLearningInteractionsByIDsUsecase(interactionReader, recorder)

	response, err := usecase.Execute(context.Background(), dto.NormalizeLearningInteractionsByIDsRequest{
		UserID:                      userID,
		LearningInteractionEventIDs: []string{"bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"},
	})
	if err == nil {
		t.Fatalf("Execute() error = nil, want incomplete session window error")
	}
	if response.ErrorCount != 1 || response.RecordedEventCount != 0 {
		t.Fatalf("response = %+v, want error=1 recorded=0", response)
	}
	if len(recorder.requests) != 0 {
		t.Fatalf("recorder requests = %d, want 0", len(recorder.requests))
	}
}

func TestNormalizeLearningInteractionsByIDsDoesNotRecordRawExposureWithoutSession3Window(t *testing.T) {
	t1 := time.Date(2026, 5, 15, 10, 0, 0, 0, time.UTC)
	userID := "11111111-1111-1111-1111-111111111111"
	interactionReader := &fakeInteractionReader{
		eventsByID: []model.RawLearningInteraction{
			validRawInteraction(userID, "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb", 102, learningenum.EventExposure, t1),
		},
	}
	recorder := &fakeRecorder{}
	usecase := service.NewNormalizeLearningInteractionsByIDsUsecase(interactionReader, recorder)

	response, err := usecase.Execute(context.Background(), dto.NormalizeLearningInteractionsByIDsRequest{
		UserID:                      userID,
		LearningInteractionEventIDs: []string{"bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if response.ReadRawCount != 1 || response.NormalizedEventCount != 0 || response.RecordedEventCount != 0 {
		t.Fatalf("response = %+v, want raw exposure read but not normalized", response)
	}
	if len(recorder.requests) != 0 {
		t.Fatalf("recorder requests = %d, want 0", len(recorder.requests))
	}
}

func TestNormalizeSelfMarkMasteredByIDReadsSelectedRowAndRecordsSetMastered(t *testing.T) {
	now := time.Date(2026, 5, 15, 10, 0, 0, 0, time.UTC)
	interactionReader := &fakeInteractionReader{eventsByID: []model.RawLearningInteraction{
		validRawInteraction("11111111-1111-1111-1111-111111111111", "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb", 102, learningenum.EventSelfMarkMastered, now.Add(time.Second)),
	}}
	recorder := &fakeRecorder{}
	usecase := service.NewNormalizeSelfMarkMasteredByIDUsecase(interactionReader, recorder)

	response, err := usecase.Execute(context.Background(), dto.NormalizeSelfMarkMasteredByIDRequest{
		UserID:                     "11111111-1111-1111-1111-111111111111",
		LearningInteractionEventID: "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !interactionReader.byIDsCalled {
		t.Fatalf("interaction by IDs reader was not called")
	}
	if response.ReadRawCount != 1 || response.NormalizedEventCount != 1 || response.RecordedEventCount != 1 {
		t.Fatalf("response = %+v, want read=1 normalized=1 recorded=1", response)
	}
	if len(recorder.requests) != 1 || len(recorder.requests[0].Events) != 1 {
		t.Fatalf("recorder requests = %+v, want one request with one event", recorder.requests)
	}
	recorded := recorder.requests[0].Events[0]
	if recorded.EventType != learningenum.EventSelfMarkMastered || recorded.ReducerEffect != learningenum.ReducerEffectSetMastered {
		t.Fatalf("recorded event = %+v, want self_mark_mastered set_mastered", recorded)
	}
	if recorded.ProgressQuality != nil {
		t.Fatalf("progress_quality = %v, want nil", recorded.ProgressQuality)
	}
}

func TestNormalizeSelfMarkMasteredByIDIgnoresOtherUserMissingRow(t *testing.T) {
	interactionReader := &fakeInteractionReader{}
	recorder := &fakeRecorder{}
	usecase := service.NewNormalizeSelfMarkMasteredByIDUsecase(interactionReader, recorder)

	response, err := usecase.Execute(context.Background(), dto.NormalizeSelfMarkMasteredByIDRequest{
		UserID:                     "11111111-1111-1111-1111-111111111111",
		LearningInteractionEventID: "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if response.ReadRawCount != 0 || response.RecordedEventCount != 0 {
		t.Fatalf("response = %+v, want no rows processed", response)
	}
	if len(recorder.requests) != 0 {
		t.Fatalf("recorder requests = %d, want 0", len(recorder.requests))
	}
}

func TestNormalizeSelfMarkMasteredByIDRejectsNonSelfMarkRawEvent(t *testing.T) {
	now := time.Date(2026, 5, 15, 10, 0, 0, 0, time.UTC)
	interactionReader := &fakeInteractionReader{eventsByID: []model.RawLearningInteraction{
		validRawInteraction("11111111-1111-1111-1111-111111111111", "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb", 102, learningenum.EventLookup, now),
	}}
	recorder := &fakeRecorder{}
	usecase := service.NewNormalizeSelfMarkMasteredByIDUsecase(interactionReader, recorder)

	response, err := usecase.Execute(context.Background(), dto.NormalizeSelfMarkMasteredByIDRequest{
		UserID:                     "11111111-1111-1111-1111-111111111111",
		LearningInteractionEventID: "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
	})
	if err == nil {
		t.Fatalf("Execute() error = nil, want validation error")
	}
	if response.ErrorCount != 1 || response.ReadRawCount != 1 {
		t.Fatalf("response = %+v, want read=1 error=1", response)
	}
	if len(recorder.requests) != 0 {
		t.Fatalf("recorder requests = %d, want 0", len(recorder.requests))
	}
}

func TestNormalizeSelfMarkMasteredByIDFailFastOnRecorderError(t *testing.T) {
	wantErr := errors.New("record failed")
	now := time.Date(2026, 5, 15, 10, 0, 0, 0, time.UTC)
	interactionReader := &fakeInteractionReader{eventsByID: []model.RawLearningInteraction{
		validRawInteraction("11111111-1111-1111-1111-111111111111", "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb", 102, learningenum.EventSelfMarkMastered, now),
	}}
	recorder := &fakeRecorder{err: wantErr}
	usecase := service.NewNormalizeSelfMarkMasteredByIDUsecase(interactionReader, recorder)

	response, err := usecase.Execute(context.Background(), dto.NormalizeSelfMarkMasteredByIDRequest{
		UserID:                     "11111111-1111-1111-1111-111111111111",
		LearningInteractionEventID: "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Execute() error = %v, want %v", err, wantErr)
	}
	if response.ErrorCount != 1 || response.NormalizedEventCount != 1 || response.RecordedEventCount != 0 {
		t.Fatalf("response = %+v, want error=1 normalized=1 recorded=0", response)
	}
}

func TestNormalizeQuizAttemptByIDReadsSelectedRowAndRecords(t *testing.T) {
	now := time.Date(2026, 5, 15, 10, 0, 0, 0, time.UTC)
	quizReader := &fakeQuizReader{eventsByID: []model.RawQuizEvent{
		validRawQuiz("11111111-1111-1111-1111-111111111111", "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", 101, now),
	}}
	recorder := &fakeRecorder{}
	usecase := service.NewNormalizeQuizAttemptByIDUsecase(quizReader, recorder)

	response, err := usecase.Execute(context.Background(), dto.NormalizeQuizAttemptByIDRequest{
		UserID:      "11111111-1111-1111-1111-111111111111",
		QuizEventID: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !quizReader.byIDsCalled {
		t.Fatalf("quiz by IDs reader was not called")
	}
	if quizReader.lastByIDsUserID != "11111111-1111-1111-1111-111111111111" {
		t.Fatalf("reader user id = %q", quizReader.lastByIDsUserID)
	}
	if response.ReadRawCount != 1 || response.NormalizedEventCount != 1 || response.RecordedEventCount != 1 {
		t.Fatalf("response = %+v, want read=1 normalized=1 recorded=1", response)
	}
	if len(recorder.requests) != 1 || len(recorder.requests[0].Events) != 1 {
		t.Fatalf("recorder requests = %+v, want one request with one event", recorder.requests)
	}
	if recorder.requests[0].Events[0].ProgressQuality == nil || *recorder.requests[0].Events[0].ProgressQuality != 5 {
		t.Fatalf("quality = %v, want 5", recorder.requests[0].Events[0].ProgressQuality)
	}
}

func TestNormalizeLearningInteractionsByIDsSkipsUnmappedLookup(t *testing.T) {
	now := time.Date(2026, 5, 15, 10, 0, 0, 0, time.UTC)
	interactionReader := &fakeInteractionReader{eventsByID: []model.RawLearningInteraction{
		{EventID: "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb", UserID: "11111111-1111-1111-1111-111111111111", EventType: learningenum.EventLookup, SourceSurface: "video_subtitle", OccurredAt: now, EventPayload: []byte(`{}`)},
	}}
	recorder := &fakeRecorder{}
	usecase := service.NewNormalizeLearningInteractionsByIDsUsecase(interactionReader, recorder)

	response, err := usecase.Execute(context.Background(), dto.NormalizeLearningInteractionsByIDsRequest{
		UserID:                      "11111111-1111-1111-1111-111111111111",
		LearningInteractionEventIDs: []string{"bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if response.ReadRawCount != 1 || response.SkippedCount != 1 || response.RecordedEventCount != 0 {
		t.Fatalf("response = %+v, want read=1 skipped=1 recorded=0", response)
	}
	if len(recorder.requests) != 0 {
		t.Fatalf("recorder requests = %d, want 0", len(recorder.requests))
	}
}

func TestNormalizeLearningInteractionsByIDsRejectsSelfMarkRawEvent(t *testing.T) {
	now := time.Date(2026, 5, 15, 10, 0, 0, 0, time.UTC)
	interactionReader := &fakeInteractionReader{eventsByID: []model.RawLearningInteraction{
		validRawInteraction("11111111-1111-1111-1111-111111111111", "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb", 102, learningenum.EventSelfMarkMastered, now),
	}}
	recorder := &fakeRecorder{}
	usecase := service.NewNormalizeLearningInteractionsByIDsUsecase(interactionReader, recorder)

	response, err := usecase.Execute(context.Background(), dto.NormalizeLearningInteractionsByIDsRequest{
		UserID:                      "11111111-1111-1111-1111-111111111111",
		LearningInteractionEventIDs: []string{"bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"},
	})
	if err == nil {
		t.Fatalf("Execute() error = nil, want validation error")
	}
	if response.ErrorCount != 1 || response.ReadRawCount != 1 {
		t.Fatalf("response = %+v, want read=1 error=1", response)
	}
	if len(recorder.requests) != 0 {
		t.Fatalf("recorder requests = %d, want 0", len(recorder.requests))
	}
}

type fakeQuizReader struct {
	events          []model.RawQuizEvent
	eventsByID      []model.RawQuizEvent
	called          bool
	byIDsCalled     bool
	lastFilter      normalizerrepo.PendingRawEventFilter
	lastByIDsUserID string
	lastByIDs       []string
}

func (r *fakeQuizReader) ListPendingQuizEvents(_ context.Context, filter normalizerrepo.PendingRawEventFilter) ([]model.RawQuizEvent, error) {
	r.called = true
	r.lastFilter = filter
	return r.events, nil
}

func (r *fakeQuizReader) ListQuizEventsByIDs(_ context.Context, userID string, eventIDs []string) ([]model.RawQuizEvent, error) {
	r.byIDsCalled = true
	r.lastByIDsUserID = userID
	r.lastByIDs = append([]string(nil), eventIDs...)
	return r.eventsByID, nil
}

type fakeInteractionReader struct {
	events             []model.RawLearningInteraction
	eventsByID         []model.RawLearningInteraction
	windows            []model.ExposureSession3Window
	windowsByIDs       []model.ExposureSession3Window
	called             bool
	byIDsCalled        bool
	windowsCalled      bool
	windowsByIDsCalled bool
	lastFilter         normalizerrepo.PendingRawEventFilter
	lastByIDsUserID    string
	lastByIDs          []string
}

func (r *fakeInteractionReader) ListPendingLearningInteractions(_ context.Context, filter normalizerrepo.PendingRawEventFilter) ([]model.RawLearningInteraction, error) {
	r.called = true
	r.lastFilter = filter
	return r.events, nil
}

func (r *fakeInteractionReader) ListLearningInteractionsByIDs(_ context.Context, userID string, eventIDs []string) ([]model.RawLearningInteraction, error) {
	r.byIDsCalled = true
	r.lastByIDsUserID = userID
	r.lastByIDs = append([]string(nil), eventIDs...)
	return r.eventsByID, nil
}

func (r *fakeInteractionReader) ListPendingExposureSession3Windows(_ context.Context, filter normalizerrepo.PendingRawEventFilter) ([]model.ExposureSession3Window, error) {
	r.windowsCalled = true
	r.lastFilter = filter
	return r.windows, nil
}

func (r *fakeInteractionReader) ListExposureSession3WindowsByIDs(_ context.Context, userID string, eventIDs []string) ([]model.ExposureSession3Window, error) {
	r.windowsByIDsCalled = true
	r.lastByIDsUserID = userID
	r.lastByIDs = append([]string(nil), eventIDs...)
	return r.windowsByIDs, nil
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

func expectedExposureSession3SourceRef(sessionIDs []string) string {
	sum := sha256.Sum256([]byte(strings.Join(sessionIDs, "|")))
	return "exposure_session3:" + hex.EncodeToString(sum[:])
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
