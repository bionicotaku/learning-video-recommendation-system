package integration

import (
	"context"
	"math"
	"testing"
	"time"

	"learning-video-recommendation-system/internal/recommendation/scheduler/application/command"
	"learning-video-recommendation-system/internal/recommendation/scheduler/domain/enum"
	"learning-video-recommendation-system/internal/recommendation/scheduler/domain/model"
	infra "learning-video-recommendation-system/internal/recommendation/scheduler/infrastructure"
	repopkg "learning-video-recommendation-system/internal/recommendation/scheduler/infrastructure/persistence/repository"
	"learning-video-recommendation-system/internal/recommendation/scheduler/infrastructure/persistence/sqlcgen"
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

	baseQuerier := sqlcgen.New(pool)
	stateRepo := repopkg.NewUserUnitStateRepository(baseQuerier)
	recordUC := newRecordEventsUseCase(pool, baseQuerier)
	replayUC := newReplayUseCase(pool, baseQuerier)

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
	corrupted.RecentQualityWindow = []int{}
	corrupted.RecentCorrectnessWindow = []bool{}
	corrupted.UpdatedAt = time.Now()
	if err := stateRepo.Upsert(ctx, &corrupted); err != nil {
		t.Fatalf("Upsert(corrupted) error = %v", err)
	}

	result, err := replayUC.Execute(ctx, command.ReplayStateCommand{UserID: userID})
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

	assertReplayedStateMatches(t, rebuiltState, onlineState)
}

func TestReplayUserUnitStatesUseCaseRebuildsAllUserUnits(t *testing.T) {
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
	unitIDs, err := loadAvailableCoarseUnitIDsFromPool(ctx, pool, userID, 2)
	if err != nil {
		t.Fatalf("loadAvailableCoarseUnitIDsFromPool() error = %v", err)
	}

	baseQuerier := sqlcgen.New(pool)
	stateRepo := repopkg.NewUserUnitStateRepository(baseQuerier)
	recordUC := newRecordEventsUseCase(pool, baseQuerier)
	replayUC := newReplayUseCase(pool, baseQuerier)

	correct := true
	quality := 4
	start := time.Date(2026, 4, 10, 10, 0, 0, 0, time.UTC)

	_, err = recordUC.Execute(ctx, command.RecordLearningEventsCommand{
		UserID: userID,
		Events: []command.LearningEventInput{
			{
				CoarseUnitID: unitIDs[0],
				EventType:    enum.EventTypeNewLearn,
				SourceType:   "integration_test",
				SourceRefID:  "full-replay-a",
				IsCorrect:    &correct,
				Quality:      &quality,
				OccurredAt:   start,
			},
			{
				CoarseUnitID: unitIDs[1],
				EventType:    enum.EventTypeNewLearn,
				SourceType:   "integration_test",
				SourceRefID:  "full-replay-b",
				IsCorrect:    &correct,
				Quality:      &quality,
				OccurredAt:   start.Add(time.Hour),
			},
		},
		IdempotencyKey: "full-replay-seed",
	})
	if err != nil {
		t.Fatalf("record Execute() error = %v", err)
	}

	beforeA, err := stateRepo.GetByUserAndUnit(ctx, userID, unitIDs[0])
	if err != nil {
		t.Fatalf("GetByUserAndUnit(unitA) error = %v", err)
	}
	beforeB, err := stateRepo.GetByUserAndUnit(ctx, userID, unitIDs[1])
	if err != nil {
		t.Fatalf("GetByUserAndUnit(unitB) error = %v", err)
	}
	if beforeA == nil || beforeB == nil {
		t.Fatal("seed states missing")
	}

	corruptedA := *beforeA
	corruptedA.Status = enum.UnitStatusSuspended
	corruptedA.RecentQualityWindow = []int{}
	corruptedA.RecentCorrectnessWindow = []bool{}
	if err := stateRepo.Upsert(ctx, &corruptedA); err != nil {
		t.Fatalf("Upsert(corruptedA) error = %v", err)
	}

	corruptedB := *beforeB
	corruptedB.Status = enum.UnitStatusSuspended
	corruptedB.ProgressPercent = 0
	corruptedB.MasteryScore = 0
	corruptedB.NextReviewAt = nil
	corruptedB.RecentQualityWindow = []int{}
	corruptedB.RecentCorrectnessWindow = []bool{}
	if err := stateRepo.Upsert(ctx, &corruptedB); err != nil {
		t.Fatalf("Upsert(corruptedB) error = %v", err)
	}

	result, err := replayUC.Execute(ctx, command.ReplayStateCommand{UserID: userID})
	if err != nil {
		t.Fatalf("replay Execute() error = %v", err)
	}
	if result.RebuiltCount != 2 {
		t.Fatalf("RebuiltCount = %d, want 2", result.RebuiltCount)
	}

	afterA, err := stateRepo.GetByUserAndUnit(ctx, userID, unitIDs[0])
	if err != nil {
		t.Fatalf("GetByUserAndUnit(afterA) error = %v", err)
	}
	afterB, err := stateRepo.GetByUserAndUnit(ctx, userID, unitIDs[1])
	if err != nil {
		t.Fatalf("GetByUserAndUnit(afterB) error = %v", err)
	}
	if afterA == nil || afterB == nil {
		t.Fatal("replayed states missing")
	}

	assertReplayedStateMatches(t, afterA, beforeA)
	assertReplayedStateMatches(t, afterB, beforeB)
}

