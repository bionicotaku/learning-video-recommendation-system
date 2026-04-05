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
