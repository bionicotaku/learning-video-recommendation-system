package tx

import (
	"context"
	"fmt"

	apirepo "learning-video-recommendation-system/internal/api/application/repository"
	learningrepo "learning-video-recommendation-system/internal/learningengine/reducer/application/repository"
	learningpersist "learning-video-recommendation-system/internal/learningengine/reducer/infrastructure/persistence/repository"
	userrepo "learning-video-recommendation-system/internal/user/application/repository"
	userpersist "learning-video-recommendation-system/internal/user/infrastructure/persistence/repository"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ActivateCollectionManager struct {
	pool *pgxpool.Pool
}

var _ apirepo.ActivateCollectionTxManager = (*ActivateCollectionManager)(nil)

func NewActivateCollectionManager(pool *pgxpool.Pool) *ActivateCollectionManager {
	return &ActivateCollectionManager{pool: pool}
}

func (m *ActivateCollectionManager) WithinUserTx(ctx context.Context, userID string, fn func(ctx context.Context, repos apirepo.ActivateCollectionRepositories) error) error {
	tx, err := m.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if _, err := tx.Exec(ctx, `select pg_advisory_xact_lock(hashtextextended('learningengine:user:' || $1, 0))`, userID); err != nil {
		return fmt.Errorf("acquire user advisory lock: %w", err)
	}

	if err := fn(ctx, activateCollectionRepositories{tx: tx}); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

type activateCollectionRepositories struct {
	tx pgx.Tx
}

func (r activateCollectionRepositories) TargetCommands() learningrepo.TargetStateCommandRepository {
	return learningpersist.NewTargetStateCommandRepository(r.tx)
}

func (r activateCollectionRepositories) UserProfiles() userrepo.ProfileRepository {
	return userpersist.NewRepository(r.tx)
}
