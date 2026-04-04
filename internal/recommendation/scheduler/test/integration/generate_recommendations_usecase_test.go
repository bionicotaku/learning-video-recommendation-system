package integration

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"learning-video-recommendation-system/internal/recommendation/scheduler/application/command"
	"learning-video-recommendation-system/internal/recommendation/scheduler/application/usecase"
	"learning-video-recommendation-system/internal/recommendation/scheduler/domain/enum"
	"learning-video-recommendation-system/internal/recommendation/scheduler/domain/model"
	domainservice "learning-video-recommendation-system/internal/recommendation/scheduler/domain/service"
	infra "learning-video-recommendation-system/internal/recommendation/scheduler/infrastructure"
	repopkg "learning-video-recommendation-system/internal/recommendation/scheduler/infrastructure/persistence/repository"
	"learning-video-recommendation-system/internal/recommendation/scheduler/infrastructure/persistence/sqlcgen"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func TestGenerateLearningUnitRecommendationsUseCase(t *testing.T) {
	cfg := infra.LoadConfig()
	if cfg.DatabaseURL == "" {
		t.Skip("DATABASE_URL is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	pool, err := infra.NewDBPool(ctx, cfg)
	if err != nil {
		t.Fatalf("NewDBPool() error = %v", err)
	}
	defer pool.Close()

	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		t.Fatalf("BeginTx() error = %v", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	q := sqlcgen.New(tx)
	stateRepo := repopkg.NewUserUnitStateRepository(q)
	settingsRepo := repopkg.NewUserSchedulerSettingsRepository(q)
	runRepo := repopkg.NewSchedulerRunRepository(q)

	userID, err := loadExistingUserID(ctx, tx)
	if err != nil {
		t.Fatalf("loadExistingUserID() error = %v", err)
	}
	unitIDs, err := loadAvailableCoarseUnitIDs(ctx, tx, userID, 4)
	if err != nil {
		t.Fatalf("loadAvailableCoarseUnitIDs() error = %v", err)
	}

	now := time.Date(2026, 4, 8, 11, 0, 0, 0, time.UTC)
	badQuality := 2

	if err := stateRepo.BatchUpsert(ctx, []*model.UserUnitState{
		{
			UserID:            userID,
			CoarseUnitID:      unitIDs[0],
			IsTarget:          true,
			Status:            enum.UnitStatusLearning,
			TargetPriority:    0.9,
			ProgressPercent:   20,
			MasteryScore:      0.2,
			EaseFactor:        2.5,
			Repetition:        1,
			IntervalDays:      1,
			NextReviewAt:      ptrTime(now.Add(-2 * time.Hour)),
			LastRecommendedAt: ptrTime(now.Add(-7 * time.Hour)),
			CreatedAt:         now,
			UpdatedAt:         now,
		},
		{
			UserID:            userID,
			CoarseUnitID:      unitIDs[1],
			IsTarget:          true,
			Status:            enum.UnitStatusReviewing,
			TargetPriority:    0.8,
			ProgressPercent:   50,
			MasteryScore:      0.4,
			EaseFactor:        2.5,
			Repetition:        2,
			IntervalDays:      3,
			NextReviewAt:      ptrTime(now.Add(-24 * time.Hour)),
			LastQuality:       &badQuality,
			LastRecommendedAt: ptrTime(now.Add(-8 * time.Hour)),
			CreatedAt:         now,
			UpdatedAt:         now,
		},
		{
			UserID:            userID,
			CoarseUnitID:      unitIDs[2],
			IsTarget:          true,
			Status:            enum.UnitStatusReviewing,
			TargetPriority:    0.6,
			ProgressPercent:   60,
			MasteryScore:      0.6,
			EaseFactor:        2.5,
			Repetition:        3,
			IntervalDays:      6,
			NextReviewAt:      ptrTime(now.Add(-6 * time.Hour)),
			LastRecommendedAt: ptrTime(now.Add(-10 * time.Hour)),
			CreatedAt:         now,
			UpdatedAt:         now,
		},
		{
			UserID:          userID,
			CoarseUnitID:    unitIDs[3],
			IsTarget:        true,
			Status:          enum.UnitStatusNew,
			TargetPriority:  0.7,
			ProgressPercent: 0,
			MasteryScore:    0,
			EaseFactor:      2.5,
			CreatedAt:       now,
			UpdatedAt:       now,
		},
	}); err != nil {
		t.Fatalf("BatchUpsert() error = %v", err)
	}

	uc := usecase.NewGenerateLearningUnitRecommendationsUseCase(
		stateRepo,
		settingsRepo,
		runRepo,
		domainservice.NewBacklogCalculator(),
		domainservice.NewQuotaAllocator(),
		domainservice.NewReviewScorer(),
		domainservice.NewNewScorer(),
		domainservice.NewPriorityZeroExtractor(),
		domainservice.NewRecommendationAssembler(),
	)

	result, err := uc.Execute(ctx, command.GenerateRecommendationsCommand{
		UserID:         userID,
		RequestedLimit: 4,
		Now:            now,
		RequestContext: map[string]any{"persist_snapshot": true},
	})
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
	if err := tx.QueryRow(ctx, `
		select due_review_count, selected_review_count, selected_new_count, context
		from learning.scheduler_runs
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

	rows, err := tx.Query(ctx, `
		select coarse_unit_id, recommend_type, rank, reason_codes
		from learning.scheduler_run_items
		where run_id = $1
		order by rank asc
	`, batch.RunID)
	if err != nil {
		t.Fatalf("Query(scheduler_run_items) error = %v", err)
	}
	defer rows.Close()

	type runItemRow struct {
		coarseUnitID  int64
		recommendType string
		rank          int
		reasonCodes   []string
	}
	runItems := make([]runItemRow, 0, len(batch.Items))
	for rows.Next() {
		var item runItemRow
		if err := rows.Scan(&item.coarseUnitID, &item.recommendType, &item.rank, &item.reasonCodes); err != nil {
			t.Fatalf("rows.Scan() error = %v", err)
		}
		runItems = append(runItems, item)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows.Err() error = %v", err)
	}
	if len(runItems) != len(batch.Items) {
		t.Fatalf("len(runItems) = %d, want %d", len(runItems), len(batch.Items))
	}
	for index, item := range runItems {
		want := batch.Items[index]
		if item.coarseUnitID != want.CoarseUnitID {
			t.Fatalf("run item coarse_unit_id = %d, want %d", item.coarseUnitID, want.CoarseUnitID)
		}
		if item.recommendType != string(want.RecommendType) {
			t.Fatalf("run item recommend_type = %q, want %q", item.recommendType, want.RecommendType)
		}
		if item.rank != want.Rank {
			t.Fatalf("run item rank = %d, want %d", item.rank, want.Rank)
		}
		if len(item.reasonCodes) == 0 {
			t.Fatalf("run item %d reason_codes is empty", item.coarseUnitID)
		}
	}
}
