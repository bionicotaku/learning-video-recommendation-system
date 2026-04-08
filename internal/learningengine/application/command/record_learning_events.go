// 作用：定义“记录学习事件”用例的输入结构，包括单条事件输入和批量命令对象。
// 输入/输出：输入是用户、事件类型、correct/quality/occurredAt 等字段；输出是 RecordLearningEventsCommand 和 LearningEventInput 两个结构定义。
// 谁调用它：上层业务调用方、集成测试、fixture/helpers.go。
// 它调用谁/传给谁：不主动调用其他文件；这些结构会传给 application/usecase/record_learning_events.go，再被转换成 domain/model.LearningEvent。
package command

import (
	"time"

	"learning-video-recommendation-system/internal/learningengine/domain/enum"

	"github.com/google/uuid"
)

// LearningEventInput is the application-layer input used to record a learning event.
type LearningEventInput struct {
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
}

// RecordLearningEventsCommand records normalized learning events for one user.
type RecordLearningEventsCommand struct {
	UserID         uuid.UUID
	Events         []LearningEventInput
	IdempotencyKey string
}
