//go:build integration

package integration

import (
	"context"
	"fmt"
	"sync"
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

	if _, err := tx.Exec(ctx, `insert into catalog.video_user_states (user_id, video_id, watch_count, completed_count, last_position_ms, max_position_ms, total_watch_ms) values ($1, $2, 3, 1, 65000, 90000, 120000)`, userID, videoID); err != nil {
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
	if states[0].LastPositionMs != 65000 || states[0].MaxPositionMs != 90000 || states[0].TotalWatchMs != 120000 {
		t.Fatalf("unexpected watch state: %+v", states[0])
	}
}

func TestLearningStateReaderListActiveByUserExcludesMasteredTargets(t *testing.T) {
	db := testDB(t)
	tx := fixture.BeginTestTx(t, db.Pool)
	ctx := context.Background()

	userID := "00000000-0000-0000-0000-000000000121"
	db.SeedUser(t, userID)
	for _, unitID := range []int64{321, 322, 323, 324, 325} {
		db.SeedCoarseUnit(t, unitID)
	}

	if _, err := tx.Exec(ctx, `
		insert into learning.user_unit_states (
			user_id,
			coarse_unit_id,
			is_target,
			target_priority,
			status,
			mastery_score
		) values
			($1, 321, true, 0.90, 'mastered', 1.0),
			($1, 322, true, 0.80, 'new', 0.0),
			($1, 323, true, 0.70, 'learning', 0.3),
			($1, 324, true, 0.60, 'reviewing', 0.7),
				($1, 325, false, 0.50, 'learning', 0.4)
	`, userID); err != nil {
		t.Fatalf("seed learning states: %v", err)
	}

	reader := repository.NewLearningStateReader(tx)
	states, err := reader.ListActiveByUser(ctx, userID)
	if err != nil {
		t.Fatalf("list learning states: %v", err)
	}

	got := make([]int64, 0, len(states))
	for _, state := range states {
		got = append(got, state.CoarseUnitID)
	}
	want := []int64{322, 323, 324}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("unit ids = %v, want %v", got, want)
	}
}

func TestRecallQueueRepositoryRebuildsAndRanksScopedCandidates(t *testing.T) {
	db := testDB(t)
	tx := fixture.BeginTestTx(t, db.Pool)
	ctx := context.Background()

	userID := "00000000-0000-0000-0000-000000000122"
	db.SeedUser(t, userID)
	for unitID := int64(331); unitID <= 334; unitID++ {
		db.SeedCoarseUnit(t, unitID)
	}
	seedInventoryVideo(t, ctx, db, tx, "00000000-0000-0000-0000-000000000331", 331, 3, 0.12000, 0.70000)
	seedInventoryVideo(t, ctx, db, tx, "00000000-0000-0000-0000-000000000332", 332, 3, 0.12000, 0.70000)
	queries := recommendationsqlc.New(tx)
	if err := queries.RefreshVideoUnitRecallIndex(ctx); err != nil {
		t.Fatalf("refresh recall index: %v", err)
	}
	if err := queries.RefreshUnitVideoInventory(ctx); err != nil {
		t.Fatalf("refresh inventory: %v", err)
	}

	now := time.Now().UTC()
	if _, err := tx.Exec(ctx, `
		insert into learning.user_unit_states (
			user_id, coarse_unit_id, is_target, target_priority, status, mastery_score, last_progress_quality, next_review_at, updated_at
		) values
			($1, 331, true, 0.90, 'reviewing', 0.7, 2, $2, $3),
			($1, 332, true, 0.80, 'new', 0.0, null, null, $3),
			($1, 333, true, 0.70, 'learning', 0.5, null, $4, $3),
			($1, 334, true, 0.60, 'mastered', 1.0, null, null, $3)
	`, userID, now.Add(-time.Minute), now.Add(-time.Hour), now.Add(24*time.Hour)); err != nil {
		t.Fatalf("seed learning states: %v", err)
	}
	if _, err := tx.Exec(ctx, `
		insert into recommendation.user_unit_serving_states (user_id, coarse_unit_id, last_served_at, served_count)
		values ($1, 331, $2, 3)
	`, userID, now.Add(-time.Hour)); err != nil {
		t.Fatalf("seed serving state: %v", err)
	}

	repo := repository.NewRecallQueueRepository(tx)
	projectionUpdatedAt, err := repo.GetProjectionUpdatedAt(ctx)
	if err != nil {
		t.Fatalf("get projection metadata: %v", err)
	}
	state, err := repo.RebuildUserQueue(ctx, userID, projectionUpdatedAt)
	if err != nil {
		t.Fatalf("rebuild queue: %v", err)
	}
	if state.ActiveTargetUnitCount != 3 {
		t.Fatalf("active target count = %d, want 3", state.ActiveTargetUnitCount)
	}

	candidates, err := repo.ListCandidates(ctx, userID, now, 10, 4)
	if err != nil {
		t.Fatalf("list queue candidates: %v", err)
	}
	if len(candidates) != 3 {
		t.Fatalf("candidate count = %d, want 3", len(candidates))
	}
	byUnit := make(map[int64]model.RecallQueueCandidate, len(candidates))
	for _, candidate := range candidates {
		byUnit[candidate.CoarseUnitID] = candidate
	}
	if byUnit[331].Bucket != "hard_review" {
		t.Fatalf("unit 331 bucket = %q, want hard_review", byUnit[331].Bucket)
	}
	if byUnit[332].Bucket != "new_now" {
		t.Fatalf("unit 332 bucket = %q, want new_now", byUnit[332].Bucket)
	}
	if byUnit[333].Bucket != "soft_review" {
		t.Fatalf("unit 333 bucket = %q, want soft_review", byUnit[333].Bucket)
	}
	if byUnit[331].ServedCount != 3 || byUnit[331].LastServedAt == nil {
		t.Fatalf("expected serving state joined into queue candidate: %+v", byUnit[331])
	}
}

