package e2e_test

import (
	"reflect"
	"slices"
	"testing"
	"time"

	lecommand "learning-video-recommendation-system/internal/learningengine/application/command"
	lemodel "learning-video-recommendation-system/internal/learningengine/domain/model"
	recdomainenum "learning-video-recommendation-system/internal/recommendation/scheduler/domain/enum"
	"learning-video-recommendation-system/internal/test/e2e/fixture"
)

type stateSnapshot struct {
	IsTarget                bool
	TargetSource            string
	TargetSourceRefID       string
	TargetPriority          float64
	Status                  string
	ProgressPercent         float64
	MasteryScore            float64
	SeenCount               int
	StrongEventCount        int
	ReviewCount             int
	CorrectCount            int
	WrongCount              int
	ConsecutiveCorrect      int
	ConsecutiveWrong        int
	LastQuality             *int
	RecentQualityWindow     []int
	RecentCorrectnessWindow []bool
	Repetition              int
	IntervalDays            float64
	EaseFactor              float64
	NextReviewAt            *time.Time
}

func TestLearningEngineToRecommendationEndToEnd_GeneratesMixedBatch(t *testing.T) {
	ctx, pool := fixture.NewTestPool(t)

	userID, err := fixture.CreateTestUser(ctx, pool)
	if err != nil {
		t.Fatalf("CreateTestUser() error = %v", err)
	}
	unitIDs, err := fixture.CreateTestCoarseUnits(ctx, pool, 6)
	if err != nil {
		t.Fatalf("CreateTestCoarseUnits() error = %v", err)
	}
	t.Cleanup(func() {
		fixture.CleanupTestData(ctx, t, pool, userID, unitIDs)
	})

	recordUC := fixture.NewRecordEventsUseCase(pool)
	stateRepo := fixture.NewStateRepository(pool)
	generateUC := fixture.NewGenerateUseCase(pool)

	baseTime := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
	eventCmd := lecommand.RecordLearningEventsCommand{
		UserID: userID,
		Events: []lecommand.LearningEventInput{
			fixture.NewLearnInput(unitIDs[0], true, 4, baseTime, "u0-new"),
			fixture.NewLearnInput(unitIDs[1], true, 5, baseTime.Add(1*time.Hour), "u1-new"),
			fixture.ReviewInput(unitIDs[1], true, 5, baseTime.Add(72*time.Hour), "u1-review"),
			fixture.NewLearnInput(unitIDs[2], true, 4, baseTime.Add(2*time.Hour), "u2-new"),
			fixture.ReviewInput(unitIDs[2], false, 2, baseTime.Add(48*time.Hour), "u2-review"),
		},
		IdempotencyKey: "e2e-mixed-batch",
	}
	if _, err := recordUC.Execute(ctx, eventCmd); err != nil {
		t.Fatalf("RecordLearningEvents.Execute() error = %v", err)
	}

	if err := fixture.SeedNewTargetState(ctx, stateRepo, userID, unitIDs[3], 0.95, "new-high", baseTime); err != nil {
		t.Fatalf("SeedNewTargetState(unit3) error = %v", err)
	}
	if err := fixture.SeedNewTargetState(ctx, stateRepo, userID, unitIDs[4], 0.85, "new-mid", baseTime); err != nil {
		t.Fatalf("SeedNewTargetState(unit4) error = %v", err)
	}
	if err := fixture.SeedNewTargetState(ctx, stateRepo, userID, unitIDs[5], 0.75, "new-recent", baseTime); err != nil {
		t.Fatalf("SeedNewTargetState(unit5) error = %v", err)
	}
	if err := fixture.InsertServingState(ctx, pool, userID, unitIDs[5], baseTime.Add(7*24*time.Hour-1*time.Hour)); err != nil {
		t.Fatalf("InsertServingState(unit5) error = %v", err)
	}

	var beforeStateCount int
	if err := pool.QueryRow(ctx, `select count(*) from learning.user_unit_states where user_id = $1`, userID).Scan(&beforeStateCount); err != nil {
		t.Fatalf("count learning.user_unit_states before error = %v", err)
	}
	var beforeEventCount int
	if err := pool.QueryRow(ctx, `select count(*) from learning.unit_learning_events where user_id = $1`, userID).Scan(&beforeEventCount); err != nil {
		t.Fatalf("count learning.unit_learning_events before error = %v", err)
	}

	now := baseTime.Add(10 * 24 * time.Hour)
	result, err := generateUC.Execute(ctx, fixture.GenerateCommand(userID, 5, now))
	if err != nil {
		t.Fatalf("GenerateRecommendations.Execute() error = %v", err)
	}

	batch := result.Batch
	if len(batch.Items) != 5 {
		t.Fatalf("len(Batch.Items) = %d, want 5", len(batch.Items))
	}
	if batch.DueReviewCount != 3 {
		t.Fatalf("Batch.DueReviewCount = %d, want 3", batch.DueReviewCount)
	}
	if batch.ReviewQuota != 3 {
		t.Fatalf("Batch.ReviewQuota = %d, want 3", batch.ReviewQuota)
	}
	if batch.NewQuota != 2 {
		t.Fatalf("Batch.NewQuota = %d, want 2", batch.NewQuota)
	}

	if batch.Items[0].CoarseUnitID != unitIDs[0] {
		t.Fatalf("Batch.Items[0].CoarseUnitID = %d, want %d", batch.Items[0].CoarseUnitID, unitIDs[0])
	}
	if batch.Items[0].RecommendType != recdomainenum.RecommendTypeReview {
		t.Fatalf("Batch.Items[0].RecommendType = %q, want %q", batch.Items[0].RecommendType, recdomainenum.RecommendTypeReview)
	}
	if batch.Items[1].CoarseUnitID != unitIDs[2] {
		t.Fatalf("Batch.Items[1].CoarseUnitID = %d, want %d", batch.Items[1].CoarseUnitID, unitIDs[2])
	}
	if batch.Items[1].RecommendType != recdomainenum.RecommendTypeReview {
		t.Fatalf("Batch.Items[1].RecommendType = %q, want %q", batch.Items[1].RecommendType, recdomainenum.RecommendTypeReview)
	}

	reviewCount := 0
	newCount := 0
	itemUnitIDs := make([]int64, 0, len(batch.Items))
	for _, item := range batch.Items {
		itemUnitIDs = append(itemUnitIDs, item.CoarseUnitID)
		switch item.RecommendType {
		case recdomainenum.RecommendTypeReview:
			reviewCount++
		case recdomainenum.RecommendTypeNew:
			newCount++
		}
	}
	if reviewCount != 3 {
		t.Fatalf("review item count = %d, want 3", reviewCount)
	}
	if newCount != 2 {
		t.Fatalf("new item count = %d, want 2", newCount)
	}
	if !slices.Contains(itemUnitIDs, unitIDs[3]) || !slices.Contains(itemUnitIDs, unitIDs[4]) {
		t.Fatalf("new candidates missing expected units: items=%v want include %d and %d", itemUnitIDs, unitIDs[3], unitIDs[4])
	}
	if slices.Contains(itemUnitIDs, unitIDs[5]) {
		t.Fatalf("items unexpectedly include recently recommended unit %d: %v", unitIDs[5], itemUnitIDs)
	}

	var afterStateCount int
	if err := pool.QueryRow(ctx, `select count(*) from learning.user_unit_states where user_id = $1`, userID).Scan(&afterStateCount); err != nil {
		t.Fatalf("count learning.user_unit_states after error = %v", err)
	}
	var afterEventCount int
	if err := pool.QueryRow(ctx, `select count(*) from learning.unit_learning_events where user_id = $1`, userID).Scan(&afterEventCount); err != nil {
		t.Fatalf("count learning.unit_learning_events after error = %v", err)
	}
	if beforeStateCount != afterStateCount {
		t.Fatalf("learning.user_unit_states count changed: before=%d after=%d", beforeStateCount, afterStateCount)
	}
	if beforeEventCount != afterEventCount {
		t.Fatalf("learning.unit_learning_events count changed: before=%d after=%d", beforeEventCount, afterEventCount)
	}

	var runCount int
	if err := pool.QueryRow(ctx, `select count(*) from recommendation.scheduler_runs where user_id = $1`, userID).Scan(&runCount); err != nil {
		t.Fatalf("count recommendation.scheduler_runs error = %v", err)
	}
	if runCount != 1 {
		t.Fatalf("recommendation.scheduler_runs count = %d, want 1", runCount)
	}

	var runItemCount int
	if err := pool.QueryRow(ctx, `
		select count(*) from recommendation.scheduler_run_items where run_id = $1
	`, batch.RunID).Scan(&runItemCount); err != nil {
		t.Fatalf("count recommendation.scheduler_run_items error = %v", err)
	}
	if runItemCount != 5 {
		t.Fatalf("recommendation.scheduler_run_items count = %d, want 5", runItemCount)
	}

	var servingCount int
	if err := pool.QueryRow(ctx, `select count(*) from recommendation.user_unit_serving_states where user_id = $1`, userID).Scan(&servingCount); err != nil {
		t.Fatalf("count recommendation.user_unit_serving_states error = %v", err)
	}
	if servingCount != 6 {
		t.Fatalf("recommendation.user_unit_serving_states count = %d, want 6", servingCount)
	}

	var touchedCount int
	if err := pool.QueryRow(ctx, `
		select count(*)
		from recommendation.user_unit_serving_states
		where user_id = $1
		  and last_recommended_at = $2
	`, userID, batch.GeneratedAt).Scan(&touchedCount); err != nil {
		t.Fatalf("count touched serving states error = %v", err)
	}
	if touchedCount != 5 {
		t.Fatalf("touched serving state count = %d, want 5", touchedCount)
	}
}

