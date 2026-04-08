// 作用：实现 replay 场景下的用户状态重建器，按事件顺序为每个 coarse unit 重算最终状态。
// 输入/输出：输入是按时间排序的 []LearningEvent；输出是 []*UserUnitState 或 error。
// 谁调用它：application/usecase/replay_user_states.go。
// 它调用谁/传给谁：调用 domain/aggregate/user_unit_reducer.go；重建结果再传回 replay use case 并交给 state repository 批量写回。
package service

import (
	"learning-video-recommendation-system/internal/learningengine/domain/aggregate"
	"learning-video-recommendation-system/internal/learningengine/domain/model"
	"learning-video-recommendation-system/internal/learningengine/domain/policy"
)

type UserStateRebuilder interface {
	Rebuild(events []model.LearningEvent) ([]*model.UserUnitState, error)
}

type userStateRebuilder struct {
	reducer aggregate.UserUnitReducer
	policy  policy.LearningPolicy
}

func NewUserStateRebuilder(
	reducer aggregate.UserUnitReducer,
	schedulerPolicy policy.LearningPolicy,
) UserStateRebuilder {
	if schedulerPolicy.MasteredIntervalDays == 0 {
		schedulerPolicy = policy.DefaultLearningPolicy()
	}

	return userStateRebuilder{
		reducer: reducer,
		policy:  schedulerPolicy,
	}
}

func (r userStateRebuilder) Rebuild(events []model.LearningEvent) ([]*model.UserUnitState, error) {
	states := make(map[int64]*model.UserUnitState, len(events))
	for _, event := range events {
		current := states[event.CoarseUnitID]
		next, err := r.reducer.Reduce(current, event, r.policy)
		if err != nil {
			return nil, err
		}
		states[event.CoarseUnitID] = next
	}

	items := make([]*model.UserUnitState, 0, len(states))
	for _, state := range states {
		items = append(items, state)
	}

	return items, nil
}
