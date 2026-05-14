package policy

import (
	"math"

	"learning-video-recommendation-system/internal/learningengine/domain/enum"
	"learning-video-recommendation-system/internal/learningengine/domain/model"
)

const (
	recentWindowLimit = 5
	easeFactorFloor   = 1.3
)

func RecentWindowLimit() int {
	return recentWindowLimit
}

func ApplyProgressSuccessSchedule(state *model.UserUnitState, quality int16) {
	state.ScheduleRepetition++

	switch state.ScheduleRepetition {
	case 1:
		state.ScheduleIntervalDays = 1
	case 2:
		state.ScheduleIntervalDays = 3
	case 3:
		state.ScheduleIntervalDays = 6
	default:
		state.ScheduleIntervalDays = math.Round(state.ScheduleIntervalDays * state.ScheduleEaseFactor)
	}

	delta := 0.1 - float64(5-quality)*(0.08+float64(5-quality)*0.02)
	state.ScheduleEaseFactor = math.Max(easeFactorFloor, state.ScheduleEaseFactor+delta)
}

func ApplyProgressFailureSchedule(state *model.UserUnitState) {
	state.ScheduleRepetition = 0
	state.ScheduleIntervalDays = 1
}

func ComputeProgressPercent(state model.UserUnitState) float64 {
	if state.ProgressEventCount == 0 {
		return 0
	}
	if ComputeActiveStatus(state) == enum.StatusMastered {
		return 100
	}

	intervalComponent := minFloat(state.ScheduleIntervalDays/21, 1)
	accuracyComponent := averageBoolWindow(state.RecentProgressPasses)
	repetitionComponent := minFloat(float64(state.ScheduleRepetition)/4, 1)
	qualityComponent := averageInt16Window(state.RecentProgressQualities) / 5

	score := 100 * clamp01(
		0.45*intervalComponent+
			0.20*accuracyComponent+
			0.20*repetitionComponent+
			0.15*qualityComponent,
	)

	return round(score, 2)
}

func ComputeMasteryScore(state model.UserUnitState) float64 {
	if state.ProgressEventCount == 0 {
		return 0
	}

	intervalComponent := minFloat(state.ScheduleIntervalDays/21, 1)
	accuracyComponent := averageBoolWindow(state.RecentProgressPasses)
	stabilityComponent := minFloat(float64(state.ConsecutiveSuccessCount)/3, 1)
	repetitionComponent := minFloat(float64(state.ScheduleRepetition)/4, 1)

	failurePenalty := 0.0
	if state.LastProgressQuality != nil && *state.LastProgressQuality < 3 {
		failurePenalty = 0.20
	}

	score := clamp01(
		0.45*intervalComponent +
			0.25*accuracyComponent +
			0.15*stabilityComponent +
			0.15*repetitionComponent -
			failurePenalty,
	)

	return round(score, 4)
}

func ComputeActiveStatus(state model.UserUnitState) string {
	if state.ProgressEventCount == 0 {
		return enum.StatusNew
	}

	if state.LastProgressQuality != nil && *state.LastProgressQuality < 3 {
		return enum.StatusReviewing
	}

	if !hasTwoRecentPassingQualities(state.RecentProgressQualities) || state.ProgressEventCount < 2 || state.ScheduleIntervalDays < 3 {
		return enum.StatusLearning
	}

	if state.ScheduleIntervalDays >= 21 && noRecentFailures(state.RecentProgressPasses) && ComputeMasteryScore(state) >= 0.8 {
		return enum.StatusMastered
	}

	return enum.StatusReviewing
}

func IsSuspendedControl(state model.UserUnitState) bool {
	return state.Status == enum.StatusSuspended || state.SuspendedReason != ""
}

func AppendRecentProgressQuality(values []int16, quality int16) []int16 {
	return trimRecentWindow(append(values, quality), recentWindowLimit)
}

func AppendRecentProgressPass(values []bool, pass bool) []bool {
	return trimRecentWindow(append(values, pass), recentWindowLimit)
}

func trimRecentWindow[T any](values []T, limit int) []T {
	if len(values) <= limit {
		return values
	}
	trimmed := make([]T, limit)
	copy(trimmed, values[len(values)-limit:])
	return trimmed
}

func hasTwoRecentPassingQualities(values []int16) bool {
	if len(values) < 2 {
		return false
	}
	last := values[len(values)-2:]
	return last[0] >= 3 && last[1] >= 3
}

func noRecentFailures(values []bool) bool {
	if len(values) == 0 {
		return false
	}

	start := 0
	if len(values) > 3 {
		start = len(values) - 3
	}

	for _, value := range values[start:] {
		if !value {
			return false
		}
	}
	return true
}

func averageBoolWindow(values []bool) float64 {
	if len(values) == 0 {
		return 0
	}

	total := 0.0
	for _, value := range values {
		if value {
			total++
		}
	}
	return total / float64(len(values))
}

func averageInt16Window(values []int16) float64 {
	if len(values) == 0 {
		return 0
	}

	total := 0.0
	for _, value := range values {
		total += float64(value)
	}
	return total / float64(len(values))
}

func clamp01(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}

func minFloat(value float64, max float64) float64 {
	if value > max {
		return max
	}
	return value
}

func round(value float64, decimals int) float64 {
	scale := math.Pow10(decimals)
	return math.Round(value*scale) / scale
}