func TestLearningEngineToRecommendationEndToEnd_ReplayRestoresStateBeforeRecommendation(t *testing.T) {
	ctx, pool := fixture.NewTestPool(t)

	userID, err := fixture.CreateTestUser(ctx, pool)
	if err != nil {
		t.Fatalf("CreateTestUser() error = %v", err)
	}
	unitIDs, err := fixture.CreateTestCoarseUnits(ctx, pool, 3)
	if err != nil {
		t.Fatalf("CreateTestCoarseUnits() error = %v", err)
	}
	t.Cleanup(func() {
		fixture.CleanupTestData(ctx, t, pool, userID, unitIDs)
	})

	recordUC := fixture.NewRecordEventsUseCase(pool)
	replayUC := fixture.NewReplayUseCase(pool)
	stateRepo := fixture.NewStateRepository(pool)
	generateUC := fixture.NewGenerateUseCase(pool)

	baseTime := time.Date(2026, 4, 2, 9, 0, 0, 0, time.UTC)
	eventCmd := lecommand.RecordLearningEventsCommand{
		UserID: userID,
		Events: []lecommand.LearningEventInput{
			fixture.NewLearnInput(unitIDs[0], true, 5, baseTime, "replay-u0-new"),
			fixture.ReviewInput(unitIDs[0], true, 5, baseTime.Add(72*time.Hour), "replay-u0-review"),
			fixture.NewLearnInput(unitIDs[1], true, 4, baseTime.Add(1*time.Hour), "replay-u1-new"),
			fixture.NewLearnInput(unitIDs[2], true, 4, baseTime.Add(2*time.Hour), "replay-u2-new"),
			fixture.ReviewInput(unitIDs[2], false, 2, baseTime.Add(48*time.Hour), "replay-u2-review"),
		},
		IdempotencyKey: "e2e-replay",
	}
	if _, err := recordUC.Execute(ctx, eventCmd); err != nil {
		t.Fatalf("RecordLearningEvents.Execute() error = %v", err)
	}

	beforeReplay := make(map[int64]stateSnapshot, len(unitIDs))
	for _, unitID := range unitIDs {
		state, err := stateRepo.GetByUserAndUnit(ctx, userID, unitID)
		if err != nil {
			t.Fatalf("GetByUserAndUnit(%d) before replay error = %v", unitID, err)
		}
		beforeReplay[unitID] = snapshotFromState(state)
	}

	if _, err := pool.Exec(ctx, `
		update learning.user_unit_states
		set status = 'mastered',
		    progress_percent = 99,
		    mastery_score = 0.99,
		    consecutive_wrong = 99,
		    next_review_at = null,
		    updated_at = $2
		where user_id = $1
	`, userID, baseTime.Add(20*24*time.Hour)); err != nil {
		t.Fatalf("corrupt states error = %v", err)
	}

	replayResult, err := replayUC.Execute(ctx, lecommand.ReplayUserStatesCommand{UserID: userID})
	if err != nil {
		t.Fatalf("ReplayUserStates.Execute() error = %v", err)
	}
	if replayResult.RebuiltCount != 3 {
		t.Fatalf("ReplayUserStates.RebuiltCount = %d, want 3", replayResult.RebuiltCount)
	}

	for _, unitID := range unitIDs {
		state, err := stateRepo.GetByUserAndUnit(ctx, userID, unitID)
		if err != nil {
			t.Fatalf("GetByUserAndUnit(%d) after replay error = %v", unitID, err)
		}
		afterReplay := snapshotFromState(state)
		if !reflect.DeepEqual(afterReplay, beforeReplay[unitID]) {
			t.Fatalf("state mismatch after replay for unit %d:\n got  %+v\n want %+v", unitID, afterReplay, beforeReplay[unitID])
		}
	}

	result, err := generateUC.Execute(ctx, fixture.GenerateCommand(userID, 3, baseTime.Add(10*24*time.Hour)))
	if err != nil {
		t.Fatalf("GenerateRecommendations.Execute() error = %v", err)
	}
	if len(result.Batch.Items) != 3 {
		t.Fatalf("len(Batch.Items) = %d, want 3", len(result.Batch.Items))
	}
	if result.Batch.Items[0].CoarseUnitID != unitIDs[1] {
		t.Fatalf("Batch.Items[0].CoarseUnitID = %d, want %d", result.Batch.Items[0].CoarseUnitID, unitIDs[1])
	}
	if result.Batch.Items[1].CoarseUnitID != unitIDs[2] {
		t.Fatalf("Batch.Items[1].CoarseUnitID = %d, want %d", result.Batch.Items[1].CoarseUnitID, unitIDs[2])
	}
}

