//go:build integration

package application_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
	"time"

	"learning-video-recommendation-system/internal/learningengine/normalizer/application/dto"
	normalizerservice "learning-video-recommendation-system/internal/learningengine/normalizer/application/service"
	normalizerrepo "learning-video-recommendation-system/internal/learningengine/normalizer/infrastructure/persistence/repository"
	"learning-video-recommendation-system/internal/learningengine/normalizer/test/fixture"
	learningservice "learning-video-recommendation-system/internal/learningengine/reducer/application/service"
	learningenum "learning-video-recommendation-system/internal/learningengine/reducer/domain/enum"
	learningrepo "learning-video-recommendation-system/internal/learningengine/reducer/infrastructure/persistence/repository"
	learningtx "learning-video-recommendation-system/internal/learningengine/reducer/infrastructure/persistence/tx"
)

func TestNormalizePendingEventsQuizUpdatesLearningState(t *testing.T) {
	t.Parallel()

	db := testDB(t)
	userID := "11111111-1111-1111-1111-111111111111"
	questionID := "22222222-2222-2222-2222-222222222222"
	db.SeedUser(t, userID)
	db.SeedCoarseUnit(t, 101)
	db.SeedQuestion(t, questionID)
	seedQuizEvent(t, db, "33333333-3333-3333-3333-333333333333", userID, questionID, 101, true, 5000, time.Date(2026, 5, 15, 10, 0, 0, 0, time.UTC))

	usecase := newNormalizerUsecase(db)
	response, err := usecase.Execute(context.Background(), dto.NormalizePendingEventsRequest{SourceKind: dto.SourceKindQuiz})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if response.RecordedEventCount != 1 {
		t.Fatalf("RecordedEventCount = %d, want 1", response.RecordedEventCount)
	}

	state := readState(t, db, userID, 101)
	if state.status != "learning" {
		t.Fatalf("status = %q, want learning", state.status)
	}
	if state.progressEventCount != 1 || state.lastProgressQuality == nil || *state.lastProgressQuality != 5 {
		t.Fatalf("progress count/quality = %d/%v, want 1/5", state.progressEventCount, state.lastProgressQuality)
	}
	if state.observationCount != 1 {
		t.Fatalf("observation_count = %d, want 1", state.observationCount)
	}
}

func TestNormalizePendingEventsSelfMarkSetsTerminalMastered(t *testing.T) {
	t.Parallel()

	db := testDB(t)
	userID := "11111111-1111-1111-1111-111111111111"
	db.SeedUser(t, userID)
	db.SeedCoarseUnit(t, 101)
	seedTargetState(t, db, userID, 101)
	seedLearningInteraction(t, db, "44444444-4444-4444-4444-444444444444", userID, 101, learningenum.EventSelfMarkMastered, time.Date(2026, 5, 15, 10, 0, 0, 0, time.UTC))

	usecase := newNormalizerUsecase(db)
	response, err := usecase.Execute(context.Background(), dto.NormalizePendingEventsRequest{SourceKind: dto.SourceKindLearningInteraction})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if response.RecordedEventCount != 1 {
		t.Fatalf("RecordedEventCount = %d, want 1", response.RecordedEventCount)
	}

	state := readState(t, db, userID, 101)
	if state.status != "mastered" {
		t.Fatalf("status = %q, want mastered", state.status)
	}
	if !state.isTarget {
		t.Fatal("is_target = false, want true")
	}
	if state.progressPercent != 100 || state.masteryScore != 1 {
		t.Fatalf("progress/mastery = %v/%v, want 100/1", state.progressPercent, state.masteryScore)
	}
	if !state.nextReviewIsNull {
		t.Fatalf("next_review_at is not null, want null")
	}
}

func TestNormalizePendingEventsLookupUpdatesObservationAndRawExposureDoesNotNormalizeDirectly(t *testing.T) {
	t.Parallel()

	db := testDB(t)
	userID := "11111111-1111-1111-1111-111111111111"
	db.SeedUser(t, userID)
	db.SeedCoarseUnit(t, 101)
	seedLearningInteraction(t, db, "55555555-5555-5555-5555-555555555555", userID, 101, learningenum.EventExposure, time.Date(2026, 5, 15, 10, 0, 0, 0, time.UTC))
	seedLearningInteraction(t, db, "66666666-6666-6666-6666-666666666666", userID, 101, learningenum.EventLookup, time.Date(2026, 5, 15, 10, 1, 0, 0, time.UTC))

	usecase := newNormalizerUsecase(db)
	response, err := usecase.Execute(context.Background(), dto.NormalizePendingEventsRequest{SourceKind: dto.SourceKindLearningInteraction})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if response.RecordedEventCount != 1 {
		t.Fatalf("RecordedEventCount = %d, want 1", response.RecordedEventCount)
	}

	state := readState(t, db, userID, 101)
	if state.status != "new" {
		t.Fatalf("status = %q, want new", state.status)
	}
	if state.observationCount != 1 || state.progressEventCount != 0 {
		t.Fatalf("observation/progress count = %d/%d, want 1/0", state.observationCount, state.progressEventCount)
	}
	if state.lastProgressQuality != nil {
		t.Fatalf("last_progress_quality = %v, want nil", state.lastProgressQuality)
	}
}

