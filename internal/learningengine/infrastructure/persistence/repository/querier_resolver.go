// 作用：决定 repository 当前应该使用事务内 querier 还是默认 querier，统一处理事务透传细节。
// 输入/输出：输入是 context 和 fallback Querier；输出是可执行 SQL 的 Querier 或 error。
// 谁调用它：unit_learning_event_repo.go、user_unit_state_repo.go。
// 它调用谁/传给谁：调用 queryctx/context.go 读取事务 querier；解析出的 querier 会传给 repository 方法内部继续执行。
package repository

import (
	"context"
	"fmt"

	"learning-video-recommendation-system/internal/learningengine/infrastructure/persistence/queryctx"
	"learning-video-recommendation-system/internal/learningengine/infrastructure/persistence/sqlcgen"
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