func TestLearningEngineToRecommendationEndToEnd_ServingStateChangesRecommendationWithoutMutatingLearningState(t *testing.T) {
	ctx, pool := fixture.NewTestPool(t)

	userID, err := fixture.CreateTestUser(ctx, pool)
	if err != nil {
		t.Fatalf("CreateTestUser() error = %v", err)
	}
	unitIDs, err := fixture.CreateTestCoarseUnits(ctx, pool, 2)
	if err != nil {
		t.Fatalf("CreateTestCoarseUnits() error = %v", err)
	}
	t.Cleanup(func() {
		fixture.CleanupTestData(ctx, t, pool, userID, unitIDs)
	})

	stateRepo := fixture.NewStateRepository(pool)
	generateUC := fixture.NewGenerateUseCase(pool)

	baseTime := time.Date(2026, 4, 4, 12, 0, 0, 0, time.UTC)
	if err := fixture.SeedNewTargetState(ctx, stateRepo, userID, unitIDs[0], 0.95, "serving-high", baseTime); err != nil {
		t.Fatalf("SeedNewTargetState(unit0) error = %v", err)
	}
	if err := fixture.SeedNewTargetState(ctx, stateRepo, userID, unitIDs[1], 0.85, "serving-low", baseTime); err != nil {
		t.Fatalf("SeedNewTargetState(unit1) error = %v", err)
	}

	beforeState0, err := stateRepo.GetByUserAndUnit(ctx, userID, unitIDs[0])
	if err != nil {
		t.Fatalf("GetByUserAndUnit(unit0) before error = %v", err)
	}
	beforeState1, err := stateRepo.GetByUserAndUnit(ctx, userID, unitIDs[1])
	if err != nil {
		t.Fatalf("GetByUserAndUnit(unit1) before error = %v", err)
	}

	firstResult, err := generateUC.Execute(ctx, fixture.GenerateCommand(userID, 1, baseTime.Add(24*time.Hour)))
	if err != nil {
		t.Fatalf("first GenerateRecommendations.Execute() error = %v", err)
	}
	if len(firstResult.Batch.Items) != 1 {
		t.Fatalf("len(first batch items) = %d, want 1", len(firstResult.Batch.Items))
	}
	if firstResult.Batch.Items[0].CoarseUnitID != unitIDs[0] {
		t.Fatalf("first batch coarseUnitID = %d, want %d", firstResult.Batch.Items[0].CoarseUnitID, unitIDs[0])
	}

	secondResult, err := generateUC.Execute(ctx, fixture.GenerateCommand(userID, 1, baseTime.Add(25*time.Hour)))
	if err != nil {
		t.Fatalf("second GenerateRecommendations.Execute() error = %v", err)
	}
	if len(secondResult.Batch.Items) != 1 {
		t.Fatalf("len(second batch items) = %d, want 1", len(secondResult.Batch.Items))
	}
	if secondResult.Batch.Items[0].CoarseUnitID != unitIDs[1] {
		t.Fatalf("second batch coarseUnitID = %d, want %d", secondResult.Batch.Items[0].CoarseUnitID, unitIDs[1])
	}

	afterState0, err := stateRepo.GetByUserAndUnit(ctx, userID, unitIDs[0])
	if err != nil {
		t.Fatalf("GetByUserAndUnit(unit0) after error = %v", err)
	}
	afterState1, err := stateRepo.GetByUserAndUnit(ctx, userID, unitIDs[1])
	if err != nil {
		t.Fatalf("GetByUserAndUnit(unit1) after error = %v", err)
	}

	if !afterState0.UpdatedAt.Equal(beforeState0.UpdatedAt) {
		t.Fatalf("unit0 UpdatedAt changed: before=%v after=%v", beforeState0.UpdatedAt, afterState0.UpdatedAt)
	}
	if !afterState1.UpdatedAt.Equal(beforeState1.UpdatedAt) {
		t.Fatalf("unit1 UpdatedAt changed: before=%v after=%v", beforeState1.UpdatedAt, afterState1.UpdatedAt)
	}

	var runCount int
	if err := pool.QueryRow(ctx, `select count(*) from recommendation.scheduler_runs where user_id = $1`, userID).Scan(&runCount); err != nil {
		t.Fatalf("count recommendation.scheduler_runs error = %v", err)
	}
	if runCount != 2 {
		t.Fatalf("recommendation.scheduler_runs count = %d, want 2", runCount)
	}
}

