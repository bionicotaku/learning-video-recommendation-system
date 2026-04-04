package rule

import (
	"fmt"

	"learning-video-recommendation-system/internal/recommendation/scheduler/domain/enum"
	"learning-video-recommendation-system/internal/recommendation/scheduler/domain/model"
)

type StrongEventHandler interface {
	Apply(current *model.UserUnitState, event model.LearningEvent) (*model.UserUnitState, error)
}

type strongEventHandler struct{}

func NewStrongEventHandler() StrongEventHandler {
	return strongEventHandler{}
}

func (strongEventHandler) Apply(current *model.UserUnitState, event model.LearningEvent) (*model.UserUnitState, error) {
	switch event.EventType {
	case enum.EventTypeNewLearn, enum.EventTypeReview, enum.EventTypeQuiz:
	default:
		return nil, fmt.Errorf("strong event handler does not support %q", event.EventType)
	}

	next := cloneOrInitState(current, event)
	next.SeenCount++
	next.StrongEventCount++
	next.LastSeenAt = &event.OccurredAt

	if event.EventType == enum.EventTypeReview || event.EventType == enum.EventTypeQuiz {
		next.ReviewCount++
		next.LastReviewedAt = &event.OccurredAt
	}

	if event.IsCorrect != nil {
		if *event.IsCorrect {
			next.CorrectCount++
			next.ConsecutiveCorrect++
			next.ConsecutiveWrong = 0
		} else {
			next.WrongCount++
			next.ConsecutiveWrong++
			next.ConsecutiveCorrect = 0
		}
	}

	if event.Quality != nil {
		quality := *event.Quality
		next.LastQuality = &quality
	}

	return next, nil
}
