package integration

import (
	"context"
	"math"
	"testing"
	"time"

	"learning-video-recommendation-system/internal/recommendation/scheduler/application/command"
	"learning-video-recommendation-system/internal/recommendation/scheduler/application/usecase"
	"learning-video-recommendation-system/internal/recommendation/scheduler/domain/enum"
	domainservice "learning-video-recommendation-system/internal/recommendation/scheduler/domain/service"
	infra "learning-video-recommendation-system/internal/recommendation/scheduler/infrastructure"
	repopkg "learning-video-recommendation-system/internal/recommendation/scheduler/infrastructure/persistence/repository"
	"learning-video-recommendation-system/internal/recommendation/scheduler/infrastructure/persistence/sqlcgen"
	txtx "learning-video-recommendation-system/internal/recommendation/scheduler/infrastructure/persistence/tx"
)

func TestReplayUserUnitStatesUseCaseRebuildsOnlineState(t *testing.T) {
	cfg := infra.LoadConfig()
	if cfg.DatabaseURL == "" {
		t.Skip("DATABASE_URL is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	pool, err := infra.NewDBPool(ctx, cfg)
	if err != nil {
		t.Fatalf("NewDBPool() error = %v", err)
	}
	defer pool.Close()

	userID, err := createTestUserIDFromPool(ctx, pool)
	if err != nil {
		t.Fatalf("createTestUserIDFromPool() error = %v", err)
	}
	defer cleanupTestUser(ctx, t, pool, userID)
	unitIDs, err := loadAvailableCoarseUnitIDsFromPool(ctx, pool, userID, 1)
	if err != nil {
		t.Fatalf("loadAvailableCoarseUnitIDsFromPool() error = %v", err)
	}
	unitID := unitIDs[0]

	txManager := txtx.NewPGXTxManager(pool)
	baseQuerier := sqlcgen.New(pool)
	stateRepo := repopkg.NewUserUnitStateRepository(baseQuerier)
	eventRepo := repopkg.NewUnitLearningEventRepository(baseQuerier)
	stateUpdater := domainservice.NewStateUpdater()

	recordUC := usecase.NewRecordLearningEventsAndUpdateStateUseCase(txManager, stateRepo, eventRepo, stateUpdater)
	replayUC := usecase.NewReplayUserUnitStatesUseCase(txManager, stateRepo, eventRepo, stateUpdater)

	correct := true
	q1 := 4
	q2 := 5
	occurredAt := time.Date(2026, 4, 9, 10, 0, 0, 0, time.UTC)

	_, err = recordUC.Execute(ctx, command.RecordLearningEventsCommand{
		UserID: userID,
		Events: []command.LearningEventInput{
			{
				CoarseUnitID: unitID,
				EventType:    enum.EventTypeNewLearn,
				SourceType:   "integration_test",
				SourceRefID:  "replay-1",
				IsCorrect:    &correct,
				Quality:      &q1,
				OccurredAt:   occurredAt,
			},
			{
				CoarseUnitID: unitID,
				EventType:    enum.EventTypeReview,
				SourceType:   "integration_test",
				SourceRefID:  "replay-2",
				IsCorrect:    &correct,
				Quality:      &q2,
				OccurredAt:   occurredAt.Add(24 * time.Hour),
			},
		},
		IdempotencyKey: "replay-seed",
	})
	if err != nil {
		t.Fatalf("record Execute() error = %v", err)
	}

	onlineState, err := stateRepo.GetByUserAndUnit(ctx, userID, unitID)
	if err != nil {
		t.Fatalf("GetByUserAndUnit() error = %v", err)
	}
	if onlineState == nil {
		t.Fatal("onlineState = nil, want value")
	}

	corrupted := *onlineState
	corrupted.Status = enum.UnitStatusSuspended
	corrupted.Repetition = 0
	corrupted.IntervalDays = 0
	corrupted.ProgressPercent = 0
	corrupted.MasteryScore = 0
	corrupted.NextReviewAt = nil
	corrupted.UpdatedAt = time.Now()
	if err := stateRepo.Upsert(ctx, &corrupted); err != nil {
		t.Fatalf("Upsert(corrupted) error = %v", err)
	}

	result, err := replayUC.Execute(ctx, command.ReplayStateCommand{
		UserID:       userID,
		CoarseUnitID: &unitID,
	})
	if err != nil {
		t.Fatalf("replay Execute() error = %v", err)
	}
	if result.RebuiltCount != 1 {
		t.Fatalf("RebuiltCount = %d, want 1", result.RebuiltCount)
	}
	if result.ErrorCount != 0 {
		t.Fatalf("ErrorCount = %d, want 0", result.ErrorCount)
	}

	rebuiltState, err := stateRepo.GetByUserAndUnit(ctx, userID, unitID)
	if err != nil {
		t.Fatalf("GetByUserAndUnit(rebuilt) error = %v", err)
	}
	if rebuiltState == nil {
		t.Fatal("rebuiltState = nil, want value")
	}

	if rebuiltState.Status != onlineState.Status {
		t.Fatalf("rebuilt Status = %q, want %q", rebuiltState.Status, onlineState.Status)
	}
	if rebuiltState.Repetition != onlineState.Repetition {
		t.Fatalf("rebuilt Repetition = %d, want %d", rebuiltState.Repetition, onlineState.Repetition)
	}
	if math.Abs(rebuiltState.IntervalDays-onlineState.IntervalDays) > 1e-9 {
		t.Fatalf("rebuilt IntervalDays = %v, want %v", rebuiltState.IntervalDays, onlineState.IntervalDays)
	}
	if math.Abs(rebuiltState.EaseFactor-onlineState.EaseFactor) > 1e-9 {
		t.Fatalf("rebuilt EaseFactor = %v, want %v", rebuiltState.EaseFactor, onlineState.EaseFactor)
	}
	if math.Abs(rebuiltState.ProgressPercent-onlineState.ProgressPercent) > 1e-9 {
		t.Fatalf("rebuilt ProgressPercent = %v, want %v", rebuiltState.ProgressPercent, onlineState.ProgressPercent)
	}
	if math.Abs(rebuiltState.MasteryScore-onlineState.MasteryScore) > 1e-9 {
		t.Fatalf("rebuilt MasteryScore = %v, want %v", rebuiltState.MasteryScore, onlineState.MasteryScore)
	}
	if !sameOptionalTime(rebuiltState.NextReviewAt, onlineState.NextReviewAt) {
		t.Fatalf("rebuilt NextReviewAt = %v, want %v", rebuiltState.NextReviewAt, onlineState.NextReviewAt)
	}
}

func sameOptionalTime(left, right *time.Time) bool {
	switch {
	case left == nil && right == nil:
		return true
	case left == nil || right == nil:
		return false
	default:
		return left.Equal(*right)
	}
}
