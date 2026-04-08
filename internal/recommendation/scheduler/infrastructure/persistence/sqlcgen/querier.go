// 文件作用：
//   - 定义 sqlc 生成层暴露给 repository 的稳定 Querier 接口
//   - 让 repository 既能接普通 Queries，也能接事务内 Queries
//
// 输入/输出：
//   - 输入：context 和各类 sqlc 参数结构
//   - 输出：查询结果、执行结果或错误
//
// 谁调用它：
//   - infrastructure/persistence/repository/*.go
//   - queryctx/context.go 和 tx/pgx_tx_manager.go 会间接传递它
//
// 它调用谁/传给谁：
//   - 具体实现由 sqlc 生成的 Queries 提供
//   - repository 通过这个接口把调用落到 db.go 中的 Queries
package sqlcgen

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

// Querier is the stable query surface used by repositories and transaction helpers.
type Querier interface {
	CountSchedulerRuns(ctx context.Context) (int64, error)
	FindDueReviewCandidates(ctx context.Context, arg FindDueReviewCandidatesParams) ([]FindDueReviewCandidatesRow, error)
	FindNewCandidates(ctx context.Context, userID pgtype.UUID) ([]FindNewCandidatesRow, error)
	UpsertSchedulerRun(ctx context.Context, arg UpsertSchedulerRunParams) error
	UpsertSchedulerRunItem(ctx context.Context, arg UpsertSchedulerRunItemParams) error
	UpsertUserUnitServingState(ctx context.Context, arg UpsertUserUnitServingStateParams) error
}
