package tx

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	learningenginesqlc "learning-video-recommendation-system/internal/learningengine/infrastructure/persistence/sqlcgen"
)

type Manager struct {
	pool *pgxpool.Pool
}

func NewManager(pool *pgxpool.Pool) *Manager {
	return &Manager{pool: pool}
}

func (m *Manager) WithinTx(ctx context.Context, fn func(ctx context.Context, queries *learningenginesqlc.Queries) error) error {
	tx, err := m.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if err := fn(ctx, learningenginesqlc.New(tx)); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
