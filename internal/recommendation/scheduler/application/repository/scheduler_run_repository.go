// 文件作用：
//   - 定义 SchedulerRunRepository，封装 Recommendation 审计写入
//   - 把 run 头和 run item 写入从 usecase 中抽离出去
//
// 输入/输出：
//   - 输入：已经组装完成的 RecommendationBatch
//   - 输出：写入成功或失败
//
// 谁调用它：
//   - application/usecase/generate_recommendations.go
//
// 它调用谁/传给谁：
//   - 接口本身不调用其他实现
//   - 由 infrastructure/persistence/repository/scheduler_run_repo.go 实现
package repository

import (
	"context"

	"learning-video-recommendation-system/internal/recommendation/scheduler/domain/model"
)

type SchedulerRunRepository interface {
	SaveRun(ctx context.Context, batch model.RecommendationBatch) error
	SaveRunItems(ctx context.Context, batch model.RecommendationBatch) error
}