func assertReplayedStateMatches(t *testing.T, got, want *model.UserUnitState) {
	t.Helper()

	if got.Status != want.Status {
		t.Fatalf("rebuilt Status = %q, want %q", got.Status, want.Status)
	}
	if got.Repetition != want.Repetition {
		t.Fatalf("rebuilt Repetition = %d, want %d", got.Repetition, want.Repetition)
	}
	if math.Abs(got.IntervalDays-want.IntervalDays) > 1e-9 {
		t.Fatalf("rebuilt IntervalDays = %v, want %v", got.IntervalDays, want.IntervalDays)
	}
	if math.Abs(got.EaseFactor-want.EaseFactor) > 1e-9 {
		t.Fatalf("rebuilt EaseFactor = %v, want %v", got.EaseFactor, want.EaseFactor)
	}
	if math.Abs(got.ProgressPercent-want.ProgressPercent) > 1e-9 {
		t.Fatalf("rebuilt ProgressPercent = %v, want %v", got.ProgressPercent, want.ProgressPercent)
	}
	if math.Abs(got.MasteryScore-want.MasteryScore) > 1e-9 {
		t.Fatalf("rebuilt MasteryScore = %v, want %v", got.MasteryScore, want.MasteryScore)
	}
	if !sameOptionalTime(got.NextReviewAt, want.NextReviewAt) {
		t.Fatalf("rebuilt NextReviewAt = %v, want %v", got.NextReviewAt, want.NextReviewAt)
	}
	if len(got.RecentQualityWindow) != len(want.RecentQualityWindow) {
		t.Fatalf("rebuilt RecentQualityWindow = %v, want %v", got.RecentQualityWindow, want.RecentQualityWindow)
	}
	for i := range got.RecentQualityWindow {
		if got.RecentQualityWindow[i] != want.RecentQualityWindow[i] {
			t.Fatalf("rebuilt RecentQualityWindow = %v, want %v", got.RecentQualityWindow, want.RecentQualityWindow)
		}
	}
	if len(got.RecentCorrectnessWindow) != len(want.RecentCorrectnessWindow) {
		t.Fatalf("rebuilt RecentCorrectnessWindow = %v, want %v", got.RecentCorrectnessWindow, want.RecentCorrectnessWindow)
	}
	for i := range got.RecentCorrectnessWindow {
		if got.RecentCorrectnessWindow[i] != want.RecentCorrectnessWindow[i] {
			t.Fatalf("rebuilt RecentCorrectnessWindow = %v, want %v", got.RecentCorrectnessWindow, want.RecentCorrectnessWindow)
		}
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
