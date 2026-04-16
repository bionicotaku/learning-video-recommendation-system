//go:build integration

package integration

import (
	"context"
	"testing"
	"time"

	appservice "learning-video-recommendation-system/internal/recommendation/application/service"
	"learning-video-recommendation-system/internal/recommendation/domain/model"
	"learning-video-recommendation-system/internal/recommendation/infrastructure/persistence/repository"
	"learning-video-recommendation-system/internal/recommendation/infrastructure/persistence/tx"
	"learning-video-recommendation-system/internal/recommendation/test/fixture"
)

func TestVideoUserStateReaderListByUserAndVideoIDs(t *testing.T) {
	pool := fixture.OpenPool(t)
	tx := fixture.BeginTestTx(t, pool)
	ctx := context.Background()

	if err := fixture.EnsureRecommendationStep1Schema(ctx, tx); err != nil {
		t.Fatalf("ensure schema: %v", err)
	}

	userID := "00000000-0000-0000-0000-000000000101"
	videoID := "00000000-0000-0000-0000-000000000201"

	if _, err := tx.Exec(ctx, `insert into catalog.video_user_states (user_id, video_id, watch_count, completed_count, last_watch_ratio, max_watch_ratio) values ($1, $2, 3, 1, 0.65000, 0.90000)`, userID, videoID); err != nil {
		t.Fatalf("seed video_user_states: %v", err)
	}

	reader := repository.NewVideoUserStateReader(tx)
	states, err := reader.ListByUserAndVideoIDs(ctx, userID, []string{videoID})
	if err != nil {
		t.Fatalf("list video user states: %v", err)
	}
	if len(states) != 1 {
		t.Fatalf("expected 1 state, got %d", len(states))
	}
	if states[0].WatchCount != 3 || states[0].CompletedCount != 1 {
		t.Fatalf("unexpected counters: %+v", states[0])
	}
}

func TestServingStateRepositoriesListAndUpsert(t *testing.T) {
	pool := fixture.OpenPool(t)
	tx := fixture.BeginTestTx(t, pool)
	ctx := context.Background()

	if err := fixture.EnsureRecommendationStep1Schema(ctx, tx); err != nil {
		t.Fatalf("ensure schema: %v", err)
	}

	userID := "00000000-0000-0000-0000-000000000102"
	videoID := "00000000-0000-0000-0000-000000000202"
	unitID := int64(301)
	runID := "00000000-0000-0000-0000-000000000401"
	now := time.Now().UTC()

	unitRepo := repository.NewUnitServingStateRepository(tx)
	videoRepo := repository.NewVideoServingStateRepository(tx)

	if err := unitRepo.Upsert(ctx, model.UserUnitServingState{
		UserID:       userID,
		CoarseUnitID: unitID,
		LastServedAt: &now,
		LastRunID:    runID,
		ServedCount:  2,
	}); err != nil {
		t.Fatalf("upsert unit serving state: %v", err)
	}

	if err := videoRepo.Upsert(ctx, model.UserVideoServingState{
		UserID:       userID,
		VideoID:      videoID,
		LastServedAt: &now,
		LastRunID:    runID,
		ServedCount:  4,
	}); err != nil {
		t.Fatalf("upsert video serving state: %v", err)
	}

	unitStates, err := unitRepo.ListByUserAndUnitIDs(ctx, userID, []int64{unitID})
	if err != nil {
		t.Fatalf("list unit serving states: %v", err)
	}
	if len(unitStates) != 1 || unitStates[0].ServedCount != 2 {
		t.Fatalf("unexpected unit states: %+v", unitStates)
	}

	videoStates, err := videoRepo.ListByUserAndVideoIDs(ctx, userID, []string{videoID})
	if err != nil {
		t.Fatalf("list video serving states: %v", err)
	}
	if len(videoStates) != 1 || videoStates[0].ServedCount != 4 {
		t.Fatalf("unexpected video states: %+v", videoStates)
	}
}

