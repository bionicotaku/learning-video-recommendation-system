package service

import (
	"fmt"

	"learning-video-recommendation-system/internal/recommendation/scheduler/domain/enum"
	"learning-video-recommendation-system/internal/recommendation/scheduler/domain/model"
	"learning-video-recommendation-system/internal/recommendation/scheduler/domain/policy"
)

type StatusTransitioner interface {
	Recompute(state *model.UserUnitState, recentQualities []int, schedulerPolicy policy.SchedulerPolicy) error
}

type statusTransitioner struct{}

func NewStatusTransitioner() StatusTransitioner {
	return statusTransitioner{}
}

func (statusTransitioner) Recompute(state *model.UserUnitState, recentQualities []int, schedulerPolicy policy.SchedulerPolicy) error {
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
