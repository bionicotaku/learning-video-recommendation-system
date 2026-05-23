//go:build integration

package service_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"learning-video-recommendation-system/internal/learningengine/reducer/application/dto"
	applearningrepo "learning-video-recommendation-system/internal/learningengine/reducer/application/repository"
	"learning-video-recommendation-system/internal/learningengine/reducer/application/service"
	"learning-video-recommendation-system/internal/learningengine/reducer/domain/model"
	persistrepo "learning-video-recommendation-system/internal/learningengine/reducer/infrastructure/persistence/repository"
	persisttx "learning-video-recommendation-system/internal/learningengine/reducer/infrastructure/persistence/tx"
	userrepo "learning-video-recommendation-system/internal/user/application/repository"
)

func TestTargetControlUsecasesWithDatabase(t *testing.T) {
	t.Parallel()

	db := testDB(t)
	db.SeedUser(t, "11111111-1111-1111-1111-111111111111")
	db.SeedCoarseUnit(t, 101)

	txManager := persisttx.NewManager(db.Pool)

	ensureUsecase := service.NewEnsureTargetUnitsUsecase(txManager)
	setInactiveUsecase := service.NewSetTargetInactiveUsecase(txManager)
	listUsecase := service.NewListUserUnitStatesUsecase(persistrepo.NewUserUnitStateRepository(db.Pool))

	if _, err := ensureUsecase.Execute(context.Background(), dto.EnsureTargetUnitsRequest{
		UserID: "11111111-1111-1111-1111-111111111111",
		Targets: []dto.TargetUnitSpec{
			{CoarseUnitID: 101, TargetSource: "curriculum", TargetSourceRefID: "lesson_1", TargetPriority: 0.9},
		},
	}); err != nil {
		t.Fatalf("EnsureTargetUnits.Execute() error = %v", err)
	}

	response, err := listUsecase.Execute(context.Background(), dto.ListUserUnitStatesRequest{
		UserID:     "11111111-1111-1111-1111-111111111111",
		OnlyTarget: true,
	})
	if err != nil {
		t.Fatalf("ListUserUnitStates.Execute() error = %v", err)
	}
	if len(response.States) != 1 {
		t.Fatalf("states len = %d, want 1", len(response.States))
	}
	if response.States[0].Status != "new" {
		t.Fatalf("status = %q, want new", response.States[0].Status)
	}

	if _, err := setInactiveUsecase.Execute(context.Background(), dto.SetTargetInactiveRequest{
		UserID:       "11111111-1111-1111-1111-111111111111",
		CoarseUnitID: 101,
	}); err != nil {
		t.Fatalf("SetTargetInactive.Execute() error = %v", err)
	}

	inactiveTargets, err := listUsecase.Execute(context.Background(), dto.ListUserUnitStatesRequest{
		UserID:     "11111111-1111-1111-1111-111111111111",
		OnlyTarget: true,
	})
	if err != nil {
		t.Fatalf("ListUserUnitStates.Execute() error = %v", err)
	}
	if len(inactiveTargets.States) != 0 {
		t.Fatalf("target states len = %d, want 0 after SetTargetInactive", len(inactiveTargets.States))
	}
}

func TestRecordLearningEventsWithDatabase(t *testing.T) {
	t.Parallel()

	db := testDB(t)
	userID := "11111111-1111-1111-1111-111111111111"
	db.SeedUser(t, userID)
	db.SeedCoarseUnit(t, 101)

	recordUsecase := service.NewRecordLearningEventsUsecase(persisttx.NewManager(db.Pool))
	stateRepo := persistrepo.NewUserUnitStateRepository(db.Pool)

	q4 := int16(4)
	t1 := time.Date(2026, 4, 16, 10, 0, 0, 0, time.UTC)
	t2 := t1.Add(24 * time.Hour)

	response, err := recordUsecase.Execute(context.Background(), dto.RecordLearningEventsRequest{
		UserID: userID,
		Events: []dto.LearningEventInput{
			{CoarseUnitID: 101, EventType: "quiz", ReducerEffect: "affects_progress", SourceType: "quiz_event", SourceRefID: "record-2", ProgressQuality: &q4, OccurredAt: t2},
			{CoarseUnitID: 101, EventType: "quiz", ReducerEffect: "affects_progress", SourceType: "quiz_event", SourceRefID: "record-1", ProgressQuality: &q4, OccurredAt: t1},
		},
	})
	if err != nil {
		t.Fatalf("RecordLearningEvents.Execute() error = %v", err)
	}
	if response.RecordedCount != 2 {
		t.Fatalf("RecordedCount = %d, want 2", response.RecordedCount)
	}

	states, err := stateRepo.ListByUser(context.Background(), userID, model.UserUnitStateFilter{})
	if err != nil {
		t.Fatalf("ListByUser() error = %v", err)
	}
	if len(states) != 1 {
		t.Fatalf("states len = %d, want 1", len(states))
	}
	if states[0].Status != "reviewing" {
		t.Fatalf("status = %q, want reviewing", states[0].Status)
	}
	if states[0].LatestLearningEventOccurredAt == nil || !states[0].LatestLearningEventOccurredAt.Equal(t2) {
		t.Fatalf("latest_learning_event_occurred_at = %v, want %v", states[0].LatestLearningEventOccurredAt, t2)
	}
	if states[0].LatestLearningEventLedgerSeq == 0 {
		t.Fatalf("latest_learning_event_ledger_seq = 0, want inserted ledger seq")
	}
}

