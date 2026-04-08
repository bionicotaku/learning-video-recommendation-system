// 文件作用：
//   - 提供 TxManager 的 pgx 实现
//   - 负责开启事务、在 context 中注入 tx querier、回滚和提交
//
// 输入/输出：
//   - 输入：调用方传入的 ctx 和事务回调 fn
//   - 输出：事务整体执行结果 error
//
// 谁调用它：
//   - application/usecase/generate_recommendations.go 通过接口间接调用
//   - fixture.NewGenerateUseCase 负责构造并注入它
//
// 它调用谁/传给谁：
//   - 调用 pgxpool.BeginTx / tx.Commit / tx.Rollback
//   - 调用 sqlcgen.New(tx) 构造事务 querier
//   - 调用 queryctx.WithQuerier 把 tx querier 传给 repository
package tx

import (
	"context"
	"errors"
	"fmt"

	apprepo "learning-video-recommendation-system/internal/recommendation/scheduler/application/repository"
	"learning-video-recommendation-system/internal/recommendation/scheduler/infrastructure/persistence/queryctx"
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
