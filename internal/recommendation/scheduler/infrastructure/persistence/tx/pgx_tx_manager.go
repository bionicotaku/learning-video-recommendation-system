package tx

import (
	"context"
	"errors"
	"fmt"

	"learning-video-recommendation-system/internal/recommendation/scheduler/infrastructure/persistence/sqlcgen"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type pgxTxManager struct {
	pool *pgxpool.Pool
}

// NewPGXTxManager creates a transaction manager backed by a pgx pool.
func NewPGXTxManager(pool *pgxpool.Pool) TxManager {
	return &pgxTxManager{pool: pool}
}

// WithinTx runs the callback inside a single database transaction.
func (m *pgxTxManager) WithinTx(ctx context.Context, fn func(ctx context.Context, q sqlcgen.Querier) error) (err error) {
	tx, err := m.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	defer func() {
		if err == nil {
			return
		}

		rollbackErr := tx.Rollback(ctx)
		if rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
			err = errors.Join(err, fmt.Errorf("rollback tx: %w", rollbackErr))
		}
	}()

	queries := sqlcgen.New(tx)
	if err = fn(ctx, queries); err != nil {
		return err
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	return nil
}