func TestRecordLearningEventsStoresConsumedWatchSessions(t *testing.T) {
	t.Parallel()

	db := testDB(t)
	userID := "11111111-1111-1111-1111-111111111111"
	db.SeedUser(t, userID)
	db.SeedCoarseUnit(t, 101)

	recordUsecase := service.NewRecordLearningEventsUsecase(persisttx.NewManager(db.Pool))
	q4 := int16(4)
	consumed := []string{
		"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaa0001",
		"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaa0002",
		"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaa0003",
	}

	response, err := recordUsecase.Execute(context.Background(), dto.RecordLearningEventsRequest{
		UserID: userID,
		Events: []dto.LearningEventInput{{
			CoarseUnitID:            101,
			EventType:               "exposure",
			ReducerEffect:           "affects_progress",
			SourceType:              "exposure_session3_v1",
			SourceRefID:             "exposure_session3:typed-consumed",
			ProgressQuality:         &q4,
			ConsumedWatchSessionIDs: consumed,
			OccurredAt:              time.Date(2026, 5, 15, 10, 0, 0, 0, time.UTC),
		}},
	})
	if err != nil {
		t.Fatalf("RecordLearningEvents.Execute() error = %v", err)
	}
	if response.RecordedCount != 1 {
		t.Fatalf("RecordedCount = %d, want 1", response.RecordedCount)
	}

	events, err := persistrepo.NewUnitLearningEventRepository(db.Pool).ListByUserOrdered(context.Background(), userID)
	if err != nil {
		t.Fatalf("ListByUserOrdered() error = %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("events = %d, want 1", len(events))
	}
	if len(events[0].ConsumedWatchSessionIDs) != 3 {
		t.Fatalf("consumed_watch_session_ids = %v, want %v", events[0].ConsumedWatchSessionIDs, consumed)
	}
	for index := range consumed {
		if events[0].ConsumedWatchSessionIDs[index] != consumed[index] {
			t.Fatalf("consumed_watch_session_ids = %v, want %v", events[0].ConsumedWatchSessionIDs, consumed)
		}
	}
}

func TestUnitLearningEventsRejectInvalidExposureSession3RowsAtDatabase(t *testing.T) {
	t.Parallel()

	db := testDB(t)
	userID := "11111111-1111-1111-1111-111111111111"
	db.SeedUser(t, userID)
	db.SeedCoarseUnit(t, 101)

	testCases := []struct {
		name             string
		eventType        string
		sourceType       string
		progressQuality  int
		countsStreak     bool
		consumedSessions string
	}{
		{
			name:             "non session3 carries consumed sessions",
			eventType:        "quiz",
			sourceType:       "quiz_event",
			progressQuality:  4,
			countsStreak:     true,
			consumedSessions: "array['aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaa0001']::uuid[]",
		},
		{
			name:             "session3 wrong event type",
			eventType:        "quiz",
			sourceType:       "exposure_session3_v1",
			progressQuality:  4,
			countsStreak:     false,
			consumedSessions: "array['aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaa0001','aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaa0002','aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaa0003']::uuid[]",
		},
		{
			name:             "session3 wrong quality",
			eventType:        "exposure",
			sourceType:       "exposure_session3_v1",
			progressQuality:  3,
			countsStreak:     false,
			consumedSessions: "array['aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaa0001','aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaa0002','aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaa0003']::uuid[]",
		},
		{
			name:             "session3 counts streak",
			eventType:        "exposure",
			sourceType:       "exposure_session3_v1",
			progressQuality:  4,
			countsStreak:     true,
			consumedSessions: "array['aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaa0001','aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaa0002','aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaa0003']::uuid[]",
		},
		{
			name:             "session3 null consumed session",
			eventType:        "exposure",
			sourceType:       "exposure_session3_v1",
			progressQuality:  4,
			countsStreak:     false,
			consumedSessions: "array['aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaa0001', null, 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaa0003']::uuid[]",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			query := `
				insert into learning.unit_learning_events (
					user_id,
					coarse_unit_id,
					event_type,
					reducer_effect,
					progress_quality,
					source_type,
					source_ref_id,
					counts_toward_success_streak,
					consumed_watch_session_ids,
					occurred_at
				)
				values (
					$1::uuid,
					101,
					$2,
					'affects_progress',
					$3,
					$4,
					$5,
					$6,
					` + testCase.consumedSessions + `,
					now()
				)
			`
			_, err := db.Pool.Exec(context.Background(), query, userID, testCase.eventType, testCase.progressQuality, testCase.sourceType, "invalid-"+testCase.name, testCase.countsStreak)
			if err == nil {
				t.Fatalf("insert invalid row error = nil, want db check violation")
			}
		})
	}
}

func TestRecordLearningEventsDuplicateSourceIsIdempotent(t *testing.T) {
	t.Parallel()

	db := testDB(t)
	userID := "11111111-1111-1111-1111-111111111111"
	db.SeedUser(t, userID)
	db.SeedCoarseUnit(t, 101)

	recordUsecase := service.NewRecordLearningEventsUsecase(persisttx.NewManager(db.Pool))
	stateRepo := persistrepo.NewUserUnitStateRepository(db.Pool)

	q4 := int16(4)
	event := dto.LearningEventInput{
		CoarseUnitID:    101,
		EventType:       "quiz",
		ReducerEffect:   "affects_progress",
		SourceType:      "quiz_event",
		SourceRefID:     "duplicate-source-1",
		ProgressQuality: &q4,
		OccurredAt:      time.Date(2026, 4, 16, 10, 0, 0, 0, time.UTC),
	}
	first, err := recordUsecase.Execute(context.Background(), dto.RecordLearningEventsRequest{
		UserID: userID,
		Events: []dto.LearningEventInput{event},
	})
	if err != nil {
		t.Fatalf("first RecordLearningEvents.Execute() error = %v", err)
	}
	second, err := recordUsecase.Execute(context.Background(), dto.RecordLearningEventsRequest{
		UserID: userID,
		Events: []dto.LearningEventInput{event},
	})
	if err != nil {
		t.Fatalf("second RecordLearningEvents.Execute() error = %v", err)
	}

	if first.RecordedCount != 1 || first.DuplicateCount != 0 {
		t.Fatalf("first response = %+v, want recorded=1 duplicate=0", first)
	}
	if second.RecordedCount != 0 || second.DuplicateCount != 1 {
		t.Fatalf("second response = %+v, want recorded=0 duplicate=1", second)
	}
	states, err := stateRepo.ListByUser(context.Background(), userID, model.UserUnitStateFilter{})
	if err != nil {
		t.Fatalf("ListByUser() error = %v", err)
	}
	if len(states) != 1 || states[0].ProgressEventCount != 1 {
		t.Fatalf("states = %+v, want one progress event", states)
	}
}

