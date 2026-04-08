// 作用：根据当前状态、最近质量窗口和策略，重算 new/learning/reviewing/mastered/suspended 之间的迁移。
// 输入/输出：输入是 UserUnitState、recentQualities、LearningPolicy；输出是被原地修改后的状态或 error。
// 谁调用它：domain/aggregate/user_unit_reducer.go、unit test。
// 它调用谁/传给谁：调用本文件内 recentPassingStreak 和 recentHasFailure；迁移结果传回 reducer。
package service

import (
	"fmt"

	"learning-video-recommendation-system/internal/learningengine/domain/enum"
	"learning-video-recommendation-system/internal/learningengine/domain/model"
	"learning-video-recommendation-system/internal/learningengine/domain/policy"
)

type StatusTransitioner interface {
	Recompute(state *model.UserUnitState, recentQualities []int, schedulerPolicy policy.LearningPolicy) error
}

type statusTransitioner struct{}

func NewStatusTransitioner() StatusTransitioner {
	return statusTransitioner{}
}

func (statusTransitioner) Recompute(state *model.UserUnitState, recentQualities []int, schedulerPolicy policy.LearningPolicy) error {
	if state == nil {
		return fmt.Errorf("state is required")
	}

	switch state.Status {
	case enum.UnitStatusNew:
		if state.StrongEventCount >= 1 {
			state.Status = enum.UnitStatusLearning
		}
	case enum.UnitStatusLearning:
		if state.StrongEventCount >= 2 && recentPassingStreak(recentQualities, 2) {
			state.Status = enum.UnitStatusReviewing
		}
	case enum.UnitStatusReviewing:
		if state.IntervalDays >= schedulerPolicy.MasteredIntervalDays && state.ConsecutiveWrong == 0 && recentPassingStreak(recentQualities, 2) {
			state.Status = enum.UnitStatusMastered
		}
	case enum.UnitStatusMastered:
		if state.ConsecutiveWrong > 0 || recentHasFailure(recentQualities) {
			state.Status = enum.UnitStatusReviewing
		}
	case enum.UnitStatusSuspended:
		return nil
	default:
		return fmt.Errorf("unsupported unit status %q", state.Status)
	}

	return nil
}

func recentPassingStreak(qualities []int, want int) bool {
	if len(qualities) < want {
		return false
	}

	start := len(qualities) - want
	for _, quality := range qualities[start:] {
		if quality < 3 {
			return false
		}
	}

	return true
}

func recentHasFailure(qualities []int) bool {
	for _, quality := range qualities {
		if quality < 3 {
			return true
		}
	}

	return false
}
