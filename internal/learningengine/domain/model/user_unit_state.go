// 作用：定义用户与 coarse unit 关系的当前学习状态模型，是 Recommendation 读取的稳定输入之一。
// 输入/输出：输入来自 reducer 或 mapper；输出供 repository、calculator、use case、测试消费。
// 谁调用它：domain/rule、domain/service、domain/aggregate、state mapper、state repository、测试。
// 它调用谁/传给谁：不主动调用其他文件；实例会在 reducer、repository 和上层读取链路之间传递。
package model

import (
	"time"

	"learning-video-recommendation-system/internal/learningengine/domain/enum"

	"github.com/google/uuid"
)

// UserUnitState is the current learning-state snapshot for a user-unit relation.
type UserUnitState struct {
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