func TestRecordSelfMarkMasteredWithDatabase(t *testing.T) {
	t.Parallel()

	db := testDB(t)
	userID := "11111111-1111-1111-1111-111111111111"
	db.SeedUser(t, userID)
	db.SeedCoarseUnit(t, 101)

	txManager := persisttx.NewManager(db.Pool)
	ensureUsecase := service.NewEnsureTargetUnitsUsecase(txManager)
	recordUsecase := service.NewRecordLearningEventsUsecase(txManager)
	replayUsecase := service.NewReplayUserStatesUsecase(txManager)
	listUsecase := service.NewListUserUnitStatesUsecase(persistrepo.NewUserUnitStateRepository(db.Pool))

	if _, err := ensureUsecase.Execute(context.Background(), dto.EnsureTargetUnitsRequest{
		UserID: userID,
		Targets: []dto.TargetUnitSpec{
			{CoarseUnitID: 101, TargetSource: "curriculum", TargetSourceRefID: "lesson_1", TargetPriority: 0.9},
		},
	}); err != nil {
		t.Fatalf("EnsureTargetUnits.Execute() error = %v", err)
	}

	selfMarkOccurredAt := time.Date(2026, 4, 16, 10, 0, 0, 0, time.UTC)
	if _, err := recordUsecase.Execute(context.Background(), dto.RecordLearningEventsRequest{
		UserID: userID,
		Events: []dto.LearningEventInput{
			{
				CoarseUnitID:  101,
				EventType:     "self_mark_mastered",
				ReducerEffect: "set_mastered",
				SourceType:    "learning_interaction_event",
				SourceRefID:   "self-mark-1",
				OccurredAt:    selfMarkOccurredAt,
			},
		},
	}); err != nil {
		t.Fatalf("RecordLearningEvents.Execute() error = %v", err)
	}

	activeTargets, err := listUsecase.Execute(context.Background(), dto.ListUserUnitStatesRequest{
		UserID:     userID,
		OnlyTarget: true,
	})
	if err != nil {
		t.Fatalf("ListUserUnitStates(active targets) error = %v", err)
	}
	if len(activeTargets.States) != 1 {
		t.Fatalf("active target states len = %d, want 1", len(activeTargets.States))
	}

	allStates, err := listUsecase.Execute(context.Background(), dto.ListUserUnitStatesRequest{
		UserID: userID,
	})
	if err != nil {
		t.Fatalf("ListUserUnitStates(all) error = %v", err)
	}
	if len(allStates.States) != 1 {
		t.Fatalf("all states len = %d, want 1", len(allStates.States))
	}
	assertCompletedMasteredState(t, allStates.States[0], selfMarkOccurredAt, true)

	if _, err := replayUsecase.Execute(context.Background(), dto.ReplayUserStatesRequest{UserID: userID}); err != nil {
		t.Fatalf("ReplayUserStates.Execute() error = %v", err)
	}

	afterReplay, err := listUsecase.Execute(context.Background(), dto.ListUserUnitStatesRequest{
		UserID: userID,
	})
	if err != nil {
		t.Fatalf("ListUserUnitStates(after replay) error = %v", err)
	}
	if len(afterReplay.States) != 1 {
		t.Fatalf("after replay states len = %d, want 1", len(afterReplay.States))
	}
	assertCompletedMasteredState(t, afterReplay.States[0], selfMarkOccurredAt, true)
}

