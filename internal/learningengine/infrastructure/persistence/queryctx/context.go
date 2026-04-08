// 作用：在 context 中写入和读取事务范围内的 sqlc Querier，实现事务 querier 的透明传播。
// 输入/输出：输入是 context 和 Querier，或仅读取 context；输出是带 querier 的新 context 或 Querier。
// 谁调用它：persistence/tx/pgx_tx_manager.go 写入，persistence/repository/querier_resolver.go 读取。
// 它调用谁/传给谁：调用标准库 context；得到的 querier 会传给 repository 内部执行 SQL。
package queryctx

import (
	"context"

	"learning-video-recommendation-system/internal/learningengine/infrastructure/persistence/sqlcgen"
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
