// 作用：提供状态克隆和初始化辅助函数，保证 reducer/rule 在修改状态时不会污染原对象。
// 输入/输出：输入是当前 state 和 event；输出是已初始化或已深拷贝的 next state。
// 谁调用它：weak_event_handler.go、strong_event_handler.go。
// 它调用谁/传给谁：不主动调用其他文件；返回值会传回各个 handler，随后再交给 reducer。
package rule

import (
	"learning-video-recommendation-system/internal/learningengine/domain/enum"
	"learning-video-recommendation-system/internal/learningengine/domain/model"
)

func cloneOrInitState(current *model.UserUnitState, event model.LearningEvent) *model.UserUnitState {
	if current == nil {
		return &model.UserUnitState{
			UserID:                  event.UserID,
			CoarseUnitID:            event.CoarseUnitID,
			IsTarget:                true,
			Status:                  enum.UnitStatusNew,
			EaseFactor:              2.5,
			RecentQualityWindow:     []int{},
			RecentCorrectnessWindow: []bool{},
		}
	}

	cloned := *current
	cloned.RecentQualityWindow = append([]int(nil), current.RecentQualityWindow...)
	cloned.RecentCorrectnessWindow = append([]bool(nil), current.RecentCorrectnessWindow...)
	return &cloned
}
