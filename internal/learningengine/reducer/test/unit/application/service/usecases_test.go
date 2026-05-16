package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"learning-video-recommendation-system/internal/learningengine/reducer/application/dto"
	applearningrepo "learning-video-recommendation-system/internal/learningengine/reducer/application/repository"
	"learning-video-recommendation-system/internal/learningengine/reducer/application/service"
	"learning-video-recommendation-system/internal/learningengine/reducer/domain/enum"
	"learning-video-recommendation-system/internal/learningengine/reducer/domain/model"
)

func TestEnsureTargetUnitsExecute(t *testing.T) {
	targetRepo := &fakeTargetStateCommandRepository{}
	txManager := &fakeTxManager{
		repositories: fakeTransactionalRepositories{
			targetCommands: targetRepo,
		},
	}
	usecase := service.NewEnsureTargetUnitsUsecase(txManager)

	response, err := usecase.Execute(context.Background(), dto.EnsureTargetUnitsRequest{
		UserID: "11111111-1111-1111-1111-111111111111",
		Targets: []dto.TargetUnitSpec{
			{CoarseUnitID: 101, TargetSource: "curriculum", TargetSourceRefID: "lesson_1", TargetPriority: 0.9},
			{CoarseUnitID: 102, TargetSource: "curriculum", TargetSourceRefID: "lesson_1", TargetPriority: 0.8},
		},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if response.TargetCount != 2 {
		t.Fatalf("TargetCount = %d, want 2", response.TargetCount)
	}
	if !txManager.withinUserCalled || txManager.lastLockedUserID != "11111111-1111-1111-1111-111111111111" {
		t.Fatalf("expected user-scoped transaction lock, got called=%v user=%q", txManager.withinUserCalled, txManager.lastLockedUserID)
	}
	if len(targetRepo.targets) != 2 {
		t.Fatalf("targets forwarded = %d, want 2", len(targetRepo.targets))
	}
}

func TestSetTargetInactiveExecute(t *testing.T) {
	targetRepo := &fakeTargetStateCommandRepository{}
	txManager := &fakeTxManager{
		repositories: fakeTransactionalRepositories{
			targetCommands: targetRepo,
		},
	}
	usecase := service.NewSetTargetInactiveUsecase(txManager)

	_, err := usecase.Execute(context.Background(), dto.SetTargetInactiveRequest{
		UserID:       "11111111-1111-1111-1111-111111111111",
		CoarseUnitID: 101,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if targetRepo.inactiveUnitID != 101 {
		t.Fatalf("inactive unit = %d, want 101", targetRepo.inactiveUnitID)
	}
	if !txManager.withinUserCalled || txManager.lastLockedUserID != "11111111-1111-1111-1111-111111111111" {
		t.Fatalf("expected user-scoped transaction lock, got called=%v user=%q", txManager.withinUserCalled, txManager.lastLockedUserID)
	}
}

func TestSuspendTargetUnitExecute(t *testing.T) {
	stateRepo := &fakeUserUnitStateRepository{
		state: &model.UserUnitState{
			UserID:             "11111111-1111-1111-1111-111111111111",
			CoarseUnitID:       101,
			IsTarget:           true,
			Status:             enum.StatusReviewing,
			ScheduleEaseFactor: 2.5,
		},
	}
	txManager := &fakeTxManager{
		repositories: fakeTransactionalRepositories{
			userUnitStates: stateRepo,
		},
	}
	usecase := service.NewSuspendTargetUnitUsecase(txManager)

	_, err := usecase.Execute(context.Background(), dto.SuspendTargetUnitRequest{
		UserID:          "11111111-1111-1111-1111-111111111111",
		CoarseUnitID:    101,
		SuspendedReason: "manual_pause",
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !txManager.withinUserCalled || txManager.lastLockedUserID != "11111111-1111-1111-1111-111111111111" {
		t.Fatalf("expected user-scoped transaction lock, got called=%v user=%q", txManager.withinUserCalled, txManager.lastLockedUserID)
	}
	if stateRepo.upserted == nil {
		t.Fatalf("state was not upserted")
	}
	if stateRepo.upserted.Status != enum.StatusSuspended {
		t.Fatalf("status = %q, want %q", stateRepo.upserted.Status, enum.StatusSuspended)
	}
	if stateRepo.upserted.SuspendedReason != "manual_pause" {
		t.Fatalf("suspended_reason = %q, want manual_pause", stateRepo.upserted.SuspendedReason)
	}
}

func TestResumeTargetUnitExecuteRecomputesStatus(t *testing.T) {
	lastProgressAt := time.Date(2026, 4, 16, 10, 0, 0, 0, time.UTC)
	lastQuality := int16(4)
	stateRepo := &fakeUserUnitStateRepository{
		state: &model.UserUnitState{
			UserID:                  "11111111-1111-1111-1111-111111111111",
			CoarseUnitID:            101,
			IsTarget:                true,
			Status:                  enum.StatusSuspended,
			SuspendedReason:         "manual_pause",
			ProgressEventCount:      2,
			ProgressSuccessCount:    2,
			ConsecutiveSuccessCount: 2,
			LastProgressQuality:     &lastQuality,
			RecentProgressQualities: []int16{4, 4},
			RecentProgressPasses:    []bool{true, true},
			ScheduleRepetition:      2,
			ScheduleIntervalDays:    3,
			ScheduleEaseFactor:      2.5,
			LastProgressAt:          &lastProgressAt,
		},
	}
	txManager := &fakeTxManager{
		repositories: fakeTransactionalRepositories{
			userUnitStates: stateRepo,
		},
	}
	usecase := service.NewResumeTargetUnitUsecase(txManager)

	_, err := usecase.Execute(context.Background(), dto.ResumeTargetUnitRequest{
		UserID:       "11111111-1111-1111-1111-111111111111",
		CoarseUnitID: 101,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if stateRepo.upserted == nil {
		t.Fatalf("state was not upserted")
	}
	if stateRepo.upserted.Status != enum.StatusReviewing {
		t.Fatalf("status = %q, want %q", stateRepo.upserted.Status, enum.StatusReviewing)
	}
	if stateRepo.upserted.SuspendedReason != "" {
		t.Fatalf("suspended_reason = %q, want empty", stateRepo.upserted.SuspendedReason)
	}
}

func TestResumeTargetUnitExecuteReturnsNotFound(t *testing.T) {
	txManager := &fakeTxManager{
		repositories: fakeTransactionalRepositories{
			userUnitStates: &fakeUserUnitStateRepository{},
		},
	}
	usecase := service.NewResumeTargetUnitUsecase(txManager)

	_, err := usecase.Execute(context.Background(), dto.ResumeTargetUnitRequest{
		UserID:       "11111111-1111-1111-1111-111111111111",
		CoarseUnitID: 101,
	})
	if !errors.Is(err, service.ErrUserUnitStateNotFound) {
		t.Fatalf("Execute() error = %v, want ErrUserUnitStateNotFound", err)
	}
}

func TestListUserUnitStatesExecuteUsesFilter(t *testing.T) {
	stateRepo := &fakeUserUnitStateRepository{
		listStates: []model.UserUnitState{{UserID: "11111111-1111-1111-1111-111111111111", CoarseUnitID: 101}},
	}
	usecase := service.NewListUserUnitStatesUsecase(stateRepo)

	response, err := usecase.Execute(context.Background(), dto.ListUserUnitStatesRequest{
		UserID:           "11111111-1111-1111-1111-111111111111",
		OnlyTarget:       true,
		ExcludeSuspended: true,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !stateRepo.lastFilter.OnlyTarget || !stateRepo.lastFilter.ExcludeSuspended {
		t.Fatalf("filter = %+v, want both flags true", stateRepo.lastFilter)
	}
	if len(response.States) != 1 {
		t.Fatalf("states len = %d, want 1", len(response.States))
	}
}

func TestRecordLearningEventsExecuteReducesSortedEvents(t *testing.T) {
	q4 := int16(4)
	t1 := time.Date(2026, 4, 16, 10, 0, 0, 0, time.UTC)
	t2 := t1.Add(24 * time.Hour)

	stateRepo := &fakeUserUnitStateRepository{}
	eventRepo := &fakeUnitLearningEventRepository{}
	txManager := &fakeTxManager{
		repositories: fakeTransactionalRepositories{
			userUnitStates: stateRepo,
			unitEvents:     eventRepo,
		},
	}
	usecase := service.NewRecordLearningEventsUsecase(txManager)

	response, err := usecase.Execute(context.Background(), dto.RecordLearningEventsRequest{
		UserID: "11111111-1111-1111-1111-111111111111",
		Events: []dto.LearningEventInput{
			{CoarseUnitID: 101, EventType: enum.EventQuiz, ReducerEffect: enum.ReducerEffectAffectsProgress, SourceType: "quiz_event", SourceRefID: "event_2", ProgressQuality: &q4, OccurredAt: t2},
			{CoarseUnitID: 101, EventType: enum.EventQuiz, ReducerEffect: enum.ReducerEffectAffectsProgress, SourceType: "quiz_event", SourceRefID: "event_1", ProgressQuality: &q4, OccurredAt: t1},
		},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if response.RecordedCount != 2 {
		t.Fatalf("RecordedCount = %d, want 2", response.RecordedCount)
	}
	if len(eventRepo.appended) != 2 {
		t.Fatalf("appended events = %d, want 2", len(eventRepo.appended))
	}
	if len(stateRepo.batchUpserted) != 1 {
		t.Fatalf("upserted states = %d, want 1", len(stateRepo.batchUpserted))
	}
	if stateRepo.batchUpserted[0].Status != enum.StatusReviewing {
		t.Fatalf("status = %q, want %q", stateRepo.batchUpserted[0].Status, enum.StatusReviewing)
	}
}

func TestRecordLearningEventsExecuteNormalizesOccurredAtToUTC(t *testing.T) {
	q4 := int16(4)
	localTime := time.Date(2026, 4, 16, 10, 0, 0, 0, time.FixedZone("PDT", -7*60*60))

	stateRepo := &fakeUserUnitStateRepository{}
	eventRepo := &fakeUnitLearningEventRepository{}
	txManager := &fakeTxManager{
		repositories: fakeTransactionalRepositories{
			userUnitStates: stateRepo,
			unitEvents:     eventRepo,
		},
	}
	usecase := service.NewRecordLearningEventsUsecase(txManager)

	_, err := usecase.Execute(context.Background(), dto.RecordLearningEventsRequest{
		UserID: "11111111-1111-1111-1111-111111111111",
		Events: []dto.LearningEventInput{
			{CoarseUnitID: 101, EventType: enum.EventQuiz, ReducerEffect: enum.ReducerEffectAffectsProgress, SourceType: "quiz_event", SourceRefID: "event_utc", ProgressQuality: &q4, OccurredAt: localTime},
		},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(eventRepo.appended) != 1 {
		t.Fatalf("appended events = %d, want 1", len(eventRepo.appended))
	}
	got := eventRepo.appended[0].OccurredAt
	if got.Location() != time.UTC {
		t.Fatalf("OccurredAt location = %v, want UTC", got.Location())
	}
	if !got.Equal(localTime) {
		t.Fatalf("OccurredAt = %v, want same instant as %v", got, localTime)
	}
}

func TestRecordLearningEventsExecuteSetMasteredTerminalState(t *testing.T) {
	t1 := time.Date(2026, 4, 16, 10, 0, 0, 0, time.UTC)
	t2 := t1.Add(24 * time.Hour)
	q1 := int16(1)

	stateRepo := &fakeUserUnitStateRepository{
		statesByUnit: map[int64]*model.UserUnitState{
			101: {
				UserID:             "11111111-1111-1111-1111-111111111111",
				CoarseUnitID:       101,
				IsTarget:           true,
				Status:             enum.StatusReviewing,
				ScheduleEaseFactor: 2.5,
			},
		},
	}
	eventRepo := &fakeUnitLearningEventRepository{}
	txManager := &fakeTxManager{
		repositories: fakeTransactionalRepositories{
			userUnitStates: stateRepo,
			unitEvents:     eventRepo,
		},
	}
	usecase := service.NewRecordLearningEventsUsecase(txManager)

	response, err := usecase.Execute(context.Background(), dto.RecordLearningEventsRequest{
		UserID: "11111111-1111-1111-1111-111111111111",
		Events: []dto.LearningEventInput{
			{CoarseUnitID: 101, EventType: enum.EventSelfMarkMastered, ReducerEffect: enum.ReducerEffectSetMastered, SourceType: "learning_interaction_event", SourceRefID: "self-mark-1", OccurredAt: t1},
			{CoarseUnitID: 101, EventType: enum.EventQuiz, ReducerEffect: enum.ReducerEffectAffectsProgress, SourceType: "quiz_event", SourceRefID: "quiz-after-mastered", ProgressQuality: &q1, OccurredAt: t2},
		},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if response.RecordedCount != 2 {
		t.Fatalf("RecordedCount = %d, want 2", response.RecordedCount)
	}
	if len(eventRepo.appended) != 2 {
		t.Fatalf("appended events = %d, want 2", len(eventRepo.appended))
	}
	if len(stateRepo.batchUpserted) != 1 {
		t.Fatalf("upserted states = %d, want 1", len(stateRepo.batchUpserted))
	}
	state := stateRepo.batchUpserted[0]
	if state.Status != enum.StatusMastered {
		t.Fatalf("status = %q, want %q", state.Status, enum.StatusMastered)
	}
	if state.IsTarget {
		t.Fatalf("is_target = true, want false")
	}
	if state.ProgressPercent != 100 {
		t.Fatalf("progress_percent = %v, want 100", state.ProgressPercent)
	}
	if state.MasteryScore != 1 {
		t.Fatalf("mastery_score = %v, want 1", state.MasteryScore)
	}
	if state.ProgressEventCount != 0 {
		t.Fatalf("progress_event_count = %d, want 0", state.ProgressEventCount)
	}
}

func TestRecordLearningEventsExecuteRejectsLateProgressEvent(t *testing.T) {
	lastProgressAt := time.Date(2026, 4, 16, 10, 0, 0, 0, time.UTC)
	q4 := int16(4)

	stateRepo := &fakeUserUnitStateRepository{
		statesByUnit: map[int64]*model.UserUnitState{
			101: {
				UserID:             "11111111-1111-1111-1111-111111111111",
				CoarseUnitID:       101,
				IsTarget:           true,
				Status:             enum.StatusReviewing,
				LastProgressAt:     &lastProgressAt,
				ScheduleEaseFactor: 2.5,
			},
		},
	}
	txManager := &fakeTxManager{
		repositories: fakeTransactionalRepositories{
			userUnitStates: stateRepo,
			unitEvents:     &fakeUnitLearningEventRepository{},
		},
	}
	usecase := service.NewRecordLearningEventsUsecase(txManager)

	_, err := usecase.Execute(context.Background(), dto.RecordLearningEventsRequest{
		UserID: "11111111-1111-1111-1111-111111111111",
		Events: []dto.LearningEventInput{
			{CoarseUnitID: 101, EventType: enum.EventQuiz, ReducerEffect: enum.ReducerEffectAffectsProgress, SourceType: "quiz_event", SourceRefID: "event_1", ProgressQuality: &q4, OccurredAt: lastProgressAt.Add(-time.Hour)},
		},
	})
	if !errors.Is(err, service.ErrLateProgressEvent) {
		t.Fatalf("Execute() error = %v, want ErrLateProgressEvent", err)
	}
}

func TestRecordLearningEventsExecuteHandlesMultipleUnits(t *testing.T) {
	q4 := int16(4)
	t1 := time.Date(2026, 4, 16, 10, 0, 0, 0, time.UTC)

	stateRepo := &fakeUserUnitStateRepository{}
	txManager := &fakeTxManager{
		repositories: fakeTransactionalRepositories{
			userUnitStates: stateRepo,
			unitEvents:     &fakeUnitLearningEventRepository{},
		},
	}
	usecase := service.NewRecordLearningEventsUsecase(txManager)

	_, err := usecase.Execute(context.Background(), dto.RecordLearningEventsRequest{
		UserID: "11111111-1111-1111-1111-111111111111",
		Events: []dto.LearningEventInput{
			{CoarseUnitID: 101, EventType: enum.EventQuiz, ReducerEffect: enum.ReducerEffectAffectsProgress, SourceType: "quiz_event", SourceRefID: "event_1", ProgressQuality: &q4, OccurredAt: t1},
			{CoarseUnitID: 102, EventType: enum.EventQuiz, ReducerEffect: enum.ReducerEffectAffectsProgress, SourceType: "quiz_event", SourceRefID: "event_2", ProgressQuality: &q4, OccurredAt: t1},
		},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if len(stateRepo.batchUpserted) != 2 {
		t.Fatalf("upserted states = %d, want 2", len(stateRepo.batchUpserted))
	}
}

func TestRecordLearningEventsExecuteLoadsAffectedStatesInOneBatch(t *testing.T) {
	q4 := int16(4)
	t1 := time.Date(2026, 4, 16, 10, 0, 0, 0, time.UTC)

	stateRepo := &fakeUserUnitStateRepository{
		statesByUnit: map[int64]*model.UserUnitState{
			101: {UserID: "11111111-1111-1111-1111-111111111111", CoarseUnitID: 101, IsTarget: true, Status: "new", ScheduleEaseFactor: 2.5},
			202: {UserID: "11111111-1111-1111-1111-111111111111", CoarseUnitID: 202, IsTarget: true, Status: "new", ScheduleEaseFactor: 2.5},
		},
	}
	txManager := &fakeTxManager{
		repositories: fakeTransactionalRepositories{
			userUnitStates: stateRepo,
			unitEvents:     &fakeUnitLearningEventRepository{},
		},
	}
	usecase := service.NewRecordLearningEventsUsecase(txManager)

	_, err := usecase.Execute(context.Background(), dto.RecordLearningEventsRequest{
		UserID: "11111111-1111-1111-1111-111111111111",
		Events: []dto.LearningEventInput{
			{CoarseUnitID: 101, EventType: enum.EventQuiz, ReducerEffect: enum.ReducerEffectAffectsProgress, SourceType: "quiz_event", SourceRefID: "batch-state-1", ProgressQuality: &q4, OccurredAt: t1},
			{CoarseUnitID: 202, EventType: enum.EventQuiz, ReducerEffect: enum.ReducerEffectAffectsProgress, SourceType: "quiz_event", SourceRefID: "batch-state-2", ProgressQuality: &q4, OccurredAt: t1.Add(time.Second)},
		},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if stateRepo.getForUpdateCalls != 0 {
		t.Fatalf("GetByUserAndUnitForUpdate calls = %d, want 0; states should be loaded in one batch", stateRepo.getForUpdateCalls)
	}
	if len(stateRepo.batchUpserted) != 2 {
		t.Fatalf("BatchUpsert states = %d, want 2", len(stateRepo.batchUpserted))
	}
}

func TestRecordLearningEventsExecuteSkipsDuplicateAppendRows(t *testing.T) {
	q4 := int16(4)
	t1 := time.Date(2026, 4, 16, 10, 0, 0, 0, time.UTC)

	stateRepo := &fakeUserUnitStateRepository{}
	eventRepo := &fakeUnitLearningEventRepository{
		appendResult: applearningrepo.AppendLearningEventsResult{
			InsertedEvents: []model.LearningEvent{
				{
					UserID:          "11111111-1111-1111-1111-111111111111",
					CoarseUnitID:    101,
					EventType:       enum.EventQuiz,
					ReducerEffect:   enum.ReducerEffectAffectsProgress,
					SourceType:      "quiz_event",
					SourceRefID:     "event_2",
					ProgressQuality: &q4,
					Metadata:        []byte("{}"),
					OccurredAt:      t1.Add(time.Hour),
				},
			},
			DuplicateCount: 1,
		},
	}
	txManager := &fakeTxManager{
		repositories: fakeTransactionalRepositories{
			userUnitStates: stateRepo,
			unitEvents:     eventRepo,
		},
	}
	usecase := service.NewRecordLearningEventsUsecase(txManager)

	response, err := usecase.Execute(context.Background(), dto.RecordLearningEventsRequest{
		UserID: "11111111-1111-1111-1111-111111111111",
		Events: []dto.LearningEventInput{
			{CoarseUnitID: 101, EventType: enum.EventQuiz, ReducerEffect: enum.ReducerEffectAffectsProgress, SourceType: "quiz_event", SourceRefID: "event_1", ProgressQuality: &q4, OccurredAt: t1},
			{CoarseUnitID: 101, EventType: enum.EventQuiz, ReducerEffect: enum.ReducerEffectAffectsProgress, SourceType: "quiz_event", SourceRefID: "event_2", ProgressQuality: &q4, OccurredAt: t1.Add(time.Hour)},
		},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if response.ReceivedCount != 2 || response.RecordedCount != 1 || response.DuplicateCount != 1 {
		t.Fatalf("response = %+v, want received=2 recorded=1 duplicate=1", response)
	}
	if len(stateRepo.batchUpserted) != 1 {
		t.Fatalf("upserted states = %d, want 1", len(stateRepo.batchUpserted))
	}
	if stateRepo.batchUpserted[0].ProgressEventCount != 1 {
		t.Fatalf("progress_event_count = %d, want 1", stateRepo.batchUpserted[0].ProgressEventCount)
	}
}

func TestReplayUserStatesExecutePreservesSetMasteredInactiveTarget(t *testing.T) {
	eventTime := time.Date(2026, 4, 16, 10, 0, 0, 0, time.UTC)
	eventsRepo := &fakeUnitLearningEventRepository{
		listByUserOrdered: []model.LearningEvent{
			{
				UserID:        "11111111-1111-1111-1111-111111111111",
				CoarseUnitID:  101,
				EventType:     enum.EventSelfMarkMastered,
				ReducerEffect: enum.ReducerEffectSetMastered,
				SourceType:    "learning_interaction_event",
				SourceRefID:   "self-mark-1",
				Metadata:      []byte("{}"),
				OccurredAt:    eventTime,
			},
		},
	}
	stateRepo := &fakeUserUnitStateRepository{
		listStates: []model.UserUnitState{
			{
				UserID:            "11111111-1111-1111-1111-111111111111",
				CoarseUnitID:      101,
				IsTarget:          true,
				TargetSource:      "curriculum",
				TargetSourceRefID: "lesson_1",
				TargetPriority:    0.9,
			},
		},
	}
	txManager := &fakeTxManager{
		repositories: fakeTransactionalRepositories{
			userUnitStates: stateRepo,
			unitEvents:     eventsRepo,
		},
	}
	usecase := service.NewReplayUserStatesUsecase(txManager)

	response, err := usecase.Execute(context.Background(), dto.ReplayUserStatesRequest{
		UserID: "11111111-1111-1111-1111-111111111111",
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if response.ProcessedEventCount != 1 {
		t.Fatalf("ProcessedEventCount = %d, want 1", response.ProcessedEventCount)
	}
	if len(stateRepo.batchUpserted) != 1 {
		t.Fatalf("upserted states = %d, want 1", len(stateRepo.batchUpserted))
	}
	state := stateRepo.batchUpserted[0]
	if state.Status != enum.StatusMastered {
		t.Fatalf("status = %q, want %q", state.Status, enum.StatusMastered)
	}
	if state.IsTarget {
		t.Fatalf("is_target = true, want false")
	}
	if state.TargetSource != "curriculum" {
		t.Fatalf("target_source = %q, want curriculum", state.TargetSource)
	}
	if state.ProgressPercent != 100 {
		t.Fatalf("progress_percent = %v, want 100", state.ProgressPercent)
	}
	if state.MasteryScore != 1 {
		t.Fatalf("mastery_score = %v, want 1", state.MasteryScore)
	}
}

func TestReplayUserStatesExecutePreservesControlSliceAndTargetOnlyRows(t *testing.T) {
	lastProgressAt := time.Date(2026, 4, 16, 10, 0, 0, 0, time.UTC)
	eventsRepo := &fakeUnitLearningEventRepository{
		listByUserOrdered: []model.LearningEvent{
			{
				UserID:          "11111111-1111-1111-1111-111111111111",
				CoarseUnitID:    101,
				EventType:       enum.EventQuiz,
				ReducerEffect:   enum.ReducerEffectAffectsProgress,
				SourceType:      "quiz_event",
				SourceRefID:     "event_1",
				ProgressQuality: int16Pointer(4),
				Metadata:        []byte("{}"),
				OccurredAt:      lastProgressAt.Add(-24 * time.Hour),
			},
			{
				UserID:          "11111111-1111-1111-1111-111111111111",
				CoarseUnitID:    101,
				EventType:       enum.EventQuiz,
				ReducerEffect:   enum.ReducerEffectAffectsProgress,
				SourceType:      "quiz_event",
				SourceRefID:     "event_2",
				ProgressQuality: int16Pointer(4),
				Metadata:        []byte("{}"),
				OccurredAt:      lastProgressAt,
			},
		},
	}
	stateRepo := &fakeUserUnitStateRepository{
		listStates: []model.UserUnitState{
			{
				UserID:            "11111111-1111-1111-1111-111111111111",
				CoarseUnitID:      101,
				IsTarget:          true,
				TargetSource:      "curriculum",
				TargetSourceRefID: "lesson_1",
				TargetPriority:    0.9,
			},
			{
				UserID:            "11111111-1111-1111-1111-111111111111",
				CoarseUnitID:      102,
				IsTarget:          true,
				TargetSource:      "curriculum",
				TargetSourceRefID: "lesson_1",
				TargetPriority:    0.8,
				Status:            enum.StatusSuspended,
				SuspendedReason:   "manual_pause",
			},
		},
	}
	txManager := &fakeTxManager{
		repositories: fakeTransactionalRepositories{
			userUnitStates: stateRepo,
			unitEvents:     eventsRepo,
		},
	}
	usecase := service.NewReplayUserStatesUsecase(txManager)

	response, err := usecase.Execute(context.Background(), dto.ReplayUserStatesRequest{
		UserID: "11111111-1111-1111-1111-111111111111",
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !stateRepo.deleteCalled {
		t.Fatalf("DeleteByUser() was not called")
	}
	if response.ProcessedEventCount != 2 {
		t.Fatalf("ProcessedEventCount = %d, want 2", response.ProcessedEventCount)
	}
	if response.RebuiltUnitCount != 2 {
		t.Fatalf("RebuiltUnitCount = %d, want 2", response.RebuiltUnitCount)
	}
	if len(stateRepo.batchUpserted) != 2 {
		t.Fatalf("upserted states = %d, want 2", len(stateRepo.batchUpserted))
	}

	rebuiltByUnit := make(map[int64]*model.UserUnitState, len(stateRepo.batchUpserted))
	for _, state := range stateRepo.batchUpserted {
		rebuiltByUnit[state.CoarseUnitID] = state
	}

	if rebuiltByUnit[101].Status != enum.StatusReviewing {
		t.Fatalf("unit 101 status = %q, want %q", rebuiltByUnit[101].Status, enum.StatusReviewing)
	}
	if rebuiltByUnit[101].TargetSource != "curriculum" {
		t.Fatalf("unit 101 target_source = %q, want curriculum", rebuiltByUnit[101].TargetSource)
	}
	if rebuiltByUnit[102].Status != enum.StatusSuspended {
		t.Fatalf("unit 102 status = %q, want %q", rebuiltByUnit[102].Status, enum.StatusSuspended)
	}
	if rebuiltByUnit[102].ProgressEventCount != 0 {
		t.Fatalf("unit 102 progress_event_count = %d, want 0", rebuiltByUnit[102].ProgressEventCount)
	}
}

type fakeTxManager struct {
	called           bool
	withinUserCalled bool
	lastLockedUserID string
	repositories     fakeTransactionalRepositories
}

func (f *fakeTxManager) WithinTx(ctx context.Context, fn func(ctx context.Context, repos service.TransactionalRepositories) error) error {
	f.called = true
	return fn(ctx, f.repositories)
}

func (f *fakeTxManager) WithinUserTx(ctx context.Context, userID string, fn func(ctx context.Context, repos service.TransactionalRepositories) error) error {
	f.withinUserCalled = true
	f.lastLockedUserID = userID
	return fn(ctx, f.repositories)
}

type fakeTransactionalRepositories struct {
	userUnitStates applearningrepo.UserUnitStateRepository
	targetCommands applearningrepo.TargetStateCommandRepository
	unitEvents     applearningrepo.UnitLearningEventRepository
}

func (f fakeTransactionalRepositories) UserUnitStates() applearningrepo.UserUnitStateRepository {
	return f.userUnitStates
}

func (f fakeTransactionalRepositories) TargetCommands() applearningrepo.TargetStateCommandRepository {
	return f.targetCommands
}

func (f fakeTransactionalRepositories) UnitLearningEvents() applearningrepo.UnitLearningEventRepository {
	return f.unitEvents
}

type fakeTargetStateCommandRepository struct {
	targets        []model.TargetUnitSpec
	inactiveUnitID int64
}

func (f *fakeTargetStateCommandRepository) EnsureTargetUnits(_ context.Context, _ string, targets []model.TargetUnitSpec) error {
	f.targets = targets
	return nil
}

func (f *fakeTargetStateCommandRepository) SetTargetInactive(_ context.Context, _ string, coarseUnitID int64) error {
	f.inactiveUnitID = coarseUnitID
	return nil
}

type fakeUserUnitStateRepository struct {
	state             *model.UserUnitState
	statesByUnit      map[int64]*model.UserUnitState
	upserted          *model.UserUnitState
	batchUpserted     []*model.UserUnitState
	listStates        []model.UserUnitState
	lastFilter        model.UserUnitStateFilter
	deleteCalled      bool
	getForUpdateCalls int
}

func (f *fakeUserUnitStateRepository) GetByUserAndUnitForUpdate(_ context.Context, _ string, coarseUnitID int64) (*model.UserUnitState, error) {
	f.getForUpdateCalls++
	if f.statesByUnit != nil {
		return f.statesByUnit[coarseUnitID], nil
	}
	return f.state, nil
}

func (f *fakeUserUnitStateRepository) ListByUserAndUnitIDsForUpdate(_ context.Context, _ string, coarseUnitIDs []int64) (map[int64]*model.UserUnitState, error) {
	result := make(map[int64]*model.UserUnitState, len(coarseUnitIDs))
	for _, coarseUnitID := range coarseUnitIDs {
		if f.statesByUnit != nil {
			if state := f.statesByUnit[coarseUnitID]; state != nil {
				result[coarseUnitID] = state
			}
			continue
		}
		if f.state != nil && f.state.CoarseUnitID == coarseUnitID {
			result[coarseUnitID] = f.state
		}
	}
	return result, nil
}

func (f *fakeUserUnitStateRepository) Upsert(_ context.Context, state *model.UserUnitState) (*model.UserUnitState, error) {
	cloned := *state
	f.upserted = &cloned
	return &cloned, nil
}

func (f *fakeUserUnitStateRepository) BatchUpsert(_ context.Context, states []*model.UserUnitState) ([]*model.UserUnitState, error) {
	f.batchUpserted = states
	return states, nil
}

func (f *fakeUserUnitStateRepository) DeleteByUser(_ context.Context, _ string) error {
	f.deleteCalled = true
	return nil
}

func (f *fakeUserUnitStateRepository) ListByUser(_ context.Context, _ string, filter model.UserUnitStateFilter) ([]model.UserUnitState, error) {
	f.lastFilter = filter
	return f.listStates, nil
}

type fakeUnitLearningEventRepository struct {
	appended          []model.LearningEvent
	appendResult      applearningrepo.AppendLearningEventsResult
	listByUserOrdered []model.LearningEvent
}

func (f *fakeUnitLearningEventRepository) Append(_ context.Context, events []model.LearningEvent) (applearningrepo.AppendLearningEventsResult, error) {
	f.appended = append(f.appended, events...)
	if f.appendResult.InsertedEvents != nil || f.appendResult.DuplicateCount != 0 {
		return f.appendResult, nil
	}
	return applearningrepo.AppendLearningEventsResult{InsertedEvents: append([]model.LearningEvent(nil), events...)}, nil
}

func (f *fakeUnitLearningEventRepository) ListByUserOrdered(_ context.Context, _ string) ([]model.LearningEvent, error) {
	return f.listByUserOrdered, nil
}

func (f *fakeUnitLearningEventRepository) ListByUserAndUnitOrdered(_ context.Context, _ string, _ int64) ([]model.LearningEvent, error) {
	return nil, nil
}

func int16Pointer(value int16) *int16 {
	return &value
}
