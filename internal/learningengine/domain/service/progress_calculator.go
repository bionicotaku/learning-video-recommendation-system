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
