// 文件作用：
//   - 定义 UserUnitServingState，表达 Recommendation 自己 owner 的投放状态快照
//   - 当前重点字段是最近推荐时间和最近推荐 run ID
//
// 输入/输出：
//   - 输入：serving state 查询结果经 mapper 转换后的字段
//   - 输出：提供给 scorer 做近期推荐抑制，并供 repository 落库相关逻辑使用
//
// 谁调用它：
//   - infrastructure/persistence/mapper/candidate_mapper.go 负责构造
//   - application/query/candidate.go 持有它
//   - domain/service/review_scorer.go 和 new_scorer.go 读取它
//
// 它调用谁/传给谁：
//   - 不直接调用其他实现
//   - 作为候选输入的一部分传给 domain/service
package model

import (
	"time"

	"github.com/google/uuid"
)

// UserUnitServingState is the Recommendation-owned serving snapshot for one user-unit pair.
type UserUnitServingState struct {
	UserID                  uuid.UUID
	CoarseUnitID            int64
	LastRecommendedAt       *time.Time
	LastRecommendationRunID *uuid.UUID
	CreatedAt               time.Time
	UpdatedAt               time.Time
}
