// 作用：验证 ReplayUserStatesUseCase 能从事件真相层重建状态，并把被污染的状态恢复为在线结果。
// 输入/输出：输入是测试用户、测试事件序列和 replay 命令；输出是测试断言结果。
// 谁调用它：go test、make check。
// 它调用谁/传给谁：调用 fixture/helpers.go、真实 record/replay use case、真实 repository；断言只返回给测试框架。
package usecase_test

import (
	"math"
	"testing"
	"time"

	"learning-video-recommendation-system/internal/learningengine/application/command"
	"learning-video-recommendation-system/internal/learningengine/domain/enum"
	"learning-video-recommendation-system/internal/learningengine/domain/model"
	repopkg "learning-video-recommendation-system/internal/learningengine/infrastructure/persistence/repository"
	"learning-video-recommendation-system/internal/learningengine/infrastructure/persistence/sqlcgen"
	"learning-video-recommendation-system/internal/learningengine/test/integration/fixture"
)

func TestReplayUserStatesUseCaseRebuildsOnlineState(t *testing.T) {
	ctx, pool := fixture.NewTestPool(t)

	userID, err := fixture.CreateTestUser(ctx, pool)
	if err != nil {
		t.Fatalf("CreateTestUser() error = %v", err)
	}
	unitIDs, err := fixture.CreateTestCoarseUnits(ctx, pool, 1)
	if err != nil {
		t.Fatalf("CreateTestCoarseUnits() error = %v", err)
	}
	t.Cleanup(func() {
		fixture.CleanupTestData(ctx, t, pool, userID, unitIDs)
	})
	unitID := unitIDs[0]

	baseQuerier := sqlcgen.New(pool)
	stateRepo := repopkg.NewUserUnitStateRepository(baseQuerier)
	recordUC := fixture.NewRecordEventsUseCase(pool, baseQuerier)
	replayUC := fixture.NewReplayUseCase(pool, baseQuerier)

	correct := true
	q1 := 4
	q2 := 5
	occurredAt := time.Date(2026, 4, 9, 10, 0, 0, 0, time.UTC)

	_, err = recordUC.Execute(ctx, command.RecordLearningEventsCommand{
		UserID: userID,
		Events: []command.LearningEventInput{
			fixture.NewLearnInput(unitID, &correct, &q1, occurredAt, "replay-1"),
			fixture.ReviewInput(unitID, &correct, &q2, occurredAt.Add(24*time.Hour), "replay-2"),
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

	result, err := replayUC.Execute(ctx, command.ReplayUserStatesCommand{UserID: userID})
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

func assertReplayedStateMatches(t *testing.T, got, want *model.UserUnitState) {
	t.Helper()

	if got.Status != want.Status {
		t.Fatalf("Status = %q, want %q", got.Status, want.Status)
	}
	if got.Repetition != want.Repetition {
		t.Fatalf("Repetition = %d, want %d", got.Repetition, want.Repetition)
	}
	if math.Abs(got.IntervalDays-want.IntervalDays) > 1e-9 {
		t.Fatalf("IntervalDays = %v, want %v", got.IntervalDays, want.IntervalDays)
	}
	if math.Abs(got.EaseFactor-want.EaseFactor) > 1e-9 {
		t.Fatalf("EaseFactor = %v, want %v", got.EaseFactor, want.EaseFactor)
	}
	if math.Abs(got.ProgressPercent-want.ProgressPercent) > 1e-9 {
		t.Fatalf("ProgressPercent = %v, want %v", got.ProgressPercent, want.ProgressPercent)
	}
	if math.Abs(got.MasteryScore-want.MasteryScore) > 1e-9 {
		t.Fatalf("MasteryScore = %v, want %v", got.MasteryScore, want.MasteryScore)
	}
	if len(got.RecentQualityWindow) != len(want.RecentQualityWindow) {
		t.Fatalf("len(RecentQualityWindow) = %d, want %d", len(got.RecentQualityWindow), len(want.RecentQualityWindow))
	}
	for i, value := range want.RecentQualityWindow {
		if got.RecentQualityWindow[i] != value {
			t.Fatalf("RecentQualityWindow[%d] = %d, want %d", i, got.RecentQualityWindow[i], value)
		}
	}
	if len(got.RecentCorrectnessWindow) != len(want.RecentCorrectnessWindow) {
		t.Fatalf("len(RecentCorrectnessWindow) = %d, want %d", len(got.RecentCorrectnessWindow), len(want.RecentCorrectnessWindow))
	}
	for i, value := range want.RecentCorrectnessWindow {
		if got.RecentCorrectnessWindow[i] != value {
			t.Fatalf("RecentCorrectnessWindow[%d] = %v, want %v", i, got.RecentCorrectnessWindow[i], value)
		}
	}
}
