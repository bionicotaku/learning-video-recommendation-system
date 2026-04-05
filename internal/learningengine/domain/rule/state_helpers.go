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
