package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"learning-video-recommendation-system/internal/learningengine/application/command"
	appservice "learning-video-recommendation-system/internal/learningengine/application/service"
	"learning-video-recommendation-system/internal/learningengine/application/usecase"
	"learning-video-recommendation-system/internal/learningengine/domain/aggregate"
	"learning-video-recommendation-system/internal/learningengine/domain/enum"
	"learning-video-recommendation-system/internal/learningengine/domain/model"
	"learning-video-recommendation-system/internal/learningengine/domain/policy"
	infra "learning-video-recommendation-system/internal/learningengine/infrastructure"
	repopkg "learning-video-recommendation-system/internal/learningengine/infrastructure/persistence/repository"
	"learning-video-recommendation-system/internal/learningengine/infrastructure/persistence/sqlcgen"
	txtx "learning-video-recommendation-system/internal/learningengine/infrastructure/persistence/tx"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func newRecordEventsUseCase(pool *pgxpool.Pool, querier sqlcgen.Querier) usecase.RecordLearningEventsUseCase {
	return usecase.NewRecordLearningEventsUseCase(
		txtx.NewPGXTxManager(pool),
		repopkg.NewUserUnitStateRepository(querier),
		repopkg.NewUnitLearningEventRepository(querier),
		aggregate.NewUserUnitReducer(),
	)
}

func newReplayUseCase(pool *pgxpool.Pool, querier sqlcgen.Querier) usecase.ReplayUserStatesUseCase {
	return usecase.NewReplayUserStatesUseCase(
		txtx.NewPGXTxManager(pool),
		repopkg.NewUserUnitStateRepository(querier),
		repopkg.NewUnitLearningEventRepository(querier),
		appservice.NewUserStateRebuilder(aggregate.NewUserUnitReducer(), policy.DefaultSchedulerPolicy()),
	)
}

func newTestPool(t *testing.T) (context.Context, *pgxpool.Pool) {
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

func createTestUser(ctx context.Context, pool *pgxpool.Pool) (uuid.UUID, error) {
	userID := uuid.New()
	if _, err := pool.Exec(ctx, `insert into auth.users (id) values ($1)`, userID); err != nil {
		return uuid.Nil, err
	}

	return userID, nil
}

func createTestCoarseUnits(ctx context.Context, pool *pgxpool.Pool, count int) ([]int64, error) {
	nowSeed := time.Now().UnixNano()
	ids := make([]int64, 0, count)

	for i := 0; i < count; i++ {
		id := nowSeed + int64(i) + 1
		kind := []enum.UnitKind{enum.UnitKindWord, enum.UnitKindPhrase, enum.UnitKindGrammar}[i%3]
		if _, err := pool.Exec(ctx, `
			insert into semantic.coarse_unit (id, kind, label, pos, english_def, chinese_def)
			values ($1, $2, $3, $4, $5, $6)
		`, id, string(kind), fmt.Sprintf("test-unit-%d", i), "n.", fmt.Sprintf("def-%d", i), fmt.Sprintf("释义-%d", i)); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}

	return ids, nil
}

func cleanupTestData(ctx context.Context, t *testing.T, pool *pgxpool.Pool, userID uuid.UUID, coarseUnitIDs []int64) {
	t.Helper()

	if len(coarseUnitIDs) > 0 {
		if _, err := pool.Exec(ctx, `delete from semantic.coarse_unit where id = any($1::bigint[])`, coarseUnitIDs); err != nil {
			t.Fatalf("delete coarse units: %v", err)
		}
	}
	if userID != uuid.Nil {
		if _, err := pool.Exec(ctx, `delete from auth.users where id = $1`, userID); err != nil {
			t.Fatalf("delete user: %v", err)
		}
	}
}

func filterEventsByUnit(events []model.LearningEvent, userID uuid.UUID, coarseUnitID int64) []model.LearningEvent {
	items := make([]model.LearningEvent, 0, len(events))
	for _, event := range events {
		if event.UserID == userID && event.CoarseUnitID == coarseUnitID {
			items = append(items, event)
		}
	}

	return items
}

func newLearnInput(coarseUnitID int64, correct *bool, quality *int, occurredAt time.Time, sourceRef string) command.LearningEventInput {
	return command.LearningEventInput{
		CoarseUnitID: coarseUnitID,
		EventType:    enum.EventTypeNewLearn,
		SourceType:   "integration_test",
		SourceRefID:  sourceRef,
		IsCorrect:    correct,
		Quality:      quality,
		OccurredAt:   occurredAt,
	}
}

func reviewInput(coarseUnitID int64, correct *bool, quality *int, occurredAt time.Time, sourceRef string) command.LearningEventInput {
	return command.LearningEventInput{
		CoarseUnitID: coarseUnitID,
		EventType:    enum.EventTypeReview,
		SourceType:   "integration_test",
		SourceRefID:  sourceRef,
		IsCorrect:    correct,
		Quality:      quality,
		OccurredAt:   occurredAt,
	}
}
