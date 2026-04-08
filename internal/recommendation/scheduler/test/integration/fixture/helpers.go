// 文件作用：
//   - 提供 scheduler 集成测试和场景测试共用的夹具与组装函数
//   - 当前也是仓库里最完整的 usecase 依赖组装示例
//
// 输入/输出：
//   - 输入：测试传入的 t、数据库连接上下文以及各类测试数据
//   - 输出：测试连接池、测试数据、组装好的 usecase 和 command
//
// 谁调用它：
//   - integration/infrastructure/*.go
//   - integration/usecase/*.go
//   - scenario/scenarios_test.go
//
// 它调用谁/传给谁：
//   - 调用 infrastructure.LoadConfig / NewDBPool
//   - 调用 repository 构造函数、tx manager 和 domain service 构造函数
//   - 把组装结果传给测试代码
package fixture

import (
	"context"
	"fmt"
	"testing"
	"time"

	"learning-video-recommendation-system/internal/recommendation/scheduler/application/command"
	"learning-video-recommendation-system/internal/recommendation/scheduler/application/usecase"
	domainservice "learning-video-recommendation-system/internal/recommendation/scheduler/domain/service"
	infra "learning-video-recommendation-system/internal/recommendation/scheduler/infrastructure"
	repopkg "learning-video-recommendation-system/internal/recommendation/scheduler/infrastructure/persistence/repository"
	"learning-video-recommendation-system/internal/recommendation/scheduler/infrastructure/persistence/sqlcgen"
	txtx "learning-video-recommendation-system/internal/recommendation/scheduler/infrastructure/persistence/tx"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func NewTestPool(t *testing.T) (context.Context, *pgxpool.Pool) {
	t.Helper()

	cfg := infra.LoadConfig()
	if cfg.DatabaseURL == "" {
		t.Skip("DATABASE_URL is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	t.Cleanup(cancel)

	pool, err := infra.NewDBPool(ctx, cfg)
	if err != nil {
		t.Fatalf("NewDBPool() error = %v", err)
	}
	t.Cleanup(pool.Close)

	return ctx, pool
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
		`, id, kinds[i%len(kinds)], fmt.Sprintf("test-unit-%d", i), "n.", fmt.Sprintf("def-%d", i), fmt.Sprintf("释义-%d", i)); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}

	return ids, nil
}

func CleanupTestData(ctx context.Context, t *testing.T, pool *pgxpool.Pool, userID uuid.UUID, coarseUnitIDs []int64) {
	t.Helper()

	if userID != uuid.Nil {
		if _, err := pool.Exec(ctx, `delete from auth.users where id = $1`, userID); err != nil {
			t.Fatalf("delete user: %v", err)
		}
	}
	if len(coarseUnitIDs) > 0 {
		if _, err := pool.Exec(ctx, `delete from semantic.coarse_unit where id = any($1::bigint[])`, coarseUnitIDs); err != nil {
			t.Fatalf("delete coarse units: %v", err)
		}
	}
}

func InsertState(ctx context.Context, pool *pgxpool.Pool, args ...any) error {
	_, err := pool.Exec(ctx, `
		insert into learning.user_unit_states (
			user_id,
			coarse_unit_id,
			is_target,
			target_source,
			target_source_ref_id,
			target_priority,
			status,
			progress_percent,
			mastery_score,
			first_seen_at,
			last_seen_at,
			last_reviewed_at,
			seen_count,
			strong_event_count,
			review_count,
			correct_count,
			wrong_count,
			consecutive_correct,
			consecutive_wrong,
			last_quality,
			recent_quality_window,
			recent_correctness_window,
			repetition,
			interval_days,
			ease_factor,
			next_review_at,
			suspended_reason,
			created_at,
			updated_at
		) values (
			$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27,$28,$29
		)
	`, args...)
	return err
}

func InsertServingState(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, coarseUnitID int64, recommendedAt time.Time) error {
	_, err := pool.Exec(ctx, `
		insert into recommendation.user_unit_serving_states (
			user_id, coarse_unit_id, last_recommended_at, created_at, updated_at
		) values ($1, $2, $3, $3, $3)
	`, userID, coarseUnitID, recommendedAt)
	return err
}

func NewGenerateUseCase(pool *pgxpool.Pool, querier sqlcgen.Querier) usecase.GenerateLearningUnitRecommendationsUseCase {
	return usecase.NewGenerateLearningUnitRecommendationsUseCase(
		txtx.NewPGXTxManager(pool),
		repopkg.NewLearningStateSnapshotReadRepository(querier),
		repopkg.NewUserUnitServingStateRepository(querier),
		repopkg.NewSchedulerRunRepository(querier),
		domainservice.NewBacklogCalculator(),
		domainservice.NewQuotaAllocator(),
		domainservice.NewReviewScorer(),
		domainservice.NewNewScorer(),
		domainservice.NewPriorityZeroExtractor(),
		domainservice.NewRecommendationAssembler(),
	)
}

func GenerateCmd(userID uuid.UUID, limit int, now time.Time) command.GenerateRecommendationsCommand {
	return command.GenerateRecommendationsCommand{
		UserID:         userID,
		RequestedLimit: limit,
		Now:            now,
	}
}
