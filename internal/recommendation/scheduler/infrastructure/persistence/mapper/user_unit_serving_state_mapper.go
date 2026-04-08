// 文件作用：
//   - 把 serving state 更新请求映射成 UpsertUserUnitServingStateParams
//   - 统一设置首次写入和更新时使用的时间字段
//
// 输入/输出：
//   - 输入：userID、coarseUnitID、runID、recommendedAt
//   - 输出：sqlcgen.UpsertUserUnitServingStateParams
//
// 谁调用它：
//   - infrastructure/persistence/repository/user_unit_serving_state_repo.go
//
// 它调用谁/传给谁：
//   - 调用 UUIDToPG 和 TimeToPG
//   - 把结果传给 sqlcgen.UpsertUserUnitServingState
package mapper

import (
	"time"

	"learning-video-recommendation-system/internal/recommendation/scheduler/infrastructure/persistence/sqlcgen"

	"github.com/google/uuid"
)

func UserUnitServingStateToUpsertParams(userID uuid.UUID, coarseUnitID int64, runID uuid.UUID, recommendedAt time.Time) sqlcgen.UpsertUserUnitServingStateParams {
	return sqlcgen.UpsertUserUnitServingStateParams{
		UserID:                  UUIDToPG(userID),
		CoarseUnitID:            coarseUnitID,
		LastRecommendedAt:       TimeToPG(recommendedAt),
		LastRecommendationRunID: UUIDToPG(runID),
		CreatedAt:               TimeToPG(recommendedAt),
		UpdatedAt:               TimeToPG(recommendedAt),
	}
}
