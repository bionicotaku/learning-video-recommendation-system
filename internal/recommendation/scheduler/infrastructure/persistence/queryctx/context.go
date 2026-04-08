// 文件作用：
//   - 在 context 中保存和读取事务作用域内的 sqlc Querier
//   - 让 repository 在事务内自动切换到 tx 版本 querier，而不是继续用普通连接池 querier
//
// 输入/输出：
//   - 输入：context 和 sqlcgen.Querier
//   - 输出：写入过 querier 的 context，或从 context 中取出的 querier
//
// 谁调用它：
//   - infrastructure/persistence/tx/pgx_tx_manager.go 写入事务 querier
//   - infrastructure/persistence/repository/querier_resolver.go 读取事务 querier
//
// 它调用谁/传给谁：
//   - 调用 context.WithValue / ctx.Value
//   - 把 tx querier 传给 repository 层
package queryctx

import (
	"context"

	"learning-video-recommendation-system/internal/recommendation/scheduler/infrastructure/persistence/sqlcgen"
)

type contextKey struct{}

// WithQuerier stores a transaction-scoped querier in context.
func WithQuerier(ctx context.Context, querier sqlcgen.Querier) context.Context {
	return context.WithValue(ctx, contextKey{}, querier)
}

// FromContext returns a transaction-scoped querier when present.
func FromContext(ctx context.Context) (sqlcgen.Querier, bool) {
	querier, ok := ctx.Value(contextKey{}).(sqlcgen.Querier)
	return querier, ok
}
