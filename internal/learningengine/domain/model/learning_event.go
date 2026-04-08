// 作用：定义标准化学习事件领域模型，是在线写入和 replay 共同使用的统一事件对象。
// 输入/输出：输入来自 application/usecase 或 mapper；输出供 reducer、repository、测试消费。
// 谁调用它：record_learning_events.go、unit_learning_event_mapper.go、replay 相关代码、测试。
// 它调用谁/传给谁：不主动调用其他文件；实例会传给 reducer 和事件 repository。
package model

import (
	"time"

	"learning-video-recommendation-system/internal/learningengine/domain/enum"

	"github.com/google/uuid"
)

// LearningEvent is a normalized learning activity record.
type LearningEvent struct {
	EventID        int64
	UserID         uuid.UUID
	CoarseUnitID   int64
	VideoID        *uuid.UUID
	EventType      enum.EventType
	SourceType     string
	SourceRefID    string
	IsCorrect      *bool
	Quality        *int
	ResponseTimeMs *int
	Metadata       map[string]any
	OccurredAt     time.Time
	CreatedAt      time.Time
}