func TestRecommendationAuditRepositoryInsertItems(t *testing.T) {
	pool := fixture.OpenPool(t)
	tx := fixture.BeginTestTx(t, pool)
	ctx := context.Background()

	if err := fixture.EnsureRecommendationStep1Schema(ctx, tx); err != nil {
		t.Fatalf("ensure schema: %v", err)
	}

	userID := "00000000-0000-0000-0000-000000000103"
	videoID := "00000000-0000-0000-0000-000000000203"
	runID := "00000000-0000-0000-0000-000000000403"

	if _, err := tx.Exec(ctx, `insert into recommendation.video_recommendation_runs (run_id, user_id, request_context, planner_snapshot, lane_budget_snapshot, candidate_summary, underfilled, result_count) values ($1, $2, '{}'::jsonb, '{}'::jsonb, '{}'::jsonb, '{}'::jsonb, false, 2)`, runID, userID); err != nil {
		t.Fatalf("seed run: %v", err)
	}

	repo := repository.NewRecommendationAuditRepository(tx)
	items := []model.RecommendationItem{
		{
			RunID:                  runID,
			Rank:                   1,
			VideoID:                videoID,
			Score:                  0.91,
			PrimaryLane:            "exact_core",
			DominantBucket:         "hard_review",
			ReasonCodes:            []string{"hard_review_covered"},
			CoveredHardReviewCount: 1,
		},
		{
			RunID:                  runID,
			Rank:                   2,
			VideoID:                videoID,
			Score:                  0.67,
			PrimaryLane:            "bundle",
			DominantBucket:         "soft_review",
			ReasonCodes:            []string{"bundle_coverage_high"},
			CoveredSoftReviewCount: 1,
		},
	}

	if err := repo.InsertItems(ctx, items); err != nil {
		t.Fatalf("insert items: %v", err)
	}

	var count int
	if err := tx.QueryRow(ctx, `select count(*) from recommendation.video_recommendation_items where run_id = $1`, runID).Scan(&count); err != nil {
		t.Fatalf("count items: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2 items, got %d", count)
	}
}

func TestRecommendationResultWriterPersistsAuditAndServingStatesInSingleFlow(t *testing.T) {
	pool := fixture.OpenPool(t)
	ctx := context.Background()

	conn, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatalf("acquire connection: %v", err)
	}
	defer conn.Release()

	if err := fixture.EnsureRecommendationStep1Schema(ctx, conn.Conn()); err != nil {
		t.Fatalf("ensure schema: %v", err)
	}

	manager := tx.NewManager(pool)
	writer := appservice.NewDefaultRecommendationResultWriter(
		manager,
		appservice.NewDefaultAuditWriter(repository.NewRecommendationAuditRepository(pool)),
		appservice.NewDefaultServingStateManager(
			repository.NewUnitServingStateRepository(pool),
			repository.NewVideoServingStateRepository(pool),
		),
	)

	runID := "00000000-0000-0000-0000-000000000501"
	userID := "00000000-0000-0000-0000-000000000111"
	videoID := "00000000-0000-0000-0000-000000000211"

	err = writer.Persist(ctx, model.RecommendationRun{
		RunID:              runID,
		UserID:             userID,
		RequestContext:     []byte(`{}`),
		PlannerSnapshot:    []byte(`{}`),
		LaneBudgetSnapshot: []byte(`{}`),
		CandidateSummary:   []byte(`{}`),
		ResultCount:        1,
	}, []model.RecommendationItem{
		{
			RunID:                  runID,
			Rank:                   1,
			VideoID:                videoID,
			Score:                  0.91,
			PrimaryLane:            "exact_core",
			DominantBucket:         "hard_review",
			DominantUnitID:         int64Ptr(301),
			ReasonCodes:            []string{"hard_review_covered"},
			CoveredHardReviewCount: 1,
		},
	}, userID, []model.FinalRecommendationItem{
		{
			VideoID:                videoID,
			CoveredUnits:           []int64{301},
			CoveredHardReviewUnits: []int64{301},
		},
	})
	if err != nil {
		t.Fatalf("persist result: %v", err)
	}

	var runCount int
	if err := pool.QueryRow(ctx, `select count(*) from recommendation.video_recommendation_runs where run_id = $1`, runID).Scan(&runCount); err != nil {
		t.Fatalf("count runs: %v", err)
	}
	if runCount != 1 {
		t.Fatalf("expected 1 run, got %d", runCount)
	}

	var itemCount int
	if err := pool.QueryRow(ctx, `select count(*) from recommendation.video_recommendation_items where run_id = $1`, runID).Scan(&itemCount); err != nil {
		t.Fatalf("count items: %v", err)
	}
	if itemCount != 1 {
		t.Fatalf("expected 1 item, got %d", itemCount)
	}

	var servedCount int
	if err := pool.QueryRow(ctx, `select served_count from recommendation.user_video_serving_states where user_id = $1 and video_id = $2`, userID, videoID).Scan(&servedCount); err != nil {
		t.Fatalf("video serving state: %v", err)
	}
	if servedCount != 1 {
		t.Fatalf("expected served_count=1, got %d", servedCount)
	}
}

func int64Ptr(value int64) *int64 {
	return &value
}
