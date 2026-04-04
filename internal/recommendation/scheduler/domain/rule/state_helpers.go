package rule

import (
	"learning-video-recommendation-system/internal/recommendation/scheduler/domain/enum"
	"learning-video-recommendation-system/internal/recommendation/scheduler/domain/model"
)

func cloneOrInitState(current *model.UserUnitState, event model.LearningEvent) *model.UserUnitState {
	if current == nil {
		return &model.UserUnitState{
			UserID:       event.UserID,
			CoarseUnitID: event.CoarseUnitID,
			IsTarget:     true,
			Status:       enum.UnitStatusNew,
			EaseFactor:   2.5,
		}
	}

	cloned := *current
	return &cloned
}