func TestRecallQueueRepositoryReturnsSuppliedCandidatesWhenNoSupplyHasHigherPriority(t *testing.T) {
	db := testDB(t)
	tx := fixture.BeginTestTx(t, db.Pool)
	ctx := context.Background()

	userID := "00000000-0000-0000-0000-000000000124"
	db.SeedUser(t, userID)
	now := time.Now().UTC()

	for unitID := int64(351); unitID <= 360; unitID++ {
		db.SeedCoarseUnit(t, unitID)
		if _, err := tx.Exec(ctx, `
			insert into learning.user_unit_states (
				user_id, coarse_unit_id, is_target, target_priority, status, mastery_score, last_progress_quality, updated_at
			) values ($1, $2, true, 1.00, 'reviewing', 0.7, 2, $3)
		`, userID, unitID, now.Add(-time.Hour)); err != nil {
			t.Fatalf("seed no-supply learning state %d: %v", unitID, err)
		}
	}
	for unitID := int64(361); unitID <= 363; unitID++ {
		seedInventoryVideo(t, ctx, db, tx, videoIDFromIndex(int(unitID)), unitID, 3, 0.12000, 0.70000)
		if _, err := tx.Exec(ctx, `
			insert into learning.user_unit_states (
				user_id, coarse_unit_id, is_target, target_priority, status, mastery_score, last_progress_quality, updated_at
			) values ($1, $2, true, 0.20, 'reviewing', 0.7, 2, $3)
		`, userID, unitID, now.Add(-time.Hour)); err != nil {
			t.Fatalf("seed supplied learning state %d: %v", unitID, err)
		}
	}

	queries := recommendationsqlc.New(tx)
	if err := queries.RefreshVideoUnitRecallIndex(ctx); err != nil {
		t.Fatalf("refresh recall index: %v", err)
	}
	if err := queries.RefreshUnitVideoInventory(ctx); err != nil {
		t.Fatalf("refresh inventory: %v", err)
	}

	repo := repository.NewRecallQueueRepository(tx)
	projectionUpdatedAt, err := repo.GetProjectionUpdatedAt(ctx)
	if err != nil {
		t.Fatalf("get projection metadata: %v", err)
	}
	if _, err := repo.RebuildUserQueue(ctx, userID, projectionUpdatedAt); err != nil {
		t.Fatalf("rebuild queue: %v", err)
	}

	candidates, err := repo.ListCandidates(ctx, userID, now, 3, 2)
	if err != nil {
		t.Fatalf("list queue candidates: %v", err)
	}
	suppliedCount := 0
	noSupplyCount := 0
	for _, candidate := range candidates {
		switch candidate.SupplyGrade {
		case "none":
			noSupplyCount++
		default:
			suppliedCount++
		}
	}
	if suppliedCount != 3 {
		t.Fatalf("supplied candidates = %d, want 3; candidates=%#v", suppliedCount, candidates)
	}
	if noSupplyCount > 2 {
		t.Fatalf("no-supply candidates = %d, want <=2; candidates=%#v", noSupplyCount, candidates)
	}
}