func TestNormalizeLearningInteractionsByIDCreatesPassiveProgressAfterThreeExposureSessions(t *testing.T) {
	t.Parallel()

	db := testDB(t)
	userID := "11111111-1111-1111-1111-111111111111"
	videoA := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa1"
	videoB := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa2"
	videoC := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa3"
	s1 := "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbb01"
	s2 := "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbb02"
	s3 := "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbb03"
	db.SeedUser(t, userID)
	db.SeedCoarseUnit(t, 101)
	db.SeedVideo(t, videoA)
	db.SeedVideo(t, videoB)
	db.SeedVideo(t, videoC)
	base := time.Date(2026, 5, 15, 10, 0, 0, 0, time.UTC)
	seedWatchSession(t, db, s1, userID, videoA, base)
	seedWatchSession(t, db, s2, userID, videoB, base.Add(time.Hour))
	seedWatchSession(t, db, s3, userID, videoC, base.Add(2*time.Hour))
	seedExposureInteraction(t, db, "55555555-5555-5555-5555-555555555551", userID, videoA, s1, 101, base)
	seedExposureInteraction(t, db, "55555555-5555-5555-5555-555555555552", userID, videoA, s1, 101, base.Add(15*time.Second))
	seedExposureInteraction(t, db, "55555555-5555-5555-5555-555555555553", userID, videoB, s2, 101, base.Add(time.Hour))
	thirdEventID := "55555555-5555-5555-5555-555555555554"
	seedExposureInteraction(t, db, thirdEventID, userID, videoC, s3, 101, base.Add(2*time.Hour))

	usecase := newNormalizeLearningInteractionsByIDsUsecase(db)
	response, err := usecase.Execute(context.Background(), dto.NormalizeLearningInteractionsByIDsRequest{
		UserID:                      userID,
		LearningInteractionEventIDs: []string{thirdEventID},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if response.RecordedEventCount != 1 {
		t.Fatalf("RecordedEventCount = %d, want 1", response.RecordedEventCount)
	}

	events, err := learningrepo.NewUnitLearningEventRepository(db.Pool).ListByUserOrdered(context.Background(), userID)
	if err != nil {
		t.Fatalf("ListByUserOrdered() error = %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("events = %d, want 1", len(events))
	}
	event := events[0]
	wantSourceRef := expectedExposureSession3SourceRef([]string{s1, s2, s3})
	if event.SourceType != "exposure_session3_v1" || event.SourceRefID != wantSourceRef {
		t.Fatalf("source = %s/%s, want %s", event.SourceType, event.SourceRefID, wantSourceRef)
	}
	if event.ProgressQuality == nil || *event.ProgressQuality != 4 || event.CountsTowardSuccessStreak {
		t.Fatalf("quality/streak = %v/%v, want q4 false", event.ProgressQuality, event.CountsTowardSuccessStreak)
	}
	wantConsumedSessions := []string{s1, s2, s3}
	if len(event.ConsumedWatchSessionIDs) != len(wantConsumedSessions) {
		t.Fatalf("consumed_watch_session_ids = %v, want %v", event.ConsumedWatchSessionIDs, wantConsumedSessions)
	}
	for index := range wantConsumedSessions {
		if event.ConsumedWatchSessionIDs[index] != wantConsumedSessions[index] {
			t.Fatalf("consumed_watch_session_ids = %v, want %v", event.ConsumedWatchSessionIDs, wantConsumedSessions)
		}
	}

	state := readState(t, db, userID, 101)
	if state.progressEventCount != 1 || state.lastProgressQuality == nil || *state.lastProgressQuality != 4 {
		t.Fatalf("state progress count/quality = %d/%v, want 1/4", state.progressEventCount, state.lastProgressQuality)
	}
	if state.consecutiveSuccessCount != 0 {
		t.Fatalf("consecutive_success_count = %d, want 0", state.consecutiveSuccessCount)
	}
}

func TestNormalizeLearningInteractionsByIDDoesNotReuseConsumedSessionAfterLateFlush(t *testing.T) {
	t.Parallel()

	db := testDB(t)
	userID := "11111111-1111-1111-1111-111111111111"
	videoID := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa1"
	db.SeedUser(t, userID)
	db.SeedCoarseUnit(t, 101)
	db.SeedVideo(t, videoID)
	base := time.Date(2026, 5, 15, 10, 0, 0, 0, time.UTC)
	sessionIDs := []string{
		"bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbb01",
		"bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbb02",
		"bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbb03",
		"bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbb04",
		"bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbb05",
	}
	eventIDs := []string{
		"55555555-5555-5555-5555-555555555551",
		"55555555-5555-5555-5555-555555555552",
		"55555555-5555-5555-5555-555555555553",
		"55555555-5555-5555-5555-555555555554",
		"55555555-5555-5555-5555-555555555555",
	}
	for index, sessionID := range sessionIDs {
		occurredAt := base.Add(time.Duration(index) * time.Hour)
		seedWatchSession(t, db, sessionID, userID, videoID, occurredAt)
		seedExposureInteraction(t, db, eventIDs[index], userID, videoID, sessionID, 101, occurredAt)
	}

	usecase := newNormalizeLearningInteractionsByIDsUsecase(db)
	firstResponse, err := usecase.Execute(context.Background(), dto.NormalizeLearningInteractionsByIDsRequest{
		UserID:                      userID,
		LearningInteractionEventIDs: []string{eventIDs[2]},
	})
	if err != nil {
		t.Fatalf("first Execute() error = %v", err)
	}
	if firstResponse.RecordedEventCount != 1 {
		t.Fatalf("first RecordedEventCount = %d, want 1", firstResponse.RecordedEventCount)
	}

	lateFlushID := "55555555-5555-5555-5555-555555555556"
	seedExposureInteraction(t, db, lateFlushID, userID, videoID, sessionIDs[2], 101, base.Add(2*time.Hour+15*time.Second))

	secondResponse, err := usecase.Execute(context.Background(), dto.NormalizeLearningInteractionsByIDsRequest{
		UserID:                      userID,
		LearningInteractionEventIDs: []string{lateFlushID, eventIDs[3], eventIDs[4]},
	})
	if err != nil {
		t.Fatalf("second Execute() error = %v", err)
	}
	if secondResponse.RecordedEventCount != 0 {
		t.Fatalf("second RecordedEventCount = %d, want 0 because session 3 was already consumed", secondResponse.RecordedEventCount)
	}

	events, err := learningrepo.NewUnitLearningEventRepository(db.Pool).ListByUserOrdered(context.Background(), userID)
	if err != nil {
		t.Fatalf("ListByUserOrdered() error = %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("events = %d, want only the first passive progress", len(events))
	}
}

func TestNormalizePendingEventsCreatesMultiplePassiveProgressWindows(t *testing.T) {
	t.Parallel()

	db := testDB(t)
	userID := "11111111-1111-1111-1111-111111111111"
	videoID := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa1"
	db.SeedUser(t, userID)
	db.SeedCoarseUnit(t, 101)
	db.SeedVideo(t, videoID)
	base := time.Date(2026, 5, 15, 10, 0, 0, 0, time.UTC)
	sessionIDs := []string{
		"bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbb01",
		"bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbb02",
		"bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbb03",
		"bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbb04",
		"bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbb05",
		"bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbb06",
	}
	eventIDs := []string{
		"55555555-5555-5555-5555-555555555551",
		"55555555-5555-5555-5555-555555555552",
		"55555555-5555-5555-5555-555555555553",
		"55555555-5555-5555-5555-555555555554",
		"55555555-5555-5555-5555-555555555555",
		"55555555-5555-5555-5555-555555555556",
	}
	for index, sessionID := range sessionIDs {
		occurredAt := base.Add(time.Duration(index) * time.Hour)
		seedWatchSession(t, db, sessionID, userID, videoID, occurredAt)
		seedExposureInteraction(t, db, eventIDs[index], userID, videoID, sessionID, 101, occurredAt)
	}

	usecase := newNormalizerUsecase(db)
	response, err := usecase.Execute(context.Background(), dto.NormalizePendingEventsRequest{SourceKind: dto.SourceKindLearningInteraction})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if response.RecordedEventCount != 2 {
		t.Fatalf("RecordedEventCount = %d, want 2", response.RecordedEventCount)
	}

	events, err := learningrepo.NewUnitLearningEventRepository(db.Pool).ListByUserOrdered(context.Background(), userID)
	if err != nil {
		t.Fatalf("ListByUserOrdered() error = %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("events = %d, want 2", len(events))
	}
	wantSourceRefs := map[string]bool{
		expectedExposureSession3SourceRef(sessionIDs[:3]): true,
		expectedExposureSession3SourceRef(sessionIDs[3:]): true,
	}
	for _, event := range events {
		if !wantSourceRefs[event.SourceRefID] {
			t.Fatalf("unexpected source_ref_id = %s", event.SourceRefID)
		}
	}
}

func TestNormalizeLearningInteractionsByIDLookupResetsExposureSession3Window(t *testing.T) {
	t.Parallel()

	db := testDB(t)
	userID := "11111111-1111-1111-1111-111111111111"
	videoID := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa1"
	db.SeedUser(t, userID)
	db.SeedCoarseUnit(t, 101)
	db.SeedVideo(t, videoID)
	base := time.Date(2026, 5, 15, 10, 0, 0, 0, time.UTC)
	sessionIDs := []string{
		"bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbb01",
		"bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbb02",
		"bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbb03",
		"bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbb04",
	}
	eventIDs := []string{
		"55555555-5555-5555-5555-555555555551",
		"55555555-5555-5555-5555-555555555552",
		"55555555-5555-5555-5555-555555555553",
		"55555555-5555-5555-5555-555555555554",
	}
	for index, sessionID := range sessionIDs {
		occurredAt := base.Add(time.Duration(index) * time.Hour)
		seedWatchSession(t, db, sessionID, userID, videoID, occurredAt)
		seedExposureInteraction(t, db, eventIDs[index], userID, videoID, sessionID, 101, occurredAt)
	}
	seedLearningInteraction(t, db, "66666666-6666-6666-6666-666666666666", userID, 101, learningenum.EventLookup, base.Add(90*time.Minute))

	usecase := newNormalizeLearningInteractionsByIDsUsecase(db)
	response, err := usecase.Execute(context.Background(), dto.NormalizeLearningInteractionsByIDsRequest{
		UserID:                      userID,
		LearningInteractionEventIDs: []string{eventIDs[3]},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if response.RecordedEventCount != 0 {
		t.Fatalf("RecordedEventCount = %d, want 0 because only two sessions after lookup", response.RecordedEventCount)
	}
}

func TestNormalizeLearningInteractionsByIDCreatesPassiveProgressAfterLookupReset(t *testing.T) {
	t.Parallel()

	db := testDB(t)
	userID := "11111111-1111-1111-1111-111111111111"
	videoID := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa1"
	db.SeedUser(t, userID)
	db.SeedCoarseUnit(t, 101)
	db.SeedVideo(t, videoID)
	base := time.Date(2026, 5, 15, 10, 0, 0, 0, time.UTC)
	sessionIDs := []string{
		"bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbb01",
		"bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbb02",
		"bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbb03",
		"bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbb04",
		"bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbb05",
	}
	eventIDs := []string{
		"55555555-5555-5555-5555-555555555551",
		"55555555-5555-5555-5555-555555555552",
		"55555555-5555-5555-5555-555555555553",
		"55555555-5555-5555-5555-555555555554",
		"55555555-5555-5555-5555-555555555555",
	}
	for index, sessionID := range sessionIDs {
		occurredAt := base.Add(time.Duration(index) * time.Hour)
		seedWatchSession(t, db, sessionID, userID, videoID, occurredAt)
		seedExposureInteraction(t, db, eventIDs[index], userID, videoID, sessionID, 101, occurredAt)
	}
	seedLearningInteraction(t, db, "66666666-6666-6666-6666-666666666666", userID, 101, learningenum.EventLookup, base.Add(90*time.Minute))

	usecase := newNormalizeLearningInteractionsByIDsUsecase(db)
	response, err := usecase.Execute(context.Background(), dto.NormalizeLearningInteractionsByIDsRequest{
		UserID:                      userID,
		LearningInteractionEventIDs: []string{eventIDs[4]},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if response.RecordedEventCount != 1 {
		t.Fatalf("RecordedEventCount = %d, want 1 from sessions after lookup", response.RecordedEventCount)
	}
	events, err := learningrepo.NewUnitLearningEventRepository(db.Pool).ListByUserOrdered(context.Background(), userID)
	if err != nil {
		t.Fatalf("ListByUserOrdered() error = %v", err)
	}
	if len(events) != 1 || events[0].SourceRefID != expectedExposureSession3SourceRef(sessionIDs[2:]) {
		t.Fatalf("events = %+v, want one session3 event from sessions 3-5", events)
	}
}

func TestNormalizePendingEventsSkipsRawBeforeResetBoundary(t *testing.T) {
	t.Parallel()

	db := testDB(t)
	userID := "11111111-1111-1111-1111-111111111111"
	questionID := "22222222-2222-2222-2222-222222222222"
	lookupEventID := "44444444-4444-4444-4444-444444444444"
	db.SeedUser(t, userID)
	db.SeedCoarseUnit(t, 101)
	db.SeedQuestion(t, questionID)
	boundary := time.Date(2026, 5, 15, 12, 0, 0, 0, time.UTC)
	seedResetBoundary(t, db, "99999999-9999-9999-9999-999999999991", userID, 101, boundary)
	seedQuizEvent(t, db, "33333333-3333-3333-3333-333333333333", userID, questionID, 101, true, 5000, boundary.Add(-time.Hour))
	seedLearningInteraction(t, db, lookupEventID, userID, 101, learningenum.EventLookup, boundary)

	usecase := newNormalizerUsecase(db)
	quizResponse, err := usecase.Execute(context.Background(), dto.NormalizePendingEventsRequest{SourceKind: dto.SourceKindQuiz})
	if err != nil {
		t.Fatalf("quiz Execute() error = %v", err)
	}
	if quizResponse.ReadRawCount != 0 || quizResponse.RecordedEventCount != 0 {
		t.Fatalf("quiz response = %+v, want old raw filtered before normalize", quizResponse)
	}

	interactionResponse, err := usecase.Execute(context.Background(), dto.NormalizePendingEventsRequest{SourceKind: dto.SourceKindLearningInteraction})
	if err != nil {
		t.Fatalf("interaction Execute() error = %v", err)
	}
	if interactionResponse.ReadRawCount != 0 || interactionResponse.RecordedEventCount != 0 {
		t.Fatalf("interaction response = %+v, want old raw filtered before normalize", interactionResponse)
	}

	quizByID := newNormalizeQuizAttemptByIDUsecase(db)
	quizByIDResponse, err := quizByID.Execute(context.Background(), dto.NormalizeQuizAttemptByIDRequest{
		UserID:      userID,
		QuizEventID: "33333333-3333-3333-3333-333333333333",
	})
	if err != nil {
		t.Fatalf("quiz by-id Execute() error = %v", err)
	}
	if quizByIDResponse.ReadRawCount != 0 || quizByIDResponse.RecordedEventCount != 0 {
		t.Fatalf("quiz by-id response = %+v, want old raw filtered before normalize", quizByIDResponse)
	}

	interactionByID := newNormalizeLearningInteractionsByIDsUsecase(db)
	interactionByIDResponse, err := interactionByID.Execute(context.Background(), dto.NormalizeLearningInteractionsByIDsRequest{
		UserID:                      userID,
		LearningInteractionEventIDs: []string{lookupEventID},
	})
	if err != nil {
		t.Fatalf("interaction by-id Execute() error = %v", err)
	}
	if interactionByIDResponse.ReadRawCount != 0 || interactionByIDResponse.RecordedEventCount != 0 {
		t.Fatalf("interaction by-id response = %+v, want old raw filtered before normalize", interactionByIDResponse)
	}

	var eventCount int
	if err := db.Pool.QueryRow(context.Background(), `select count(*) from learning.unit_learning_events where source_type <> 'learning_unit_reset'`).Scan(&eventCount); err != nil {
		t.Fatalf("count normalized learning events: %v", err)
	}
	if eventCount != 0 {
		t.Fatalf("normalized learning events = %d, want 0", eventCount)
	}
}

func TestNormalizeExposureSession3UsesResetBoundaryAsWindowStart(t *testing.T) {
	t.Parallel()

	db := testDB(t)
	userID := "11111111-1111-1111-1111-111111111111"
	videoID := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa1"
	db.SeedUser(t, userID)
	db.SeedCoarseUnit(t, 101)
	db.SeedVideo(t, videoID)
	base := time.Date(2026, 5, 15, 10, 0, 0, 0, time.UTC)
	boundary := base.Add(150 * time.Minute)
	seedResetBoundary(t, db, "99999999-9999-9999-9999-999999999992", userID, 101, boundary)

	sessionIDs := []string{
		"bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbb01",
		"bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbb02",
		"bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbb03",
		"bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbb04",
		"bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbb05",
	}
	eventIDs := []string{
		"55555555-5555-5555-5555-555555555551",
		"55555555-5555-5555-5555-555555555552",
		"55555555-5555-5555-5555-555555555553",
		"55555555-5555-5555-5555-555555555554",
		"55555555-5555-5555-5555-555555555555",
	}
	for index, sessionID := range sessionIDs {
		occurredAt := base.Add(time.Duration(index) * time.Hour)
		seedWatchSession(t, db, sessionID, userID, videoID, occurredAt)
		seedExposureInteraction(t, db, eventIDs[index], userID, videoID, sessionID, 101, occurredAt)
	}

	usecase := newNormalizeLearningInteractionsByIDsUsecase(db)
	firstResponse, err := usecase.Execute(context.Background(), dto.NormalizeLearningInteractionsByIDsRequest{
		UserID:                      userID,
		LearningInteractionEventIDs: []string{eventIDs[4]},
	})
	if err != nil {
		t.Fatalf("first Execute() error = %v", err)
	}
	if firstResponse.RecordedEventCount != 0 {
		t.Fatalf("first RecordedEventCount = %d, want 0 because only two sessions after reset", firstResponse.RecordedEventCount)
	}

	afterBoundarySession := "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbb06"
	afterBoundaryEvent := "55555555-5555-5555-5555-555555555556"
	seedWatchSession(t, db, afterBoundarySession, userID, videoID, boundary.Add(3*time.Hour))
	seedExposureInteraction(t, db, afterBoundaryEvent, userID, videoID, afterBoundarySession, 101, boundary.Add(3*time.Hour))

	secondResponse, err := usecase.Execute(context.Background(), dto.NormalizeLearningInteractionsByIDsRequest{
		UserID:                      userID,
		LearningInteractionEventIDs: []string{afterBoundaryEvent},
	})
	if err != nil {
		t.Fatalf("second Execute() error = %v", err)
	}
	if secondResponse.RecordedEventCount != 1 {
		t.Fatalf("second RecordedEventCount = %d, want one session3 event after reset", secondResponse.RecordedEventCount)
	}

	events, err := learningrepo.NewUnitLearningEventRepository(db.Pool).ListByUserOrdered(context.Background(), userID)
	if err != nil {
		t.Fatalf("ListByUserOrdered() error = %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("events = %d, want reset plus one session3 event", len(events))
	}
	wantSourceRef := expectedExposureSession3SourceRef([]string{sessionIDs[3], sessionIDs[4], afterBoundarySession})
	if events[1].SourceRefID != wantSourceRef {
		t.Fatalf("source_ref_id = %s, want %s", events[1].SourceRefID, wantSourceRef)
	}
}

func TestNormalizeByIDsOnlyProcessesRequestedUserRows(t *testing.T) {
	t.Parallel()

	db := testDB(t)
	userID := "11111111-1111-1111-1111-111111111111"
	otherUserID := "99999999-9999-9999-9999-999999999999"
	questionID := "22222222-2222-2222-2222-222222222222"
	db.SeedUser(t, userID)
	db.SeedUser(t, otherUserID)
	db.SeedCoarseUnit(t, 101)
	db.SeedCoarseUnit(t, 102)
	db.SeedQuestion(t, questionID)

	quizEventID := "33333333-3333-3333-3333-333333333333"
	lookupEventID := "44444444-4444-4444-4444-444444444444"
	otherUserEventID := "55555555-5555-5555-5555-555555555555"
	occurredAt := time.Date(2026, 5, 15, 10, 0, 0, 0, time.UTC)
	seedQuizEvent(t, db, quizEventID, userID, questionID, 101, true, 5000, occurredAt)
	seedLearningInteraction(t, db, lookupEventID, userID, 102, learningenum.EventLookup, occurredAt.Add(time.Second))
	seedLearningInteraction(t, db, otherUserEventID, otherUserID, 102, learningenum.EventLookup, occurredAt.Add(2*time.Second))

	quizUsecase := newNormalizeQuizAttemptByIDUsecase(db)
	quizResponse, err := quizUsecase.Execute(context.Background(), dto.NormalizeQuizAttemptByIDRequest{
		UserID:      userID,
		QuizEventID: quizEventID,
	})
	if err != nil {
		t.Fatalf("quiz Execute() error = %v", err)
	}
	if quizResponse.ReadRawCount != 1 || quizResponse.RecordedEventCount != 1 {
		t.Fatalf("quiz response = %+v, want read=1 recorded=1", quizResponse)
	}

	interactionUsecase := newNormalizeLearningInteractionsByIDsUsecase(db)
	interactionResponse, err := interactionUsecase.Execute(context.Background(), dto.NormalizeLearningInteractionsByIDsRequest{
		UserID:                      userID,
		LearningInteractionEventIDs: []string{lookupEventID, otherUserEventID},
	})
	if err != nil {
		t.Fatalf("interaction Execute() error = %v", err)
	}
	if interactionResponse.ReadRawCount != 1 || interactionResponse.RecordedEventCount != 1 {
		t.Fatalf("interaction response = %+v, want read=1 recorded=1", interactionResponse)
	}

	quizState := readState(t, db, userID, 101)
	if quizState.progressEventCount != 1 || quizState.lastProgressQuality == nil || *quizState.lastProgressQuality != 5 {
		t.Fatalf("quiz state = %+v, want one quality=5 progress event", quizState)
	}
	lookupState := readState(t, db, userID, 102)
	if lookupState.observationCount != 1 || lookupState.progressEventCount != 0 {
		t.Fatalf("lookup state = %+v, want observation only", lookupState)
	}

	var otherUserStateCount int
	if err := db.Pool.QueryRow(context.Background(), `select count(*) from learning.user_unit_states where user_id = $1`, otherUserID).Scan(&otherUserStateCount); err != nil {
		t.Fatalf("count other user states: %v", err)
	}
	if otherUserStateCount != 0 {
		t.Fatalf("other user states = %d, want 0", otherUserStateCount)
	}
}

func TestNormalizeSelfMarkMasteredByIDSetsTerminalMastered(t *testing.T) {
	t.Parallel()

	db := testDB(t)
	userID := "11111111-1111-1111-1111-111111111111"
	eventID := "44444444-4444-4444-4444-444444444444"
	db.SeedUser(t, userID)
	db.SeedCoarseUnit(t, 101)
	seedTargetState(t, db, userID, 101)
	seedLearningInteraction(t, db, eventID, userID, 101, learningenum.EventSelfMarkMastered, time.Date(2026, 5, 15, 10, 0, 0, 0, time.UTC))

	usecase := newNormalizeSelfMarkMasteredByIDUsecase(db)
	response, err := usecase.Execute(context.Background(), dto.NormalizeSelfMarkMasteredByIDRequest{
		UserID:                     userID,
		LearningInteractionEventID: eventID,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if response.ReadRawCount != 1 || response.RecordedEventCount != 1 {
		t.Fatalf("response = %+v, want read=1 recorded=1", response)
	}

	state := readState(t, db, userID, 101)
	if state.status != "mastered" || !state.isTarget || state.progressPercent != 100 || state.masteryScore != 1 || !state.nextReviewIsNull {
		t.Fatalf("state = %+v, want terminal mastered", state)
	}
}

func TestNormalizeSelfMarkMasteredByIDRejectsNonSelfMarkRawRow(t *testing.T) {
	t.Parallel()

	db := testDB(t)
	userID := "11111111-1111-1111-1111-111111111111"
	eventID := "44444444-4444-4444-4444-444444444444"
	db.SeedUser(t, userID)
	db.SeedCoarseUnit(t, 101)
	seedLearningInteraction(t, db, eventID, userID, 101, learningenum.EventLookup, time.Date(2026, 5, 15, 10, 0, 0, 0, time.UTC))

	usecase := newNormalizeSelfMarkMasteredByIDUsecase(db)
	response, err := usecase.Execute(context.Background(), dto.NormalizeSelfMarkMasteredByIDRequest{
		UserID:                     userID,
		LearningInteractionEventID: eventID,
	})
	if err == nil {
		t.Fatalf("Execute() error = nil, want event type error")
	}
	if response.ReadRawCount != 1 || response.ErrorCount != 1 {
		t.Fatalf("response = %+v, want read=1 error=1", response)
	}

	var eventCount int
	if err := db.Pool.QueryRow(context.Background(), `select count(*) from learning.unit_learning_events`).Scan(&eventCount); err != nil {
		t.Fatalf("count learning events: %v", err)
	}
	if eventCount != 0 {
		t.Fatalf("learning events = %d, want 0", eventCount)
	}
}

func TestNormalizeByIDWritesLearningEventOccurredAtAsUTCInstant(t *testing.T) {
	t.Parallel()

	db := testDB(t)
	userID := "11111111-1111-1111-1111-111111111111"
	questionID := "22222222-2222-2222-2222-222222222222"
	quizEventID := "33333333-3333-3333-3333-333333333333"
	completedAt := time.Date(2026, 5, 15, 10, 0, 0, 0, time.FixedZone("PDT", -7*60*60))
	db.SeedUser(t, userID)
	db.SeedCoarseUnit(t, 101)
	db.SeedQuestion(t, questionID)
	seedQuizEvent(t, db, quizEventID, userID, questionID, 101, true, 5000, completedAt)

	usecase := newNormalizeQuizAttemptByIDUsecase(db)
	response, err := usecase.Execute(context.Background(), dto.NormalizeQuizAttemptByIDRequest{
		UserID:      userID,
		QuizEventID: quizEventID,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if response.RecordedEventCount != 1 {
		t.Fatalf("RecordedEventCount = %d, want 1", response.RecordedEventCount)
	}

	events, err := learningrepo.NewUnitLearningEventRepository(db.Pool).ListByUserOrdered(context.Background(), userID)
	if err != nil {
		t.Fatalf("ListByUserOrdered() error = %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("events = %d, want 1", len(events))
	}
	occurredAt := events[0].OccurredAt
	if occurredAt.Location() != time.UTC {
		t.Fatalf("occurred_at location = %v, want UTC", occurredAt.Location())
	}
	if !occurredAt.Equal(completedAt) {
		t.Fatalf("occurred_at = %v, want same instant as %v", occurredAt, completedAt)
	}
}

func newNormalizerUsecase(db *fixture.TestDatabase) *normalizerservice.NormalizePendingEventsUsecase {
	recordUsecase := learningservice.NewRecordLearningEventsUsecase(learningtx.NewManager(db.Pool))
	return normalizerservice.NewNormalizePendingEventsUsecase(
		normalizerrepo.NewRawQuizEventReader(db.Pool),
		normalizerrepo.NewRawLearningInteractionReader(db.Pool),
		recordUsecase,
	)
}

func newNormalizeLearningInteractionsByIDsUsecase(db *fixture.TestDatabase) *normalizerservice.NormalizeLearningInteractionsByIDsUsecase {
	recordUsecase := learningservice.NewRecordLearningEventsUsecase(learningtx.NewManager(db.Pool))
	return normalizerservice.NewNormalizeLearningInteractionsByIDsUsecase(
		normalizerrepo.NewRawLearningInteractionReader(db.Pool),
		recordUsecase,
	)
}

func newNormalizeQuizAttemptByIDUsecase(db *fixture.TestDatabase) *normalizerservice.NormalizeQuizAttemptByIDUsecase {
	recordUsecase := learningservice.NewRecordLearningEventsUsecase(learningtx.NewManager(db.Pool))
	return normalizerservice.NewNormalizeQuizAttemptByIDUsecase(
		normalizerrepo.NewRawQuizEventReader(db.Pool),
		recordUsecase,
	)
}

func newNormalizeSelfMarkMasteredByIDUsecase(db *fixture.TestDatabase) *normalizerservice.NormalizeSelfMarkMasteredByIDUsecase {
	recordUsecase := learningservice.NewRecordLearningEventsUsecase(learningtx.NewManager(db.Pool))
	return normalizerservice.NewNormalizeSelfMarkMasteredByIDUsecase(
		normalizerrepo.NewRawLearningInteractionReader(db.Pool),
		recordUsecase,
	)
}

type stateRow struct {
	status                  string
	isTarget                bool
	progressPercent         float64
	masteryScore            float64
	observationCount        int32
	progressEventCount      int32
	consecutiveSuccessCount int32
	lastProgressQuality     *int16
	nextReviewIsNull        bool
}

func readState(t *testing.T, db *fixture.TestDatabase, userID string, unitID int64) stateRow {
	t.Helper()

	var row stateRow
	if err := db.Pool.QueryRow(context.Background(), `
		select
			status,
			is_target,
			progress_percent::float8,
			mastery_score::float8,
			observation_count,
			progress_event_count,
			consecutive_success_count,
			last_progress_quality,
			next_review_at is null
		from learning.user_unit_states
		where user_id = $1 and coarse_unit_id = $2
	`, userID, unitID).Scan(
		&row.status,
		&row.isTarget,
		&row.progressPercent,
		&row.masteryScore,
		&row.observationCount,
		&row.progressEventCount,
		&row.consecutiveSuccessCount,
		&row.lastProgressQuality,
		&row.nextReviewIsNull,
	); err != nil {
		t.Fatalf("read learning.user_unit_states: %v", err)
	}
	return row
}

func seedWatchSession(t *testing.T, db *fixture.TestDatabase, sessionID, userID, videoID string, startedAt time.Time) {
	t.Helper()
	if _, err := db.Pool.Exec(context.Background(), `
		insert into analytics.video_watch_events (
			watch_session_id,
			user_id,
			video_id,
			started_at,
			last_seen_at
		) values (
			$1::uuid,
			$2::uuid,
			$3::uuid,
			$4,
			$4
		)`, sessionID, userID, videoID, startedAt); err != nil {
		t.Fatalf("seed analytics.video_watch_events: %v", err)
	}
}

func seedTargetState(t *testing.T, db *fixture.TestDatabase, userID string, unitID int64) {
	t.Helper()
	if _, err := db.Pool.Exec(context.Background(), `
		insert into learning.user_unit_states (
			user_id,
			coarse_unit_id,
			is_target,
			target_source,
			target_source_ref_id,
			target_priority
		) values (
			$1::uuid,
			$2,
			true,
			'curriculum',
			'lesson_1',
			0.9
		)`, userID, unitID); err != nil {
		t.Fatalf("seed learning.user_unit_states: %v", err)
	}
}

func seedResetBoundary(t *testing.T, db *fixture.TestDatabase, eventID, userID string, unitID int64, boundary time.Time) {
	t.Helper()
	if _, err := db.Pool.Exec(context.Background(), `
		insert into learning.unit_learning_events (
			event_id,
			user_id,
			coarse_unit_id,
			event_type,
			reducer_effect,
			source_type,
			source_ref_id,
			metadata,
			occurred_at,
			reset_boundary_at
		) values (
			$1::uuid,
			$2::uuid,
			$3,
			'reset_unlearned',
			'reset_unlearned',
			'learning_unit_reset',
			$1::text,
			'{}'::jsonb,
			$4,
			$4
		)`, eventID, userID, unitID, boundary); err != nil {
		t.Fatalf("seed reset boundary: %v", err)
	}
}

func seedExposureInteraction(t *testing.T, db *fixture.TestDatabase, eventID, userID, videoID, watchSessionID string, unitID int64, occurredAt time.Time) {
	t.Helper()
	if _, err := db.Pool.Exec(context.Background(), `
		insert into analytics.learning_interaction_events (
			event_id,
			client_event_id,
			user_id,
			event_type,
			source_surface,
			video_id,
			watch_session_id,
			coarse_unit_id,
			token_text,
			sentence_index,
			span_index,
			occurred_at,
			exposure_start_ms,
			exposure_end_ms,
			exposure_count,
			event_payload
		) values (
			$1::uuid,
			'client-' || $1::text,
			$2::uuid,
			'exposure',
			'video_subtitle',
			$3::uuid,
			$4::uuid,
			$5,
			'example',
			1,
			1,
			$6,
			100,
			1200,
			1,
			'{}'::jsonb
		)`, eventID, userID, videoID, watchSessionID, unitID, occurredAt); err != nil {
		t.Fatalf("seed exposure interaction: %v", err)
	}
}

func seedQuizEvent(t *testing.T, db *fixture.TestDatabase, eventID, userID, questionID string, unitID int64, correct bool, elapsedMS int32, completedAt time.Time) {
	t.Helper()
	selectedOptionIDs := []string{"correct"}
	selectionIntervalMS := []int32{elapsedMS}
	if !correct {
		selectedOptionIDs = []string{"wrong", "correct"}
		selectionIntervalMS = []int32{1000, elapsedMS - 1000}
	}
	if _, err := db.Pool.Exec(context.Background(), `
		insert into analytics.quiz_events (
			event_id,
			client_event_id,
			user_id,
			question_id,
			coarse_unit_id,
			trigger_type,
			selected_option_ids,
			selection_interval_ms,
			is_first_try_correct,
			total_elapsed_ms,
			shown_at,
			completed_at
		) values (
			$1::uuid,
			'client-' || $1::text,
			$2::uuid,
			$3::uuid,
			$4,
			'manual',
			$5,
			$6,
			$7,
			$8,
			$9::timestamptz - interval '1 second',
			$9
		)`, eventID, userID, questionID, unitID, selectedOptionIDs, selectionIntervalMS, correct, elapsedMS, completedAt); err != nil {
		t.Fatalf("seed analytics.quiz_events: %v", err)
	}
}

func seedLearningInteraction(t *testing.T, db *fixture.TestDatabase, eventID, userID string, unitID int64, eventType string, occurredAt time.Time) {
	t.Helper()
	if _, err := db.Pool.Exec(context.Background(), `
		insert into analytics.learning_interaction_events (
			event_id,
			client_event_id,
			user_id,
			event_type,
			source_surface,
			coarse_unit_id,
			token_text,
			occurred_at,
			exposure_start_ms,
			exposure_end_ms,
			exposure_count,
			lookup_visible_ms,
			event_payload
		) values (
			$1::uuid,
			'client-' || $1::text,
			$2::uuid,
			$3,
			'video_subtitle',
			$4,
			'example',
			$5,
			100,
			1200,
			1,
			9000,
			'{}'::jsonb
		)`, eventID, userID, eventType, unitID, occurredAt); err != nil {
		t.Fatalf("seed analytics.learning_interaction_events: %v", err)
	}
}

func expectedExposureSession3SourceRef(sessionIDs []string) string {
	sum := sha256.Sum256([]byte(strings.Join(sessionIDs, "|")))
	return "exposure_session3:" + hex.EncodeToString(sum[:])
}
