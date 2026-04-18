//go:build integration

package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	appservice "learning-video-recommendation-system/internal/recommendation/application/service"
	"learning-video-recommendation-system/internal/recommendation/domain/model"
	"learning-video-recommendation-system/internal/recommendation/infrastructure/persistence/repository"
	recommendationsqlc "learning-video-recommendation-system/internal/recommendation/infrastructure/persistence/sqlcgen"
	"learning-video-recommendation-system/internal/recommendation/infrastructure/persistence/tx"
	"learning-video-recommendation-system/internal/recommendation/test/fixture"
)

type execer interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

func TestVideoUserStateReaderListByUserAndVideoIDs(t *testing.T) {
	db := testDB(t)
	tx := fixture.BeginTestTx(t, db.Pool)
	ctx := context.Background()

	userID := "00000000-0000-0000-0000-000000000101"
	videoID := "00000000-0000-0000-0000-000000000201"
	seedBaseRefs(t, ctx, db, tx, userID, videoID, 301)

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
	db := testDB(t)
	tx := fixture.BeginTestTx(t, db.Pool)
	ctx := context.Background()

	userID := "00000000-0000-0000-0000-000000000102"
	videoID := "00000000-0000-0000-0000-000000000202"
	unitID := int64(301)
	runID := "00000000-0000-0000-0000-000000000401"
	now := time.Now().UTC()
	seedBaseRefs(t, ctx, db, tx, userID, videoID, unitID)

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
	db := testDB(t)
	tx := fixture.BeginTestTx(t, db.Pool)
	ctx := context.Background()

	userID := "00000000-0000-0000-0000-000000000103"
	videoID := "00000000-0000-0000-0000-000000000203"
	runID := "00000000-0000-0000-0000-000000000403"
	unitID := int64(301)
	seedBaseRefs(t, ctx, db, tx, userID, videoID, unitID)

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
			DominantUnitID:         &unitID,
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
			DominantUnitID:         &unitID,
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

func TestReadModelRepositoriesUseRealMaterializedViews(t *testing.T) {
	db := testDB(t)
	tx := fixture.BeginTestTx(t, db.Pool)
	ctx := context.Background()

	userID := "00000000-0000-0000-0000-000000000104"
	videoID := "00000000-0000-0000-0000-000000000204"
	unitID := int64(401)
	seedBaseRefs(t, ctx, db, tx, userID, videoID, unitID)

	if _, err := tx.Exec(ctx, `insert into catalog.video_transcripts (video_id, mapped_span_ratio) values ($1, 0.70000)`, videoID); err != nil {
		t.Fatalf("seed transcript: %v", err)
	}
	if _, err := tx.Exec(ctx, `
		insert into catalog.video_unit_index (
			video_id, coarse_unit_id, mention_count, sentence_count, first_start_ms, last_end_ms, coverage_ms, coverage_ratio,
			sentence_indexes, evidence_span_refs, sample_surface_forms
		) values ($1, $2, 3, 2, 1000, 5000, 4000, 0.12000, '{1,2}', '[{"sentence_index":1,"span_index":1}]'::jsonb, '{"surface"}')
	`, videoID, unitID); err != nil {
		t.Fatalf("seed unit index: %v", err)
	}
	if _, err := tx.Exec(ctx, `insert into catalog.video_semantic_spans (video_id, sentence_index, span_index, coarse_unit_id, start_ms, end_ms, text) values ($1, 1, 1, $2, 1000, 1500, 'span')`, videoID, unitID); err != nil {
		t.Fatalf("seed semantic span: %v", err)
	}
	if _, err := tx.Exec(ctx, `insert into catalog.video_transcript_sentences (video_id, sentence_index, text, start_ms, end_ms) values ($1, 1, 'Sentence 1', 900, 1600)`, videoID); err != nil {
		t.Fatalf("seed transcript sentence: %v", err)
	}

	queries := recommendationsqlc.New(tx)
	if err := queries.RefreshRecommendableVideoUnits(ctx); err != nil {
		t.Fatalf("refresh recommendable: %v", err)
	}
	if err := queries.RefreshUnitVideoInventory(ctx); err != nil {
		t.Fatalf("refresh inventory: %v", err)
	}

	recommendableReader := repository.NewRecommendableVideoUnitReader(tx)
	rows, err := recommendableReader.ListByUnitIDs(ctx, []int64{unitID})
	if err != nil {
		t.Fatalf("list recommendable rows: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 recommendable row, got %#v", rows)
	}
	if rows[0].VideoID != videoID || rows[0].CoarseUnitID != unitID {
		t.Fatalf("unexpected recommendable row: %+v", rows[0])
	}

	inventoryReader := repository.NewUnitInventoryReader(tx)
	inventory, err := inventoryReader.ListByUnitIDs(ctx, []int64{unitID})
	if err != nil {
		t.Fatalf("list unit inventory: %v", err)
	}
	if len(inventory) != 1 {
		t.Fatalf("expected 1 inventory row, got %#v", inventory)
	}
	if inventory[0].DistinctVideoCount != 1 || inventory[0].SupplyGrade != "weak" {
		t.Fatalf("unexpected inventory row: %+v", inventory[0])
	}
}

func TestUnitInventoryReadModelCoversSupplyGradesAndNone(t *testing.T) {
	db := testDB(t)
	tx := fixture.BeginTestTx(t, db.Pool)
	ctx := context.Background()

	userID := "00000000-0000-0000-0000-000000000105"
	seedUser(t, ctx, db, tx, userID)
	seedCoarseUnit(t, ctx, db, tx, 501)
	seedCoarseUnit(t, ctx, db, tx, 502)
	seedCoarseUnit(t, ctx, db, tx, 503)
	seedCoarseUnit(t, ctx, db, tx, 504)
	seedCoarseUnit(t, ctx, db, tx, 505)

	seedInventoryVideo(t, ctx, db, tx, "00000000-0000-0000-0000-000000000301", 501, 2, 0.05000, 0.60000)
	seedInventoryVideo(t, ctx, db, tx, "00000000-0000-0000-0000-000000000302", 502, 2, 0.05000, 0.60000)
	seedInventoryVideo(t, ctx, db, tx, "00000000-0000-0000-0000-000000000303", 502, 2, 0.05000, 0.60000)
	for i := 0; i < 4; i++ {
		videoID := videoIDFromIndex(400 + i)
		seedInventoryVideo(t, ctx, db, tx, videoID, 503, 2, 0.05000, 0.60000)
	}
	for i := 0; i < 4; i++ {
		videoID := videoIDFromIndex(500 + i)
		seedInventoryVideo(t, ctx, db, tx, videoID, 504, 2, 0.05000, 0.60000)
	}

	queries := recommendationsqlc.New(tx)
	if err := queries.RefreshRecommendableVideoUnits(ctx); err != nil {
		t.Fatalf("refresh recommendable: %v", err)
	}
	if err := queries.RefreshUnitVideoInventory(ctx); err != nil {
		t.Fatalf("refresh inventory: %v", err)
	}

	inventoryReader := repository.NewUnitInventoryReader(tx)
	inventory, err := inventoryReader.ListByUnitIDs(ctx, []int64{501, 502, 503, 504, 505})
	if err != nil {
		t.Fatalf("list unit inventory: %v", err)
	}

	byUnit := make(map[int64]model.UnitVideoInventory, len(inventory))
	for _, row := range inventory {
		byUnit[row.CoarseUnitID] = row
	}

	if byUnit[501].SupplyGrade != "weak" {
		t.Fatalf("unit 501 supply grade = %q, want weak", byUnit[501].SupplyGrade)
	}
	if byUnit[502].SupplyGrade != "ok" {
		t.Fatalf("unit 502 supply grade = %q, want ok", byUnit[502].SupplyGrade)
	}
	if byUnit[503].SupplyGrade != "strong" {
		t.Fatalf("unit 503 supply grade = %q, want strong", byUnit[503].SupplyGrade)
	}
	if byUnit[504].SupplyGrade != "strong" {
		t.Fatalf("unit 504 supply grade = %q, want strong", byUnit[504].SupplyGrade)
	}
	if byUnit[505].SupplyGrade != "none" {
		t.Fatalf("unit 505 supply grade = %q, want none", byUnit[505].SupplyGrade)
	}
}

func TestRecommendationResultWriterPersistsAuditAndServingStatesInSingleFlow(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	conn, err := db.Pool.Acquire(ctx)
	if err != nil {
		t.Fatalf("acquire connection: %v", err)
	}
	defer conn.Release()

	manager := tx.NewManager(db.Pool)
	writer := appservice.NewDefaultRecommendationResultWriter(
		manager,
		appservice.NewDefaultAuditWriter(repository.NewRecommendationAuditRepository(db.Pool)),
		appservice.NewDefaultServingStateManager(
			repository.NewUnitServingStateRepository(db.Pool),
			repository.NewVideoServingStateRepository(db.Pool),
		),
	)

	runID := "00000000-0000-0000-0000-000000000501"
	userID := "00000000-0000-0000-0000-000000000111"
	videoID := "00000000-0000-0000-0000-000000000211"
	seedBaseRefs(t, ctx, db, conn.Conn(), userID, videoID, 301)

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
	if err := db.Pool.QueryRow(ctx, `select count(*) from recommendation.video_recommendation_runs where run_id = $1`, runID).Scan(&runCount); err != nil {
		t.Fatalf("count runs: %v", err)
	}
	if runCount != 1 {
		t.Fatalf("expected 1 run, got %d", runCount)
	}

	var itemCount int
	if err := db.Pool.QueryRow(ctx, `select count(*) from recommendation.video_recommendation_items where run_id = $1`, runID).Scan(&itemCount); err != nil {
		t.Fatalf("count items: %v", err)
	}
	if itemCount != 1 {
		t.Fatalf("expected 1 item, got %d", itemCount)
	}

	var servedCount int
	if err := db.Pool.QueryRow(ctx, `select served_count from recommendation.user_video_serving_states where user_id = $1 and video_id = $2`, userID, videoID).Scan(&servedCount); err != nil {
		t.Fatalf("video serving state: %v", err)
	}
	if servedCount != 1 {
		t.Fatalf("expected served_count=1, got %d", servedCount)
	}
}

func seedBaseRefs(t *testing.T, ctx context.Context, testDB *fixture.TestDatabase, db execer, userID string, videoID string, unitID int64) {
	t.Helper()
	seedUser(t, ctx, testDB, db, userID)
	seedCoarseUnit(t, ctx, testDB, db, unitID)
	testDB.SeedVideo(t, videoID)
}

func seedUser(t *testing.T, ctx context.Context, testDB *fixture.TestDatabase, db execer, userID string) {
	t.Helper()
	testDB.SeedUser(t, userID)
}

func seedCoarseUnit(t *testing.T, ctx context.Context, testDB *fixture.TestDatabase, db execer, unitID int64) {
	t.Helper()
	testDB.SeedCoarseUnit(t, unitID)
}

func seedInventoryVideo(t *testing.T, ctx context.Context, testDB *fixture.TestDatabase, db execer, videoID string, unitID int64, mentionCount int, coverageRatio float64, mappedSpanRatio float64) {
	t.Helper()
	seedCoarseUnit(t, ctx, testDB, db, unitID)
	testDB.SeedVideo(t, videoID)
	if _, err := db.Exec(ctx, `insert into catalog.video_transcripts (video_id, mapped_span_ratio) values ($1, $2) on conflict (video_id) do update set mapped_span_ratio = excluded.mapped_span_ratio`, videoID, mappedSpanRatio); err != nil {
		t.Fatalf("seed inventory transcript: %v", err)
	}
	if _, err := db.Exec(ctx, `
		insert into catalog.video_unit_index (
			video_id, coarse_unit_id, mention_count, sentence_count, first_start_ms, last_end_ms, coverage_ms, coverage_ratio,
			sentence_indexes, evidence_span_refs, sample_surface_forms
		) values ($1, $2, $3, 2, 1000, 5000, 4000, $4, '{1,2}', '[{"sentence_index":1,"span_index":1}]'::jsonb, '{"surface"}')
		on conflict do nothing
	`, videoID, unitID, mentionCount, coverageRatio); err != nil {
		t.Fatalf("seed inventory unit index: %v", err)
	}
}

func videoIDFromIndex(index int) string {
	return fmt.Sprintf("00000000-0000-0000-0000-%012d", index)
}

func int64Ptr(value int64) *int64 {
	return &value
}
