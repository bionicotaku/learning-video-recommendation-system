// 文件作用：
//   - 统一解析 repository 当前应该使用哪个 querier
//   - 优先使用事务上下文里的 tx querier，没有时才回退到构造注入的普通 querier
//
// 输入/输出：
//   - 输入：context 和默认 querier
//   - 输出：当前应使用的 sqlcgen.Querier 或错误
//
// 谁调用它：
//   - 所有 repository 实现都会先调用它
//
// 它调用谁/传给谁：
//   - 调用 queryctx.FromContext
//   - 把解析出的 querier 传给后续 sqlc 调用
package repository

import (
	"context"
	"fmt"

	"learning-video-recommendation-system/internal/recommendation/scheduler/infrastructure/persistence/queryctx"
	"learning-video-recommendation-system/internal/recommendation/scheduler/infrastructure/persistence/sqlcgen"
)

func resolveQuerier(ctx context.Context, fallback sqlcgen.Querier) (sqlcgen.Querier, error) {
	if querier, ok := queryctx.FromContext(ctx); ok {
		return querier, nil
	}
	if fallback == nil {
		return nil, fmt.Errorf("repository querier is not configured")
	}

	return fallback, nil
}
