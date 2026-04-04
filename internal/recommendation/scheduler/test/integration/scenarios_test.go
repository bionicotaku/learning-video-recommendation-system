package integration

import (
	"context"
	"errors"
	"fmt"
	"math"
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
	txtx "learning-video-recommendation-system/internal/recommendation/scheduler/infrastructure/persistence/tx"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func TestIntegrationScenarioANewUserInitialLearning(t *testing.T) {
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

	userID, err := createTestUserIDFromPool(ctx, pool)
	if err != nil {
		t.Fatalf("createTestUserIDFromPool() error = %v", err)
	}
	defer cleanupTestUser(ctx, t, pool, userID)
	unitIDs, err := loadAvailableCoarseUnitIDsFromPool(ctx, pool, userID, 3)
	if err != nil {
		t.Fatalf("loadAvailableCoarseUnitIDsFromPool() error = %v", err)
	}

	baseQuerier := sqlcgen.New(pool)
	stateRepo := repopkg.NewUserUnitStateRepository(baseQuerier)
	eventRepo := repopkg.NewUnitLearningEventRepository(baseQuerier)
	now := time.Date(2026, 4, 10, 9, 0, 0, 0, time.UTC)

	if err := stateRepo.BatchUpsert(ctx, []*model.UserUnitState{
		{
			UserID:          userID,
			CoarseUnitID:    unitIDs[0],
			IsTarget:        true,
			Status:          enum.UnitStatusNew,
			TargetPriority:  0.9,
			EaseFactor:      2.5,
			ProgressPercent: 0,
			MasteryScore:    0,
			CreatedAt:       now,
			UpdatedAt:       now,
		},
		{
			UserID:          userID,
			CoarseUnitID:    unitIDs[1],
			IsTarget:        true,
			Status:          enum.UnitStatusNew,
			TargetPriority:  0.7,
			EaseFactor:      2.5,
			ProgressPercent: 0,
			MasteryScore:    0,
			CreatedAt:       now,
			UpdatedAt:       now,
		},
		{
			UserID:          userID,
			CoarseUnitID:    unitIDs[2],
			IsTarget:        true,
			Status:          enum.UnitStatusNew,
			TargetPriority:  0.5,
			EaseFactor:      2.5,
			ProgressPercent: 0,
			MasteryScore:    0,
			CreatedAt:       now,
			UpdatedAt:       now,
		},
	}); err != nil {
		t.Fatalf("BatchUpsert() error = %v", err)
	}

	generateUC := newGenerateRecommendationsUseCase(baseQuerier)
	generated, err := generateUC.Execute(ctx, command.GenerateRecommendationsCommand{
		UserID:         userID,
		RequestedLimit: 3,
		Now:            now,
	})
	if err != nil {
		t.Fatalf("Generate Execute() error = %v", err)
	}
	if len(generated.Batch.Items) != 3 {
		t.Fatalf("len(Batch.Items) = %d, want 3", len(generated.Batch.Items))
	}
	for _, item := range generated.Batch.Items {
		if item.RecommendType != enum.RecommendTypeNew {
			t.Fatalf("RecommendType = %q, want %q", item.RecommendType, enum.RecommendTypeNew)
		}
	}

	recordUC := usecase.NewRecordLearningEventsAndUpdateStateUseCase(
		txtx.NewPGXTxManager(pool),
		stateRepo,
		eventRepo,
		domainservice.NewStateUpdater(),
	)
	correct := true
	quality := 4
	_, err = recordUC.Execute(ctx, command.RecordLearningEventsCommand{
		UserID: userID,
		Events: []command.LearningEventInput{
			{
				CoarseUnitID: generated.Batch.Items[0].CoarseUnitID,
				EventType:    enum.EventTypeNewLearn,
				SourceType:   "integration_test",
				SourceRefID:  "scenario-a-new",
				IsCorrect:    &correct,
				Quality:      &quality,
				OccurredAt:   now,
			},
		},
		IdempotencyKey: "scenario-a",
	})
	if err != nil {
		t.Fatalf("Record Execute() error = %v", err)
	}

	state, err := stateRepo.GetByUserAndUnit(ctx, userID, generated.Batch.Items[0].CoarseUnitID)
	if err != nil {
		t.Fatalf("GetByUserAndUnit() error = %v", err)
	}
	if state == nil {
		t.Fatal("state = nil, want value")
	}
	if state.Status != enum.UnitStatusLearning {
		t.Fatalf("state.Status = %q, want %q", state.Status, enum.UnitStatusLearning)
	}
	if state.StrongEventCount != 1 {
		t.Fatalf("state.StrongEventCount = %d, want 1", state.StrongEventCount)
	}
	if state.IntervalDays != 1 {
		t.Fatalf("state.IntervalDays = %v, want 1", state.IntervalDays)
	}
}

func TestIntegrationScenarioBNormalReviewProgressionToMastered(t *testing.T) {
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
	recordUC := usecase.NewRecordLearningEventsAndUpdateStateUseCase(
		txtx.NewPGXTxManager(pool),
		repopkg.NewUserUnitStateRepository(baseQuerier),
		repopkg.NewUnitLearningEventRepository(baseQuerier),
		domainservice.NewStateUpdater(),
	)
	stateRepo := repopkg.NewUserUnitStateRepository(baseQuerier)

	correct := true
	quality := 5
	start := time.Date(2026, 4, 11, 8, 0, 0, 0, time.UTC)

	steps := []struct {
		eventType      enum.EventType
		occurredAt     time.Time
		wantStatus     enum.UnitStatus
		wantRepetition int
		wantInterval   float64
		wantEaseFactor float64
	}{
		{eventType: enum.EventTypeNewLearn, occurredAt: start, wantStatus: enum.UnitStatusLearning, wantRepetition: 1, wantInterval: 1, wantEaseFactor: 2.6},
		{eventType: enum.EventTypeReview, occurredAt: start.Add(24 * time.Hour), wantStatus: enum.UnitStatusReviewing, wantRepetition: 2, wantInterval: 3, wantEaseFactor: 2.7},
		{eventType: enum.EventTypeReview, occurredAt: start.Add(4 * 24 * time.Hour), wantStatus: enum.UnitStatusReviewing, wantRepetition: 3, wantInterval: 6, wantEaseFactor: 2.8},
		{eventType: enum.EventTypeReview, occurredAt: start.Add(10 * 24 * time.Hour), wantStatus: enum.UnitStatusReviewing, wantRepetition: 4, wantInterval: 17, wantEaseFactor: 2.9},
		{eventType: enum.EventTypeReview, occurredAt: start.Add(27 * 24 * time.Hour), wantStatus: enum.UnitStatusMastered, wantRepetition: 5, wantInterval: 49, wantEaseFactor: 3.0},
	}

	for index, step := range steps {
		_, err := recordUC.Execute(ctx, command.RecordLearningEventsCommand{
			UserID: userID,
			Events: []command.LearningEventInput{
				{
					CoarseUnitID: unitID,
					EventType:    step.eventType,
					SourceType:   "integration_test",
					SourceRefID:  fmt.Sprintf("scenario-b-step-%d", index),
					IsCorrect:    &correct,
					Quality:      &quality,
					OccurredAt:   step.occurredAt,
				},
			},
			IdempotencyKey: fmt.Sprintf("scenario-b-%d", index),
		})
		if err != nil {
			t.Fatalf("Record Execute() step %d error = %v", index, err)
		}

		state, err := stateRepo.GetByUserAndUnit(ctx, userID, unitID)
		if err != nil {
			t.Fatalf("GetByUserAndUnit() step %d error = %v", index, err)
		}
		if state == nil {
			t.Fatalf("state step %d = nil, want value", index)
		}
		if state.Status != step.wantStatus {
			t.Fatalf("state.Status step %d = %q, want %q", index, state.Status, step.wantStatus)
		}
		if state.Repetition != step.wantRepetition {
			t.Fatalf("state.Repetition step %d = %d, want %d", index, state.Repetition, step.wantRepetition)
		}
		if math.Abs(state.IntervalDays-step.wantInterval) > 1e-9 {
			t.Fatalf("state.IntervalDays step %d = %v, want %v", index, state.IntervalDays, step.wantInterval)
		}
		if math.Abs(state.EaseFactor-step.wantEaseFactor) > 1e-9 {
			t.Fatalf("state.EaseFactor step %d = %v, want %v", index, state.EaseFactor, step.wantEaseFactor)
		}
	}
}

func TestIntegrationScenarioCFailureRegressionResetsIntervalWithoutChangingEF(t *testing.T) {
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
	recordUC := usecase.NewRecordLearningEventsAndUpdateStateUseCase(
		txtx.NewPGXTxManager(pool),
		repopkg.NewUserUnitStateRepository(baseQuerier),
		repopkg.NewUnitLearningEventRepository(baseQuerier),
		domainservice.NewStateUpdater(),
	)
	stateRepo := repopkg.NewUserUnitStateRepository(baseQuerier)

	correct := true
	quality := 5
	start := time.Date(2026, 4, 12, 8, 0, 0, 0, time.UTC)
	for index, input := range []command.LearningEventInput{
		{CoarseUnitID: unitID, EventType: enum.EventTypeNewLearn, SourceType: "integration_test", SourceRefID: "scenario-c-new", IsCorrect: &correct, Quality: &quality, OccurredAt: start},
		{CoarseUnitID: unitID, EventType: enum.EventTypeReview, SourceType: "integration_test", SourceRefID: "scenario-c-review-1", IsCorrect: &correct, Quality: &quality, OccurredAt: start.Add(24 * time.Hour)},
		{CoarseUnitID: unitID, EventType: enum.EventTypeReview, SourceType: "integration_test", SourceRefID: "scenario-c-review-2", IsCorrect: &correct, Quality: &quality, OccurredAt: start.Add(4 * 24 * time.Hour)},
	} {
		_, err := recordUC.Execute(ctx, command.RecordLearningEventsCommand{
			UserID:         userID,
			Events:         []command.LearningEventInput{input},
			IdempotencyKey: fmt.Sprintf("scenario-c-seed-%d", index),
		})
		if err != nil {
			t.Fatalf("seed Execute() step %d error = %v", index, err)
		}
	}

	beforeFailure, err := stateRepo.GetByUserAndUnit(ctx, userID, unitID)
	if err != nil {
		t.Fatalf("GetByUserAndUnit(beforeFailure) error = %v", err)
	}
	if beforeFailure == nil {
		t.Fatal("beforeFailure = nil, want value")
	}

	wrong := false
	failureQuality := 2
	failureAt := start.Add(10 * 24 * time.Hour)
	_, err = recordUC.Execute(ctx, command.RecordLearningEventsCommand{
		UserID: userID,
		Events: []command.LearningEventInput{
			{
				CoarseUnitID: unitID,
				EventType:    enum.EventTypeReview,
				SourceType:   "integration_test",
				SourceRefID:  "scenario-c-failure",
				IsCorrect:    &wrong,
				Quality:      &failureQuality,
				OccurredAt:   failureAt,
			},
		},
		IdempotencyKey: "scenario-c-failure",
	})
	if err != nil {
		t.Fatalf("failure Execute() error = %v", err)
	}

	afterFailure, err := stateRepo.GetByUserAndUnit(ctx, userID, unitID)
	if err != nil {
		t.Fatalf("GetByUserAndUnit(afterFailure) error = %v", err)
	}
	if afterFailure == nil {
		t.Fatal("afterFailure = nil, want value")
	}
	if afterFailure.Status != enum.UnitStatusReviewing {
		t.Fatalf("state.Status = %q, want %q", afterFailure.Status, enum.UnitStatusReviewing)
	}
	if afterFailure.Repetition != 0 {
		t.Fatalf("state.Repetition = %d, want 0", afterFailure.Repetition)
	}
	if afterFailure.IntervalDays != 1 {
		t.Fatalf("state.IntervalDays = %v, want 1", afterFailure.IntervalDays)
	}
	if math.Abs(afterFailure.EaseFactor-beforeFailure.EaseFactor) > 1e-9 {
		t.Fatalf("state.EaseFactor = %v, want unchanged %v", afterFailure.EaseFactor, beforeFailure.EaseFactor)
	}
	if afterFailure.NextReviewAt == nil {
		t.Fatal("state.NextReviewAt = nil, want value")
	}
	wantNext := failureAt.Add(24 * time.Hour)
	if !afterFailure.NextReviewAt.Equal(wantNext) {
		t.Fatalf("state.NextReviewAt = %v, want %v", afterFailure.NextReviewAt, wantNext)
	}
}

func TestIntegrationScenarioDBacklogProtectionSuppressesNewUnits(t *testing.T) {
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
	userID, err := createTestUserID(ctx, tx)
	if err != nil {
		t.Fatalf("createTestUserID() error = %v", err)
	}
	unitIDs, err := loadAvailableCoarseUnitIDs(ctx, tx, userID, 6)
	if err != nil {
		t.Fatalf("loadAvailableCoarseUnitIDs() error = %v", err)
	}

	if _, err := tx.Exec(ctx, `
		insert into learning.user_scheduler_settings (
			user_id,
			session_default_limit,
			daily_new_unit_quota,
			daily_review_soft_limit,
			daily_review_hard_limit,
			timezone
		) values ($1, 10, 8, 2, 3, 'UTC')
		on conflict (user_id) do update
		set session_default_limit = excluded.session_default_limit,
		    daily_new_unit_quota = excluded.daily_new_unit_quota,
		    daily_review_soft_limit = excluded.daily_review_soft_limit,
		    daily_review_hard_limit = excluded.daily_review_hard_limit,
		    timezone = excluded.timezone
	`, userID); err != nil {
		t.Fatalf("insert user_scheduler_settings error = %v", err)
	}

	now := time.Date(2026, 4, 13, 10, 0, 0, 0, time.UTC)
	states := make([]*model.UserUnitState, 0, 6)
	for _, unitID := range unitIDs[:4] {
		states = append(states, &model.UserUnitState{
			UserID:          userID,
			CoarseUnitID:    unitID,
			IsTarget:        true,
			Status:          enum.UnitStatusReviewing,
			TargetPriority:  0.9,
			ProgressPercent: 60,
			MasteryScore:    0.5,
			EaseFactor:      2.5,
			Repetition:      3,
			IntervalDays:    6,
			NextReviewAt:    ptrTime(now.Add(-2 * time.Hour)),
			CreatedAt:       now,
			UpdatedAt:       now,
		})
	}
	for _, unitID := range unitIDs[4:] {
		states = append(states, &model.UserUnitState{
			UserID:          userID,
			CoarseUnitID:    unitID,
			IsTarget:        true,
			Status:          enum.UnitStatusNew,
			TargetPriority:  0.8,
			ProgressPercent: 0,
			MasteryScore:    0,
			EaseFactor:      2.5,
			CreatedAt:       now,
			UpdatedAt:       now,
		})
	}
	if err := stateRepo.BatchUpsert(ctx, states); err != nil {
		t.Fatalf("BatchUpsert() error = %v", err)
	}

	uc := newGenerateRecommendationsUseCase(q)
	result, err := uc.Execute(ctx, command.GenerateRecommendationsCommand{
		UserID:         userID,
		RequestedLimit: 10,
		Now:            now,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !result.Batch.BacklogProtection {
		t.Fatal("Batch.BacklogProtection = false, want true")
	}
	if result.Batch.NewQuota != 0 {
		t.Fatalf("Batch.NewQuota = %d, want 0", result.Batch.NewQuota)
	}
	if len(result.Batch.Items) != 3 {
		t.Fatalf("len(Batch.Items) = %d, want 3", len(result.Batch.Items))
	}
	for _, item := range result.Batch.Items {
		if item.RecommendType != enum.RecommendTypeReview {
			t.Fatalf("RecommendType = %q, want all review items", item.RecommendType)
		}
	}
}

func TestIntegrationScenarioEUnifiedKindsScheduling(t *testing.T) {
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
	userID, err := createTestUserID(ctx, tx)
	if err != nil {
		t.Fatalf("createTestUserID() error = %v", err)
	}
	kindUnitIDs, err := loadAvailableCoarseUnitIDByKind(ctx, tx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			t.Skip("word/phrase/grammar units are not all available in this database")
		}
		t.Fatalf("loadAvailableCoarseUnitIDByKind() error = %v", err)
	}

	now := time.Date(2026, 4, 14, 10, 0, 0, 0, time.UTC)
	if err := stateRepo.BatchUpsert(ctx, []*model.UserUnitState{
		{
			UserID:          userID,
			CoarseUnitID:    kindUnitIDs[enum.UnitKindWord],
			IsTarget:        true,
			Status:          enum.UnitStatusReviewing,
			TargetPriority:  0.9,
			ProgressPercent: 55,
			MasteryScore:    0.4,
			EaseFactor:      2.5,
			Repetition:      2,
			IntervalDays:    3,
			NextReviewAt:    ptrTime(now.Add(-2 * time.Hour)),
			CreatedAt:       now,
			UpdatedAt:       now,
		},
		{
			UserID:          userID,
			CoarseUnitID:    kindUnitIDs[enum.UnitKindPhrase],
			IsTarget:        true,
			Status:          enum.UnitStatusReviewing,
			TargetPriority:  0.8,
			ProgressPercent: 40,
			MasteryScore:    0.3,
			EaseFactor:      2.5,
			Repetition:      1,
			IntervalDays:    1,
			NextReviewAt:    ptrTime(now.Add(-1 * time.Hour)),
			CreatedAt:       now,
			UpdatedAt:       now,
		},
		{
			UserID:          userID,
			CoarseUnitID:    kindUnitIDs[enum.UnitKindGrammar],
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

	uc := newGenerateRecommendationsUseCase(q)
	result, err := uc.Execute(ctx, command.GenerateRecommendationsCommand{
		UserID:         userID,
		RequestedLimit: 3,
		Now:            now,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(result.Batch.Items) != 3 {
		t.Fatalf("len(Batch.Items) = %d, want 3", len(result.Batch.Items))
	}

	kinds := make(map[enum.UnitKind]struct{}, 3)
	for _, item := range result.Batch.Items {
		kinds[item.Kind] = struct{}{}
	}
	for _, kind := range []enum.UnitKind{enum.UnitKindWord, enum.UnitKindPhrase, enum.UnitKindGrammar} {
		if _, ok := kinds[kind]; !ok {
			t.Fatalf("kind %q not found in recommendation batch", kind)
		}
	}
}

func newGenerateRecommendationsUseCase(querier sqlcgen.Querier) usecase.GenerateLearningUnitRecommendationsUseCase {
	return usecase.NewGenerateLearningUnitRecommendationsUseCase(
		repopkg.NewUserUnitStateRepository(querier),
		repopkg.NewUserSchedulerSettingsRepository(querier),
		repopkg.NewSchedulerRunRepository(querier),
		domainservice.NewBacklogCalculator(),
		domainservice.NewQuotaAllocator(),
		domainservice.NewReviewScorer(),
		domainservice.NewNewScorer(),
		domainservice.NewPriorityZeroExtractor(),
		domainservice.NewRecommendationAssembler(),
	)
}

func loadAvailableCoarseUnitIDByKind(ctx context.Context, tx pgx.Tx, userID uuid.UUID) (map[enum.UnitKind]int64, error) {
	result := make(map[enum.UnitKind]int64, 3)
	for _, kind := range []enum.UnitKind{enum.UnitKindWord, enum.UnitKindPhrase, enum.UnitKindGrammar} {
		var unitID int64
		err := tx.QueryRow(ctx, `
			select c.id
			from semantic.coarse_unit c
			left join learning.user_unit_states s
			  on s.coarse_unit_id = c.id
			 and s.user_id = $1
			left join learning.unit_learning_events e
			  on e.coarse_unit_id = c.id
			 and e.user_id = $1
			where s.coarse_unit_id is null
			  and e.coarse_unit_id is null
			  and c.kind = $2
			order by c.id
			limit 1
		`, userID, string(kind)).Scan(&unitID)
		if err != nil {
			return nil, err
		}
		result[kind] = unitID
	}

	return result, nil
}
