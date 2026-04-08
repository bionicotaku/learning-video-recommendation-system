// 作用：声明事件真相层仓储接口，抽象事件追加写入和按用户顺序回放读取。
// 输入/输出：输入是 []LearningEvent 或 userID；输出是 error 或按时序排序的 []LearningEvent。
// 谁调用它：record_learning_events.go、replay_user_states.go 两个 use case。
// 它调用谁/传给谁：接口本身不调用其他文件；由 infrastructure/persistence/repository/unit_learning_event_repo.go 实现。
package repository

import (
	"context"

	"learning-video-recommendation-system/internal/learningengine/domain/model"

	"github.com/google/uuid"
)

type UnitLearningEventRepository interface {
	Append(ctx context.Context, events []model.LearningEvent) error
	ListByUserOrdered(ctx context.Context, userID uuid.UUID) ([]model.LearningEvent, error)
}
