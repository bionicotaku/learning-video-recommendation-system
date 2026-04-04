package integration

import (
	"context"
	"errors"
	"testing"
	"time"

	"learning-video-recommendation-system/internal/recommendation/scheduler/application/command"
	appquery "learning-video-recommendation-system/internal/recommendation/scheduler/application/query"
	apprepo "learning-video-recommendation-system/internal/recommendation/scheduler/application/repository"
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
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestRecordLearningEventsAndUpdateStateUseCase(t *testing.T) {
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

	userID, err := loadExistingUserIDFromPool(ctx, pool)
	if err != nil {
		t.Fatalf("loadExistingUserIDFromPool() error = %v", err)
	}
	unitIDs, err := loadAvailableCoarseUnitIDsFromPool(ctx, pool, userID, 2)
	if err != nil {
		t.Fatalf("loadAvailableCoarseUnitIDsFromPool() error = %v", err)
	}
	defer cleanupLearningRows(ctx, t, pool, userID, unitIDs)

	txManager := txtx.NewPGXTxManager(pool)
	stateRepo := repopkg.NewUserUnitStateRepository()
	eventRepo := repopkg.NewUnitLearningEventRepository()
	stateUpdater := domainservice.NewStateUpdater()

	uc := usecase.NewRecordLearningEventsAndUpdateStateUseCase(txManager, stateRepo, eventRepo, stateUpdater)

	correct := true
	quality := 4
	occurredAt := time.Date(2026, 4, 8, 15, 0, 0, 0, time.UTC)

	result, err := uc.Execute(ctx, command.RecordLearningEventsCommand{
		UserID: userID,
		Events: []command.LearningEventInput{
			{
				CoarseUnitID: unitIDs[0],
				EventType:    enum.EventTypeNewLearn,
				SourceType:   "integration_test",
				SourceRefID:  "record-events",
				IsCorrect:    &correct,
				Quality:      &quality,
				OccurredAt:   occurredAt,
			},
		},
		IdempotencyKey: "integration-success",
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if result.AcceptedCount != 1 {
		t.Fatalf("AcceptedCount = %d, want 1", result.AcceptedCount)
	}
	if len(result.UpdatedUnits) != 1 || result.UpdatedUnits[0] != unitIDs[0] {
		t.Fatalf("UpdatedUnits = %v, want [%d]", result.UpdatedUnits, unitIDs[0])
	}

	q := sqlcgen.New(pool)
	events, err := eventRepo.FindForReplay(ctx, q, userID, &unitIDs[0], nil)
	if err != nil {
		t.Fatalf("FindForReplay() error = %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("len(events) = %d, want 1", len(events))
	}

	state, err := stateRepo.GetByUserAndUnit(ctx, q, userID, unitIDs[0])
	if err != nil {
		t.Fatalf("GetByUserAndUnit() error = %v", err)
	}
	if state == nil {
		t.Fatal("state = nil, want value")
	}
	if state.Status != enum.UnitStatusLearning {
		t.Fatalf("state.Status = %q, want %q", state.Status, enum.UnitStatusLearning)
	}
}

func TestRecordLearningEventsAndUpdateStateUseCaseRollsBackOnStateFailure(t *testing.T) {
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

	userID, err := loadExistingUserIDFromPool(ctx, pool)
	if err != nil {
		t.Fatalf("loadExistingUserIDFromPool() error = %v", err)
	}
	unitIDs, err := loadAvailableCoarseUnitIDsFromPool(ctx, pool, userID, 1)
	if err != nil {
		t.Fatalf("loadAvailableCoarseUnitIDsFromPool() error = %v", err)
	}
	defer cleanupLearningRows(ctx, t, pool, userID, unitIDs)

	baseStateRepo := repopkg.NewUserUnitStateRepository()
	eventRepo := repopkg.NewUnitLearningEventRepository()
	uc := usecase.NewRecordLearningEventsAndUpdateStateUseCase(
		txtx.NewPGXTxManager(pool),
		failingUserUnitStateRepository{delegate: baseStateRepo},
		eventRepo,
		domainservice.NewStateUpdater(),
	)

	correct := true
	quality := 4

	_, err = uc.Execute(ctx, command.RecordLearningEventsCommand{
		UserID: userID,
		Events: []command.LearningEventInput{
			{
				CoarseUnitID: unitIDs[0],
				EventType:    enum.EventTypeNewLearn,
				SourceType:   "integration_test",
				SourceRefID:  "rollback",
				IsCorrect:    &correct,
				Quality:      &quality,
				OccurredAt:   time.Date(2026, 4, 8, 16, 0, 0, 0, time.UTC),
			},
		},
		IdempotencyKey: "integration-rollback",
	})
	if err == nil {
		t.Fatal("Execute() error = nil, want rollback error")
	}

	q := sqlcgen.New(pool)
	events, err := eventRepo.FindForReplay(ctx, q, userID, &unitIDs[0], nil)
	if err != nil {
		t.Fatalf("FindForReplay() error = %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("len(events) = %d, want 0 after rollback", len(events))
	}

	state, err := baseStateRepo.GetByUserAndUnit(ctx, q, userID, unitIDs[0])
	if err != nil {
		t.Fatalf("GetByUserAndUnit() error = %v", err)
	}
	if state != nil {
		t.Fatalf("state = %+v, want nil after rollback", state)
	}
}

type failingUserUnitStateRepository struct {
	delegate apprepo.UserUnitStateRepository
}

func (f failingUserUnitStateRepository) GetByUserAndUnit(ctx context.Context, q sqlcgen.Querier, userID uuid.UUID, coarseUnitID int64) (*model.UserUnitState, error) {
	return f.delegate.GetByUserAndUnit(ctx, q, userID, coarseUnitID)
}

func (f failingUserUnitStateRepository) Upsert(ctx context.Context, q sqlcgen.Querier, state *model.UserUnitState) error {
	return errors.New("forced upsert failure")
}

func (f failingUserUnitStateRepository) BatchUpsert(ctx context.Context, q sqlcgen.Querier, states []*model.UserUnitState) error {
	return f.delegate.BatchUpsert(ctx, q, states)
}

func (f failingUserUnitStateRepository) DeleteForReplay(ctx context.Context, q sqlcgen.Querier, userID uuid.UUID, coarseUnitID *int64) error {
	return f.delegate.DeleteForReplay(ctx, q, userID, coarseUnitID)
}

func (f failingUserUnitStateRepository) FindDueReviewCandidates(ctx context.Context, q sqlcgen.Querier, userID uuid.UUID, now time.Time) ([]appquery.ReviewCandidate, error) {
	return f.delegate.FindDueReviewCandidates(ctx, q, userID, now)
}

func (f failingUserUnitStateRepository) FindNewCandidates(ctx context.Context, q sqlcgen.Querier, userID uuid.UUID) ([]appquery.NewCandidate, error) {
	return f.delegate.FindNewCandidates(ctx, q, userID)
}

func loadExistingUserIDFromPool(ctx context.Context, pool *pgxpool.Pool) (uuid.UUID, error) {
	var userID uuid.UUID
	err := pool.QueryRow(ctx, `select id from auth.users limit 1`).Scan(&userID)
	return userID, err
}

func loadAvailableCoarseUnitIDsFromPool(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, count int) ([]int64, error) {
	rows, err := pool.Query(ctx, `
		select c.id
		from semantic.coarse_unit c
		left join learning.user_unit_states s
		  on s.coarse_unit_id = c.id
		 and s.user_id = $1
		where s.coarse_unit_id is null
		order by c.id
		limit $2
	`, userID, count)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(ids) < count {
		return nil, pgx.ErrNoRows
	}

	return ids, nil
}

func cleanupLearningRows(ctx context.Context, t *testing.T, pool *pgxpool.Pool, userID uuid.UUID, unitIDs []int64) {
	t.Helper()
	if len(unitIDs) == 0 {
		return
	}

	if _, err := pool.Exec(ctx, `delete from learning.unit_learning_events where user_id = $1 and coarse_unit_id = any($2)`, userID, unitIDs); err != nil {
		t.Fatalf("cleanup unit_learning_events error = %v", err)
	}
	if _, err := pool.Exec(ctx, `delete from learning.user_unit_states where user_id = $1 and coarse_unit_id = any($2)`, userID, unitIDs); err != nil {
		t.Fatalf("cleanup user_unit_states error = %v", err)
	}
	if _, err := pool.Exec(ctx, `delete from learning.scheduler_run_items where user_id = $1 and coarse_unit_id = any($2)`, userID, unitIDs); err != nil {
		t.Fatalf("cleanup scheduler_run_items error = %v", err)
	}
}
