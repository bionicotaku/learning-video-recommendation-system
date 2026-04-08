// 文件作用：
//   - 定义 UserUnitServingStateRepository，封装 Recommendation 自有 serving state 的更新入口
//   - 把最近推荐时间和最近 run ID 的落库动作从 usecase 中隔离出去
//
// 输入/输出：
//   - 输入：userID、runID、coarseUnitIDs、recommendedAt
//   - 输出：serving state 更新成功或失败
//
// 谁调用它：
//   - application/usecase/generate_recommendations.go
//
// 它调用谁/传给谁：
//   - 接口本身不调用其他实现
//   - 由 infrastructure/persistence/repository/user_unit_serving_state_repo.go 实现
package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type UserUnitServingStateRepository interface {
	TouchRecommendedAt(ctx context.Context, userID uuid.UUID, runID uuid.UUID, coarseUnitIDs []int64, recommendedAt time.Time) error
}
