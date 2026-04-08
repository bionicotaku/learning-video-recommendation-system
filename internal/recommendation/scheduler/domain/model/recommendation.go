// 文件作用：
//   - 定义 RecommendationItem 和 RecommendationBatch，表达 scheduler 的最终领域输出
//   - 让 assembler、持久化 mapper 和调用方围绕统一结果结构协作
//
// 输入/输出：
//   - 输入：domain/service/recommendation_assembler.go 组装出的结果字段
//   - 输出：返回给 usecase 调用方，并写入 scheduler_runs / scheduler_run_items
//
// 谁调用它：
//   - domain/service/recommendation_assembler.go 负责构造
//   - application/usecase/generate_recommendations.go 返回它
//   - infrastructure/persistence/mapper/scheduler_run_mapper.go 读取它做落库映射
//   - 测试读取它做断言
//
// 它调用谁/传给谁：
//   - 不直接调用其他实现
//   - 作为最终领域结果传给 DTO 和持久化层
package model

import (
	"time"

	"learning-video-recommendation-system/internal/recommendation/scheduler/domain/enum"

	"github.com/google/uuid"
)

// RecommendationItem is a scheduler output item consumed by downstream stages.
type RecommendationItem struct {
	CoarseUnitID int64
	Kind         enum.UnitKind
	Label        string

	RecommendType enum.RecommendType
	Status        enum.UnitStatus
	Rank          int
	Score         float64
	ReasonCodes   []string

	TargetPriority  float64
	ProgressPercent float64
	MasteryScore    float64
	NextReviewAt    *time.Time
}

// RecommendationBatch is a full recommendation output batch.
type RecommendationBatch struct {
	RunID             uuid.UUID
	UserID            uuid.UUID
	GeneratedAt       time.Time
	SessionLimit      int
	DueReviewCount    int
	ReviewQuota       int
	NewQuota          int
	BacklogProtection bool
	Items             []RecommendationItem
}
