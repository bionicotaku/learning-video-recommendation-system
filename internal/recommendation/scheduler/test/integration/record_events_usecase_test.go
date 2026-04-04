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

	userID, err := createTestUserIDFromPool(ctx, pool)
	if err != nil {
		t.Fatalf("createTestUserIDFromPool() error = %v", err)
	}
	defer cleanupTestUser(ctx, t, pool, userID)
	unitIDs, err := loadAvailableCoarseUnitIDsFromPool(ctx, pool, userID, 2)
	if err != nil {
		t.Fatalf("loadAvailableCoarseUnitIDsFromPool() error = %v", err)
	}

	txManager := txtx.NewPGXTxManager(pool)
	baseQuerier := sqlcgen.New(pool)
	stateRepo := repopkg.NewUserUnitStateRepository(baseQuerier)
	eventRepo := repopkg.NewUnitLearningEventRepository(baseQuerier)
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

	events, err := eventRepo.FindForReplay(ctx, userID, &unitIDs[0], nil)
	if err != nil {
		t.Fatalf("FindForReplay() error = %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("len(events) = %d, want 1", len(events))
	}

	state, err := stateRepo.GetByUserAndUnit(ctx, userID, unitIDs[0])
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

	userID, err := createTestUserIDFromPool(ctx, pool)
	if err != nil {
		t.Fatalf("createTestUserIDFromPool() error = %v", err)
	}
	defer cleanupTestUser(ctx, t, pool, userID)
	unitIDs, err := loadAvailableCoarseUnitIDsFromPool(ctx, pool, userID, 1)
	if err != nil {
		t.Fatalf("loadAvailableCoarseUnitIDsFromPool() error = %v", err)
	}

	baseQuerier := sqlcgen.New(pool)
	baseStateRepo := repopkg.NewUserUnitStateRepository(baseQuerier)
	eventRepo := repopkg.NewUnitLearningEventRepository(baseQuerier)
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

	events, err := eventRepo.FindForReplay(ctx, userID, &unitIDs[0], nil)
	if err != nil {
		t.Fatalf("FindForReplay() error = %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("len(events) = %d, want 0 after rollback", len(events))
	}

	state, err := baseStateRepo.GetByUserAndUnit(ctx, userID, unitIDs[0])
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

func (f failingUserUnitStateRepository) GetByUserAndUnit(ctx context.Context, userID uuid.UUID, coarseUnitID int64) (*model.UserUnitState, error) {
	return f.delegate.GetByUserAndUnit(ctx, userID, coarseUnitID)
}

func (f failingUserUnitStateRepository) Upsert(ctx context.Context, state *model.UserUnitState) error {
	return errors.New("forced upsert failure")
}

func (f failingUserUnitStateRepository) BatchUpsert(ctx context.Context, states []*model.UserUnitState) error {
	return f.delegate.BatchUpsert(ctx, states)
}

func (f failingUserUnitStateRepository) DeleteForReplay(ctx context.Context, userID uuid.UUID, coarseUnitID *int64) error {
	return f.delegate.DeleteForReplay(ctx, userID, coarseUnitID)
}

func (f failingUserUnitStateRepository) FindDueReviewCandidates(ctx context.Context, userID uuid.UUID, now time.Time) ([]appquery.ReviewCandidate, error) {
	return f.delegate.FindDueReviewCandidates(ctx, userID, now)
}

func (f failingUserUnitStateRepository) FindNewCandidates(ctx context.Context, userID uuid.UUID) ([]appquery.NewCandidate, error) {
	return f.delegate.FindNewCandidates(ctx, userID)
}

func createTestUserIDFromPool(ctx context.Context, pool *pgxpool.Pool) (uuid.UUID, error) {
	userID := uuid.New()
	if _, err := pool.Exec(ctx, `insert into auth.users (id) values ($1)`, userID); err != nil {
		return uuid.Nil, err
	}

	return userID, nil
}

func cleanupTestUser(ctx context.Context, t *testing.T, pool *pgxpool.Pool, userID uuid.UUID) {
	t.Helper()
	if _, err := pool.Exec(ctx, `delete from auth.users where id = $1`, userID); err != nil {
		t.Fatalf("cleanup auth.users error = %v", err)
	}
}

func loadAvailableCoarseUnitIDsFromPool(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, count int) ([]int64, error) {
	rows, err := pool.Query(ctx, `
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
