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
