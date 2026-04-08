// 作用：处理弱事件 exposure/lookup，只更新允许由弱事件改变的曝光类字段。
// 输入/输出：输入是当前 UserUnitState 和弱事件 LearningEvent；输出是只更新 SeenCount/LastSeenAt 的 state。
// 谁调用它：domain/aggregate/user_unit_reducer.go、unit test。
// 它调用谁/传给谁：调用 state_helpers.go 的 cloneOrInitState；处理结果传回 reducer。
package rule

import (
	"fmt"

	"learning-video-recommendation-system/internal/learningengine/domain/enum"
	"learning-video-recommendation-system/internal/learningengine/domain/model"
)

type WeakEventHandler interface {
	Apply(current *model.UserUnitState, event model.LearningEvent) (*model.UserUnitState, error)
}

type weakEventHandler struct{}

func NewWeakEventHandler() WeakEventHandler {
	return weakEventHandler{}
}

func (weakEventHandler) Apply(current *model.UserUnitState, event model.LearningEvent) (*model.UserUnitState, error) {
	if event.EventType != enum.EventTypeExposure && event.EventType != enum.EventTypeLookup {
		return nil, fmt.Errorf("weak event handler does not support %q", event.EventType)
	}

	next := cloneOrInitState(current, event)
	next.SeenCount++
	next.LastSeenAt = &event.OccurredAt

	return next, nil
}