func TestRecallQueueRepositorySerializesConcurrentUserRebuilds(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	userID := "00000000-0000-0000-0000-000000000123"
	db.SeedUser(t, userID)
	for unitID := int64(341); unitID <= 345; unitID++ {
		seedInventoryVideo(t, ctx, db, db.Pool, videoIDFromIndex(int(unitID)), unitID, 3, 0.12000, 0.70000)
	}
	queries := recommendationsqlc.New(db.Pool)
	if err := queries.RefreshVideoUnitRecallIndex(ctx); err != nil {
		t.Fatalf("refresh recall index: %v", err)
	}
	if err := queries.RefreshUnitVideoInventory(ctx); err != nil {
		t.Fatalf("refresh inventory: %v", err)
	}

	now := time.Now().UTC()
	for unitID := int64(341); unitID <= 345; unitID++ {
		if _, err := db.Pool.Exec(ctx, `
			insert into learning.user_unit_states (
				user_id, coarse_unit_id, is_target, target_priority, status, mastery_score, updated_at
			) values ($1, $2, true, 0.80, 'learning', 0.5, $3)
		`, userID, unitID, now.Add(-time.Hour)); err != nil {
			t.Fatalf("seed learning state %d: %v", unitID, err)
		}
	}

	repo := repository.NewRecallQueueRepository(db.Pool)
	projectionUpdatedAt, err := repo.GetProjectionUpdatedAt(ctx)
	if err != nil {
		t.Fatalf("get projection metadata: %v", err)
	}
	if _, err := repo.RebuildUserQueue(ctx, userID, projectionUpdatedAt); err != nil {
		t.Fatalf("initial rebuild queue: %v", err)
	}

	const workers = 8
	var wg sync.WaitGroup
	errs := make(chan error, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := repo.RebuildUserQueue(ctx, userID, projectionUpdatedAt)
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent rebuild failed: %v", err)
		}
	}

	var rowCount int
	if err := db.Pool.QueryRow(ctx, `select count(*) from recommendation.user_unit_recall_queue where user_id = $1`, userID).Scan(&rowCount); err != nil {
		t.Fatalf("count queue rows: %v", err)
	}
	if rowCount != 5 {
		t.Fatalf("queue row count = %d, want 5", rowCount)
	}
}

