package service

import (
	"fmt"
	"math"
	"time"

	"learning-video-recommendation-system/internal/learningengine/domain/model"
	"learning-video-recommendation-system/internal/learningengine/domain/policy"
)

type SM2Updater interface {
	Apply(state *model.UserUnitState, quality int, occurredAt time.Time, schedulerPolicy policy.SchedulerPolicy) error
}

type sm2Updater struct{}

func NewSM2Updater() SM2Updater {
	return sm2Updater{}
}

func (sm2Updater) Apply(state *model.UserUnitState, quality int, occurredAt time.Time, schedulerPolicy policy.SchedulerPolicy) error {
	if state == nil {
		return fmt.Errorf("state is required")
	}
	if quality < 0 || quality > 5 {
		return fmt.Errorf("quality must be between 0 and 5")
	}

	if quality >= 3 {
		state.Repetition++

		switch state.Repetition {
		case 1:
			state.IntervalDays = schedulerPolicy.InitialIntervals[0]
		case 2:
			state.IntervalDays = schedulerPolicy.InitialIntervals[1]
		case 3:
			state.IntervalDays = schedulerPolicy.InitialIntervals[2]
		default:
			state.IntervalDays = math.Round(state.IntervalDays * state.EaseFactor)
		}

		state.EaseFactor = maxEaseFactor(updatedEaseFactor(state.EaseFactor, quality), schedulerPolicy.MinEaseFactor)
		state.NextReviewAt = timePtr(occurredAt.Add(durationFromDays(state.IntervalDays)))
		return nil
	}

	state.Repetition = 0
	state.IntervalDays = 1
	state.NextReviewAt = timePtr(occurredAt.Add(24 * time.Hour))
	return nil
}

func updatedEaseFactor(current float64, quality int) float64 {
	return current + (0.1 - float64(5-quality)*(0.08+float64(5-quality)*0.02))
}

func maxEaseFactor(current, min float64) float64 {
	if current < min {
		return min
	}

	return current
}

func durationFromDays(days float64) time.Duration {
	return time.Duration(days * float64(24*time.Hour))
}

func timePtr(value time.Time) *time.Time {
	return &value
}
