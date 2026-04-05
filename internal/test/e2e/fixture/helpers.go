package fixture

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	lecommand "learning-video-recommendation-system/internal/learningengine/application/command"
	leapprepo "learning-video-recommendation-system/internal/learningengine/application/repository"
	leservice "learning-video-recommendation-system/internal/learningengine/application/service"
	leusecase "learning-video-recommendation-system/internal/learningengine/application/usecase"
	leaggregate "learning-video-recommendation-system/internal/learningengine/domain/aggregate"
	leenum "learning-video-recommendation-system/internal/learningengine/domain/enum"
	lemodel "learning-video-recommendation-system/internal/learningengine/domain/model"
	lepolicy "learning-video-recommendation-system/internal/learningengine/domain/policy"
	lerepo "learning-video-recommendation-system/internal/learningengine/infrastructure/persistence/repository"
	lesqlc "learning-video-recommendation-system/internal/learningengine/infrastructure/persistence/sqlcgen"
	letx "learning-video-recommendation-system/internal/learningengine/infrastructure/persistence/tx"
	reccommand "learning-video-recommendation-system/internal/recommendation/scheduler/application/command"
	recusecase "learning-video-recommendation-system/internal/recommendation/scheduler/application/usecase"
	recservice "learning-video-recommendation-system/internal/recommendation/scheduler/domain/service"
	recrepo "learning-video-recommendation-system/internal/recommendation/scheduler/infrastructure/persistence/repository"
	recsqlc "learning-video-recommendation-system/internal/recommendation/scheduler/infrastructure/persistence/sqlcgen"
	rectx "learning-video-recommendation-system/internal/recommendation/scheduler/infrastructure/persistence/tx"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func NewTestPool(t *testing.T) (context.Context, *pgxpool.Pool) {
	t.Helper()

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	t.Cleanup(cancel)

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("pgxpool.New() error = %v", err)
	}
	t.Cleanup(pool.Close)

	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("pool.Ping() error = %v", err)
	}

	return ctx, pool
}

func NewRecordEventsUseCase(pool *pgxpool.Pool) leusecase.RecordLearningEventsUseCase {
	querier := lesqlc.New(pool)
	return leusecase.NewRecordLearningEventsUseCase(
		letx.NewPGXTxManager(pool),
		lerepo.NewUserUnitStateRepository(querier),
		lerepo.NewUnitLearningEventRepository(querier),
		leaggregate.NewUserUnitReducer(),
	)
}

func NewReplayUseCase(pool *pgxpool.Pool) leusecase.ReplayUserStatesUseCase {
	querier := lesqlc.New(pool)
	return leusecase.NewReplayUserStatesUseCase(
		letx.NewPGXTxManager(pool),
		lerepo.NewUserUnitStateRepository(querier),
		lerepo.NewUnitLearningEventRepository(querier),
		leservice.NewUserStateRebuilder(leaggregate.NewUserUnitReducer(), lepolicy.DefaultLearningPolicy()),
	)
}

func NewGenerateUseCase(pool *pgxpool.Pool) recusecase.GenerateLearningUnitRecommendationsUseCase {
	querier := recsqlc.New(pool)
	return recusecase.NewGenerateLearningUnitRecommendationsUseCase(
		rectx.NewPGXTxManager(pool),
		recrepo.NewLearningStateSnapshotReadRepository(querier),
		recrepo.NewUserUnitServingStateRepository(querier),
		recrepo.NewSchedulerRunRepository(querier),
		recservice.NewBacklogCalculator(),
		recservice.NewQuotaAllocator(),
		recservice.NewReviewScorer(),
		recservice.NewNewScorer(),
		recservice.NewPriorityZeroExtractor(),
		recservice.NewRecommendationAssembler(),
	)
}

func NewStateRepository(pool *pgxpool.Pool) leapprepo.UserUnitStateRepository {
	return lerepo.NewUserUnitStateRepository(lesqlc.New(pool))
}

func NewEventRepository(pool *pgxpool.Pool) leapprepo.UnitLearningEventRepository {
	return lerepo.NewUnitLearningEventRepository(lesqlc.New(pool))
}

func CreateTestUser(ctx context.Context, pool *pgxpool.Pool) (uuid.UUID, error) {
	userID := uuid.New()
	if _, err := pool.Exec(ctx, `insert into auth.users (id) values ($1)`, userID); err != nil {
		return uuid.Nil, err
	}

	return userID, nil
}

func CreateTestCoarseUnits(ctx context.Context, pool *pgxpool.Pool, count int) ([]int64, error) {
	nowSeed := time.Now().UnixNano()
	ids := make([]int64, 0, count)
	kinds := []string{"word", "phrase", "grammar"}

	for i := 0; i < count; i++ {
		id := nowSeed + int64(i) + 1
		if _, err := pool.Exec(ctx, `
			insert into semantic.coarse_unit (id, kind, label, pos, english_def, chinese_def)
			values ($1, $2, $3, $4, $5, $6)
		`, id, kinds[i%len(kinds)], fmt.Sprintf("e2e-unit-%d", i), "n.", fmt.Sprintf("def-%d", i), fmt.Sprintf("释义-%d", i)); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}

	return ids, nil
}

