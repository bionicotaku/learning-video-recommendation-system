// 文件作用：
//   - 定义 LearningStateSnapshot，表示 Recommendation 读取到的学习状态快照
//   - 承接 learning.user_unit_states 中 scheduler 需要的字段集合
//
// 输入/输出：
//   - 输入：候选 SQL 查询结果经 mapper 转换后的状态字段
//   - 输出：提供给 scorer、priority extractor、assembler 和测试
//
// 谁调用它：
//   - infrastructure/persistence/mapper/candidate_mapper.go 负责构造
//   - application/query/candidate.go 持有它
//   - domain/service/*.go 负责读取它的字段
//
// 它调用谁/传给谁：
//   - 不直接调用其他实现
//   - 作为跨模块输入模型传给 scheduler 的领域服务
package model

import (
	"time"

	"learning-video-recommendation-system/internal/recommendation/scheduler/domain/enum"

	"github.com/google/uuid"
)

// LearningStateSnapshot is the Learning engine snapshot consumed by Recommendation.
type LearningStateSnapshot struct {
	UserID       uuid.UUID
	CoarseUnitID int64

	IsTarget          bool
	TargetSource      string
	TargetSourceRefID string
	TargetPriority    float64

	Status enum.UnitStatus

	ProgressPercent float64
	MasteryScore    float64

	FirstSeenAt    *time.Time
	LastSeenAt     *time.Time
	LastReviewedAt *time.Time

	SeenCount          int
	StrongEventCount   int
	ReviewCount        int
	CorrectCount       int
	WrongCount         int
	ConsecutiveCorrect int
	ConsecutiveWrong   int

	LastQuality *int

	RecentQualityWindow     []int
	RecentCorrectnessWindow []bool

	Repetition   int
	IntervalDays float64
	EaseFactor   float64
	NextReviewAt *time.Time

	SuspendedReason string

	CreatedAt time.Time
	UpdatedAt time.Time
}
