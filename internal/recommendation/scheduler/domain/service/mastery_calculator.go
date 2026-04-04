package service

import (
	"math"

	"learning-video-recommendation-system/internal/recommendation/scheduler/domain/model"
	"learning-video-recommendation-system/internal/recommendation/scheduler/domain/policy"
)

type MasteryScoreCalculator interface {
	Compute(state *model.UserUnitState, recentAccuracy float64, schedulerPolicy policy.SchedulerPolicy) float64
}

type masteryScoreCalculator struct{}

func NewMasteryScoreCalculator() MasteryScoreCalculator {
	return masteryScoreCalculator{}
}

func (masteryScoreCalculator) Compute(state *model.UserUnitState, recentAccuracy float64, schedulerPolicy policy.SchedulerPolicy) float64 {
	if state == nil {
		return 0
	}

	if recentAccuracy < 0 {
		recentAccuracy = 0
	}
	if recentAccuracy > 1 {
		recentAccuracy = 1
	}

	stabilityScore := math.Min(1, state.IntervalDays/schedulerPolicy.MasteredIntervalDays)
	masteryScore := 0.45*(state.ProgressPercent/100) + 0.35*recentAccuracy + 0.20*stabilityScore
	if masteryScore < 0 {
		return 0
	}
	if masteryScore > 1 {
		return 1
	}

	return masteryScore
}
