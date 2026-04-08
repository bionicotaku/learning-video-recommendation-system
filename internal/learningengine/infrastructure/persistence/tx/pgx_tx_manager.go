// 作用：用 pgx 实现 TxManager，在事务内创建 sqlc querier 并通过 context 向 repository 传播。
// 输入/输出：输入是 pool、context 和事务回调；输出是回调执行后的 error。
// 谁调用它：application/usecase 的构造逻辑、fixture/helpers.go。
// 它调用谁/传给谁：调用 pgxpool.BeginTx、sqlcgen.New(tx)、queryctx.WithQuerier；带事务的 context 再传给 repository。
package tx

import (
	"context"
	"errors"
	"fmt"

	apprepo "learning-video-recommendation-system/internal/learningengine/application/repository"
	"learning-video-recommendation-system/internal/learningengine/infrastructure/persistence/queryctx"
	"learning-video-recommendation-system/internal/learningengine/infrastructure/persistence/sqlcgen"

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
func (m *pgxTxManager) WithinTx(ctx context.Context, fn func(ctx context.Context) error) (err error) {
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

	ctxWithTx := queryctx.WithQuerier(ctx, sqlcgen.New(tx))
	if err = fn(ctxWithTx); err != nil {
		return err
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	return nil
}

type TxManager = apprepo.TxManager
