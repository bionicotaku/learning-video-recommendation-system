package tx

import (
	"context"

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
	tx, err := m.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

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
