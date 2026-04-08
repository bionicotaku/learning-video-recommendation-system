// 作用：声明用户学习状态仓储接口，抽象状态表的读取、upsert、批量回写和按用户删除。
// 输入/输出：输入是 userID、coarseUnitID 或 UserUnitState；输出是 state、error 或批量写结果。
// 谁调用它：record_learning_events.go、replay_user_states.go 两个 use case。
// 它调用谁/传给谁：接口本身不调用其他文件；由 infrastructure/persistence/repository/user_unit_state_repo.go 实现。
package repository

import (
	"context"

	"learning-video-recommendation-system/internal/learningengine/domain/model"

	"github.com/google/uuid"
)

type UserUnitStateRepository interface {
	GetByUserAndUnit(ctx context.Context, userID uuid.UUID, coarseUnitID int64) (*model.UserUnitState, error)
	Upsert(ctx context.Context, state *model.UserUnitState) error
	BatchUpsert(ctx context.Context, states []*model.UserUnitState) error
	DeleteByUser(ctx context.Context, userID uuid.UUID) error
}