func snapshotFromState(state *lemodel.UserUnitState) stateSnapshot {
	var lastQuality *int
	if state.LastQuality != nil {
		value := *state.LastQuality
		lastQuality = &value
	}

	var nextReviewAt *time.Time
	if state.NextReviewAt != nil {
		value := *state.NextReviewAt
		nextReviewAt = &value
	}

	return stateSnapshot{
		IsTarget:                state.IsTarget,
		TargetSource:            state.TargetSource,
		TargetSourceRefID:       state.TargetSourceRefID,
		TargetPriority:          state.TargetPriority,
		Status:                  string(state.Status),
		ProgressPercent:         state.ProgressPercent,
		MasteryScore:            state.MasteryScore,
		SeenCount:               state.SeenCount,
		StrongEventCount:        state.StrongEventCount,
		ReviewCount:             state.ReviewCount,
		CorrectCount:            state.CorrectCount,
		WrongCount:              state.WrongCount,
		ConsecutiveCorrect:      state.ConsecutiveCorrect,
		ConsecutiveWrong:        state.ConsecutiveWrong,
		LastQuality:             lastQuality,
		RecentQualityWindow:     append([]int(nil), state.RecentQualityWindow...),
		RecentCorrectnessWindow: append([]bool(nil), state.RecentCorrectnessWindow...),
		Repetition:              state.Repetition,
		IntervalDays:            state.IntervalDays,
		EaseFactor:              state.EaseFactor,
		NextReviewAt:            nextReviewAt,
	}
}