func TestResetUserUnitProgressWithDatabaseAndReplay(t *testing.T) {
	t.Parallel()

	db := testDB(t)
	userID := "11111111-1111-1111-1111-111111111111"
	db.SeedUser(t, userID)
	db.SeedCoarseUnit(t, 101)

	txManager := persisttx.NewManager(db.Pool)
	ensureUsecase := service.NewEnsureTargetUnitsUsecase(txManager)
	recordUsecase := service.NewRecordLearningEventsUsecase(txManager)
	setInactiveUsecase := service.NewSetTargetInactiveUsecase(txManager)
	resetUsecase := service.NewResetUserUnitProgressUsecase(txManager)
	replayUsecase := service.NewReplayUserStatesUsecase(txManager)
	listUsecase := service.NewListUserUnitStatesUsecase(persistrepo.NewUserUnitStateRepository(db.Pool))

	if _, err := ensureUsecase.Execute(context.Background(), dto.EnsureTargetUnitsRequest{
		UserID: userID,
		Targets: []dto.TargetUnitSpec{
			{CoarseUnitID: 101, TargetSource: "curriculum", TargetSourceRefID: "lesson_1", TargetPriority: 0.9},
		},
	}); err != nil {
		t.Fatalf("EnsureTargetUnits.Execute() error = %v", err)
	}

	q4 := int16(4)
	occurredAt := time.Date(2026, 4, 16, 10, 0, 0, 0, time.UTC)
	if _, err := recordUsecase.Execute(context.Background(), dto.RecordLearningEventsRequest{
		UserID: userID,
		Events: []dto.LearningEventInput{
			{CoarseUnitID: 101, EventType: "quiz", ReducerEffect: "affects_progress", SourceType: "quiz_event", SourceRefID: "progress-1", ProgressQuality: &q4, OccurredAt: occurredAt},
		},
	}); err != nil {
		t.Fatalf("RecordLearningEvents.Execute() error = %v", err)
	}

	if _, err := setInactiveUsecase.Execute(context.Background(), dto.SetTargetInactiveRequest{
		UserID:       userID,
		CoarseUnitID: 101,
	}); err != nil {
		t.Fatalf("SetTargetInactive.Execute() error = %v", err)
	}

	resetOccurredAt := occurredAt.Add(time.Hour)
	first, err := resetUsecase.Execute(context.Background(), dto.ResetUserUnitProgressRequest{
		UserID:        userID,
		ClientContext: []byte(`{"platform":"ios"}`),
		ClientEventID: "reset-1",
		CoarseUnitID:  101,
		SourceSurface: "word_detail",
		TokenText:     "apple",
		OccurredAt:    resetOccurredAt,
		EventPayload:  []byte(`{"origin":"manual"}`),
	})
	if err != nil {
		t.Fatalf("ResetUserUnitProgress.Execute() error = %v", err)
	}
	if !first.Accepted || !first.Inserted || first.UnitLearningEventID == "" {
		t.Fatalf("first reset response = %+v, want accepted inserted event id", first)
	}

	allStates, err := listUsecase.Execute(context.Background(), dto.ListUserUnitStatesRequest{UserID: userID})
	if err != nil {
		t.Fatalf("ListUserUnitStates(all) error = %v", err)
	}
	if len(allStates.States) != 1 {
		t.Fatalf("all states len = %d, want 1", len(allStates.States))
	}
	assertResetUnlearnedState(t, allStates.States[0], false)

	second, err := resetUsecase.Execute(context.Background(), dto.ResetUserUnitProgressRequest{
		UserID:        userID,
		ClientContext: []byte(`{"platform":"ios"}`),
		ClientEventID: "reset-1",
		CoarseUnitID:  101,
		SourceSurface: "word_detail",
		OccurredAt:    resetOccurredAt,
		EventPayload:  []byte(`{"origin":"manual"}`),
	})
	if err != nil {
		t.Fatalf("duplicate ResetUserUnitProgress.Execute() error = %v", err)
	}
	if !second.Accepted || second.Inserted || second.UnitLearningEventID != first.UnitLearningEventID {
		t.Fatalf("second reset response = %+v, want accepted duplicate same event id %q", second, first.UnitLearningEventID)
	}

	if _, err := replayUsecase.Execute(context.Background(), dto.ReplayUserStatesRequest{UserID: userID}); err != nil {
		t.Fatalf("ReplayUserStates.Execute() error = %v", err)
	}
	afterReplay, err := listUsecase.Execute(context.Background(), dto.ListUserUnitStatesRequest{UserID: userID})
	if err != nil {
		t.Fatalf("ListUserUnitStates(after replay) error = %v", err)
	}
	if len(afterReplay.States) != 1 {
		t.Fatalf("after replay states len = %d, want 1", len(afterReplay.States))
	}
	assertResetUnlearnedState(t, afterReplay.States[0], false)
}

