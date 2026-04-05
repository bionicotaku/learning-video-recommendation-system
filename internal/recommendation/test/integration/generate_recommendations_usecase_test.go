package integration

import (
	"encoding/json"
	"testing"
	"time"

	"learning-video-recommendation-system/internal/recommendation/domain/enum"
	"learning-video-recommendation-system/internal/recommendation/infrastructure/persistence/sqlcgen"

	"github.com/google/uuid"
)

func TestGenerateLearningUnitRecommendationsUseCase(t *testing.T) {
	ctx, pool := newTestPool(t)

	userID, err := createTestUser(ctx, pool)
	if err != nil {
		t.Fatalf("createTestUser() error = %v", err)
	}
	unitIDs, err := createTestCoarseUnits(ctx, pool, 4)
	if err != nil {
		t.Fatalf("createTestCoarseUnits() error = %v", err)
	}
	t.Cleanup(func() {
		cleanupTestData(ctx, t, pool, userID, unitIDs)
	})

	now := time.Date(2026, 4, 8, 11, 0, 0, 0, time.UTC)
	nilTime := any(nil)
	nilText := any(nil)
	badQuality := int16(2)

	if err := insertState(ctx, pool,
		userID, unitIDs[0], true, "lesson", "l-1", 0.9, "learning", 20.0, 0.2,
		nilTime, nilTime, nilTime, 0, 0, 0, 0, 0, 0, 0, nil, []int16{}, []bool{}, 1, 1.0, 2.5, now.Add(-2*time.Hour), nilText, now, now,
	); err != nil {
		t.Fatalf("insertState(unit0) error = %v", err)
	}
	if err := insertState(ctx, pool,
		userID, unitIDs[1], true, "lesson", "l-2", 0.8, "reviewing", 50.0, 0.4,
		nilTime, nilTime, nilTime, 0, 0, 0, 0, 0, 0, 0, &badQuality, []int16{}, []bool{}, 2, 3.0, 2.5, now.Add(-24*time.Hour), nilText, now, now,
	); err != nil {
		t.Fatalf("insertState(unit1) error = %v", err)
	}
	if err := insertState(ctx, pool,
		userID, unitIDs[2], true, "lesson", "l-3", 0.6, "reviewing", 60.0, 0.6,
		nilTime, nilTime, nilTime, 0, 0, 0, 0, 0, 0, 0, nil, []int16{}, []bool{}, 3, 6.0, 2.5, now.Add(-6*time.Hour), nilText, now, now,
	); err != nil {
		t.Fatalf("insertState(unit2) error = %v", err)
	}
	if err := insertState(ctx, pool,
		userID, unitIDs[3], true, "lesson", "l-4", 0.7, "new", 0.0, 0.0,
		nilTime, nilTime, nilTime, 0, 0, 0, 0, 0, 0, 0, nil, []int16{}, []bool{}, 0, 0.0, 2.5, nilTime, nilText, now, now,
	); err != nil {
		t.Fatalf("insertState(unit3) error = %v", err)
	}

	if err := insertServingState(ctx, pool, userID, unitIDs[0], now.Add(-7*time.Hour)); err != nil {
		t.Fatalf("insertServingState(unit0) error = %v", err)
	}
	if err := insertServingState(ctx, pool, userID, unitIDs[1], now.Add(-8*time.Hour)); err != nil {
		t.Fatalf("insertServingState(unit1) error = %v", err)
	}
	if err := insertServingState(ctx, pool, userID, unitIDs[2], now.Add(-10*time.Hour)); err != nil {
		t.Fatalf("insertServingState(unit2) error = %v", err)
	}

	q := sqlcgen.New(pool)
	uc := newGenerateUseCase(pool, q)

	var beforeStateCount int
	if err := pool.QueryRow(ctx, `select count(*) from learning.user_unit_states where user_id = $1`, userID).Scan(&beforeStateCount); err != nil {
		t.Fatalf("count learning.user_unit_states before error = %v", err)
	}
	var beforeEventCount int
	if err := pool.QueryRow(ctx, `select count(*) from learning.unit_learning_events where user_id = $1`, userID).Scan(&beforeEventCount); err != nil {
		t.Fatalf("count learning.unit_learning_events before error = %v", err)
	}

	result, err := uc.Execute(ctx, generateCmd(userID, 4, now))
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	batch := result.Batch
	if batch.UserID != userID {
		t.Fatalf("Batch.UserID = %v, want %v", batch.UserID, userID)
	}
	if batch.RunID == uuid.Nil {
		t.Fatal("Batch.RunID = nil, want generated UUID")
	}
	if len(batch.Items) != 3 {
		t.Fatalf("len(Batch.Items) = %d, want 3", len(batch.Items))
	}
	if batch.Items[0].CoarseUnitID != unitIDs[0] {
		t.Fatalf("Batch.Items[0].CoarseUnitID = %d, want %d", batch.Items[0].CoarseUnitID, unitIDs[0])
	}
	if batch.Items[1].CoarseUnitID != unitIDs[1] {
		t.Fatalf("Batch.Items[1].CoarseUnitID = %d, want %d", batch.Items[1].CoarseUnitID, unitIDs[1])
	}
	if batch.Items[2].RecommendType != enum.RecommendTypeNew {
		t.Fatalf("Batch.Items[2].RecommendType = %q, want %q", batch.Items[2].RecommendType, enum.RecommendTypeNew)
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

	runCount, err := q.CountSchedulerRuns(ctx)
	if err != nil {
		t.Fatalf("CountSchedulerRuns() error = %v", err)
	}
	if runCount != 1 {
		t.Fatalf("CountSchedulerRuns() = %d, want 1", runCount)
	}

	var (
		dueReviewCount      int
		selectedReviewCount int
		selectedNewCount    int
		contextPayload      []byte
	)
	if err := pool.QueryRow(ctx, `
		select due_review_count, selected_review_count, selected_new_count, context
		from recommendation.scheduler_runs
		where run_id = $1
	`, batch.RunID).Scan(&dueReviewCount, &selectedReviewCount, &selectedNewCount, &contextPayload); err != nil {
		t.Fatalf("QueryRow(scheduler_runs) error = %v", err)
	}
	if dueReviewCount != 3 {
		t.Fatalf("due_review_count = %d, want 3", dueReviewCount)
	}
	if selectedReviewCount != 2 {
		t.Fatalf("selected_review_count = %d, want 2", selectedReviewCount)
	}
	if selectedNewCount != 1 {
		t.Fatalf("selected_new_count = %d, want 1", selectedNewCount)
	}

	var contextMap map[string]any
	if err := json.Unmarshal(contextPayload, &contextMap); err != nil {
		t.Fatalf("json.Unmarshal(context) error = %v", err)
	}
	if contextMap["backlog_protection"] != false {
		t.Fatalf("context[backlog_protection] = %v, want false", contextMap["backlog_protection"])
	}

	rows, err := pool.Query(ctx, `
		select coarse_unit_id, recommend_type, rank, reason_codes
		from recommendation.scheduler_run_items
		where run_id = $1
		order by rank asc
	`, batch.RunID)
	if err != nil {
		t.Fatalf("Query(scheduler_run_items) error = %v", err)
	}
	defer rows.Close()

	type runItemRow struct {
		CoarseUnitID  int64
		RecommendType string
		Rank          int
		ReasonCodes   []string
	}
	var items []runItemRow
	for rows.Next() {
		var item runItemRow
		if err := rows.Scan(&item.CoarseUnitID, &item.RecommendType, &item.Rank, &item.ReasonCodes); err != nil {
			t.Fatalf("rows.Scan() error = %v", err)
		}
		items = append(items, item)
	}
	if len(items) != 3 {
		t.Fatalf("len(run_items) = %d, want 3", len(items))
	}

	var servingCount int
	if err := pool.QueryRow(ctx, `select count(*) from recommendation.user_unit_serving_states where user_id = $1`, userID).Scan(&servingCount); err != nil {
		t.Fatalf("count serving states error = %v", err)
	}
	if servingCount != 4 {
		t.Fatalf("serving state count = %d, want 4", servingCount)
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
	if touchedCount != 3 {
		t.Fatalf("touched serving state count = %d, want 3", touchedCount)
	}

	var untouchedRecommendedAt time.Time
	if err := pool.QueryRow(ctx, `
		select last_recommended_at
		from recommendation.user_unit_serving_states
		where user_id = $1
		  and coarse_unit_id = $2
	`, userID, unitIDs[2]).Scan(&untouchedRecommendedAt); err != nil {
		t.Fatalf("QueryRow(untouched serving state) error = %v", err)
	}
	if !untouchedRecommendedAt.Equal(now.Add(-10 * time.Hour)) {
		t.Fatalf("untouched last_recommended_at = %v, want %v", untouchedRecommendedAt, now.Add(-10*time.Hour))
	}
}