func CleanupTestData(ctx context.Context, t *testing.T, pool *pgxpool.Pool, userID uuid.UUID, coarseUnitIDs []int64) {
	t.Helper()

	if userID != uuid.Nil {
		if _, err := pool.Exec(ctx, `
			delete from recommendation.scheduler_run_items
			where run_id in (
				select run_id from recommendation.scheduler_runs where user_id = $1
			)
		`, userID); err != nil {
			t.Fatalf("delete scheduler_run_items: %v", err)
		}
		if _, err := pool.Exec(ctx, `delete from recommendation.scheduler_runs where user_id = $1`, userID); err != nil {
			t.Fatalf("delete scheduler_runs: %v", err)
		}
		if _, err := pool.Exec(ctx, `delete from recommendation.user_unit_serving_states where user_id = $1`, userID); err != nil {
			t.Fatalf("delete user_unit_serving_states: %v", err)
		}
		if _, err := pool.Exec(ctx, `delete from learning.unit_learning_events where user_id = $1`, userID); err != nil {
			t.Fatalf("delete unit_learning_events: %v", err)
		}
		if _, err := pool.Exec(ctx, `delete from learning.user_unit_states where user_id = $1`, userID); err != nil {
			t.Fatalf("delete user_unit_states: %v", err)
		}
		if _, err := pool.Exec(ctx, `delete from auth.users where id = $1`, userID); err != nil {
			t.Fatalf("delete auth.users: %v", err)
		}
	}

	if len(coarseUnitIDs) > 0 {
		if _, err := pool.Exec(ctx, `delete from semantic.coarse_unit where id = any($1::bigint[])`, coarseUnitIDs); err != nil {
			t.Fatalf("delete semantic.coarse_unit: %v", err)
		}
	}
}

func SeedNewTargetState(ctx context.Context, repo leapprepo.UserUnitStateRepository, userID uuid.UUID, coarseUnitID int64, priority float64, sourceRef string, now time.Time) error {
	return repo.Upsert(ctx, &lemodel.UserUnitState{
		UserID:                  userID,
		CoarseUnitID:            coarseUnitID,
		IsTarget:                true,
		TargetSource:            "lesson",
		TargetSourceRefID:       sourceRef,
		TargetPriority:          priority,
		Status:                  leenum.UnitStatusNew,
		ProgressPercent:         0,
		MasteryScore:            0,
		RecentQualityWindow:     []int{},
		RecentCorrectnessWindow: []bool{},
		Repetition:              0,
		IntervalDays:            0,
		EaseFactor:              2.5,
		CreatedAt:               now,
		UpdatedAt:               now,
	})
}

func InsertServingState(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, coarseUnitID int64, recommendedAt time.Time) error {
	_, err := pool.Exec(ctx, `
		insert into recommendation.user_unit_serving_states (
			user_id, coarse_unit_id, last_recommended_at, created_at, updated_at
		) values ($1, $2, $3, $3, $3)
		on conflict (user_id, coarse_unit_id)
		do update set last_recommended_at = excluded.last_recommended_at, updated_at = excluded.updated_at
	`, userID, coarseUnitID, recommendedAt)
	return err
}

func BoolPtr(value bool) *bool {
	return &value
}

func IntPtr(value int) *int {
	return &value
}

func NewLearnInput(coarseUnitID int64, correct bool, quality int, occurredAt time.Time, sourceRef string) lecommand.LearningEventInput {
	return lecommand.LearningEventInput{
		CoarseUnitID: coarseUnitID,
		EventType:    leenum.EventTypeNewLearn,
		SourceType:   "e2e_test",
		SourceRefID:  sourceRef,
		IsCorrect:    BoolPtr(correct),
		Quality:      IntPtr(quality),
		OccurredAt:   occurredAt,
	}
}

func ReviewInput(coarseUnitID int64, correct bool, quality int, occurredAt time.Time, sourceRef string) lecommand.LearningEventInput {
	return lecommand.LearningEventInput{
		CoarseUnitID: coarseUnitID,
		EventType:    leenum.EventTypeReview,
		SourceType:   "e2e_test",
		SourceRefID:  sourceRef,
		IsCorrect:    BoolPtr(correct),
		Quality:      IntPtr(quality),
		OccurredAt:   occurredAt,
	}
}

func GenerateCommand(userID uuid.UUID, limit int, now time.Time) reccommand.GenerateRecommendationsCommand {
	return reccommand.GenerateRecommendationsCommand{
		UserID:         userID,
		RequestedLimit: limit,
		Now:            now,
	}
}
