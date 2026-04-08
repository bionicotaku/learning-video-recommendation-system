// 作用：根据进度、最近正确率和稳定性计算 0..1 的 MasteryScore。
// 输入/输出：输入是 UserUnitState、recentAccuracy、LearningPolicy；输出是 float64 mastery score。
// 谁调用它：domain/aggregate/user_unit_reducer.go、unit test。
// 它调用谁/传给谁：不主动调用其他文件；结果会传回 reducer 并写入 UserUnitState.MasteryScore。
package service

import (
	"math"

	"learning-video-recommendation-system/internal/learningengine/domain/model"
	"learning-video-recommendation-system/internal/learningengine/domain/policy"
)

type MasteryScoreCalculator interface {
	Compute(state *model.UserUnitState, recentAccuracy float64, schedulerPolicy policy.LearningPolicy) float64
}

type masteryScoreCalculator struct{}

func NewMasteryScoreCalculator() MasteryScoreCalculator {
	return masteryScoreCalculator{}
}

func (masteryScoreCalculator) Compute(state *model.UserUnitState, recentAccuracy float64, schedulerPolicy policy.LearningPolicy) float64 {
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
