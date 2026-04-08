// 作用：根据当前 IntervalDays 和 mastered 阈值计算 ProgressPercent。
// 输入/输出：输入是 intervalDays 和 LearningPolicy；输出是 0..100 的 float64。
// 谁调用它：domain/aggregate/user_unit_reducer.go、unit test。
// 它调用谁/传给谁：不主动调用其他文件；结果会传回 reducer 并写入 UserUnitState.ProgressPercent。
package service

import (
	"math"

	"learning-video-recommendation-system/internal/learningengine/domain/policy"
)

type ProgressCalculator interface {
	Compute(intervalDays float64, schedulerPolicy policy.LearningPolicy) float64
}

type progressCalculator struct{}

func NewProgressCalculator() ProgressCalculator {
	return progressCalculator{}
}

func (progressCalculator) Compute(intervalDays float64, schedulerPolicy policy.LearningPolicy) float64 {
	if intervalDays <= 0 {
		return 0
	}

	target := schedulerPolicy.MasteredIntervalDays + 1
	progress := math.Log(intervalDays+1) / math.Log(target) * 100
	if progress > 100 {
		return 100
	}
	if progress < 0 {
		return 0
	}

	return progress
}