func TestResetUserUnitProgressUsesLedgerOrderAndBoundaryForReplay(t *testing.T) {
	t.Parallel()

	db := testDB(t)
	userID := "11111111-1111-1111-1111-111111111111"
	db.SeedUser(t, userID)
	db.SeedCoarseUnit(t, 101)

	txManager := persisttx.NewManager(db.Pool)
	ensureUsecase := service.NewEnsureTargetUnitsUsecase(txManager)
	recordUsecase := service.NewRecordLearningEventsUsecase(txManager)
	resetUsecase := service.NewResetUserUnitProgressUsecase(txManager)
	replayUsecase := service.NewReplayUserStatesUsecase(txManager)
	listUsecase := service.NewListUserUnitStatesUsecase(persistrepo.NewUserUnitStateRepository(db.Pool))
	eventRepo := persistrepo.NewUnitLearningEventRepository(db.Pool)

	if _, err := ensureUsecase.Execute(context.Background(), dto.EnsureTargetUnitsRequest{
		UserID: userID,
		Targets: []dto.TargetUnitSpec{
			{CoarseUnitID: 101, TargetSource: "curriculum", TargetSourceRefID: "lesson_1", TargetPriority: 0.9},
		},
	}); err != nil {
		t.Fatalf("EnsureTargetUnits.Execute() error = %v", err)
	}

	q4 := int16(4)
	progressAt := time.Date(2026, 4, 16, 12, 0, 0, 0, time.UTC)
	if _, err := recordUsecase.Execute(context.Background(), dto.RecordLearningEventsRequest{
		UserID: userID,
		Events: []dto.LearningEventInput{
			{CoarseUnitID: 101, EventType: "quiz", ReducerEffect: "affects_progress", SourceType: "quiz_event", SourceRefID: "reset-boundary-progress-1", ProgressQuality: &q4, OccurredAt: progressAt},
		},
	}); err != nil {
		t.Fatalf("RecordLearningEvents.Execute(progress) error = %v", err)
	}

	resetOccurredAt := progressAt.Add(-2 * time.Hour)
	response, err := resetUsecase.Execute(context.Background(), dto.ResetUserUnitProgressRequest{
		UserID:        userID,
		ClientEventID: "reset-boundary-1",
		CoarseUnitID:  101,
		SourceSurface: "word_detail",
		OccurredAt:    resetOccurredAt,
	})
	if err != nil {
		t.Fatalf("ResetUserUnitProgress.Execute() error = %v", err)
	}
	if !response.Accepted || !response.Inserted {
		t.Fatalf("reset response = %+v, want inserted", response)
	}

	events, err := eventRepo.ListByUserOrdered(context.Background(), userID)
	if err != nil {
		t.Fatalf("ListByUserOrdered() error = %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("events len = %d, want 2", len(events))
	}
	if events[1].ReducerEffect != "reset_unlearned" {
		t.Fatalf("event order = %+v, want reset second by ledger order", events)
	}
	if events[1].ResetBoundaryAt == nil || !events[1].ResetBoundaryAt.Equal(progressAt) {
		t.Fatalf("reset_boundary_at = %v, want %v", events[1].ResetBoundaryAt, progressAt)
	}
	beforeReplay, err := listUsecase.Execute(context.Background(), dto.ListUserUnitStatesRequest{UserID: userID})
	if err != nil {
		t.Fatalf("ListUserUnitStates.Execute(before replay) error = %v", err)
	}
	if len(beforeReplay.States) != 1 {
		t.Fatalf("before replay states len = %d, want 1", len(beforeReplay.States))
	}
	if beforeReplay.States[0].LatestResetBoundaryAt == nil || !beforeReplay.States[0].LatestResetBoundaryAt.Equal(progressAt) {
		t.Fatalf("latest_reset_boundary_at = %v, want %v", beforeReplay.States[0].LatestResetBoundaryAt, progressAt)
	}

	stale, err := recordUsecase.Execute(context.Background(), dto.RecordLearningEventsRequest{
		UserID: userID,
		Events: []dto.LearningEventInput{
			{CoarseUnitID: 101, EventType: "quiz", ReducerEffect: "affects_progress", SourceType: "quiz_event", SourceRefID: "reset-boundary-stale-1", ProgressQuality: &q4, OccurredAt: progressAt},
		},
	})
	if err != nil {
		t.Fatalf("RecordLearningEvents.Execute(stale) error = %v", err)
	}
	if stale.RecordedCount != 0 || stale.SkippedBeforeResetCount != 1 {
		t.Fatalf("stale response = %+v, want skipped before reset", stale)
	}

	if _, err := replayUsecase.Execute(context.Background(), dto.ReplayUserStatesRequest{UserID: userID}); err != nil {
		t.Fatalf("ReplayUserStates.Execute() error = %v", err)
	}
	states, err := listUsecase.Execute(context.Background(), dto.ListUserUnitStatesRequest{UserID: userID})
	if err != nil {
		t.Fatalf("ListUserUnitStates.Execute() error = %v", err)
	}
	if len(states.States) != 1 {
		t.Fatalf("states len = %d, want 1", len(states.States))
	}
	if states.States[0].LatestResetBoundaryAt == nil || !states.States[0].LatestResetBoundaryAt.Equal(progressAt) {
		t.Fatalf("replayed latest_reset_boundary_at = %v, want %v", states.States[0].LatestResetBoundaryAt, progressAt)
	}
	assertResetUnlearnedState(t, states.States[0], true)
}

func TestResetUserUnitProgressDuplicateClientEventAcrossUnitsIsIdempotent(t *testing.T) {
	t.Parallel()

	db := testDB(t)
	userID := "11111111-1111-1111-1111-111111111111"
	db.SeedUser(t, userID)
	db.SeedCoarseUnit(t, 101)
	db.SeedCoarseUnit(t, 102)

	txManager := persisttx.NewManager(db.Pool)
	ensureUsecase := service.NewEnsureTargetUnitsUsecase(txManager)
	recordUsecase := service.NewRecordLearningEventsUsecase(txManager)
	resetUsecase := service.NewResetUserUnitProgressUsecase(txManager)
	listUsecase := service.NewListUserUnitStatesUsecase(persistrepo.NewUserUnitStateRepository(db.Pool))

	if _, err := ensureUsecase.Execute(context.Background(), dto.EnsureTargetUnitsRequest{
		UserID: userID,
		Targets: []dto.TargetUnitSpec{
			{CoarseUnitID: 101, TargetSource: "curriculum", TargetSourceRefID: "lesson_1", TargetPriority: 0.9},
			{CoarseUnitID: 102, TargetSource: "curriculum", TargetSourceRefID: "lesson_1", TargetPriority: 0.8},
		},
	}); err != nil {
		t.Fatalf("EnsureTargetUnits.Execute() error = %v", err)
	}

	q4 := int16(4)
	occurredAt := time.Date(2026, 4, 16, 10, 0, 0, 0, time.UTC)
	if _, err := recordUsecase.Execute(context.Background(), dto.RecordLearningEventsRequest{
		UserID: userID,
		Events: []dto.LearningEventInput{
			{CoarseUnitID: 102, EventType: "quiz", ReducerEffect: "affects_progress", SourceType: "quiz_event", SourceRefID: "progress-102", ProgressQuality: &q4, OccurredAt: occurredAt},
		},
	}); err != nil {
		t.Fatalf("RecordLearningEvents.Execute() error = %v", err)
	}

	first, err := resetUsecase.Execute(context.Background(), dto.ResetUserUnitProgressRequest{
		UserID:        userID,
		ClientEventID: "reset-duplicate-1",
		CoarseUnitID:  101,
		SourceSurface: "word_detail",
		OccurredAt:    occurredAt.Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("first ResetUserUnitProgress.Execute() error = %v", err)
	}
	if !first.Accepted || !first.Inserted || first.UnitLearningEventID == "" {
		t.Fatalf("first response = %+v, want inserted reset event", first)
	}

	second, err := resetUsecase.Execute(context.Background(), dto.ResetUserUnitProgressRequest{
		UserID:        userID,
		ClientEventID: "reset-duplicate-1",
		CoarseUnitID:  102,
		SourceSurface: "word_detail",
		OccurredAt:    occurredAt.Add(2 * time.Hour),
	})
	if err != nil {
		t.Fatalf("second ResetUserUnitProgress.Execute() error = %v", err)
	}
	if !second.Accepted || second.Inserted || second.UnitLearningEventID != first.UnitLearningEventID {
		t.Fatalf("second response = %+v, want duplicate same event id %q", second, first.UnitLearningEventID)
	}

	allStates, err := listUsecase.Execute(context.Background(), dto.ListUserUnitStatesRequest{UserID: userID})
	if err != nil {
		t.Fatalf("ListUserUnitStates(all) error = %v", err)
	}
	statesByUnit := indexStatesByUnit(allStates.States)
	if statesByUnit[101].Status != "new" || statesByUnit[101].ProgressEventCount != 0 {
		t.Fatalf("unit 101 state = %+v, want reset", statesByUnit[101])
	}
	if statesByUnit[102].ProgressEventCount != 1 || statesByUnit[102].ProgressPercent == 0 {
		t.Fatalf("unit 102 state = %+v, want unchanged progress", statesByUnit[102])
	}

	events, err := persistrepo.NewUnitLearningEventRepository(db.Pool).ListByUserOrdered(context.Background(), userID)
	if err != nil {
		t.Fatalf("ListByUserOrdered() error = %v", err)
	}
	resetCount := 0
	for _, event := range events {
		if event.SourceType == "learning_unit_reset" && event.SourceRefID == "reset-duplicate-1" {
			resetCount++
			if event.CoarseUnitID != 101 {
				t.Fatalf("reset duplicate event coarse_unit_id = %d, want 101", event.CoarseUnitID)
			}
		}
	}
	if resetCount != 1 {
		t.Fatalf("reset event count = %d, want 1", resetCount)
	}
}

func TestRecordLearningEventsRollsBackWhenStateWriteFails(t *testing.T) {
	t.Parallel()

	db := testDB(t)
	userID := "11111111-1111-1111-1111-111111111111"
	db.SeedUser(t, userID)
	db.SeedCoarseUnit(t, 101)

	usecase := service.NewRecordLearningEventsUsecase(&failingBatchUpsertTxManager{pool: db.Pool})
	q4 := int16(4)
	t1 := time.Date(2026, 4, 16, 10, 0, 0, 0, time.UTC)

	_, err := usecase.Execute(context.Background(), dto.RecordLearningEventsRequest{
		UserID: userID,
		Events: []dto.LearningEventInput{
			{CoarseUnitID: 101, EventType: "quiz", ReducerEffect: "affects_progress", SourceType: "quiz_event", SourceRefID: "rollback-1", ProgressQuality: &q4, OccurredAt: t1},
		},
	})
	if !errors.Is(err, errForcedBatchUpsertFailure) {
		t.Fatalf("Execute() error = %v, want errForcedBatchUpsertFailure", err)
	}

	var count int
	if err := db.Pool.QueryRow(context.Background(), `select count(*) from learning.unit_learning_events where user_id = $1`, userID).Scan(&count); err != nil {
		t.Fatalf("QueryRow() error = %v", err)
	}
	if count != 0 {
		t.Fatalf("event count = %d, want 0 after rollback", count)
	}
}

func TestReplayUserStatesWithDatabase(t *testing.T) {
	t.Parallel()

	db := testDB(t)
	userID := "11111111-1111-1111-1111-111111111111"
	db.SeedUser(t, userID)
	db.SeedCoarseUnit(t, 101)
	db.SeedCoarseUnit(t, 102)

	stateRepo := persistrepo.NewUserUnitStateRepository(db.Pool)
	txManager := persisttx.NewManager(db.Pool)

	ensureUsecase := service.NewEnsureTargetUnitsUsecase(txManager)
	recordUsecase := service.NewRecordLearningEventsUsecase(txManager)
	replayUsecase := service.NewReplayUserStatesUsecase(txManager)

	if _, err := ensureUsecase.Execute(context.Background(), dto.EnsureTargetUnitsRequest{
		UserID: userID,
		Targets: []dto.TargetUnitSpec{
			{CoarseUnitID: 101, TargetSource: "curriculum", TargetSourceRefID: "lesson_1", TargetPriority: 0.9},
			{CoarseUnitID: 102, TargetSource: "curriculum", TargetSourceRefID: "lesson_1", TargetPriority: 0.8},
		},
	}); err != nil {
		t.Fatalf("EnsureTargetUnits.Execute() error = %v", err)
	}

	q4 := int16(4)
	t1 := time.Date(2026, 4, 16, 10, 0, 0, 0, time.UTC)
	t2 := t1.Add(24 * time.Hour)
	if _, err := recordUsecase.Execute(context.Background(), dto.RecordLearningEventsRequest{
		UserID: userID,
		Events: []dto.LearningEventInput{
			{CoarseUnitID: 101, EventType: "quiz", ReducerEffect: "affects_progress", SourceType: "quiz_event", SourceRefID: "replay-1", ProgressQuality: &q4, OccurredAt: t1},
			{CoarseUnitID: 101, EventType: "quiz", ReducerEffect: "affects_progress", SourceType: "quiz_event", SourceRefID: "replay-2", ProgressQuality: &q4, OccurredAt: t2},
		},
	}); err != nil {
		t.Fatalf("RecordLearningEvents.Execute() error = %v", err)
	}

	beforeReplay, err := stateRepo.ListByUser(context.Background(), userID, model.UserUnitStateFilter{})
	if err != nil {
		t.Fatalf("ListByUser(before replay) error = %v", err)
	}

	response, err := replayUsecase.Execute(context.Background(), dto.ReplayUserStatesRequest{UserID: userID})
	if err != nil {
		t.Fatalf("ReplayUserStates.Execute() error = %v", err)
	}
	if response.ProcessedEventCount != 2 {
		t.Fatalf("ProcessedEventCount = %d, want 2", response.ProcessedEventCount)
	}
	if response.RebuiltUnitCount != 2 {
		t.Fatalf("RebuiltUnitCount = %d, want 2", response.RebuiltUnitCount)
	}

	afterReplay, err := stateRepo.ListByUser(context.Background(), userID, model.UserUnitStateFilter{})
	if err != nil {
		t.Fatalf("ListByUser(after replay) error = %v", err)
	}

	beforeByUnit := indexStatesByUnit(beforeReplay)
	afterByUnit := indexStatesByUnit(afterReplay)

	if afterByUnit[101].Status != "reviewing" {
		t.Fatalf("unit 101 status = %q, want reviewing", afterByUnit[101].Status)
	}
	if afterByUnit[102].Status != "new" {
		t.Fatalf("unit 102 status = %q, want new", afterByUnit[102].Status)
	}
	if afterByUnit[102].ProgressEventCount != 0 {
		t.Fatalf("unit 102 progress_event_count = %d, want 0", afterByUnit[102].ProgressEventCount)
	}
	if afterByUnit[101].ScheduleRepetition != beforeByUnit[101].ScheduleRepetition || afterByUnit[101].ScheduleIntervalDays != beforeByUnit[101].ScheduleIntervalDays {
		t.Fatalf("unit 101 replay progression mismatch: before=%+v after=%+v", beforeByUnit[101], afterByUnit[101])
	}
}

func TestReplayAndRecordSerializeForSameUser(t *testing.T) {
	t.Parallel()

	db := testDB(t)
	userID := "11111111-1111-1111-1111-111111111111"
	db.SeedUser(t, userID)
	db.SeedCoarseUnit(t, 101)

	baseManager := persisttx.NewManager(db.Pool)
	replayGate := newBlockingUserTxManager(baseManager, userID)
	recordUsecase := service.NewRecordLearningEventsUsecase(baseManager)
	replayUsecase := service.NewReplayUserStatesUsecase(replayGate)
	stateRepo := persistrepo.NewUserUnitStateRepository(db.Pool)

	firstQuality := int16(4)
	firstOccurredAt := time.Date(2026, 4, 16, 10, 0, 0, 0, time.UTC)
	if _, err := recordUsecase.Execute(context.Background(), dto.RecordLearningEventsRequest{
		UserID: userID,
		Events: []dto.LearningEventInput{
			{CoarseUnitID: 101, EventType: "quiz", ReducerEffect: "affects_progress", SourceType: "quiz_event", SourceRefID: "serialize-1", ProgressQuality: &firstQuality, OccurredAt: firstOccurredAt},
		},
	}); err != nil {
		t.Fatalf("seed RecordLearningEvents.Execute() error = %v", err)
	}

	var replayErr error
	var replayWG sync.WaitGroup
	replayWG.Add(1)
	go func() {
		defer replayWG.Done()
		_, replayErr = replayUsecase.Execute(context.Background(), dto.ReplayUserStatesRequest{UserID: userID})
	}()

	<-replayGate.started

	recordDone := make(chan error, 1)
	go func() {
		secondQuality := int16(4)
		_, err := recordUsecase.Execute(context.Background(), dto.RecordLearningEventsRequest{
			UserID: userID,
			Events: []dto.LearningEventInput{
				{CoarseUnitID: 101, EventType: "quiz", ReducerEffect: "affects_progress", SourceType: "quiz_event", SourceRefID: "serialize-2", ProgressQuality: &secondQuality, OccurredAt: firstOccurredAt.Add(24 * time.Hour)},
			},
		})
		recordDone <- err
	}()

	select {
	case err := <-recordDone:
		t.Fatalf("record completed before replay released lock: %v", err)
	case <-time.After(150 * time.Millisecond):
	}

	close(replayGate.release)
	replayWG.Wait()
	if replayErr != nil {
		t.Fatalf("ReplayUserStates.Execute() error = %v", replayErr)
	}

	select {
	case err := <-recordDone:
		if err != nil {
			t.Fatalf("RecordLearningEvents.Execute() error = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for blocked record to finish")
	}

	states, err := stateRepo.ListByUser(context.Background(), userID, model.UserUnitStateFilter{})
	if err != nil {
		t.Fatalf("ListByUser() error = %v", err)
	}
	if len(states) != 1 {
		t.Fatalf("states len = %d, want 1", len(states))
	}
	if states[0].Status != "reviewing" || states[0].ProgressEventCount != 2 {
		t.Fatalf("unexpected final state after replay+record serialization: %+v", states[0])
	}

	replayed, err := service.NewReplayUserStatesUsecase(baseManager).Execute(context.Background(), dto.ReplayUserStatesRequest{UserID: userID})
	if err != nil {
		t.Fatalf("final replay Execute() error = %v", err)
	}
	if replayed.ProcessedEventCount != 2 {
		t.Fatalf("ProcessedEventCount after final replay = %d, want 2", replayed.ProcessedEventCount)
	}
}

var errForcedBatchUpsertFailure = errors.New("forced batch upsert failure")

type failingBatchUpsertTxManager struct {
	pool *pgxpool.Pool
}

func (m *failingBatchUpsertTxManager) WithinTx(ctx context.Context, fn func(ctx context.Context, repos service.TransactionalRepositories) error) error {
	tx, err := m.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	repos := failingBatchUpsertRepositories{
		userUnitStates: &failingBatchUpsertUserUnitStateRepository{
			inner: persistrepo.NewUserUnitStateRepository(tx),
		},
		unitLearningEvents: persistrepo.NewUnitLearningEventRepository(tx),
	}
	if err := fn(ctx, repos); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (m *failingBatchUpsertTxManager) WithinUserTx(ctx context.Context, _ string, fn func(ctx context.Context, repos service.TransactionalRepositories) error) error {
	return m.WithinTx(ctx, fn)
}

type failingBatchUpsertRepositories struct {
	userUnitStates     applearningrepo.UserUnitStateRepository
	unitLearningEvents applearningrepo.UnitLearningEventRepository
}

func (r failingBatchUpsertRepositories) UserUnitStates() applearningrepo.UserUnitStateRepository {
	return r.userUnitStates
}

func (r failingBatchUpsertRepositories) TargetCommands() applearningrepo.TargetStateCommandRepository {
	return nil
}

func (r failingBatchUpsertRepositories) UnitLearningEvents() applearningrepo.UnitLearningEventRepository {
	return r.unitLearningEvents
}

func (r failingBatchUpsertRepositories) ActivityStats() userrepo.ActivityStatsRecorder {
	return nil
}

type failingBatchUpsertUserUnitStateRepository struct {
	inner applearningrepo.UserUnitStateRepository
}

func (r *failingBatchUpsertUserUnitStateRepository) GetByUserAndUnit(ctx context.Context, userID string, coarseUnitID int64) (*model.UserUnitState, error) {
	return r.inner.GetByUserAndUnit(ctx, userID, coarseUnitID)
}

func (r *failingBatchUpsertUserUnitStateRepository) GetByUserAndUnitForUpdate(ctx context.Context, userID string, coarseUnitID int64) (*model.UserUnitState, error) {
	return r.inner.GetByUserAndUnitForUpdate(ctx, userID, coarseUnitID)
}

func (r *failingBatchUpsertUserUnitStateRepository) ListByUserAndUnitIDsForUpdate(ctx context.Context, userID string, coarseUnitIDs []int64) (map[int64]*model.UserUnitState, error) {
	return r.inner.ListByUserAndUnitIDsForUpdate(ctx, userID, coarseUnitIDs)
}

func (r *failingBatchUpsertUserUnitStateRepository) Upsert(ctx context.Context, state *model.UserUnitState) (*model.UserUnitState, error) {
	return r.inner.Upsert(ctx, state)
}

func (r *failingBatchUpsertUserUnitStateRepository) BatchUpsert(context.Context, []*model.UserUnitState) ([]*model.UserUnitState, error) {
	return nil, errForcedBatchUpsertFailure
}

func (r *failingBatchUpsertUserUnitStateRepository) DeleteByUser(ctx context.Context, userID string) error {
	return r.inner.DeleteByUser(ctx, userID)
}

func (r *failingBatchUpsertUserUnitStateRepository) ListByUser(ctx context.Context, userID string, filter model.UserUnitStateFilter) ([]model.UserUnitState, error) {
	return r.inner.ListByUser(ctx, userID, filter)
}

func indexStatesByUnit(states []model.UserUnitState) map[int64]model.UserUnitState {
	indexed := make(map[int64]model.UserUnitState, len(states))
	for _, state := range states {
		indexed[state.CoarseUnitID] = state
	}
	return indexed
}

func assertCompletedMasteredState(t *testing.T, state model.UserUnitState, lastProgressAt time.Time, wantTarget bool) {
	t.Helper()

	if state.Status != "mastered" {
		t.Fatalf("status = %q, want mastered", state.Status)
	}
	if state.IsTarget != wantTarget {
		t.Fatalf("is_target = %v, want %v", state.IsTarget, wantTarget)
	}
	if state.ProgressPercent != 100 {
		t.Fatalf("progress_percent = %v, want 100", state.ProgressPercent)
	}
	if state.MasteryScore != 1 {
		t.Fatalf("mastery_score = %v, want 1", state.MasteryScore)
	}
	if state.NextReviewAt != nil {
		t.Fatalf("next_review_at = %v, want nil", state.NextReviewAt)
	}
	if state.LastProgressAt == nil || !state.LastProgressAt.Equal(lastProgressAt) {
		t.Fatalf("last_progress_at = %v, want %v", state.LastProgressAt, lastProgressAt)
	}
}

func assertResetUnlearnedState(t *testing.T, state model.UserUnitState, wantTarget bool) {
	t.Helper()

	if state.IsTarget != wantTarget {
		t.Fatalf("is_target = %v, want %v", state.IsTarget, wantTarget)
	}
	if state.Status != "new" {
		t.Fatalf("status = %q, want new", state.Status)
	}
	if state.ProgressPercent != 0 {
		t.Fatalf("progress_percent = %v, want 0", state.ProgressPercent)
	}
	if state.MasteryScore != 0 {
		t.Fatalf("mastery_score = %v, want 0", state.MasteryScore)
	}
	if state.ObservationCount != 0 || state.ProgressEventCount != 0 {
		t.Fatalf("counts = observation:%d progress:%d, want 0", state.ObservationCount, state.ProgressEventCount)
	}
	if state.LastObservedAt != nil || state.LastProgressAt != nil {
		t.Fatalf("observed/progress timestamps = %v/%v, want nil", state.LastObservedAt, state.LastProgressAt)
	}
	if state.NextReviewAt != nil {
		t.Fatalf("next_review_at = %v, want nil", state.NextReviewAt)
	}
	if state.ScheduleRepetition != 0 || state.ScheduleIntervalDays != 0 || state.ScheduleEaseFactor != 2.5 {
		t.Fatalf("schedule = repetition:%d interval:%v ease:%v, want reset defaults", state.ScheduleRepetition, state.ScheduleIntervalDays, state.ScheduleEaseFactor)
	}
}

type blockingUserTxManager struct {
	inner   service.TxManager
	userID  string
	started chan struct{}
	release chan struct{}
	once    sync.Once
}

func newBlockingUserTxManager(inner service.TxManager, userID string) *blockingUserTxManager {
	return &blockingUserTxManager{
		inner:   inner,
		userID:  userID,
		started: make(chan struct{}),
		release: make(chan struct{}),
	}
}

func (m *blockingUserTxManager) WithinTx(ctx context.Context, fn func(ctx context.Context, repos service.TransactionalRepositories) error) error {
	return m.inner.WithinTx(ctx, fn)
}

func (m *blockingUserTxManager) WithinUserTx(ctx context.Context, userID string, fn func(ctx context.Context, repos service.TransactionalRepositories) error) error {
	return m.inner.WithinUserTx(ctx, userID, func(ctx context.Context, repos service.TransactionalRepositories) error {
		if userID == m.userID {
			m.once.Do(func() { close(m.started) })
			<-m.release
		}
		return fn(ctx, repos)
	})
}