func TestServingStateRepositoriesListAndIncrement(t *testing.T) {
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

	if err := unitRepo.IncrementServedCounts(ctx, userID, runID, now, []int64{unitID}); err != nil {
		t.Fatalf("increment unit serving state: %v", err)
	}
	if err := unitRepo.IncrementServedCounts(ctx, userID, runID, now, []int64{unitID}); err != nil {
		t.Fatalf("increment unit serving state again: %v", err)
	}

	if err := videoRepo.IncrementServedCounts(ctx, userID, runID, now, []string{videoID}); err != nil {
		t.Fatalf("increment video serving state: %v", err)
	}
	if err := videoRepo.IncrementServedCounts(ctx, userID, runID, now, []string{videoID}); err != nil {
		t.Fatalf("increment video serving state again: %v", err)
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
	if len(videoStates) != 1 || videoStates[0].ServedCount != 2 {
		t.Fatalf("unexpected video states: %+v", videoStates)
	}
}

func TestServingStateRepositoriesIncrementConcurrently(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	userID := "00000000-0000-0000-0000-000000000112"
	videoID := "00000000-0000-0000-0000-000000000212"
	unitID := int64(312)
	runID := "00000000-0000-0000-0000-000000000412"
	now := time.Now().UTC()
	seedBaseRefs(t, ctx, db, db.Pool, userID, videoID, unitID)

	unitRepo := repository.NewUnitServingStateRepository(db.Pool)
	videoRepo := repository.NewVideoServingStateRepository(db.Pool)

	var wg sync.WaitGroup
	errs := make(chan error, 4)
	for i := 0; i < 2; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			errs <- unitRepo.IncrementServedCounts(ctx, userID, runID, now, []int64{unitID})
		}()
		go func() {
			defer wg.Done()
			errs <- videoRepo.IncrementServedCounts(ctx, userID, runID, now, []string{videoID})
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("increment serving state concurrently: %v", err)
		}
	}

	unitStates, err := unitRepo.ListByUserAndUnitIDs(ctx, userID, []int64{unitID})
	if err != nil {
		t.Fatalf("list unit serving states: %v", err)
	}
	if len(unitStates) != 1 || unitStates[0].ServedCount != 2 {
		t.Fatalf("unexpected unit states after concurrent increments: %+v", unitStates)
	}

	videoStates, err := videoRepo.ListByUserAndVideoIDs(ctx, userID, []string{videoID})
	if err != nil {
		t.Fatalf("list video serving states: %v", err)
	}
	if len(videoStates) != 1 || videoStates[0].ServedCount != 2 {
		t.Fatalf("unexpected video states after concurrent increments: %+v", videoStates)
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
			RunID:          runID,
			Rank:           1,
			VideoID:        videoID,
			Score:          0.91,
			PrimaryLane:    "exact_core",
			DominantRole:   model.LearningRoleHardReview,
			DominantUnitID: &unitID,
			ReasonCodes:    []string{"hard_review_covered"},
			LearningUnits:  []model.ExpectedLearningUnit{learningUnit(unitID, model.LearningRoleHardReview)},
		},
		{
			RunID:          runID,
			Rank:           2,
			VideoID:        videoID,
			Score:          0.67,
			PrimaryLane:    "bundle",
			DominantRole:   model.LearningRoleSoftReview,
			DominantUnitID: &unitID,
			ReasonCodes:    []string{"bundle_coverage_high"},
			LearningUnits:  []model.ExpectedLearningUnit{learningUnit(unitID, model.LearningRoleSoftReview)},
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

	var learningUnitCount int
	if err := tx.QueryRow(ctx, `select jsonb_array_length(learning_units) from recommendation.video_recommendation_items where run_id = $1 and rank = 1`, runID).Scan(&learningUnitCount); err != nil {
		t.Fatalf("read learning_units: %v", err)
	}
	if learningUnitCount != 1 {
		t.Fatalf("expected 1 learning unit in audit item, got %d", learningUnitCount)
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
	if _, err := tx.Exec(ctx, `insert into catalog.video_transcript_sentences (video_id, sentence_index, start_ms, end_ms, text, translation) values ($1, 1, 900, 1600, 'fixture sentence', '测试句子')`, videoID); err != nil {
		t.Fatalf("seed transcript sentence: %v", err)
	}
	if _, err := tx.Exec(ctx, `insert into catalog.video_semantic_spans (video_id, sentence_index, span_index, coarse_unit_id, start_ms, end_ms, surface_text, explanation, base_form, translation, dictionary, mapping_reason) values ($1, 1, 1, $2, 1000, 1500, 'fixture span', 'fixture explanation', 'fixture', '测试', 'fixture dictionary', 'fixture reason')`, videoID, unitID); err != nil {
		t.Fatalf("seed semantic span: %v", err)
	}
	if _, err := tx.Exec(ctx, `
			insert into catalog.video_unit_index (
				video_id, coarse_unit_id, mention_count, sentence_count, coverage_ms, coverage_ratio,
				sentence_indexes, best_evidence_sentence_index, best_evidence_span_index,
				best_evidence_start_ms, best_evidence_end_ms,
				best_evidence_scores, best_evidence_question_reject_reason, best_evidence_selection_reason,
				best_evidence_candidate_score, best_evidence_target_text
			) values ($1, $2, 3, 2, 4000, 0.12000, '{1,2}', 1, 1, 1000, 1500, '{}'::jsonb, null, 'test fixture', 8.3500, 'fixture span')
		`, videoID, unitID); err != nil {
		t.Fatalf("seed unit index: %v", err)
	}

	queries := recommendationsqlc.New(tx)
	if err := queries.RefreshVideoUnitRecallIndex(ctx); err != nil {
		t.Fatalf("refresh recall index: %v", err)
	}
	if err := queries.RefreshUnitVideoInventory(ctx); err != nil {
		t.Fatalf("refresh inventory: %v", err)
	}

	recommendableReader := repository.NewRecommendableVideoUnitReader(tx)
	rows, err := recommendableReader.ListByUnitIDs(ctx, []int64{unitID}, 20)
	if err != nil {
		t.Fatalf("list recommendable rows: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 recommendable row, got %#v", rows)
	}
	if rows[0].VideoID != videoID || rows[0].CoarseUnitID != unitID {
		t.Fatalf("unexpected recommendable row: %+v", rows[0])
	}
	if rows[0].BestEvidenceCandidateScore == nil || *rows[0].BestEvidenceCandidateScore != 8.35 {
		t.Fatalf("expected best evidence candidate score, got %+v", rows[0])
	}
	if rows[0].BestEvidenceTargetText == nil || *rows[0].BestEvidenceTargetText != "fixture span" {
		t.Fatalf("expected best evidence target text, got %+v", rows[0])
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
	if err := queries.RefreshVideoUnitRecallIndex(ctx); err != nil {
		t.Fatalf("refresh recall index: %v", err)
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
	if _, ok := byUnit[505]; ok {
		t.Fatalf("unit 505 should not have inventory row without recall supply")
	}
}

func TestVideoFillCandidateReaderListsMasteredTargetBeforePopularFill(t *testing.T) {
	db := testDB(t)
	tx := fixture.BeginTestTx(t, db.Pool)
	ctx := context.Background()

	userID := "00000000-0000-0000-0000-000000000106"
	unitID := int64(601)
	masteredVideoID := "00000000-0000-0000-0000-000000000601"
	excludedVideoID := "00000000-0000-0000-0000-000000000602"
	popularVideoID := "00000000-0000-0000-0000-000000000603"
	seedBaseRefs(t, ctx, db, tx, userID, masteredVideoID, unitID)
	seedInventoryVideo(t, ctx, db, tx, masteredVideoID, unitID, 7, 0.24000, 0.82000)
	seedInventoryVideo(t, ctx, db, tx, excludedVideoID, unitID, 9, 0.32000, 0.90000)
	db.SeedVideo(t, popularVideoID)

	if _, err := tx.Exec(ctx, `
		insert into learning.user_unit_states (user_id, coarse_unit_id, is_target, target_priority, status, mastery_score)
		values ($1, $2, true, 1.0000, 'mastered', 1.0000)
	`, userID, unitID); err != nil {
		t.Fatalf("seed mastered user unit state: %v", err)
	}
	if _, err := tx.Exec(ctx, `
		insert into catalog.video_engagement_stats (video_id, view_count, like_count, favorite_count)
		values
			($1, 100, 10, 5),
			($2, 1000, 100, 50),
			($3, 900, 90, 45)
		on conflict (video_id) do update set
			view_count = excluded.view_count,
			like_count = excluded.like_count,
			favorite_count = excluded.favorite_count
	`, masteredVideoID, excludedVideoID, popularVideoID); err != nil {
		t.Fatalf("seed engagement stats: %v", err)
	}

	queries := recommendationsqlc.New(tx)
	if err := queries.RefreshVideoUnitRecallIndex(ctx); err != nil {
		t.Fatalf("refresh recall index: %v", err)
	}

	reader := repository.NewVideoFillCandidateReader(tx)
	masteredRows, err := reader.ListMasteredTargetFillCandidates(ctx, userID, []string{excludedVideoID}, 10)
	if err != nil {
		t.Fatalf("list mastered target fill candidates: %v", err)
	}
	if len(masteredRows) != 1 {
		t.Fatalf("expected 1 mastered target fill candidate, got %#v", masteredRows)
	}
	if masteredRows[0].VideoID != masteredVideoID {
		t.Fatalf("mastered fill video id = %q, want %q", masteredRows[0].VideoID, masteredVideoID)
	}
	if masteredRows[0].MatchedUnitCount != 1 {
		t.Fatalf("unexpected mastered fill candidate: %+v", masteredRows[0])
	}
	if masteredRows[0].ViewCount != 100 || masteredRows[0].LikeCount != 10 || masteredRows[0].FavoriteCount != 5 {
		t.Fatalf("unexpected mastered fill engagement counts: %+v", masteredRows[0])
	}

	popularRows, err := reader.ListPopularFillCandidates(ctx, userID, []string{excludedVideoID, masteredVideoID}, 10)
	if err != nil {
		t.Fatalf("list popular fill candidates: %v", err)
	}
	if len(popularRows) != 1 {
		t.Fatalf("expected 1 popular fill candidate, got %#v", popularRows)
	}
	if popularRows[0].VideoID != popularVideoID {
		t.Fatalf("popular fill video id = %q, want %q", popularRows[0].VideoID, popularVideoID)
	}
	if popularRows[0].MatchedUnitCount != 0 || popularRows[0].ViewCount != 900 {
		t.Fatalf("unexpected popular fill candidate: %+v", popularRows[0])
	}
}

func TestVideoFillCandidateReaderPopularFillIncludesVideosWithoutEngagementStats(t *testing.T) {
	db := testDB(t)
	tx := fixture.BeginTestTx(t, db.Pool)
	ctx := context.Background()

	userID := "00000000-0000-0000-0000-000000000107"
	videoWithStatsID := "00000000-0000-0000-0000-000000000611"
	videoWithoutStatsID := "00000000-0000-0000-0000-000000000612"
	inactiveVideoID := "00000000-0000-0000-0000-000000000613"
	privateVideoID := "00000000-0000-0000-0000-000000000614"
	futureVideoID := "00000000-0000-0000-0000-000000000615"
	seedUser(t, ctx, db, tx, userID)
	db.SeedVideo(t, videoWithStatsID)
	db.SeedVideo(t, videoWithoutStatsID)
	db.SeedVideo(t, inactiveVideoID)
	db.SeedVideo(t, privateVideoID)
	db.SeedVideo(t, futureVideoID)

	if _, err := tx.Exec(ctx, `
		update catalog.videos
		set status = 'inactive'
		where video_id = $1
	`, inactiveVideoID); err != nil {
		t.Fatalf("seed inactive video: %v", err)
	}
	if _, err := tx.Exec(ctx, `
		update catalog.videos
		set visibility_status = 'private'
		where video_id = $1
	`, privateVideoID); err != nil {
		t.Fatalf("seed private video: %v", err)
	}
	if _, err := tx.Exec(ctx, `
		update catalog.videos
		set publish_at = now() + interval '24 hours'
		where video_id = $1
	`, futureVideoID); err != nil {
		t.Fatalf("seed future video: %v", err)
	}
	if _, err := tx.Exec(ctx, `
		insert into catalog.video_engagement_stats (video_id, view_count, like_count, favorite_count)
		values ($1, 1000, 100, 50)
	`, videoWithStatsID); err != nil {
		t.Fatalf("seed engagement stats: %v", err)
	}

	reader := repository.NewVideoFillCandidateReader(tx)
	popularRows, err := reader.ListPopularFillCandidates(ctx, userID, nil, 10)
	if err != nil {
		t.Fatalf("list popular fill candidates: %v", err)
	}

	got := make([]string, 0, len(popularRows))
	for _, row := range popularRows {
		got = append(got, row.VideoID)
	}
	want := []string{videoWithStatsID, videoWithoutStatsID}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("popular fill video ids = %v, want %v", got, want)
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
			RunID:          runID,
			Rank:           1,
			VideoID:        videoID,
			Score:          0.91,
			PrimaryLane:    "exact_core",
			DominantRole:   model.LearningRoleHardReview,
			DominantUnitID: int64Ptr(301),
			ReasonCodes:    []string{"hard_review_covered"},
			LearningUnits:  []model.ExpectedLearningUnit{learningUnit(301, model.LearningRoleHardReview)},
		},
	}, userID, []model.FinalRecommendationItem{
		{
			VideoID:       videoID,
			LearningUnits: []model.ExpectedLearningUnit{learningUnit(301, model.LearningRoleHardReview)},
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
	if _, err := db.Exec(ctx, `insert into catalog.video_transcript_sentences (video_id, sentence_index, start_ms, end_ms, text, translation) values ($1, 1, 900, 1600, 'inventory sentence', '库存句子') on conflict do nothing`, videoID); err != nil {
		t.Fatalf("seed inventory transcript sentence: %v", err)
	}
	if _, err := db.Exec(ctx, `insert into catalog.video_semantic_spans (video_id, sentence_index, span_index, coarse_unit_id, start_ms, end_ms, surface_text, explanation, base_form, translation, dictionary, mapping_reason) values ($1, 1, 1, $2, 1000, 1500, 'inventory span', 'inventory explanation', 'inventory', '库存', 'inventory dictionary', 'inventory reason') on conflict do nothing`, videoID, unitID); err != nil {
		t.Fatalf("seed inventory semantic span: %v", err)
	}
	if _, err := db.Exec(ctx, `
			insert into catalog.video_unit_index (
				video_id, coarse_unit_id, mention_count, sentence_count, coverage_ms, coverage_ratio,
				sentence_indexes, best_evidence_sentence_index, best_evidence_span_index,
				best_evidence_start_ms, best_evidence_end_ms,
				best_evidence_scores, best_evidence_question_reject_reason, best_evidence_selection_reason,
				best_evidence_candidate_score, best_evidence_target_text
			) values ($1, $2, $3, 2, 4000, $4, '{1,2}', 1, 1, 1000, 1500, '{}'::jsonb, null, 'test fixture', 8.3500, 'inventory span')
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

func learningUnit(unitID int64, role model.LearningRole) model.ExpectedLearningUnit {
	return model.ExpectedLearningUnit{
		CoarseUnitID: unitID,
		Role:         role,
		IsPrimary:    model.IsCoreLearningRole(role),
	}
}
