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
	userrepo "learning-video-recommendation-system/internal/user/application/repository"
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

func TestActivateUnitCollectionTargetExecuteUsesUserScopedTransaction(t *testing.T) {
	targetRepo := &fakeTargetStateCommandRepository{
		activation: model.ActivatedUnitCollectionTarget{
			CollectionID:   "11111111-1111-4111-8111-111111111111",
			CollectionSlug: "toefl-core",
			TargetCount:    1000,
		},
	}
	txManager := &fakeTxManager{
		repositories: fakeTransactionalRepositories{
			targetCommands: targetRepo,
		},
	}
	usecase := service.NewActivateUnitCollectionTargetUsecase(txManager)

	response, err := usecase.Execute(context.Background(), dto.ActivateUnitCollectionTargetRequest{
		UserID:         "22222222-2222-4222-8222-222222222222",
		CollectionSlug: "toefl-core",
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !txManager.withinUserCalled || txManager.lastLockedUserID != "22222222-2222-4222-8222-222222222222" {
		t.Fatalf("expected user-scoped transaction lock, got called=%v user=%q", txManager.withinUserCalled, txManager.lastLockedUserID)
	}
	if targetRepo.activatedUserID != "22222222-2222-4222-8222-222222222222" || targetRepo.activatedSlug != "toefl-core" {
		t.Fatalf("activation args = user:%q slug:%q", targetRepo.activatedUserID, targetRepo.activatedSlug)
	}
	if response.CollectionSlug != "toefl-core" || response.TargetCount != 1000 {
		t.Fatalf("response = %+v", response)
	}
}

func TestGetActiveUnitCollectionExecuteReturnsActiveAndPropagatesReaderError(t *testing.T) {
	t.Run("active profile", func(t *testing.T) {
		reader := &fakeActiveUnitCollectionReader{
			active: &model.ActiveUnitCollection{
				CollectionID:   "11111111-1111-4111-8111-111111111111",
				CollectionSlug: "toefl-core",
			},
		}
		usecase := service.NewGetActiveUnitCollectionUsecase(reader)

		response, err := usecase.Execute(context.Background(), dto.GetActiveUnitCollectionRequest{
			UserID: "22222222-2222-4222-8222-222222222222",
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if reader.userID != "22222222-2222-4222-8222-222222222222" {
			t.Fatalf("reader userID = %q", reader.userID)
		}
		if response.ActiveCollection == nil || response.ActiveCollection.CollectionSlug != "toefl-core" {
			t.Fatalf("ActiveCollection = %+v", response.ActiveCollection)
		}
	})

	t.Run("reader error", func(t *testing.T) {
		wantErr := errors.New("database unavailable")
		usecase := service.NewGetActiveUnitCollectionUsecase(&fakeActiveUnitCollectionReader{err: wantErr})

		_, err := usecase.Execute(context.Background(), dto.GetActiveUnitCollectionRequest{
			UserID: "22222222-2222-4222-8222-222222222222",
		})
		if !errors.Is(err, wantErr) {
			t.Fatalf("Execute() error = %v, want %v", err, wantErr)
		}
	})
}

func TestGetActiveLearningTargetCoarseUnitIDsExecute(t *testing.T) {
	t.Run("active targets", func(t *testing.T) {
		active := "toefl-core"
		reader := &fakeActiveUnitCollectionReader{
			activeTargets: model.ActiveLearningTargetCoarseUnitIDs{
				ActiveCollection: &active,
				CoarseUnitIDs:    []int64{101, 205},
			},
		}
		usecase := service.NewGetActiveLearningTargetCoarseUnitIDsUsecase(reader)

		response, err := usecase.Execute(context.Background(), dto.GetActiveLearningTargetCoarseUnitIDsRequest{
			UserID: "22222222-2222-4222-8222-222222222222",
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if reader.activeTargetUserID != "22222222-2222-4222-8222-222222222222" {
			t.Fatalf("reader userID = %q", reader.activeTargetUserID)
		}
		if response.ActiveCollection == nil || *response.ActiveCollection != active {
			t.Fatalf("ActiveCollection = %v, want %q", response.ActiveCollection, active)
		}
		if response.TargetCount != 2 || len(response.CoarseUnitIDs) != 2 || response.CoarseUnitIDs[0] != 101 || response.CoarseUnitIDs[1] != 205 {
			t.Fatalf("response = %+v", response)
		}
	})

	t.Run("no profile returns empty response", func(t *testing.T) {
		usecase := service.NewGetActiveLearningTargetCoarseUnitIDsUsecase(&fakeActiveUnitCollectionReader{})

		response, err := usecase.Execute(context.Background(), dto.GetActiveLearningTargetCoarseUnitIDsRequest{
			UserID: "22222222-2222-4222-8222-222222222222",
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if response.ActiveCollection != nil || response.TargetCount != 0 || len(response.CoarseUnitIDs) != 0 {
			t.Fatalf("response = %+v, want null active collection and empty ids", response)
		}
	})

	t.Run("missing user id", func(t *testing.T) {
		usecase := service.NewGetActiveLearningTargetCoarseUnitIDsUsecase(&fakeActiveUnitCollectionReader{})

		_, err := usecase.Execute(context.Background(), dto.GetActiveLearningTargetCoarseUnitIDsRequest{})
		if err == nil {
			t.Fatalf("Execute() error = nil, want error")
		}
	})
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

func TestListUserUnitStatesExecuteUsesFilter(t *testing.T) {
	stateRepo := &fakeUserUnitStateRepository{
		listStates: []model.UserUnitState{{UserID: "11111111-1111-1111-1111-111111111111", CoarseUnitID: 101}},
	}
	usecase := service.NewListUserUnitStatesUsecase(stateRepo)

	response, err := usecase.Execute(context.Background(), dto.ListUserUnitStatesRequest{
		UserID:     "11111111-1111-1111-1111-111111111111",
		OnlyTarget: true,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !stateRepo.lastFilter.OnlyTarget {
		t.Fatalf("filter = %+v, want only_target true", stateRepo.lastFilter)
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
	if !state.IsTarget {
		t.Fatalf("is_target = false, want true")
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
	if state.LastProgressAt == nil || !state.LastProgressAt.Equal(t1) {
		t.Fatalf("last_progress_at = %v, want %v", state.LastProgressAt, t1)
	}
}

func TestResetUserUnitProgressExecuteWritesResetEventAndClearsState(t *testing.T) {
	occurredAt := time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC)
	lastProgressAt := occurredAt.Add(-24 * time.Hour)
	lastQuality := int16(5)
	stateRepo := &fakeUserUnitStateRepository{
		state: &model.UserUnitState{
			UserID:                  "11111111-1111-1111-1111-111111111111",
			CoarseUnitID:            101,
			IsTarget:                false,
			TargetSource:            "unit_collection",
			TargetSourceRefID:       "collection-1",
			TargetPriority:          0.7,
			Status:                  enum.StatusMastered,
			ProgressPercent:         100,
			MasteryScore:            1,
			ObservationCount:        3,
			ProgressEventCount:      2,
			LastProgressAt:          &lastProgressAt,
			LastProgressQuality:     &lastQuality,
			RecentProgressQualities: []int16{4, 5},
			RecentProgressPasses:    []bool{true, true},
			ProgressSuccessCount:    2,
			ConsecutiveSuccessCount: 2,
			ScheduleRepetition:      2,
			ScheduleIntervalDays:    1,
			ScheduleEaseFactor:      2.6,
		},
	}
	eventRepo := &fakeUnitLearningEventRepository{
		appendResult: applearningrepo.AppendLearningEventsResult{
			InsertedEvents: []model.LearningEvent{{
				EventID:         "55555555-5555-5555-5555-555555555555",
				UserID:          "11111111-1111-1111-1111-111111111111",
				CoarseUnitID:    101,
				EventType:       enum.EventResetUnlearned,
				ReducerEffect:   enum.ReducerEffectResetUnlearned,
				SourceType:      "learning_unit_reset",
				SourceRefID:     "reset-1",
				Metadata:        []byte("{}"),
				OccurredAt:      occurredAt,
				ResetBoundaryAt: &occurredAt,
			}},
		},
	}
	txManager := &fakeTxManager{
		repositories: fakeTransactionalRepositories{
			userUnitStates: stateRepo,
			unitEvents:     eventRepo,
		},
	}
	usecase := service.NewResetUserUnitProgressUsecase(txManager)

	response, err := usecase.Execute(context.Background(), dto.ResetUserUnitProgressRequest{
		UserID:        "11111111-1111-1111-1111-111111111111",
		ClientEventID: "reset-1",
		CoarseUnitID:  101,
		SourceSurface: "word_detail",
		OccurredAt:    occurredAt,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !response.Accepted || !response.Inserted || response.UnitLearningEventID != "55555555-5555-5555-5555-555555555555" {
		t.Fatalf("response = %+v", response)
	}
	if len(eventRepo.appended) != 1 {
		t.Fatalf("appended events = %d, want 1", len(eventRepo.appended))
	}
	event := eventRepo.appended[0]
	if event.EventType != enum.EventResetUnlearned || event.ReducerEffect != enum.ReducerEffectResetUnlearned || event.SourceType != "learning_unit_reset" || event.SourceRefID != "reset-1" {
		t.Fatalf("reset event = %+v", event)
	}
	if len(stateRepo.batchUpserted) != 1 {
		t.Fatalf("batch upserted states = %d, want 1", len(stateRepo.batchUpserted))
	}
	next := stateRepo.batchUpserted[0]
	if next.IsTarget || next.TargetSource != "unit_collection" || next.TargetSourceRefID != "collection-1" || next.TargetPriority != 0.7 {
		t.Fatalf("control fields not preserved: %+v", next)
	}
	if next.Status != enum.StatusNew || next.ProgressPercent != 0 || next.MasteryScore != 0 || next.ProgressEventCount != 0 || next.ObservationCount != 0 {
		t.Fatalf("state not reset: %+v", next)
	}
}

func TestResetUserUnitProgressExecuteUsesStateProjectionForBoundary(t *testing.T) {
	clientOccurredAt := time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC)
	latestOccurredAt := clientOccurredAt.Add(2 * time.Hour)
	latestResetBoundaryAt := clientOccurredAt.Add(time.Hour)
	stateRepo := &fakeUserUnitStateRepository{
		state: &model.UserUnitState{
			UserID:                        "11111111-1111-1111-1111-111111111111",
			CoarseUnitID:                  101,
			IsTarget:                      true,
			Status:                        enum.StatusReviewing,
			ProgressPercent:               50,
			MasteryScore:                  0.5,
			ScheduleEaseFactor:            2.5,
			LatestLearningEventOccurredAt: &latestOccurredAt,
			LatestResetBoundaryAt:         &latestResetBoundaryAt,
		},
	}
	eventRepo := &fakeUnitLearningEventRepository{
		appendResult: applearningrepo.AppendLearningEventsResult{
			InsertedEvents: []model.LearningEvent{{
				EventID:         "55555555-5555-5555-5555-555555555555",
				LedgerSeq:       77,
				UserID:          "11111111-1111-1111-1111-111111111111",
				CoarseUnitID:    101,
				EventType:       enum.EventResetUnlearned,
				ReducerEffect:   enum.ReducerEffectResetUnlearned,
				SourceType:      "learning_unit_reset",
				SourceRefID:     "reset-1",
				Metadata:        []byte("{}"),
				OccurredAt:      clientOccurredAt,
				ResetBoundaryAt: &latestOccurredAt,
			}},
		},
	}
	txManager := &fakeTxManager{
		repositories: fakeTransactionalRepositories{
			userUnitStates: stateRepo,
			unitEvents:     eventRepo,
		},
	}
	usecase := service.NewResetUserUnitProgressUsecase(txManager)

	_, err := usecase.Execute(context.Background(), dto.ResetUserUnitProgressRequest{
		UserID:        "11111111-1111-1111-1111-111111111111",
		ClientEventID: "reset-1",
		CoarseUnitID:  101,
		SourceSurface: "word_detail",
		OccurredAt:    clientOccurredAt,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if len(eventRepo.appended) != 1 {
		t.Fatalf("appended events = %d, want 1", len(eventRepo.appended))
	}
	if eventRepo.appended[0].ResetBoundaryAt == nil || !eventRepo.appended[0].ResetBoundaryAt.Equal(latestOccurredAt) {
		t.Fatalf("reset boundary = %v, want %v", eventRepo.appended[0].ResetBoundaryAt, latestOccurredAt)
	}
	if len(stateRepo.batchUpserted) != 1 {
		t.Fatalf("batch upserted states = %d, want 1", len(stateRepo.batchUpserted))
	}
	next := stateRepo.batchUpserted[0]
	if next.LatestResetBoundaryAt == nil || !next.LatestResetBoundaryAt.Equal(latestOccurredAt) {
		t.Fatalf("state latest reset boundary = %v, want %v", next.LatestResetBoundaryAt, latestOccurredAt)
	}
	if next.LatestLearningEventLedgerSeq != 77 {
		t.Fatalf("state latest ledger seq = %d, want 77", next.LatestLearningEventLedgerSeq)
	}
}

func TestResetUserUnitProgressExecuteTreatsClientEventIDAsUserScopedDuplicate(t *testing.T) {
	occurredAt := time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC)
	stateRepo := &fakeUserUnitStateRepository{
		state: &model.UserUnitState{
			UserID:             "11111111-1111-1111-1111-111111111111",
			CoarseUnitID:       102,
			IsTarget:           true,
			Status:             enum.StatusReviewing,
			ProgressPercent:    40,
			MasteryScore:       0.4,
			ProgressEventCount: 1,
			ScheduleEaseFactor: 2.5,
		},
	}
	eventRepo := &fakeUnitLearningEventRepository{
		eventBySource: &model.LearningEvent{
			EventID:       "55555555-5555-5555-5555-555555555555",
			UserID:        "11111111-1111-1111-1111-111111111111",
			CoarseUnitID:  101,
			EventType:     enum.EventResetUnlearned,
			ReducerEffect: enum.ReducerEffectResetUnlearned,
			SourceType:    "learning_unit_reset",
			SourceRefID:   "reset-1",
			Metadata:      []byte("{}"),
			OccurredAt:    occurredAt,
		},
	}
	txManager := &fakeTxManager{
		repositories: fakeTransactionalRepositories{
			userUnitStates: stateRepo,
			unitEvents:     eventRepo,
		},
	}
	usecase := service.NewResetUserUnitProgressUsecase(txManager)

	response, err := usecase.Execute(context.Background(), dto.ResetUserUnitProgressRequest{
		UserID:        "11111111-1111-1111-1111-111111111111",
		ClientEventID: "reset-1",
		CoarseUnitID:  102,
		SourceSurface: "word_detail",
		OccurredAt:    occurredAt,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !response.Accepted || response.Inserted || response.UnitLearningEventID != "55555555-5555-5555-5555-555555555555" {
		t.Fatalf("response = %+v, want duplicate existing event", response)
	}
	if len(eventRepo.appended) != 0 {
		t.Fatalf("appended events = %d, want 0", len(eventRepo.appended))
	}
	if len(stateRepo.batchUpserted) != 0 {
		t.Fatalf("batch upserted states = %d, want 0", len(stateRepo.batchUpserted))
	}
}

func TestResetUserUnitProgressExecuteTreatsAppendDuplicateAsIdempotent(t *testing.T) {
	occurredAt := time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC)
	stateRepo := &fakeUserUnitStateRepository{
		state: &model.UserUnitState{
			UserID:             "11111111-1111-1111-1111-111111111111",
			CoarseUnitID:       102,
			IsTarget:           true,
			Status:             enum.StatusReviewing,
			ProgressPercent:    40,
			MasteryScore:       0.4,
			ProgressEventCount: 1,
			ScheduleEaseFactor: 2.5,
		},
	}
	eventRepo := &fakeUnitLearningEventRepository{
		appendErr: applearningrepo.ErrDuplicateResetClientEvent,
		eventBySourceResults: []*model.LearningEvent{
			nil,
			{
				EventID:       "55555555-5555-5555-5555-555555555555",
				UserID:        "11111111-1111-1111-1111-111111111111",
				CoarseUnitID:  101,
				EventType:     enum.EventResetUnlearned,
				ReducerEffect: enum.ReducerEffectResetUnlearned,
				SourceType:    "learning_unit_reset",
				SourceRefID:   "reset-1",
				Metadata:      []byte("{}"),
				OccurredAt:    occurredAt,
			},
		},
	}
	txManager := &fakeTxManager{
		repositories: fakeTransactionalRepositories{
			userUnitStates: stateRepo,
			unitEvents:     eventRepo,
		},
	}
	usecase := service.NewResetUserUnitProgressUsecase(txManager)

	response, err := usecase.Execute(context.Background(), dto.ResetUserUnitProgressRequest{
		UserID:        "11111111-1111-1111-1111-111111111111",
		ClientEventID: "reset-1",
		CoarseUnitID:  102,
		SourceSurface: "word_detail",
		OccurredAt:    occurredAt,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !response.Accepted || response.Inserted || response.UnitLearningEventID != "55555555-5555-5555-5555-555555555555" {
		t.Fatalf("response = %+v, want duplicate existing event", response)
	}
	if len(eventRepo.appended) != 1 {
		t.Fatalf("appended attempts = %d, want 1", len(eventRepo.appended))
	}
	if len(stateRepo.batchUpserted) != 0 {
		t.Fatalf("batch upserted states = %d, want 0", len(stateRepo.batchUpserted))
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

func TestRecordLearningEventsExecuteSkipsEventsBeforeStateResetBoundary(t *testing.T) {
	boundary := time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC)
	beforeBoundary := boundary.Add(-time.Minute)
	stateRepo := &fakeUserUnitStateRepository{
		statesByUnit: map[int64]*model.UserUnitState{
			101: {
				UserID:                "11111111-1111-1111-1111-111111111111",
				CoarseUnitID:          101,
				IsTarget:              true,
				Status:                enum.StatusNew,
				ScheduleEaseFactor:    2.5,
				LatestResetBoundaryAt: &boundary,
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
	quality := int16(4)
	usecase := service.NewRecordLearningEventsUsecase(txManager)

	response, err := usecase.Execute(context.Background(), dto.RecordLearningEventsRequest{
		UserID: "11111111-1111-1111-1111-111111111111",
		Events: []dto.LearningEventInput{{
			CoarseUnitID:              101,
			EventType:                 enum.EventQuiz,
			ReducerEffect:             enum.ReducerEffectAffectsProgress,
			SourceType:                "quiz_event",
			SourceRefID:               "quiz-before-reset",
			ProgressQuality:           &quality,
			CountsTowardSuccessStreak: true,
			OccurredAt:                beforeBoundary,
		}},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if response.RecordedCount != 0 || response.SkippedBeforeResetCount != 1 {
		t.Fatalf("response = %+v, want skipped before reset", response)
	}
	if len(eventRepo.appended) != 0 {
		t.Fatalf("appended events = %d, want 0", len(eventRepo.appended))
	}
	if len(stateRepo.batchUpserted) != 0 {
		t.Fatalf("batch upserted states = %d, want 0", len(stateRepo.batchUpserted))
	}
}

func TestRecordLearningEventsExecuteIncrementsStartedUnitWhenProgressCrossesZero(t *testing.T) {
	q4 := int16(4)
	t1 := time.Date(2026, 4, 16, 10, 0, 0, 0, time.UTC)

	stateRepo := &fakeUserUnitStateRepository{
		statesByUnit: map[int64]*model.UserUnitState{
			202: {
				UserID:             "11111111-1111-1111-1111-111111111111",
				CoarseUnitID:       202,
				Status:             enum.StatusReviewing,
				ProgressPercent:    50,
				ScheduleEaseFactor: 2.5,
			},
		},
	}
	statsRecorder := &fakeActivityStatsRecorder{}
	txManager := &fakeTxManager{
		repositories: fakeTransactionalRepositories{
			userUnitStates: stateRepo,
			unitEvents:     &fakeUnitLearningEventRepository{},
			activityStats:  statsRecorder,
		},
	}
	usecase := service.NewRecordLearningEventsUsecase(txManager)

	_, err := usecase.Execute(context.Background(), dto.RecordLearningEventsRequest{
		UserID: "11111111-1111-1111-1111-111111111111",
		Events: []dto.LearningEventInput{
			{CoarseUnitID: 101, EventType: enum.EventQuiz, ReducerEffect: enum.ReducerEffectAffectsProgress, SourceType: "quiz_event", SourceRefID: "new-unit-1", ProgressQuality: &q4, OccurredAt: t1},
			{CoarseUnitID: 101, EventType: enum.EventQuiz, ReducerEffect: enum.ReducerEffectAffectsProgress, SourceType: "quiz_event", SourceRefID: "new-unit-2", ProgressQuality: &q4, OccurredAt: t1.Add(time.Second)},
			{CoarseUnitID: 202, EventType: enum.EventQuiz, ReducerEffect: enum.ReducerEffectAffectsProgress, SourceType: "quiz_event", SourceRefID: "existing-unit-1", ProgressQuality: &q4, OccurredAt: t1},
		},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if statsRecorder.startedUnitIncrements != 1 {
		t.Fatalf("started unit increments = %d, want 1", statsRecorder.startedUnitIncrements)
	}
}

func TestRecordLearningEventsExecuteValidatesConsumedWatchSessions(t *testing.T) {
	q4 := int16(4)
	t1 := time.Date(2026, 4, 16, 10, 0, 0, 0, time.UTC)
	usecase := service.NewRecordLearningEventsUsecase(&fakeTxManager{
		repositories: fakeTransactionalRepositories{
			unitEvents:     &fakeUnitLearningEventRepository{},
			userUnitStates: &fakeUserUnitStateRepository{},
		},
	})

	_, err := usecase.Execute(context.Background(), dto.RecordLearningEventsRequest{
		UserID: "11111111-1111-1111-1111-111111111111",
		Events: []dto.LearningEventInput{{
			CoarseUnitID:              101,
			EventType:                 enum.EventQuiz,
			ReducerEffect:             enum.ReducerEffectAffectsProgress,
			SourceType:                "quiz_event",
			SourceRefID:               "quiz-1",
			ProgressQuality:           &q4,
			CountsTowardSuccessStreak: true,
			ConsumedWatchSessionIDs:   []string{"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaa0001"},
			OccurredAt:                t1,
		}},
	})
	if err == nil {
		t.Fatalf("Execute() error = nil, want non-session3 consumed session validation error")
	}

	_, err = usecase.Execute(context.Background(), dto.RecordLearningEventsRequest{
		UserID: "11111111-1111-1111-1111-111111111111",
		Events: []dto.LearningEventInput{{
			CoarseUnitID:            101,
			EventType:               enum.EventExposure,
			ReducerEffect:           enum.ReducerEffectAffectsProgress,
			SourceType:              "exposure_session3_v1",
			SourceRefID:             "exposure_session3:too-short",
			ProgressQuality:         &q4,
			ConsumedWatchSessionIDs: []string{"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaa0001", "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaa0002"},
			OccurredAt:              t1,
		}},
	})
	if err == nil {
		t.Fatalf("Execute() error = nil, want session3 consumed session count validation error")
	}

	_, err = usecase.Execute(context.Background(), dto.RecordLearningEventsRequest{
		UserID: "11111111-1111-1111-1111-111111111111",
		Events: []dto.LearningEventInput{{
			CoarseUnitID:            101,
			EventType:               enum.EventQuiz,
			ReducerEffect:           enum.ReducerEffectAffectsProgress,
			SourceType:              "exposure_session3_v1",
			SourceRefID:             "exposure_session3:wrong-event-type",
			ProgressQuality:         &q4,
			ConsumedWatchSessionIDs: []string{"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaa0001", "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaa0002", "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaa0003"},
			OccurredAt:              t1,
		}},
	})
	if err == nil {
		t.Fatalf("Execute() error = nil, want session3 event_type validation error")
	}

	q3 := int16(3)
	_, err = usecase.Execute(context.Background(), dto.RecordLearningEventsRequest{
		UserID: "11111111-1111-1111-1111-111111111111",
		Events: []dto.LearningEventInput{{
			CoarseUnitID:            101,
			EventType:               enum.EventExposure,
			ReducerEffect:           enum.ReducerEffectAffectsProgress,
			SourceType:              "exposure_session3_v1",
			SourceRefID:             "exposure_session3:wrong-quality",
			ProgressQuality:         &q3,
			ConsumedWatchSessionIDs: []string{"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaa0001", "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaa0002", "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaa0003"},
			OccurredAt:              t1,
		}},
	})
	if err == nil {
		t.Fatalf("Execute() error = nil, want session3 quality validation error")
	}

	_, err = usecase.Execute(context.Background(), dto.RecordLearningEventsRequest{
		UserID: "11111111-1111-1111-1111-111111111111",
		Events: []dto.LearningEventInput{{
			CoarseUnitID:              101,
			EventType:                 enum.EventExposure,
			ReducerEffect:             enum.ReducerEffectAffectsProgress,
			SourceType:                "exposure_session3_v1",
			SourceRefID:               "exposure_session3:wrong-streak",
			ProgressQuality:           &q4,
			CountsTowardSuccessStreak: true,
			ConsumedWatchSessionIDs:   []string{"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaa0001", "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaa0002", "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaa0003"},
			OccurredAt:                t1,
		}},
	})
	if err == nil {
		t.Fatalf("Execute() error = nil, want session3 streak validation error")
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
	if !state.IsTarget {
		t.Fatalf("is_target = false, want true")
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
	if state.LastProgressAt == nil || !state.LastProgressAt.Equal(eventTime) {
		t.Fatalf("last_progress_at = %v, want %v", state.LastProgressAt, eventTime)
	}
}

func TestReplayUserStatesExecuteRebuildsProjectionWatermarks(t *testing.T) {
	t1 := time.Date(2026, 5, 21, 10, 0, 0, 0, time.UTC)
	resetOccurredAt := t1.Add(-time.Hour)
	resetBoundaryAt := t1.Add(time.Hour)
	quality := int16(4)
	stateRepo := &fakeUserUnitStateRepository{
		listStates: []model.UserUnitState{{
			UserID:             "11111111-1111-1111-1111-111111111111",
			CoarseUnitID:       101,
			IsTarget:           true,
			TargetSource:       "unit_collection",
			TargetSourceRefID:  "collection-1",
			TargetPriority:     0.9,
			Status:             enum.StatusReviewing,
			ScheduleEaseFactor: 2.5,
			CreatedAt:          t1.Add(-24 * time.Hour),
		}},
	}
	eventRepo := &fakeUnitLearningEventRepository{
		listByUserOrdered: []model.LearningEvent{
			{
				EventID:         "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
				LedgerSeq:       10,
				UserID:          "11111111-1111-1111-1111-111111111111",
				CoarseUnitID:    101,
				EventType:       enum.EventQuiz,
				ReducerEffect:   enum.ReducerEffectAffectsProgress,
				SourceType:      "quiz_event",
				SourceRefID:     "quiz-1",
				ProgressQuality: &quality,
				OccurredAt:      t1,
				Metadata:        []byte("{}"),
			},
			{
				EventID:         "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
				LedgerSeq:       11,
				UserID:          "11111111-1111-1111-1111-111111111111",
				CoarseUnitID:    101,
				EventType:       enum.EventResetUnlearned,
				ReducerEffect:   enum.ReducerEffectResetUnlearned,
				SourceType:      "learning_unit_reset",
				SourceRefID:     "reset-1",
				OccurredAt:      resetOccurredAt,
				ResetBoundaryAt: &resetBoundaryAt,
				Metadata:        []byte("{}"),
			},
		},
	}
	txManager := &fakeTxManager{
		repositories: fakeTransactionalRepositories{
			userUnitStates: stateRepo,
			unitEvents:     eventRepo,
		},
	}
	usecase := service.NewReplayUserStatesUsecase(txManager)

	if _, err := usecase.Execute(context.Background(), dto.ReplayUserStatesRequest{
		UserID: "11111111-1111-1111-1111-111111111111",
	}); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if len(stateRepo.batchUpserted) != 1 {
		t.Fatalf("upserted states = %d, want 1", len(stateRepo.batchUpserted))
	}
	state := stateRepo.batchUpserted[0]
	if state.LatestLearningEventOccurredAt == nil || !state.LatestLearningEventOccurredAt.Equal(t1) {
		t.Fatalf("latest occurred = %v, want %v", state.LatestLearningEventOccurredAt, t1)
	}
	if state.LatestResetBoundaryAt == nil || !state.LatestResetBoundaryAt.Equal(resetBoundaryAt) {
		t.Fatalf("latest reset boundary = %v, want %v", state.LatestResetBoundaryAt, resetBoundaryAt)
	}
	if state.LatestLearningEventLedgerSeq != 11 {
		t.Fatalf("latest ledger seq = %d, want 11", state.LatestLearningEventLedgerSeq)
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
	if rebuiltByUnit[102].Status != enum.StatusNew {
		t.Fatalf("unit 102 status = %q, want %q", rebuiltByUnit[102].Status, enum.StatusNew)
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
	activityStats  userrepo.ActivityStatsRecorder
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

func (f fakeTransactionalRepositories) ActivityStats() userrepo.ActivityStatsRecorder {
	return f.activityStats
}

type fakeActivityStatsRecorder struct {
	startedUnitIncrements int
}

func (f *fakeActivityStatsRecorder) AddWatchDuration(_ context.Context, _ string, _ time.Time, _ int64) error {
	return nil
}

func (f *fakeActivityStatsRecorder) IncrementQuizAttempt(_ context.Context, _ string, _ time.Time) error {
	return nil
}

func (f *fakeActivityStatsRecorder) IncrementStartedUnit(_ context.Context, _ string) error {
	f.startedUnitIncrements++
	return nil
}

func (f *fakeActivityStatsRecorder) IncrementLearningInteraction(_ context.Context, _ string, _ time.Time) error {
	return nil
}

type fakeTargetStateCommandRepository struct {
	targets         []model.TargetUnitSpec
	inactiveUnitID  int64
	activatedUserID string
	activatedSlug   string
	activation      model.ActivatedUnitCollectionTarget
}

func (f *fakeTargetStateCommandRepository) EnsureTargetUnits(_ context.Context, _ string, targets []model.TargetUnitSpec) error {
	f.targets = targets
	return nil
}

func (f *fakeTargetStateCommandRepository) ActivateUnitCollectionTarget(_ context.Context, userID string, collectionSlug string) (model.ActivatedUnitCollectionTarget, error) {
	f.activatedUserID = userID
	f.activatedSlug = collectionSlug
	return f.activation, nil
}

func (f *fakeTargetStateCommandRepository) SetTargetInactive(_ context.Context, _ string, coarseUnitID int64) error {
	f.inactiveUnitID = coarseUnitID
	return nil
}

type fakeActiveUnitCollectionReader struct {
	userID             string
	active             *model.ActiveUnitCollection
	activeTargetUserID string
	activeTargets      model.ActiveLearningTargetCoarseUnitIDs
	err                error
}

func (f *fakeActiveUnitCollectionReader) GetActiveUnitCollection(_ context.Context, userID string) (*model.ActiveUnitCollection, error) {
	f.userID = userID
	return f.active, f.err
}

func (f *fakeActiveUnitCollectionReader) GetActiveLearningTargetCoarseUnitIDs(_ context.Context, userID string) (model.ActiveLearningTargetCoarseUnitIDs, error) {
	f.activeTargetUserID = userID
	return f.activeTargets, f.err
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

func (f *fakeUserUnitStateRepository) GetByUserAndUnit(_ context.Context, _ string, coarseUnitID int64) (*model.UserUnitState, error) {
	if f.statesByUnit != nil {
		return f.statesByUnit[coarseUnitID], nil
	}
	if f.state != nil && f.state.CoarseUnitID == coarseUnitID {
		return f.state, nil
	}
	return nil, nil
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
	appended             []model.LearningEvent
	appendResult         applearningrepo.AppendLearningEventsResult
	appendErr            error
	eventBySource        *model.LearningEvent
	eventBySourceResults []*model.LearningEvent
	eventBySourceCalls   int
	listByUserOrdered    []model.LearningEvent
}

func (f *fakeUnitLearningEventRepository) Append(_ context.Context, events []model.LearningEvent) (applearningrepo.AppendLearningEventsResult, error) {
	f.appended = append(f.appended, events...)
	if f.appendErr != nil {
		return applearningrepo.AppendLearningEventsResult{}, f.appendErr
	}
	if f.appendResult.InsertedEvents != nil || f.appendResult.DuplicateCount != 0 {
		return f.appendResult, nil
	}
	return applearningrepo.AppendLearningEventsResult{InsertedEvents: append([]model.LearningEvent(nil), events...)}, nil
}

func (f *fakeUnitLearningEventRepository) GetByUserSourceRef(_ context.Context, _ string, _ string, _ string) (*model.LearningEvent, error) {
	if len(f.eventBySourceResults) > 0 {
		index := f.eventBySourceCalls
		f.eventBySourceCalls++
		if index >= len(f.eventBySourceResults) {
			index = len(f.eventBySourceResults) - 1
		}
		return f.eventBySourceResults[index], nil
	}
	return f.eventBySource, nil
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
