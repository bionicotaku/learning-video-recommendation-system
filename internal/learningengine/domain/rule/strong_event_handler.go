// 作用：处理强事件的基础字段更新，负责计数器、最近时间、正确/错误 streak、LastQuality 的维护。
// 输入/输出：输入是当前 UserUnitState 和强事件 LearningEvent；输出是已更新基础统计但尚未做 SM-2/状态迁移的 state。
// 谁调用它：domain/aggregate/user_unit_reducer.go、unit test。
// 它调用谁/传给谁：调用 state_helpers.go 的 cloneOrInitState；处理结果传回 reducer，后续再交给 domain/service。
package rule

import (
	"fmt"

	"learning-video-recommendation-system/internal/learningengine/domain/enum"
	"learning-video-recommendation-system/internal/learningengine/domain/model"
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
