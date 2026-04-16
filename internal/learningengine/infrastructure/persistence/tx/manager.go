package tx

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	apprepo "learning-video-recommendation-system/internal/learningengine/application/repository"
	"learning-video-recommendation-system/internal/learningengine/application/service"
	persistrepo "learning-video-recommendation-system/internal/learningengine/infrastructure/persistence/repository"
)

type Manager struct {
	pool *pgxpool.Pool
}

func NewManager(pool *pgxpool.Pool) *Manager {
	return &Manager{pool: pool}
}

var _ service.TxManager = (*Manager)(nil)

func (m *Manager) WithinTx(ctx context.Context, fn func(ctx context.Context, repos service.TransactionalRepositories) error) error {
	return m.withinTx(ctx, "", false, fn)
}

func (m *Manager) WithinUserTx(ctx context.Context, userID string, fn func(ctx context.Context, repos service.TransactionalRepositories) error) error {
	return m.withinTx(ctx, userID, true, fn)
}

func (m *Manager) withinTx(ctx context.Context, userID string, lockUser bool, fn func(ctx context.Context, repos service.TransactionalRepositories) error) error {
	tx, err := m.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if lockUser {
		if _, err := tx.Exec(ctx, `select pg_advisory_xact_lock(hashtextextended('learningengine:user:' || $1, 0))`, userID); err != nil {
			return fmt.Errorf("acquire user advisory lock: %w", err)
		}
	}

	repositories := repositories{tx: tx}
	if err := fn(ctx, repositories); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

type repositories struct {
	tx pgx.Tx
}

func (r repositories) UserUnitStates() apprepo.UserUnitStateRepository {
	return persistrepo.NewUserUnitStateRepository(r.tx)
}

func (r repositories) TargetCommands() apprepo.TargetStateCommandRepository {
	return persistrepo.NewTargetStateCommandRepository(r.tx)
}

func (r repositories) UnitLearningEvents() apprepo.UnitLearningEventRepository {
	return persistrepo.NewUnitLearningEventRepository(r.tx)
}
