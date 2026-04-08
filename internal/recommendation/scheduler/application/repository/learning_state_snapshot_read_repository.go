// 文件作用：
//   - 定义 LearningStateSnapshotReadRepository，只暴露 scheduler 需要的只读候选查询
//   - 明确 Recommendation 只消费 Learning engine 输出，不提供回写能力
//
// 输入/输出：
//   - 输入：userID，以及 review 查询所需的 now
//   - 输出：ReviewCandidate 或 NewCandidate 列表
//
// 谁调用它：
//   - application/usecase/generate_recommendations.go
//
// 它调用谁/传给谁：
//   - 接口本身不调用其他实现
//   - 由 infrastructure/persistence/repository/learning_state_snapshot_read_repo.go 实现
package repository

import (
	"context"
	"time"

	"learning-video-recommendation-system/internal/recommendation/scheduler/application/query"

	"github.com/google/uuid"
)

// LearningStateSnapshotReadRepository only exposes candidate reads from Learning engine data.
type LearningStateSnapshotReadRepository interface {
	FindDueReviewCandidates(ctx context.Context, userID uuid.UUID, now time.Time) ([]query.ReviewCandidate, error)
	FindNewCandidates(ctx context.Context, userID uuid.UUID) ([]query.NewCandidate, error)
}
